package grpc

import (
	"context"
	"testing"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/user/v1"
	"github.com/abgdnv/gocommerce/user_service/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockUserService is a mock implementation of the UserService interface.
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(ctx context.Context, user service.CreateUserDto) (*string, error) {
	args := m.Called(ctx, user)

	var result *string
	if id, ok := args.Get(0).(string); ok {
		result = &id
	}
	return result, args.Error(1)
}

func TestServer_Register(t *testing.T) {
	ctx := context.Background()
	req := &pb.RegisterRequest{
		UserName:  "jdoe",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "jdoe@example.com",
		Password:  "password",
	}
	dto := service.CreateUserDto{
		UserName:  req.UserName,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	}
	successID := "123456"

	// given
	testCases := []struct {
		name         string
		retID        string
		retErr       error
		expectedCode codes.Code
	}{
		{
			name:         "success",
			retID:        successID,
			expectedCode: codes.OK,
		},
		{
			name:         "already exists",
			retErr:       service.ErrUserAlreadyExists,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "invalid data",
			retErr:       service.ErrInvalidUserData,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "internal error",
			retErr:       service.ErrIdPInteractionFailed,
			expectedCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			mockSvc := new(MockUserService)
			server := NewServer(mockSvc)
			mockSvc.On("Register", mock.Anything, dto).Return(tc.retID, tc.retErr)

			// when
			res, err := server.Register(ctx, req)

			// then
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, successID, res.Id)
			} else {
				require.Nil(t, res)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.expectedCode, st.Code())
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
