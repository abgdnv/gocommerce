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

	"github.com/abgdnv/gocommerce/notification_service/internal/config"
	"github.com/abgdnv/gocommerce/notification_service/internal/subscriber"
	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"github.com/abgdnv/gocommerce/pkg/nats"
	"github.com/abgdnv/gocommerce/pkg/telemetry"
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

	logger := bootstrap.NewLogger(cfg.Log.Level)
	slog.SetDefault(logger)

	// create tracer provider
	tracerProvider, err := telemetry.NewTracerProvider(ctx, serviceName, cfg.Telemetry)
	if err != nil {
		logger.Error("error creating tracer provider", slog.Any("error", err))
		return err
	}

	natsConn, err := nats.NewClient(cfg.Nats.Url, cfg.Nats.Timeout)
	if err != nil {
		return fmt.Errorf("failed to create NATS connection: %w", err)
	}
	js, err := nats.NewJetStreamContext(natsConn)
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// create readiness probe file and remove it on shutdown
	if err := os.WriteFile(cfg.ProbesConfig.ReadinessFileName, []byte("ok"), 0644); err != nil {
		slog.Error("failed to create readiness probe file", "error", err)
	}
	defer func() {
		err := os.Remove(cfg.ProbesConfig.ReadinessFileName)
		if err != nil {
			logger.Error("Can't delete file", "file", cfg.ProbesConfig.ReadinessFileName)
		}
	}()

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

	// Create liveness probe file and update it periodically
	g.Go(func() error {
		if err := os.WriteFile(cfg.ProbesConfig.LivenessFileName, []byte("ok"), 0644); err != nil {
			return fmt.Errorf("failed to create liveness probe file: %w", err)
		}
		ticker := time.NewTicker(cfg.ProbesConfig.LivenessInterval)
		defer ticker.Stop()
		for {
			select {
			case <-gCtx.Done():
				_ = os.Remove(cfg.ProbesConfig.LivenessFileName)
				return nil
			case <-ticker.C:
				if err := os.Chtimes(cfg.ProbesConfig.LivenessFileName, time.Now(), time.Now()); err != nil {
					slog.Error("Failed to update liveness probe file", "error", err)
				}
			}
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

	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("errgroup encountered an error: %w", err)
		}
	}

	return nil
}
