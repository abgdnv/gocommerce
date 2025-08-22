package server

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// RegistrationFunc registers a grpc service with the server.
type RegistrationFunc func(*grpc.Server)

// NewGRPCServer creates a new gRPC server instance with optional reflection and service registration.
func NewGRPCServer(enableReflection bool, registerFunc ...RegistrationFunc) *grpc.Server {
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

	if enableReflection {
		reflection.Register(grpcServer)
	}

	for _, regFunc := range registerFunc {
		regFunc(grpcServer)
	}

	return grpcServer
}
