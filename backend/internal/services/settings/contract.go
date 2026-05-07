// Package settings provides settings business logic including user settings
// CRUD with auto-creation of defaults and base currency management.
//
// Invariants:
//   - GetSettings auto-creates default settings if none exist (get-or-create pattern).
//   - UpdateBaseCurrency validates the currency exists before updating.
//   - All repository interfaces declare only the methods this service actually uses.
package settings

import (
	"time"

	"github.com/go-budget/backend/internal/models"
)

// ServiceInterface defines the public API of the settings service.
type ServiceInterface interface {
	GetSettings(userID int) (*UserSettings, error)
	UpdateSettings(userID int, params UpdateParams) (*UserSettings, error)
	GetBaseCurrency(userID int) (*Currency, error)
	UpdateBaseCurrency(userID int, currencyID int) (*Currency, error)
	GetLanguages() ([]*Language, error)
}

// UserSettings represents user settings in the settings service domain.
type UserSettings struct {
	ID        int
	UserID    int
	Settings  SettingsData
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SettingsData holds the settings fields.
type SettingsData struct {
	Language          string
	ProjectionEndDate *string
	ProjectionPeriod  *string
}

// Currency represents a currency in the settings service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// Language represents a language in the settings service domain.
type Language struct {
	ID        int
	Code      string
	Name      string
	IsDeleted bool
}

// SettingsRepository defines settings data access methods needed by the service.
type SettingsRepository interface {
	CreateDefault(userID int) error
	GetByUserID(userID int) (*models.UserSettings, error)
	Update(settings *models.UserSettings) error
}

// UserRepository defines user data access methods needed by the service.
type UserRepository interface {
	GetByID(id int) (*models.User, error)
	Update(user *models.User) error
}

// CurrencyRepository defines currency data access methods needed by the service.
type CurrencyRepository interface {
	GetByID(id int) (*models.Currency, error)
}

// LanguageRepository defines language data access methods needed by the service.
type LanguageRepository interface {
	GetAll() ([]*models.Language, error)
}

// UpdateParams holds the domain parameters for updating user settings.
type UpdateParams struct {
	Language          string
	ProjectionEndDate *string
	ProjectionPeriod  *string
}
