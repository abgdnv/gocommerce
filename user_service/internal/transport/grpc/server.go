// Package grpc provides a gRPC server for the user service.
package grpc

import (
	"context"
	"errors"
	"log/slog"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
	"github.com/abgdnv/gocommerce/user_service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService defines the interface for the user service.
type UserService interface {
	Register(ctx context.Context, user service.CreateUserDto) (*string, error)
}

type Server struct {
	// Embed the unimplemented server for forward compatibility
	pb.UnimplementedUserServiceServer
	service UserService
}

func NewServer(service UserService) *Server {
	return &Server{service: service}
}

// Register creates a new Keycloak user
func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	slog.InfoContext(ctx, "received grpc request Register", slog.Any("username", req.UserName))
	userDto := service.CreateUserDto{
		UserName:  req.UserName,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	}
	userID, err := s.service.Register(ctx, userDto)
	if err != nil {
		slog.ErrorContext(ctx, "service.Register failed", "error", err)
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if errors.Is(err, service.ErrInvalidUserData) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	slog.InfoContext(ctx, "send grpc response", "userID", *userID)
	return &pb.RegisterResponse{Id: *userID}, nil
}
