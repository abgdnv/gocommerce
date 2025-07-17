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

	"github.com/abgdnv/gocommerce/notification_service/internal/config"
	"github.com/abgdnv/gocommerce/notification_service/internal/subscriber"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/abgdnv/gocommerce/pkg/nats"
	"golang.org/x/sync/errgroup"
)

const serviceName = "notification"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Printf("application run failed: %v", err)
		os.Exit(1)
	}
	log.Println("application stopped gracefully")
}

// run initializes the application, starts the NATS subscriber, and optionally starts the pprof server if enabled.
func run(ctx context.Context) error {
	cfg, cfgErr := configloader.Load[*config.Config](serviceName)
	if cfgErr != nil {
		return fmt.Errorf("failed to load configuration: %w", cfgErr)
	}
	log.Printf("Configuration loaded: %v", cfg)

	logger := newLogger(cfg.Log.Level)
	slog.SetDefault(logger)

	natsConn, err := nats.NewClient(cfg.Nats.Url, cfg.Nats.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create NATS connection: %w", err)
	}
	js, err := nats.NewJetStreamContext(natsConn)
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("NATS subscriber started")
		err := subscriber.Start(gCtx, js, cfg.Subscriber, logger)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("subscriber failed", "error", err)
			return err
		}
		logger.Info("subscriber stopped gracefully.")
		return nil
	})

	// Start the pprof server if enabled
	if cfg.PProf.Enabled {
		pprofServer := &http.Server{
			Addr: cfg.PProf.Addr,
		}
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
			logger.Info("Shutting down pprof server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
			defer cancel()
			return pprofServer.Shutdown(shutdownCtx)
		})
	}

	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("errgroup encountered an error: %w", err)
		}
	}

	return nil
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
