// Package app contains the application setup for the ProductService.
package app

import (
	"log/slog"
	"net/http"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/server"
	"github.com/abgdnv/gocommerce/product_service/internal/config"
	"github.com/abgdnv/gocommerce/product_service/internal/service"
	"github.com/abgdnv/gocommerce/product_service/internal/store"
	grpcImpl "github.com/abgdnv/gocommerce/product_service/internal/transport/grpc"
	"github.com/abgdnv/gocommerce/product_service/internal/transport/rest"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
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
	mux := server.NewChiRouter(deps.Logger)
	wireRoutes(mux, deps)
	return mux
}

// wireRoutes sets up the HTTP routes for the ProductService application.
func wireRoutes(mux *chi.Mux, deps *Dependencies) {
	productHandler := rest.NewHandler(deps.ProductService, deps.Logger)
	productHandler.RegisterRoutes(mux)
}

// SetupHttpServer creates and configures an HTTP server for the ProductService application.
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

// SetupGrpcServer initializes the gRPC server for the ProductService application.
func SetupGrpcServer(deps *Dependencies, reflectionEnabled bool) *grpc.Server {
	// Service registration function for gRPC server
	productRegisterFunc := func(s *grpc.Server) {
		productGRPCServer := grpcImpl.NewServer(deps.ProductService)
		pb.RegisterProductServiceServer(s, productGRPCServer)
	}
	// create a new gRPC server with reflection if enabled
	return server.NewGRPCServer(reflectionEnabled, productRegisterFunc)
}
