// Package main implements a simple HTTP server for managing products.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/abgdnv/gocommerce/product_service/internal/app"
	"github.com/abgdnv/gocommerce/product_service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const serviceName = "product"

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Printf("application run failed: %v", err)
		os.Exit(1)
	}
	log.Println("application stopped gracefully")
}

// run initializes the application, sets up the database connection, and starts the HTTP, gRPC and pprof servers.
func run(ctx context.Context) error {
	cfg, cfgErr := configloader.Load[*config.Config](serviceName)
	if cfgErr != nil {
		return fmt.Errorf("failed to load configuration: %w", cfgErr)
	}
	log.Printf("Configuration loaded: %v", cfg)

	logger := bootstrap.NewLogger(cfg.Log.Level)
	slog.SetDefault(logger)

	dbPool, err := bootstrap.NewDbPool(ctx, cfg.Database.URL, cfg.Database.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}
	defer dbPool.Close()
	logger.Info("Successfully connected to the database!")

	httpServer, pprofServer, grpcServer := setupServers(dbPool, logger, cfg)

	g, gCtx := errgroup.WithContext(ctx)

	// Start the HTTP server
	g.Go(func() error {
		logger.Info("HTTP server listening", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server failed: %w", err)
		}
		return nil
	})
	// gracefully shutdown HTTP server on context cancellation
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	})

	// Start the gRPC server
	g.Go(func() error {
		grpcAddr := ":" + cfg.GRPC.Port
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on gRPC port: %w", err)
		}
		logger.Info("gRPC server listening", slog.String("addr", grpcAddr))
		return grpcServer.Serve(lis)
	})
	// gracefully shutdown gRPC server on context cancellation
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down gRPC server...")
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
			logger.Info("gRPC server stopped gracefully.")
			return nil
		case <-time.After(cfg.Shutdown.Timeout):
			logger.Warn("gRPC server graceful stop timed out. Forcing stop.")
			grpcServer.Stop()
			return fmt.Errorf("grpc server graceful stop timed out")
		}
	})

	// Start the pprof server if enabled
	if cfg.PProf.Enabled {
		g.Go(func() error {
			logger.Info("Pprof server listening", slog.String("addr", pprofServer.Addr))
			if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("pprof server failed: %w", err)
			}
			return nil
		})
		// gracefully shutdown pprof server on context cancellation
		g.Go(func() error {
			<-gCtx.Done()
			logger.Info("Shutting down pprof server...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
			defer cancel()
			return pprofServer.Shutdown(shutdownCtx)
		})
	}
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("errgroup encountered an error: %w", err)
	}
	return nil
}

// setupServers initializes the HTTP, pprof, and gRPC servers with the provided database pool, logger, and configuration.
func setupServers(dbPool *pgxpool.Pool, logger *slog.Logger, cfg *config.Config) (*http.Server, *http.Server, *grpc.Server) {
	deps := app.SetupDependencies(dbPool, logger)
	httpServer := app.SetupHttpServer(deps, cfg)
	grpcServer := app.SetupGrpcServer(deps, cfg.GRPC.ReflectionEnabled)
	pprofServer := &http.Server{
		Addr: cfg.PProf.Addr,
	}
	return httpServer, pprofServer, grpcServer
}
