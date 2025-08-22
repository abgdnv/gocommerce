package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPConfig has the configuration for the HTTP server.
type HTTPConfig struct {
	Port           int
	MaxHeaderBytes int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	ReadHeader     time.Duration
}

// NewHTTPServer creates and configures a new HTTP server instance.
func NewHTTPServer(cfg config.HTTPConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadTimeout:       cfg.Timeout.Read,
		WriteTimeout:      cfg.Timeout.Write,
		IdleTimeout:       cfg.Timeout.Idle,
		ReadHeaderTimeout: cfg.Timeout.ReadHeader,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}

// NewChiRouter creates a new Chi router with a set of
// middleware for request ID injection, structured logging, telemetry, and recovery.
func NewChiRouter(logger *slog.Logger) *chi.Mux {
	mux := chi.NewRouter()
	mux.Use(web.Recoverer(logger))
	mux.Use(middleware.RequestID)
	mux.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http.server")
	})
	mux.Use(web.TelemetryEnricher)
	mux.Use(web.StructuredLogger(logger))
	return mux
}
