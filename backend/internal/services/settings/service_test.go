package settings

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories

type mockSettingsRepo struct {
	GetByUserIDFunc   func(userID int) (*models.UserSettings, error)
	CreateDefaultFunc func(userID int) error
	UpdateFunc        func(settings *models.UserSettings) error
}

func (m *mockSettingsRepo) GetByUserID(userID int) (*models.UserSettings, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *mockSettingsRepo) CreateDefault(userID int) error {
	if m.CreateDefaultFunc != nil {
		return m.CreateDefaultFunc(userID)
	}
	return nil
}

func (m *mockSettingsRepo) Update(settings *models.UserSettings) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(settings)
	}
	return nil
}

type mockUserRepo struct {
	GetByIDFunc func(id int) (*models.User, error)
	UpdateFunc  func(user *models.User) error
}

func (m *mockUserRepo) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.User{
		ID:             id,
		IsActive:       true,
		BaseCurrencyID: 1,
		BaseCurrency:   &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"},
	}, nil
}

func (m *mockUserRepo) Update(user *models.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(user)
	}
	return nil
}

type mockCurrencyRepo struct {
	GetByIDFunc func(id int) (*models.Currency, error)
}

func (m *mockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Currency{ID: id, Code: "USD", Name: "US Dollar"}, nil
}

type mockLanguageRepo struct {
	GetAllFunc func() ([]*models.Language, error)
}

func (m *mockLanguageRepo) GetAll() ([]*models.Language, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []*models.Language{{ID: 1, Code: "en", Name: "English"}}, nil
}

func newTestService() (*Service, *mockSettingsRepo, *mockUserRepo, *mockCurrencyRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	settingsRepo := &mockSettingsRepo{}
	userRepo := &mockUserRepo{}
	currencyRepo := &mockCurrencyRepo{}
	languageRepo := &mockLanguageRepo{}
	svc := New(settingsRepo, userRepo, currencyRepo, languageRepo, logger)
	return svc, settingsRepo, userRepo, currencyRepo
}

func defaultSettings(userID int) *models.UserSettings {
	return &models.UserSettings{
		ID:        1,
		UserID:    userID,
		Settings:  models.SettingsData{Language: "en"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ==================== GetSettings Tests ====================

func TestGetSettings_Success(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return defaultSettings(userID), nil
	}

	result, err := svc.GetSettings(1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.UserID)
	assert.Equal(t, "en", result.Settings.Language)
}

func TestGetSettings_AutoCreate(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	callCount := 0
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		callCount++
		if callCount == 1 {
			return nil, sql.ErrNoRows
		}
		return defaultSettings(userID), nil
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		assert.Equal(t, 1, userID)
		return nil
	}

	result, err := svc.GetSettings(1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.UserID)
	assert.Equal(t, 2, callCount)
}

func TestGetSettings_CreateDefaultFails(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, sql.ErrNoRows
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return errors.New("create failed")
	}

	result, err := svc.GetSettings(1)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrSettingsCreateFailed))
	assert.Contains(t, err.Error(), "create failed")
}

func TestGetSettings_ReFetchAfterCreateFails(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	callCount := 0
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		callCount++
		if callCount == 1 {
			return nil, sql.ErrNoRows
		}
		return nil, errors.New("db error after create")
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return nil
	}

	result, err := svc.GetSettings(1)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrSettingsNotFound))
	assert.Contains(t, err.Error(), "db error after create")
}

func TestGetSettings_NonErrNoRowsDBError(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	dbErr := errors.New("connection refused")
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, dbErr
	}

	result, err := svc.GetSettings(1)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestGetSettings_CorrectUserIDPassed(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	var receivedUserID int
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		receivedUserID = userID
		return defaultSettings(userID), nil
	}

	_, err := svc.GetSettings(42)
	require.NoError(t, err)
	assert.Equal(t, 42, receivedUserID)
}

// ==================== UpdateSettings Tests ====================

func TestUpdateSettings_Success(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return defaultSettings(userID), nil
	}

	endDate := "2025-12-31"
	period := "daily"
	params := UpdateParams{
		Language:          "de",
		ProjectionEndDate: &endDate,
		ProjectionPeriod:  &period,
	}

	result, err := svc.UpdateSettings(1, params)
	require.NoError(t, err)
	assert.Equal(t, "de", result.Settings.Language)
	assert.Equal(t, &endDate, result.Settings.ProjectionEndDate)
	assert.Equal(t, &period, result.Settings.ProjectionPeriod)
}

func TestUpdateSettings_AutoCreatePath(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	callCount := 0
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		callCount++
		if callCount == 1 {
			return nil, sql.ErrNoRows
		}
		return defaultSettings(userID), nil
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return nil
	}

	var updatedSettings *models.UserSettings
	settingsRepo.UpdateFunc = func(settings *models.UserSettings) error {
		updatedSettings = settings
		return nil
	}

	params := UpdateParams{Language: "fr"}
	result, err := svc.UpdateSettings(1, params)
	require.NoError(t, err)
	assert.Equal(t, "fr", result.Settings.Language)
	assert.Equal(t, "fr", updatedSettings.Settings.Language)
}

func TestUpdateSettings_NilProjectionFields(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		endDate := "2025-12-31"
		period := "daily"
		s := defaultSettings(userID)
		s.Settings.ProjectionEndDate = &endDate
		s.Settings.ProjectionPeriod = &period
		return s, nil
	}

	params := UpdateParams{
		Language:          "en",
		ProjectionEndDate: nil,
		ProjectionPeriod:  nil,
	}

	result, err := svc.UpdateSettings(1, params)
	require.NoError(t, err)
	assert.Nil(t, result.Settings.ProjectionEndDate)
	assert.Nil(t, result.Settings.ProjectionPeriod)
}

func TestUpdateSettings_UpdateRepoError(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return defaultSettings(userID), nil
	}
	settingsRepo.UpdateFunc = func(settings *models.UserSettings) error {
		return errors.New("update failed")
	}

	result, err := svc.UpdateSettings(1, UpdateParams{Language: "en"})
	assert.Nil(t, result)
	assert.EqualError(t, err, "update failed")
}

func TestUpdateSettings_GetOrCreateFailure(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, sql.ErrNoRows
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return errors.New("create failed")
	}

	result, err := svc.UpdateSettings(1, UpdateParams{Language: "en"})
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrSettingsCreateFailed))
}

func TestUpdateSettings_FieldsAppliedCorrectly(t *testing.T) {
	svc, settingsRepo, _, _ := newTestService()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return defaultSettings(userID), nil
	}

	var savedSettings *models.UserSettings
	settingsRepo.UpdateFunc = func(settings *models.UserSettings) error {
		savedSettings = settings
		return nil
	}

	endDate := "2026-06-30"
	period := "monthly"
	params := UpdateParams{
		Language:          "uk",
		ProjectionEndDate: &endDate,
		ProjectionPeriod:  &period,
	}

	_, err := svc.UpdateSettings(1, params)
	require.NoError(t, err)
	assert.Equal(t, "uk", savedSettings.Settings.Language)
	assert.Equal(t, &endDate, savedSettings.Settings.ProjectionEndDate)
	assert.Equal(t, &period, savedSettings.Settings.ProjectionPeriod)
}

// ==================== GetBaseCurrency Tests ====================

func TestGetBaseCurrency_Success(t *testing.T) {
	svc, _, userRepo, _ := newTestService()
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{
			ID:             id,
			BaseCurrencyID: 1,
			BaseCurrency:   &models.Currency{ID: 1, Code: "EUR", Name: "Euro"},
		}, nil
	}

	result, err := svc.GetBaseCurrency(1)
	require.NoError(t, err)
	assert.Equal(t, "EUR", result.Code)
	assert.Equal(t, "Euro", result.Name)
}

func TestGetBaseCurrency_NotSet(t *testing.T) {
	svc, _, userRepo, _ := newTestService()
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{
			ID:           id,
			BaseCurrency: nil,
		}, nil
	}

	result, err := svc.GetBaseCurrency(1)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrBaseCurrencyNotSet))
}

func TestGetBaseCurrency_UserFetchError(t *testing.T) {
	svc, _, userRepo, _ := newTestService()
	dbErr := errors.New("user not found")
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return nil, dbErr
	}

	result, err := svc.GetBaseCurrency(1)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestGetBaseCurrency_CorrectUserIDPassed(t *testing.T) {
	svc, _, userRepo, _ := newTestService()
	var receivedID int
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		receivedID = id
		return &models.User{
			ID:           id,
			BaseCurrency: &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"},
		}, nil
	}

	_, err := svc.GetBaseCurrency(99)
	require.NoError(t, err)
	assert.Equal(t, 99, receivedID)
}

// ==================== UpdateBaseCurrency Tests ====================

func TestUpdateBaseCurrency_Success(t *testing.T) {
	svc, _, userRepo, currencyRepo := newTestService()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return &models.Currency{ID: id, Code: "GBP", Name: "British Pound"}, nil
	}
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{ID: id, BaseCurrencyID: 1}, nil
	}

	var updatedUser *models.User
	userRepo.UpdateFunc = func(user *models.User) error {
		updatedUser = user
		return nil
	}

	result, err := svc.UpdateBaseCurrency(1, 5)
	require.NoError(t, err)
	assert.Equal(t, "GBP", result.Code)
	assert.Equal(t, "British Pound", result.Name)
	assert.Equal(t, 5, updatedUser.BaseCurrencyID)
}

func TestUpdateBaseCurrency_InvalidCurrency(t *testing.T) {
	svc, _, _, currencyRepo := newTestService()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, sql.ErrNoRows
	}

	result, err := svc.UpdateBaseCurrency(1, 999)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrInvalidCurrency))
}

func TestUpdateBaseCurrency_CurrencyRepoDBError(t *testing.T) {
	svc, _, _, currencyRepo := newTestService()
	dbErr := errors.New("connection refused")
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, dbErr
	}

	result, err := svc.UpdateBaseCurrency(1, 999)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "connection refused")
}

func TestUpdateBaseCurrency_UserFetchError(t *testing.T) {
	svc, _, userRepo, currencyRepo := newTestService()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return &models.Currency{ID: id, Code: "USD", Name: "US Dollar"}, nil
	}
	dbErr := errors.New("user fetch failed")
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return nil, dbErr
	}

	result, err := svc.UpdateBaseCurrency(1, 1)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestUpdateBaseCurrency_UserUpdateError(t *testing.T) {
	svc, _, userRepo, currencyRepo := newTestService()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return &models.Currency{ID: id, Code: "USD", Name: "US Dollar"}, nil
	}
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{ID: id, BaseCurrencyID: 1}, nil
	}
	updateErr := errors.New("update failed")
	userRepo.UpdateFunc = func(user *models.User) error {
		return updateErr
	}

	result, err := svc.UpdateBaseCurrency(1, 1)
	assert.Nil(t, result)
	assert.Equal(t, updateErr, err)
}

func TestUpdateBaseCurrency_BaseCurrencyIDSetCorrectly(t *testing.T) {
	svc, _, userRepo, currencyRepo := newTestService()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return &models.Currency{ID: 42, Code: "JPY", Name: "Japanese Yen"}, nil
	}
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{ID: id, BaseCurrencyID: 1}, nil
	}

	var capturedUser *models.User
	userRepo.UpdateFunc = func(user *models.User) error {
		capturedUser = user
		return nil
	}

	result, err := svc.UpdateBaseCurrency(1, 42)
	require.NoError(t, err)
	assert.Equal(t, 42, capturedUser.BaseCurrencyID)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "JPY", result.Code)
}

func TestUpdateBaseCurrency_CorrectCurrencyReturned(t *testing.T) {
	svc, _, userRepo, currencyRepo := newTestService()
	expectedCurrency := &models.Currency{ID: 7, Code: "CHF", Name: "Swiss Franc"}
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return expectedCurrency, nil
	}
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{ID: id, BaseCurrencyID: 1}, nil
	}

	result, err := svc.UpdateBaseCurrency(1, 7)
	require.NoError(t, err)
	assert.Equal(t, expectedCurrency.ID, result.ID)
	assert.Equal(t, expectedCurrency.Code, result.Code)
	assert.Equal(t, expectedCurrency.Name, result.Name)
}
