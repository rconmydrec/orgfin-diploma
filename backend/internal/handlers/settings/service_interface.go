package settings

import (
	settingsservice "github.com/go-budget/backend/internal/services/settings"
)

// SettingsService defines the interface for settings service operations
// needed by this handler.
type SettingsService interface {
	GetSettings(userID int) (*settingsservice.UserSettings, error)
	UpdateSettings(userID int, params settingsservice.UpdateParams) (*settingsservice.UserSettings, error)
	GetBaseCurrency(userID int) (*settingsservice.Currency, error)
	UpdateBaseCurrency(userID int, currencyID int) (*settingsservice.Currency, error)
	GetLanguages() ([]*settingsservice.Language, error)
}
