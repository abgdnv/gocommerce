package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/client/grpc/interceptors"
	"github.com/abgdnv/gocommerce/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/abgdnv/gocommerce/order_service/internal/app"
	"github.com/abgdnv/gocommerce/order_service/internal/config"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const serviceName = "order"

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

	// Create a gRPC client connection to the Product service
	grpcClient, err := grpc.NewClient(
		cfg.Services.Product.Grpc.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			interceptors.UnaryClientTimeoutInterceptor(cfg.Services.Product.Grpc.Timeout),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client connection: %w", err)
	}
	productClient := pb.NewProductServiceClient(grpcClient)

	natsConn, err := nats.NewClient(cfg.Nats.Url, cfg.Nats.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create NATS connection: %w", err)
	}
	js, err := nats.NewJetStreamContext(natsConn)
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Set up HTTP and pprof servers
	httpServer, pprofServer := setupServers(dbPool, productClient, js, logger, cfg)

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

	// gracefully shutdown grpc client
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down grpc client")

		closeDone := make(chan struct{})
		go func() {
			err := grpcClient.Close()
			if err != nil {
				logger.Error("failed to close gRPC client connection")
			}
			close(closeDone)
		}()

		select {
		case <-closeDone:
			logger.Info("gRPC client connection closed successfully")
			return nil
		case <-time.After(cfg.Shutdown.Timeout):
			return fmt.Errorf("failed to close gRPC client connection")
		}
	})

	// gracefully shutdown NATS connection on context cancellation
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Draining NATS connection...")

		drainDone := make(chan struct{})
		go func() {
			if err := natsConn.Drain(); err != nil {
				logger.Error("failed to drain nats connection", "error", err)
			}
			close(drainDone)
		}()

		select {
		case <-drainDone:
			logger.Info("NATS connection drained successfully.")
			return nil
		case <-time.After(cfg.Shutdown.Timeout):
			return fmt.Errorf("nats drain timeout")
		}
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("errgroup encountered an error: %w", err)
	}
	return nil
}

// setupServers initializes the HTTP, pprof, and gRPC servers with the provided database pool, logger, and configuration.
func setupServers(dbPool *pgxpool.Pool, productClient pb.ProductServiceClient, js jetstream.JetStream, logger *slog.Logger, cfg *config.Config) (*http.Server, *http.Server) {
	deps := app.SetupDependencies(dbPool, productClient, js, logger)
	httpServer := app.SetupHttpServer(deps, cfg)
	pprofServer := &http.Server{
		Addr: cfg.PProf.Addr,
	}
	return httpServer, pprofServer
}
