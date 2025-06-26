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

	_ "net/http/pprof"

	"github.com/abgdnv/gocommerce/internal/config"
	"github.com/abgdnv/gocommerce/internal/product/app"
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
	logLevel, logger := newLogger(cfg)
	logger.Info("Product service starting...", "config_log_level", cfg.Log.Level, "actual_slog_level", logLevel.String())

	// Set up pprof server for profiling
	if cfg.PProf.Enabled {
		go pprofServer(cfg.PProf.Addr, logger)
	} else {
		logger.Info("Pprof server is disabled")
	}

	// Create context with timeout for database connection
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	// Create a new database connection pool
	dbPool := newDbPool(poolCtx, cfg, logger)
	defer dbPool.Close()
	// Ping the database to ensure the connection is established (fail early if not)
	if err := dbPool.Ping(poolCtx); err != nil {
		logger.Error("Unable to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Successfully connected to the database!")

	mux, err := app.SetupApplication(cfg, dbPool, logger)
	if err != nil {
		logger.Error("Error setting up application", "error", err)
		os.Exit(1)
	}

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

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
	// Wait for the server to shut down gracefully
	<-idleConnectionsClosed
}

func newLogger(cfg *config.Config) (slog.Level, *slog.Logger) {
	logLevel := toLevel(cfg.Log.Level)
	loggerOpts := &slog.HandlerOptions{
		AddSource: logLevel == slog.LevelDebug,
		Level:     logLevel,
	}
	logHandler := slog.NewJSONHandler(os.Stdout, loggerOpts)
	logger := slog.New(logHandler)
	return logLevel, logger
}

// pprofServer starts a pprof server for profiling the application.
func pprofServer(pprofListenAddr string, logger *slog.Logger) {
	logger.Info("Starting pprof server", "address", pprofListenAddr)
	// http.ListenAndServe will use the http.DefaultServeMux, which includes pprof handlers
	if err := http.ListenAndServe(pprofListenAddr, nil); err != nil {
		logger.Error("Pprof server failed to start", "error", err)
	}
}

// newDbPool creates a new database connection pool with the provided context and configuration,
// logs an error and exits the application if the pool cannot be created.
func newDbPool(poolCtx context.Context, cfg *config.Config, logger *slog.Logger) *pgxpool.Pool {
	dbPool, errPool := pgxpool.New(poolCtx, cfg.Database.URL)
	if errPool != nil {
		logger.Error("Unable to create connection pool", "error", errPool)
		os.Exit(1)
	}
	return dbPool
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
