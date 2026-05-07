package auth

import (
	authservice "github.com/go-budget/backend/internal/services/auth"
)

// AuthService defines the interface for auth service operations
type AuthService interface {
	Register(email, password, firstName, lastName string) (*authservice.RegisterResult, error)
	Login(email, password string) (string, error)
	GetProfile(userID int) (*authservice.User, *authservice.UserSettings, error)
	ActivateUser(token string) error
	ChangePassword(userID int, currentPassword, newPassword string) error
	LoginOrRegisterOAuth(email, firstName, lastName string) (string, error)
}
