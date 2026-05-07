package settings

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

// Service encapsulates all settings business logic.
type Service struct {
	settingsRepo SettingsRepository
	userRepo     UserRepository
	currencyRepo CurrencyRepository
	languageRepo LanguageRepository
	logger       *slog.Logger
}

// New creates a new settings Service with the given dependencies.
func New(
	settingsRepo SettingsRepository,
	userRepo UserRepository,
	currencyRepo CurrencyRepository,
	languageRepo LanguageRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		settingsRepo: settingsRepo,
		userRepo:     userRepo,
		currencyRepo: currencyRepo,
		languageRepo: languageRepo,
		logger:       logger,
	}
}

// GetLanguages returns all available languages.
func (s *Service) GetLanguages() ([]*Language, error) {
	langs, err := s.languageRepo.GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]*Language, len(langs))
	for i, l := range langs {
		result[i] = toLanguage(l)
	}
	return result, nil
}

// GetSettings returns the user's settings, auto-creating defaults if they don't exist.
func (s *Service) GetSettings(userID int) (*UserSettings, error) {
	m, err := s.getOrCreateSettings(userID)
	if err != nil {
		return nil, err
	}
	return toUserSettings(m), nil
}

// UpdateSettings updates the user's settings, auto-creating defaults if they don't exist.
func (s *Service) UpdateSettings(userID int, params UpdateParams) (*UserSettings, error) {
	settings, err := s.getOrCreateSettings(userID)
	if err != nil {
		return nil, err
	}

	settings.Settings.Language = params.Language
	settings.Settings.ProjectionEndDate = params.ProjectionEndDate
	settings.Settings.ProjectionPeriod = params.ProjectionPeriod

	if err := s.settingsRepo.Update(settings); err != nil {
		return nil, err
	}

	return toUserSettings(settings), nil
}

// GetBaseCurrency returns the user's base currency.
func (s *Service) GetBaseCurrency(userID int) (*Currency, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.BaseCurrency == nil {
		return nil, ErrBaseCurrencyNotSet
	}

	return toCurrency(user.BaseCurrency), nil
}

// UpdateBaseCurrency validates the currency, updates the user's base currency, and returns the currency.
func (s *Service) UpdateBaseCurrency(userID, currencyID int) (*Currency, error) {
	cur, err := s.currencyRepo.GetByID(currencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCurrency
		}
		return nil, fmt.Errorf("get currency: %w", err)
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.BaseCurrencyID = cur.ID
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return toCurrency(cur), nil
}

// getOrCreateSettings fetches settings for the user, creating defaults if they don't exist.
func (s *Service) getOrCreateSettings(userID int) (*models.UserSettings, error) {
	settings, err := s.settingsRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if createErr := s.settingsRepo.CreateDefault(userID); createErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrSettingsCreateFailed, createErr)
			}
			settings, err = s.settingsRepo.GetByUserID(userID)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrSettingsNotFound, err)
			}
			return settings, nil
		}
		return nil, err
	}

	return settings, nil
}

// toUserSettings converts a models.UserSettings to the service domain type.
func toUserSettings(m *models.UserSettings) *UserSettings {
	if m == nil {
		return nil
	}
	return &UserSettings{
		ID:     m.ID,
		UserID: m.UserID,
		Settings: SettingsData{
			Language:          m.Settings.Language,
			ProjectionEndDate: m.Settings.ProjectionEndDate,
			ProjectionPeriod:  m.Settings.ProjectionPeriod,
		},
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// toCurrency converts a models.Currency to the service domain type.
func toCurrency(m *models.Currency) *Currency {
	if m == nil {
		return nil
	}
	return &Currency{
		ID:   m.ID,
		Code: m.Code,
		Name: m.Name,
	}
}

// toLanguage converts a models.Language to the service domain type.
func toLanguage(m *models.Language) *Language {
	if m == nil {
		return nil
	}
	return &Language{
		ID:        m.ID,
		Code:      m.Code,
		Name:      m.Name,
		IsDeleted: m.IsDeleted,
	}
}
