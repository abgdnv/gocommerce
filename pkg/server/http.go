package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/go-chi/chi/v5"
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
func NewHTTPServer(cfg HTTPConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeader,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}

// NewChiRouter creates a new Chi router with a set of
// middleware for request ID injection, structured logging, and recovery.
func NewChiRouter(logger *slog.Logger) *chi.Mux {
	mux := chi.NewRouter()
	mux.Use(web.RequestIDInjector)
	mux.Use(web.StructuredLogger(logger))
	mux.Use(web.Recoverer(logger))
	return mux
}
