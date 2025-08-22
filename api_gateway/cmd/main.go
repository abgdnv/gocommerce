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

	"github.com/abgdnv/gocommerce/api_gateway/internal/config"
	"github.com/abgdnv/gocommerce/api_gateway/internal/service"
	"github.com/abgdnv/gocommerce/api_gateway/internal/transport/rest"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
	"github.com/abgdnv/gocommerce/pkg/auth"
	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/client/grpc/interceptors"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/abgdnv/gocommerce/pkg/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const serviceName = "gw"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Printf("application run failed: %v", err)
		os.Exit(1)
	}
	log.Println("application stopped gracefully")
}

// run initializes the application, starts the HTTP and pprof servers.
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

	// Create a gRPC client connection to the User service
	grpcClient, err := grpc.NewClient(
		cfg.Services.User.Grpc.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			interceptors.UnaryClientTimeoutInterceptor(cfg.Services.User.Grpc.Timeout),
		),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client connection: %w", err)
	}
	userClient := pb.NewUserServiceClient(grpcClient)
	healthClient := healthpb.NewHealthClient(grpcClient)
	userService := service.NewUserService(userClient, healthClient)

	g, gCtx := errgroup.WithContext(ctx)

	// Start the API Gateway
	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	verifier, err := auth.NewJWTVerifier(startupCtx, cfg.IdP)
	if err != nil {
		return fmt.Errorf("failed to create JWT verifier: %w", err)
	}

	gw := rest.NewGW(cfg.HTTPServer, userService, cfg.Services, cfg.IdP.JwksURL, logger)
	httpServer, err := gw.SetupHTTPServer(verifier)
	if err != nil {
		return err
	}
	g.Go(func() error {
		logger.Info("API Gateway started", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("API gateway failed: %w", err)
		}
		return nil
	})
	// gracefully shutdown the API Gateway
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("Shutting down API Gateway...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	})

	// Start the pprof server if enabled
	pprofServer := &http.Server{
		Addr: cfg.PProf.Addr,
	}
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
