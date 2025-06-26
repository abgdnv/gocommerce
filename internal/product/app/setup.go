// Package app contains the application setup for the ProductService.
package app

import (
	"log/slog"
	"net/http"
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

// SetupApplication initializes the HTTP server and routes for the ProductService application.
func SetupApplication(cfg *config.Config, dbPool *pgxpool.Pool, logger *slog.Logger) (http.Handler, error) {
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

	return mux, nil
}
