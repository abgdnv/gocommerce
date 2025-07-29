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
	"github.com/abgdnv/gocommerce/api_gateway/internal/transport/rest"
	"github.com/abgdnv/gocommerce/pkg/auth"
	"github.com/abgdnv/gocommerce/pkg/bootstrap"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
	"golang.org/x/sync/errgroup"
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

	g, gCtx := errgroup.WithContext(ctx)

	// Start the API Gateway
	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	verifier, err := auth.NewJWTVerifier(startupCtx, cfg.IdP)
	if err != nil {
		return fmt.Errorf("failed to create JWT verifier: %w", err)
	}

	gw := rest.NewGW(cfg.HTTPServer, cfg.Services, logger)
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

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("errgroup encountered an error: %w", err)
	}
	return nil
}
