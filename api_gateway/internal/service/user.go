package service

import (
	"context"
	"fmt"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type UserService struct {
	userClient   pb.UserServiceClient
	healthClient healthpb.HealthClient
}

// UserDto represents the data transfer object for user registration
type UserDto struct {
	UserName  string `json:"user_name"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// NewUserService creates a service for interact with User service via gRPC
func NewUserService(userClient pb.UserServiceClient, healthClient healthpb.HealthClient) *UserService {
	return &UserService{
		userClient:   userClient,
		healthClient: healthClient,
	}
}

// Register registers a new user using the User service via gRPC.
// It returns the user ID if successful, or an error if registration fails.
func (u *UserService) Register(ctx context.Context, user UserDto) (*string, error) {
	request := &pb.RegisterRequest{
		UserName:  user.UserName,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Password:  user.Password,
	}
	userID, err := u.userClient.Register(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("user registration error: %w", err)
	}
	return &userID.Id, nil
}

// Check checks the health status of the User service via gRPC.
func (u *UserService) Check(ctx context.Context) error {
	resp, err := u.healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("status: %v", resp.Status.String())
	}
	return nil
}
