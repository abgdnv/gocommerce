// Package app contains the application setup for the UserService.
package app

import (
	"log/slog"

	"github.com/Nerzal/gocloak/v13"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
	"github.com/abgdnv/gocommerce/pkg/server"
	"github.com/abgdnv/gocommerce/user_service/internal/service"
	grpcImpl "github.com/abgdnv/gocommerce/user_service/internal/transport/grpc"
	"google.golang.org/grpc"
)

type Dependencies struct {
	UserService *service.UserService
	Logger      *slog.Logger
}

func SetupDependencies(logger *slog.Logger, gocloak *gocloak.GoCloak, clientID, secret, realm string) *Dependencies {
	uService := service.NewService(gocloak, realm, clientID, secret)
	return &Dependencies{
		UserService: uService,
		Logger:      logger,
	}
}

// SetupGrpcServer initializes the gRPC server
func SetupGrpcServer(deps *Dependencies, reflectionEnabled bool) *grpc.Server {
	// Service registration function for gRPC server
	userRegisterFunc := func(s *grpc.Server) {
		userGRPCServer := grpcImpl.NewServer(deps.UserService)
		pb.RegisterUserServiceServer(s, userGRPCServer)
	}
	// create a new gRPC server with reflection if enabled
	return server.NewGRPCServer(reflectionEnabled, userRegisterFunc)
}
