package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	authservice "github.com/go-budget/backend/internal/services/auth"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/idtoken"
)

// MockGoogleValidator for unit testing
type MockGoogleValidator struct {
	Email         string
	EmailVerified bool
	FirstName     string
	LastName      string
	ShouldFail    bool
	FailError     error
}

func (m *MockGoogleValidator) Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	if m.ShouldFail {
		if m.FailError != nil {
			return nil, m.FailError
		}
		return nil, errors.New("invalid token")
	}

	return &idtoken.Payload{
		Claims: map[string]interface{}{
			"email":          m.Email,
			"email_verified": m.EmailVerified,
			"given_name":     m.FirstName,
			"family_name":    m.LastName,
		},
	}, nil
}

// CustomValidator for echo
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// MockAuthService implements AuthService for testing
type MockAuthService struct {
	RegisterFunc             func(email, password, firstName, lastName string) (*authservice.RegisterResult, error)
	LoginFunc                func(email, password string) (string, error)
	GetProfileFunc           func(userID int) (*authservice.User, *authservice.UserSettings, error)
	ActivateUserFunc         func(token string) error
	ChangePasswordFunc       func(userID int, current_password, new_password string) error
	LoginOrRegisterOAuthFunc func(email, firstName, lastName string) (string, error)
}

func (m *MockAuthService) Register(email, password, firstName, lastName string) (*authservice.RegisterResult, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(email, password, firstName, lastName)
	}
	return nil, nil
}

func (m *MockAuthService) Login(email, password string) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(email, password)
	}
	return "", nil
}

func (m *MockAuthService) GetProfile(userID int) (*authservice.User, *authservice.UserSettings, error) {
	if m.GetProfileFunc != nil {
		return m.GetProfileFunc(userID)
	}
	return nil, nil, nil
}

func (m *MockAuthService) ActivateUser(token string) error {
	if m.ActivateUserFunc != nil {
		return m.ActivateUserFunc(token)
	}
	return nil
}

func (m *MockAuthService) ChangePassword(userID int, current_password, new_password string) error {
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(userID, current_password, new_password)
	}
	return nil
}

func (m *MockAuthService) LoginOrRegisterOAuth(email, firstName, lastName string) (string, error) {
	if m.LoginOrRegisterOAuthFunc != nil {
		return m.LoginOrRegisterOAuthFunc(email, firstName, lastName)
	}
	return "", nil
}

func setupTestHandler(mockService *MockAuthService) (*Handler, *echo.Echo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := New(mockService, "test-client-id", logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return handler, e
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
}

// ==================== Register Error Tests ====================

func TestRegisterDBError(t *testing.T) {
	mockService := &MockAuthService{
		RegisterFunc: func(email, password, firstName, lastName string) (*authservice.RegisterResult, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"email": "test@test.com", "password": "Test123!", "firstName": "Test", "lastName": "User"}`
	req := httptest.NewRequest(http.MethodPost, "/register/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Registration failed")
}

func TestRegisterBindError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/register/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request data")
}

func TestRegisterValidationError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	// Missing required fields
	body := `{"email": "test@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/register/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Register(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Validation failed")
}

// ==================== Login Error Tests ====================

func TestLoginDBError(t *testing.T) {
	mockService := &MockAuthService{
		LoginFunc: func(email, password string) (string, error) {
			return "", errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"email": "test@test.com", "password": "Test123!"}`
	req := httptest.NewRequest(http.MethodPost, "/login/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestLoginBindError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/login/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestLoginValidationError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{"email": "test@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/login/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== GetProfile Error Tests ====================

func TestGetProfileDBError(t *testing.T) {
	mockService := &MockAuthService{
		GetProfileFunc: func(userID int) (*authservice.User, *authservice.UserSettings, error) {
			return nil, nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/profile/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetProfile(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get profile")
}

func TestGetProfileUserNotFound(t *testing.T) {
	mockService := &MockAuthService{
		GetProfileFunc: func(userID int) (*authservice.User, *authservice.UserSettings, error) {
			return nil, nil, authservice.ErrUserNotFound
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/profile/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetProfile(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not found")
}

func TestGetProfileSuccessWithoutSettings(t *testing.T) {
	firstName := "Test"
	lastName := "User"
	mockService := &MockAuthService{
		GetProfileFunc: func(userID int) (*authservice.User, *authservice.UserSettings, error) {
			return &authservice.User{
				ID:           1,
				Email:        "test@test.com",
				FirstName:    &firstName,
				LastName:     &lastName,
				BaseCurrency: &authservice.Currency{Code: "USD"},
			}, nil, nil
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/profile/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetProfile(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "test@test.com")
}

// ==================== Activate Error Tests ====================

func TestActivateDBError(t *testing.T) {
	mockService := &MockAuthService{
		ActivateUserFunc: func(token string) error {
			return errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/activate/sometoken", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues("sometoken")

	err := handler.Activate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Activation failed")
}

func TestActivateTokenNotFound(t *testing.T) {
	mockService := &MockAuthService{
		ActivateUserFunc: func(token string) error {
			return authservice.ErrTokenNotFound
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/activate/invalidtoken", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues("invalidtoken")

	err := handler.Activate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Token not found")
}

func TestActivateTokenExpired(t *testing.T) {
	mockService := &MockAuthService{
		ActivateUserFunc: func(token string) error {
			return authservice.ErrTokenExpired
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/activate/expiredtoken", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("token")
	c.SetParamValues("expiredtoken")

	err := handler.Activate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Token expired")
}

// ==================== ChangePassword Error Tests ====================

func TestChangePasswordDBError(t *testing.T) {
	mockService := &MockAuthService{
		ChangePasswordFunc: func(userID int, current_password, new_password string) error {
			return errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"current_password": "OldPass123!", "new_password": "NewPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/change-password/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePassword(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to change password")
}

func TestChangePasswordUserNotFound(t *testing.T) {
	mockService := &MockAuthService{
		ChangePasswordFunc: func(userID int, current_password, new_password string) error {
			return authservice.ErrUserNotFound
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"current_password": "OldPass123!", "new_password": "NewPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/change-password/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePassword(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not found")
}

func TestChangePasswordIncorrect(t *testing.T) {
	mockService := &MockAuthService{
		ChangePasswordFunc: func(userID int, current_password, new_password string) error {
			return authservice.ErrIncorrectPassword
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"current_password": "WrongPass!", "new_password": "NewPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/change-password/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePassword(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Current password is incorrect")
}

func TestChangePasswordBindError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/change-password/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePassword(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestChangePasswordValidationError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{"current_password": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/change-password/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePassword(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== OAuth Error Tests ====================

func TestOAuthDBError(t *testing.T) {
	mockService := &MockAuthService{
		LoginOrRegisterOAuthFunc: func(email, firstName, lastName string) (string, error) {
			return "", errors.New("database error")
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockValidator := &MockGoogleValidator{
		Email:         "test@test.com",
		EmailVerified: true,
		FirstName:     "Test",
		LastName:      "User",
	}
	handler := NewWithValidator(mockService, "test-client-id", mockValidator, logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	body := `{"credential": "valid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.OAuth(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestOAuthBindError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.OAuth(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestOAuthValidationError(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.OAuth(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== RegisterRoutes Test ====================

func TestRegisterRoutes(t *testing.T) {
	mockService := &MockAuthService{}
	handler, e := setupTestHandler(mockService)

	g := e.Group("/auth")
	handler.RegisterRoutes(g)

	// Verify routes are registered by checking that they exist
	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/auth/register/"])
	assert.True(t, routePaths["/auth/login/"])
	assert.True(t, routePaths["/auth/profile/"])
	assert.True(t, routePaths["/auth/activate/:token"])
	assert.True(t, routePaths["/auth/oauth/"])
	assert.True(t, routePaths["/auth/change-password/"])
}
