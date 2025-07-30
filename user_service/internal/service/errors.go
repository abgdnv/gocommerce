package service

import "errors"

var (
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidUserData      = errors.New("invalid user data")
	ErrIdPInteractionFailed = errors.New("identity provider interaction failed")
)
