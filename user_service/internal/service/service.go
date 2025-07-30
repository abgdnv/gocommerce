package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Nerzal/gocloak/v13"
)

type UserService struct {
	gocloak  *gocloak.GoCloak
	realm    string
	clientID string
	secret   string
}

type CreateUserDto struct {
	UserName  string `json:"user_name"   validate:"required"`
	FirstName string `json:"first_name"  validate:"required"`
	LastName  string `json:"last_name"   validate:"required"`
	Email     string `json:"email"       validate:"email"`
	Password  string `json:"password"    validate:"required"`
}

func NewService(gocloak *gocloak.GoCloak, realm, clientID, secret string) *UserService {
	return &UserService{gocloak: gocloak, realm: realm, clientID: clientID, secret: secret}
}

func (u *UserService) Register(ctx context.Context, userDto CreateUserDto) (*string, error) {
	user := gocloak.User{
		Username:  gocloak.StringP(userDto.UserName),
		Email:     gocloak.StringP(userDto.Email),
		Enabled:   gocloak.BoolP(true),
		FirstName: gocloak.StringP(userDto.FirstName),
		LastName:  gocloak.StringP(userDto.LastName),
	}

	token, err := u.gocloak.LoginClient(ctx, u.clientID, u.secret, u.realm)
	if err != nil {
		slog.Error("Failed to login", "error", err)
		return nil, fmt.Errorf("%w: failed to login to Keycloak: %v", ErrIdPInteractionFailed, err)

	}

	userID, err := u.gocloak.CreateUser(ctx, token.AccessToken, u.realm, user)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		var apiErr *gocloak.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case http.StatusConflict:
				return nil, ErrUserAlreadyExists
			case http.StatusBadRequest:
				return nil, ErrInvalidUserData
			}
		}
		return nil, ErrIdPInteractionFailed
	}

	err = u.gocloak.SetPassword(ctx, token.AccessToken, userID, u.realm, userDto.Password, false)
	if err != nil {
		slog.Error("Failed to set password", "error", err)
		errSetPassword := fmt.Errorf("%w: failed to set password: %v", ErrIdPInteractionFailed, err)
		_ = u.gocloak.DeleteUser(ctx, token.AccessToken, u.realm, userID)
		return nil, errSetPassword
	}

	return &userID, nil
}
