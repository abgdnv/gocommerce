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

	"github.com/Nerzal/gocloak/v13"
	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/abgdnv/gocommerce/pkg/telemetry"
	"github.com/abgdnv/gocommerce/user_service/internal/app"
	"github.com/abgdnv/gocommerce/user_service/internal/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const serviceName = "user"

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

	// create tracer provider
	tracerProvider, err := telemetry.NewTracerProvider(ctx, serviceName, cfg.Telemetry)
	if err != nil {
		logger.Error("error creating tracer provider", slog.Any("error", err))
		return err
	}

	pprofServer, grpcServer, grpcHealth, err := setupServers(ctx, logger, cfg)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)

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
		grpcHealth.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			grpcHealth.Shutdown()
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
	// gracefully shutdown tracer provider
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down tracer provider")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
		defer cancel()
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %v", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("errgroup encountered an error: %w", err)
	}
	return nil
}

// setupServers initializes the HTTP, pprof, and gRPC servers with the provided database pool, logger, and configuration.
func setupServers(ctx context.Context, logger *slog.Logger, cfg *config.Config) (*http.Server, *grpc.Server, *health.Server, error) {
	client := gocloak.NewClient(cfg.IdP.URL)
	//fail-fast
	_, err := client.LoginClient(ctx, cfg.IdP.ClientID, cfg.IdP.Secret, cfg.IdP.Realm)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("login failed: %w", err)
	}
	deps := app.SetupDependencies(logger, client, cfg.IdP.ClientID, cfg.IdP.Secret, cfg.IdP.Realm)
	grpcServer := app.SetupGrpcServer(deps, cfg.GRPC.ReflectionEnabled)
	pprofServer := &http.Server{
		Addr: cfg.PProf.Addr,
	}

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	return pprofServer, grpcServer, healthServer, nil
}
