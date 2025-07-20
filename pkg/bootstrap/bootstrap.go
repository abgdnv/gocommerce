package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewLogger creates a new slog.Logger instance with the specified log level.
func NewLogger(level string) *slog.Logger {
	logLevel := toLevel(level)
	loggerOpts := &slog.HandlerOptions{
		AddSource: logLevel == slog.LevelDebug,
		Level:     logLevel,
	}
	logHandler := slog.NewJSONHandler(os.Stdout, loggerOpts)
	logger := slog.New(logHandler)
	return logger
}

// NewDbPool creates a new database connection pool with the provided context and configuration,
func NewDbPool(ctx context.Context, url string, connectTimeout time.Duration) (*pgxpool.Pool, error) {
	// Create context with timeout for database connection
	poolCtx, cancel := context.WithTimeout(ctx, connectTimeout)
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
