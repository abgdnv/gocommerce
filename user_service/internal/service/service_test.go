package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGoCloakClient is a mock implementation of the gocloak.GoCloak interface
type mockGoCloakClient struct {
	loginToken *gocloak.JWT
	loginErr   error

	createID  string
	createErr error

	setPwdErr    error
	deleteCalled bool
}

func (m *mockGoCloakClient) LoginClient(context.Context, string, string, string, ...string) (*gocloak.JWT, error) {
	return m.loginToken, m.loginErr
}

func (m *mockGoCloakClient) CreateUser(context.Context, string, string, gocloak.User) (string, error) {
	return m.createID, m.createErr
}

func (m *mockGoCloakClient) SetPassword(context.Context, string, string, string, string, bool) error {
	return m.setPwdErr
}

func (m *mockGoCloakClient) DeleteUser(context.Context, string, string, string) error {
	m.deleteCalled = true
	return nil
}

// TestUserService_Register tests the Register method of the UserService
func TestUserService_Register(t *testing.T) {
	ctx := context.Background()
	validUser := CreateUserDto{
		UserName:  "jdoe",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "jdoe@example.com",
		Password:  "password",
	}
	invalidUser := CreateUserDto{}
	successToken := &gocloak.JWT{AccessToken: "token"}

	// given
	tests := []struct {
		name         string
		mock         *mockGoCloakClient
		userDto      CreateUserDto
		expectedErr  error
		expectDelete bool
	}{
		{
			name: "success",
			mock: &mockGoCloakClient{
				loginToken: successToken,
				createID:   "uid",
			},
			userDto: validUser,
		},
		{
			name:        "invalid user data",
			mock:        &mockGoCloakClient{},
			userDto:     invalidUser,
			expectedErr: ErrInvalidUserData,
		},
		{
			name: "login error",
			mock: &mockGoCloakClient{
				loginErr: errors.New("login fail"),
			},
			userDto:     validUser,
			expectedErr: ErrIdPInteractionFailed,
		},
		{
			name: "user exists",
			mock: &mockGoCloakClient{
				loginToken: successToken,
				createErr:  &gocloak.APIError{Code: http.StatusConflict},
			},
			userDto:     validUser,
			expectedErr: ErrUserAlreadyExists,
		},
		{
			name: "invalid data",
			mock: &mockGoCloakClient{
				loginToken: successToken,
				createErr:  &gocloak.APIError{Code: http.StatusBadRequest},
			},
			userDto:     validUser,
			expectedErr: ErrInvalidUserData,
		},
		{
			name: "create error",
			mock: &mockGoCloakClient{
				loginToken: successToken,
				createErr:  errors.New("fail"),
			},
			userDto:     validUser,
			expectedErr: ErrIdPInteractionFailed,
		},
		{
			name: "set password error",
			mock: &mockGoCloakClient{
				loginToken: successToken,
				createID:   "uid",
				setPwdErr:  errors.New("fail"),
			},
			userDto:      validUser,
			expectedErr:  ErrIdPInteractionFailed,
			expectDelete: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := NewService(tc.mock, "realm", "client", "secret")

			// when
			id, err := svc.Register(ctx, tc.userDto)

			// then
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, id)
			} else {
				require.NoError(t, err)
				require.NotNil(t, id)
				assert.Equal(t, tc.mock.createID, *id)
			}
			assert.Equal(t, tc.expectDelete, tc.mock.deleteCalled)
		})
	}
}
