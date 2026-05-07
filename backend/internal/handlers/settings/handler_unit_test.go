package settings

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	settingsservice "github.com/go-budget/backend/internal/services/settings"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock repositories for the service

type MockSettingsRepo struct {
	GetByUserIDFunc   func(userID int) (*models.UserSettings, error)
	CreateDefaultFunc func(userID int) error
	UpdateFunc        func(settings *models.UserSettings) error
}

func (m *MockSettingsRepo) GetByUserID(userID int) (*models.UserSettings, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *MockSettingsRepo) CreateDefault(userID int) error {
	if m.CreateDefaultFunc != nil {
		return m.CreateDefaultFunc(userID)
	}
	return nil
}

func (m *MockSettingsRepo) Update(settings *models.UserSettings) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(settings)
	}
	return nil
}

type MockUserRepo struct {
	GetByIDFunc func(id int) (*models.User, error)
	UpdateFunc  func(user *models.User) error
}

func (m *MockUserRepo) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.User{ID: 1, IsActive: true, BaseCurrencyID: 1, BaseCurrency: &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}}, nil
}

func (m *MockUserRepo) Update(user *models.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(user)
	}
	return nil
}

type MockCurrencyRepo struct {
	GetByIDFunc func(id int) (*models.Currency, error)
}

func (m *MockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
}

type MockLanguageRepo struct {
	GetAllFunc func() ([]*models.Language, error)
}

func (m *MockLanguageRepo) GetAll() ([]*models.Language, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []*models.Language{
		{ID: 1, Code: "en", Name: "English"},
	}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockSettingsRepo, *MockUserRepo, *MockCurrencyRepo, *MockLanguageRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	settingsRepo := &MockSettingsRepo{}
	userRepo := &MockUserRepo{}
	currencyRepo := &MockCurrencyRepo{}
	languageRepo := &MockLanguageRepo{}

	svc := settingsservice.New(settingsRepo, userRepo, currencyRepo, languageRepo, logger)
	handler := New(svc, logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return handler, e, settingsRepo, userRepo, currencyRepo, languageRepo
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
}

// ==================== GetLanguages Tests ====================

func TestGetLanguagesDBError(t *testing.T) {
	handler, e, _, _, _, languageRepo := setupTestHandler()
	languageRepo.GetAllFunc = func() ([]*models.Language, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/languages", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetLanguages(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get languages")
}

// ==================== GetSettings Tests ====================

func TestGetSettingsDBError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetSettingsCreateDefaultError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, sql.ErrNoRows
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return errors.New("create failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetSettingsGetAfterCreateError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== UpdateSettings Tests ====================

func TestUpdateSettingsDBError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, errors.New("db error")
	}

	body := `{"language": "en"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateSettingsCreateDefaultError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return nil, sql.ErrNoRows
	}
	settingsRepo.CreateDefaultFunc = func(userID int) error {
		return errors.New("create failed")
	}

	body := `{"language": "en"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateSettingsUpdateError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
	settingsRepo.GetByUserIDFunc = func(userID int) (*models.UserSettings, error) {
		return &models.UserSettings{
			ID:        1,
			UserID:    1,
			Settings:  models.SettingsData{Language: "en"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	settingsRepo.UpdateFunc = func(settings *models.UserSettings) error {
		return errors.New("update failed")
	}

	body := `{"language": "de"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== GetBaseCurrency Tests ====================

func TestGetBaseCurrencyNotSet(t *testing.T) {
	handler, e, _, userRepo, _, _ := setupTestHandler()
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return &models.User{ID: id, IsActive: true, BaseCurrency: nil}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/base-currency/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBaseCurrency(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Base currency not set")
}

func TestGetBaseCurrencyDBError(t *testing.T) {
	handler, e, _, userRepo, _, _ := setupTestHandler()
	userRepo.GetByIDFunc = func(id int) (*models.User, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/base-currency/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBaseCurrency(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get base currency")
}

// ==================== UpdateBaseCurrency Tests ====================

func TestUpdateBaseCurrencyInvalidCurrencyUnit(t *testing.T) {
	handler, e, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, sql.ErrNoRows
	}

	body := `{"currencyId": 999}`
	req := httptest.NewRequest(http.MethodPut, "/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateBaseCurrency(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid currency")
}

func TestUpdateBaseCurrencyUpdateError(t *testing.T) {
	handler, e, _, userRepo, _, _ := setupTestHandler()
	userRepo.UpdateFunc = func(user *models.User) error {
		return errors.New("update failed")
	}

	body := `{"currencyId": 1}`
	req := httptest.NewRequest(http.MethodPut, "/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateBaseCurrency(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== UpdateSettings Bug Fix Test ====================

func TestUpdateSettingsGetAfterCreateError(t *testing.T) {
	handler, e, settingsRepo, _, _, _ := setupTestHandler()
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

	body := `{"language": "en"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateSettings(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to")
}
