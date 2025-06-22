// Package main implements a simple HTTP server for managing products.
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
	"time"

	"github.com/abgdnv/gocommerce/internal/config"
	"github.com/abgdnv/gocommerce/internal/platform/web"
	"github.com/abgdnv/gocommerce/internal/product/handler"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/abgdnv/gocommerce/internal/product/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		log.Fatalf("Error loading configuration: %v", cfgErr)
	}
	log.Printf("Configuration loaded: %v", cfg)

	// Set up structured logging
	logLevel := toLevel(cfg.Log.Level)
	loggerOpts := &slog.HandlerOptions{
		AddSource: logLevel == slog.LevelDebug,
		Level:     logLevel,
	}
	logHandler := slog.NewJSONHandler(os.Stdout, loggerOpts)
	logger := slog.New(logHandler)
	logger.Info("Product service starting...", "config_log_level", cfg.Log.Level, "actual_slog_level", logLevel.String())

	// Create context with timeout for database connection
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	dbPool, errPool := pgxpool.New(poolCtx, cfg.Database.URL)
	if errPool != nil {
		logger.Error("Unable to create connection pool", "error", errPool)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(poolCtx); err != nil {
		logger.Error("Unable to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Successfully connected to the database!")

	pService := service.NewService(store.NewPgStore(dbPool))

	pApi := handler.NewAPI(pService, logger)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(web.RequestIDInjector)
	mux.Use(web.StructuredLogger(logger))
	mux.Use(middleware.Timeout(cfg.HTTPServer.Timeout.Read + 2*time.Second))
	mux.Use(web.Recoverer(logger))

	mux.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", pApi.FindAll)
		r.Post("/", pApi.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", pApi.FindByID)
			r.Delete("/", pApi.DeleteByID)
			r.Put("/", pApi.Update)
			r.Put("/stock", pApi.UpdateStock)
		})
	})

	mux.Get("/healthz", pApi.HealthCheck)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler:           mux,
		ReadTimeout:       cfg.HTTPServer.Timeout.Read,
		WriteTimeout:      cfg.HTTPServer.Timeout.Write,
		IdleTimeout:       cfg.HTTPServer.Timeout.Idle,
		ReadHeaderTimeout: cfg.HTTPServer.Timeout.ReadHeader,
		MaxHeaderBytes:    cfg.HTTPServer.MaxHeaderBytes,
	}

	// Graceful shutdown handling
	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		logger.Info("Server is shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP server Shutdown", "error", err)
		}
		close(idleConnectionsClosed)
	}()

	logger.Info("Starting server", "address", server.Addr)

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
	// Wait for the server to shut down gracefully
	<-idleConnectionsClosed
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
