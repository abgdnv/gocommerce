// Package app contains the application setup for the Order service.
package app

import (
	"log/slog"
	"net/http"

	"github.com/abgdnv/gocommerce/order_service/internal/config"
	"github.com/abgdnv/gocommerce/order_service/internal/service"
	"github.com/abgdnv/gocommerce/order_service/internal/store"
	"github.com/abgdnv/gocommerce/order_service/internal/transport/rest"
	"github.com/abgdnv/gocommerce/pkg/server"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	OrderService service.OrderService
	Logger       *slog.Logger
}

func SetupDependencies(dbPool *pgxpool.Pool, logger *slog.Logger) *Dependencies {
	pService := service.NewService(store.NewPgStore(dbPool))

	return &Dependencies{
		OrderService: pService,
		Logger:       logger,
	}
}

// SetupHttpHandler initializes the HTTP server and routes for the OrderService application.
// Used by E2E tests to set up the HTTP server with the necessary routes and middleware.
func SetupHttpHandler(deps *Dependencies) http.Handler {
	mux := server.NewChiRouter(deps.Logger)
	wireRoutes(mux, deps)
	return mux
}

// wireRoutes sets up the HTTP routes for the OrderService application.
func wireRoutes(mux *chi.Mux, deps *Dependencies) {
	productHandler := rest.NewHandler(deps.OrderService, deps.Logger)
	productHandler.RegisterRoutes(mux)
}

// SetupHttpServer creates and configures an HTTP server for the OrderService application.
func SetupHttpServer(deps *Dependencies, cfg *config.Config) *http.Server {

	mux := SetupHttpHandler(deps)

	httpCfg := server.HTTPConfig{
		Port:           cfg.HTTPServer.Port,
		MaxHeaderBytes: cfg.HTTPServer.MaxHeaderBytes,
		ReadTimeout:    cfg.HTTPServer.Timeout.Read,
		WriteTimeout:   cfg.HTTPServer.Timeout.Write,
		IdleTimeout:    cfg.HTTPServer.Timeout.Idle,
		ReadHeader:     cfg.HTTPServer.Timeout.ReadHeader,
	}

	return server.NewHTTPServer(httpCfg, mux)
}
