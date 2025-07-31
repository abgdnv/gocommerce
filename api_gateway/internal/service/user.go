package service

import (
	"context"
	"fmt"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
)

type UserService struct {
	userClient pb.UserServiceClient
}

// UserDto represents the data transfer object for user registration
type UserDto struct {
	UserName  string
	FirstName string
	LastName  string
	Email     string
	Password  string
}

// NewUserService creates a service for interact with User service via gRPC
func NewUserService(userClient pb.UserServiceClient) *UserService {
	return &UserService{
		userClient: userClient,
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
