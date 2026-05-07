// Package auth provides authentication business logic including user
// registration, login, OAuth, password management, and account activation.
//
// Invariants:
//   - Registration creates default settings, copies default categories, and
//     creates a trial subscription (all non-blocking: failures are logged but
//     do not prevent registration).
//   - Activation is idempotent: re-activating an already-active user just cleans up the token.
//   - OAuth users are auto-activated (no activation token needed).
//   - The enqueuer and subscriptionSvc dependencies are optional (nil-safe).
package auth

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/hibiken/asynq"
)

// ServiceInterface defines the public API of the auth service.
type ServiceInterface interface {
	Register(email, password, firstName, lastName string) (*RegisterResult, error)
	Login(email, password string) (string, error)
	GetProfile(userID int) (*User, *UserSettings, error)
	ActivateUser(token string) error
	ChangePassword(userID int, currentPassword, newPassword string) error
	LoginOrRegisterOAuth(email, firstName, lastName string) (string, error)
}

// User represents a user in the auth service domain.
type User struct {
	ID             int
	Email          string
	IsActive       bool
	FirstName      *string
	LastName       *string
	BaseCurrencyID int
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Relations
	BaseCurrency *Currency
}

// Currency represents a currency in the auth service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// UserSettings represents user settings in the auth service domain.
type UserSettings struct {
	ID       int
	UserID   int
	Settings SettingsData
}

// SettingsData holds the settings fields.
type SettingsData struct {
	Language          string
	ProjectionEndDate *string
	ProjectionPeriod  *string
}

// UserRepository defines user data access methods needed by the service.
type UserRepository interface {
	Create(user *models.User) (*models.User, error)
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Activate(userID int) error
	UpdatePassword(userID int, passwordHash string) error
}

// ActivationTokenRepository defines activation token data access methods needed by the service.
type ActivationTokenRepository interface {
	Create(token *models.ActivationToken) error
	GetByToken(token string) (*models.ActivationToken, error)
	Delete(id int) error
}

// UserSettingsRepository defines user settings data access methods needed by the service.
type UserSettingsRepository interface {
	CreateDefault(userID int) error
	GetByUserID(userID int) (*models.UserSettings, error)
}

// CategoryRepository defines category data access methods needed by the service.
type CategoryRepository interface {
	CopyDefaultCategories(userID int) error
}

// CurrencyRepository defines currency data access methods needed by the service.
type CurrencyRepository interface {
	GetByCode(code string) (*models.Currency, error)
}

// TaskEnqueuer is the interface for enqueuing async tasks.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// SubscriptionTrialCreator creates a trial subscription for a newly registered user.
// This is an optional dependency; when nil, trial creation is skipped.
type SubscriptionTrialCreator interface {
	CreateTrialSubscription(userID int) error
}

// RegisterResult holds the result of a successful user registration.
type RegisterResult struct {
	User            *User
	ActivationToken string
}
