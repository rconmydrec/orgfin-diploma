package auth

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the auth service.
var (
	ErrUserExists         = serviceerrors.New(serviceerrors.Conflict, "user with this email already exists")
	ErrInvalidCredentials = serviceerrors.New(serviceerrors.InvalidInput, "invalid credentials or user not found")
	ErrUserNotActivated   = serviceerrors.New(serviceerrors.Unauthorized, "user not activated")
	ErrUserNotFound       = serviceerrors.New(serviceerrors.NotFound, "user not found")
	ErrTokenNotFound      = serviceerrors.New(serviceerrors.NotFound, "token not found")
	ErrTokenExpired       = serviceerrors.New(serviceerrors.Unauthorized, "token expired")
	ErrIncorrectPassword  = serviceerrors.New(serviceerrors.Unauthorized, "current password is incorrect")
)
