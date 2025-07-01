// Package app contains the application setup for the ProductService.
package app

import (
	"fmt"
	"log/slog"
	"net/http"

	pb "github.com/abgdnv/gocommerce/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/internal/platform/web"
	"github.com/abgdnv/gocommerce/internal/product/config"
	grpcImpl "github.com/abgdnv/gocommerce/internal/product/grpc"
	"github.com/abgdnv/gocommerce/internal/product/handler"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/abgdnv/gocommerce/internal/product/store"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Dependencies struct {
	ProductService service.ProductService
	Logger         *slog.Logger
}

func SetupDependencies(dbPool *pgxpool.Pool, logger *slog.Logger) *Dependencies {
	pService := service.NewService(store.NewPgStore(dbPool))

	return &Dependencies{
		ProductService: pService,
		Logger:         logger,
	}
}

// SetupHttpHandler initializes the HTTP server and routes for the ProductService application.
// Used by E2E tests to set up the HTTP server with the necessary routes and middleware.
func SetupHttpHandler(deps *Dependencies) http.Handler {

	pApi := handler.NewAPI(deps.ProductService, deps.Logger)

	mux := chi.NewRouter()
	mux.Use(web.RequestIDInjector)
	mux.Use(web.StructuredLogger(deps.Logger))
	mux.Use(web.Recoverer(deps.Logger))

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

	return mux
}

// SetupHttpServer creates and configures an HTTP server for the ProductService application.
func SetupHttpServer(deps *Dependencies, cfg *config.Config) *http.Server {
	mux := SetupHttpHandler(deps)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler:           mux,
		ReadTimeout:       cfg.HTTPServer.Timeout.Read,
		WriteTimeout:      cfg.HTTPServer.Timeout.Write,
		IdleTimeout:       cfg.HTTPServer.Timeout.Idle,
		ReadHeaderTimeout: cfg.HTTPServer.Timeout.ReadHeader,
		MaxHeaderBytes:    cfg.HTTPServer.MaxHeaderBytes,
	}
	return server
}

// SetupGrpcServer initializes the gRPC server for the ProductService application.
func SetupGrpcServer(deps *Dependencies, reflectionEnabled bool) *grpc.Server {
	grpcServer := grpc.NewServer()
	if reflectionEnabled {
		reflection.Register(grpcServer)
	}
	pb.RegisterProductServiceServer(grpcServer, grpcImpl.NewServer(deps.ProductService))
	return grpcServer
}
