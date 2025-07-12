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

	"github.com/abgdnv/gocommerce/order_service/internal/app"
	"github.com/abgdnv/gocommerce/order_service/internal/config"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/configloader"
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

	logger := newLogger(cfg.Log.Level)
	slog.SetDefault(logger)

	dbPool, err := newDbPool(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}
	defer dbPool.Close()
	logger.Info("Successfully connected to the database!")

	// Create a gRPC client connection to the Product service
	conn, err := grpc.NewClient(cfg.Services.ProductGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to create gRPC client connection: %w", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			logger.Error("Failed to close gRPC client connection", slog.String("error", err.Error()))
		} else {
			logger.Info("gRPC client connection closed successfully")
		}
	}(conn)
	productClient := pb.NewProductServiceClient(conn)

	// Set up HTTP and pprof servers
	httpServer, pprofServer := setupServers(dbPool, productClient, logger, cfg)

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
func setupServers(dbPool *pgxpool.Pool, productClient pb.ProductServiceClient, logger *slog.Logger, cfg *config.Config) (*http.Server, *http.Server) {
	deps := app.SetupDependencies(dbPool, productClient, logger)
	httpServer := app.SetupHttpServer(deps, cfg)
	pprofServer := &http.Server{
		Addr: cfg.PProf.Addr,
	}
	return httpServer, pprofServer
}

// newLogger creates a new slog.Logger instance with the specified log level.
func newLogger(level string) *slog.Logger {
	logLevel := toLevel(level)
	loggerOpts := &slog.HandlerOptions{
		AddSource: logLevel == slog.LevelDebug,
		Level:     logLevel,
	}
	logHandler := slog.NewJSONHandler(os.Stdout, loggerOpts)
	logger := slog.New(logHandler)
	return logger
}

// newDbPool creates a new database connection pool with the provided context and configuration,
func newDbPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	// Create context with timeout for database connection
	poolCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dbPool, errPool := pgxpool.New(poolCtx, url)
	if errPool != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", errPool)
	}
	// Ping the database to ensure the connection is established (fail early if not)
	if err := dbPool.Ping(poolCtx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return dbPool, nil
}

// toLevel converts a string representation of a log level to slog.Level.
func toLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
