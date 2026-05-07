package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authhandler "github.com/go-budget/backend/internal/handlers/auth"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

// MockGoogleValidator for testing OAuth
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

// TestRegisterEndpoint tests for POST /auth/register/
func TestRegisterSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Cleanup before and after test
	testutil.CleanupUserByEmail(t, app.DB, "newuser@example.com")
	defer testutil.CleanupUserByEmail(t, app.DB, "newuser@example.com")

	// Create request
	reqBody := `{
		"email": "newuser@example.com",
		"password": "Secure_password_123",
		"first_name": "New",
		"last_name": "User"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Assert status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Assert response fields
	if response["email"] != "newuser@example.com" {
		t.Errorf("Expected email 'newuser@example.com', got '%v'", response["email"])
	}
	if response["firstName"] != "New" {
		t.Errorf("Expected firstName 'New', got '%v'", response["firstName"])
	}
	if response["lastName"] != "User" {
		t.Errorf("Expected lastName 'User', got '%v'", response["lastName"])
	}
	if _, ok := response["id"]; !ok {
		t.Error("Expected 'id' field in response")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "duplicate@example.com"

	// Cleanup and create existing user
	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, "Test_password_123")

	// Try to register with same email
	reqBody := `{
		"email": "duplicate@example.com",
		"password": "Secure_password_123",
		"first_name": "Another",
		"last_name": "User"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterInvalidEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "invalid-email",
		"password": "Secure_password_123",
		"first_name": "Test",
		"last_name": "User"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	testutil.CleanupUserByEmail(t, app.DB, "weakpass@example.com")
	defer testutil.CleanupUserByEmail(t, app.DB, "weakpass@example.com")

	// API accepts short passwords (min 3 chars per validation)
	reqBody := `{
		"email": "weakpass@example.com",
		"password": "123",
		"first_name": "Test",
		"last_name": "User"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// API does not validate password strength beyond min length
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRegisterMissingRequiredFields(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Missing password
	reqBody := `{
		"email": "test@example.com"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterEmptyEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "",
		"password": "Secure_password_123",
		"first_name": "Test",
		"last_name": "User"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterWithoutOptionalFields(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	testutil.CleanupUserByEmail(t, app.DB, "minimal@example.com")
	defer testutil.CleanupUserByEmail(t, app.DB, "minimal@example.com")

	reqBody := `{
		"email": "minimal@example.com",
		"password": "Secure_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["email"] != "minimal@example.com" {
		t.Errorf("Expected email 'minimal@example.com', got '%v'", response["email"])
	}
}

// TestLoginEndpoint tests for POST /auth/login/
func TestLoginSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "logintest@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)

	reqBody := `{
		"email": "logintest@example.com",
		"password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["accessToken"]; !ok {
		t.Error("Expected 'accessToken' field in response")
	}
	if response["tokenType"] != "bearer" {
		t.Errorf("Expected tokenType 'bearer', got '%v'", response["tokenType"])
	}
}

func TestLoginWrongPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "wrongpass@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)

	reqBody := `{
		"email": "wrongpass@example.com",
		"password": "wrong_password"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "nonexistent@example.com",
		"password": "any_password"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestLoginInvalidEmailFormat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "not-an-email",
		"password": "password123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginMissingPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "test@example.com"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create user but don't activate
	_, err := app.AuthService.Register(email, password, "", "")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	// User is inactive by default

	reqBody := `{
		"email": "inactive@example.com",
		"password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

// TestProfileEndpoint tests for GET /auth/profile/
func TestGetProfileSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "profile@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["email"] != email {
		t.Errorf("Expected email '%s', got '%v'", email, response["email"])
	}
	if _, ok := response["baseCurrency"]; !ok {
		t.Error("Expected 'baseCurrency' field in response")
	}
}

func TestGetProfileUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Without token, should return 422 (missing header) or 401
	if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d or %d, got %d. Body: %s",
			http.StatusUnprocessableEntity, http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetProfileInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", "invalid_token_here")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetProfileExpiredToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Create an obviously expired/invalid token
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiZXhwIjoxfQ.invalid"

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", expiredToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetProfileInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "profile-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetProfileDeletedUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "profile-deleted@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Soft-delete user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_deleted = true WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

// TestActivateEndpoint tests for GET /auth/activate/{token}
func TestActivateSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "toactivate@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create inactive user with activation token
	result, err := app.AuthService.Register(email, password, "Test", "User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// User should have activation token
	activationToken := result.ActivationToken

	req := httptest.NewRequest(http.MethodGet, "/auth/activate/"+activationToken, nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify response is true
	var response bool
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !response {
		t.Error("Expected response to be true")
	}

	// Verify user is now active
	var isActive bool
	err = app.DB.QueryRow("SELECT is_active FROM users WHERE id = $1", result.User.ID).Scan(&isActive)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if !isActive {
		t.Error("Expected user to be active after activation")
	}
}

func TestActivateInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	req := httptest.NewRequest(http.MethodGet, "/auth/activate/invalid_token_12345", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestActivateExpiredToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "expired_token@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create user
	result, err := app.AuthService.Register(email, password, "", "")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Manually expire the token
	_, err = app.DB.Exec(
		"UPDATE activation_tokens SET expires_at = NOW() - INTERVAL '1 hour' WHERE user_id = $1",
		result.User.ID,
	)
	if err != nil {
		t.Fatalf("Failed to expire token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/activate/"+result.ActivationToken, nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestActivateDeletedUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "activate-deleted@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create inactive user with activation token
	result, err := app.AuthService.Register(email, password, "Test", "User")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Soft-delete the user before activation
	_, err = app.DB.Exec("UPDATE users SET is_deleted = true WHERE id = $1", result.User.ID)
	if err != nil {
		t.Fatalf("Failed to soft-delete user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/activate/"+result.ActivationToken, nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

// TestChangePasswordEndpoint tests for POST /auth/change-password/
func TestChangePasswordSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "changepass@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"current_password": "Test_password_123",
		"new_password": "New_secure_password_456"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	// Verify can login with new password
	loginReqBody := `{
		"email": "changepass@example.com",
		"password": "New_secure_password_456"
	}`
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(loginReqBody))
	loginReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	loginRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Errorf("Expected to login with new password, got status %d. Body: %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "wrongcurrent@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"current_password": "wrong_current_password",
		"new_password": "New_secure_password_456"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"current_password": "old_password",
		"new_password": "new_password"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Without token, should return 422 or 401
	if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d or %d, got %d. Body: %s",
			http.StatusUnprocessableEntity, http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordSameAsCurrent(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "samepass@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"current_password": "Test_password_123",
		"new_password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// API accepts same password
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

// ==================== Additional Auth Tests ====================

func TestRegisterInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "invalidjson@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := ``

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := ``

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "emptybody@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := ``

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginEmptyEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "",
		"password": "password123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestLoginEmptyPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "test@example.com",
		"password": ""
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterEmptyPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{
		"email": "test@example.com",
		"password": ""
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestRegisterVeryLongEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Very long email
	longEmail := strings.Repeat("a", 200) + "@example.com"
	testutil.CleanupUserByEmail(t, app.DB, longEmail)
	defer testutil.CleanupUserByEmail(t, app.DB, longEmail)

	reqBody := `{
		"email": "` + longEmail + `",
		"password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// API accepts long emails
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestGetProfileWithExpiredToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Malformed JWT
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

	req := httptest.NewRequest(http.MethodGet, "/auth/profile/", nil)
	req.Header.Set("auth-token", expiredToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordMissingCurrentPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "missingcurrent@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"new_password": "New_password_456"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordMissingNewPassword(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "missingnew@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"current_password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "changepass-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	reqBody := `{
		"current_password": "Test_password_123",
		"new_password": "New_secure_password_456"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestActivateEmptyToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	req := httptest.NewRequest(http.MethodGet, "/auth/activate/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Empty token path - requires auth middleware
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestActivateAlreadyActivatedUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "alreadyactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create and activate user
	result, err := app.AuthService.Register(email, password, "", "")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	activationToken := result.ActivationToken

	// First activation
	req1 := httptest.NewRequest(http.MethodGet, "/auth/activate/"+activationToken, nil)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("First activation should succeed, got %d", rec1.Code)
	}

	// Second activation with same token
	req2 := httptest.NewRequest(http.MethodGet, "/auth/activate/"+activationToken, nil)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)

	// Should fail - token already used
	if rec2.Code != http.StatusNotFound && rec2.Code != http.StatusBadRequest {
		t.Errorf("Second activation should fail, got %d", rec2.Code)
	}
}

func TestRegisterWithSpecialCharactersInName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	testutil.CleanupUserByEmail(t, app.DB, "specialname@example.com")
	defer testutil.CleanupUserByEmail(t, app.DB, "specialname@example.com")

	reqBody := `{
		"email": "specialname@example.com",
		"password": "Secure_password_123",
		"first_name": "José",
		"last_name": "García"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/register/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["firstName"] != "José" {
		t.Errorf("Expected firstName 'José', got '%v'", response["firstName"])
	}
}

func TestLoginCaseSensitiveEmail(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	email := "casetest@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)

	// Try login with uppercase email
	reqBody := `{
		"email": "CASETEST@EXAMPLE.COM",
		"password": "Test_password_123"
	}`

	req := httptest.NewRequest(http.MethodPost, "/auth/login/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// May or may not be case-sensitive - just document behavior
	// Most email systems are case-insensitive
	if rec.Code != http.StatusOK && rec.Code != http.StatusUnauthorized {
		t.Errorf("Unexpected status %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ==================== OAuth Tests ====================

func TestOAuthMissingCredential(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Missing credential field
	reqBody := `{}`

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestOAuthInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestOAuthEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := ``

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestOAuthEmptyCredential(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	reqBody := `{"credential": ""}`

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestOAuthInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAuthRoutes()

	// Invalid/fake credential - will fail Google validation
	reqBody := `{"credential": "invalid.google.token"}`

	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should fail with unauthorized because token validation will fail
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

// ==================== OAuth Tests with Mock ====================

func setupOAuthTest(t *testing.T, mockValidator *MockGoogleValidator) (*testutil.TestApp, *echo.Echo) {
	t.Helper()
	app := testutil.NewTestApp(t)

	// Create handler with mock validator
	authHandler := authhandler.NewWithValidator(
		app.AuthService,
		testutil.TestConfig.GoogleClientID,
		mockValidator,
		app.Logger,
	)

	// Setup routes with mocked handler
	e := echo.New()
	e.Validator = app.Echo.Validator
	authGroup := e.Group("/auth")
	authGroup.POST("/oauth/", authHandler.OAuth)

	return app, e
}

func TestOAuthSuccessNewUser(t *testing.T) {
	email := "oauth-new@example.com"

	mockValidator := &MockGoogleValidator{
		Email:         email,
		EmailVerified: true,
		FirstName:     "OAuth",
		LastName:      "User",
	}

	app, e := setupOAuthTest(t, mockValidator)

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	reqBody := `{"credential": "valid-google-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["accessToken"]; !ok {
		t.Error("Expected 'accessToken' field in response")
	}
	if response["tokenType"] != "bearer" {
		t.Errorf("Expected tokenType 'bearer', got '%v'", response["tokenType"])
	}
}

func TestOAuthSuccessExistingUser(t *testing.T) {
	email := "oauth-existing@example.com"
	password := "Test_password_123"

	app, e := setupOAuthTest(t, &MockGoogleValidator{
		Email:         email,
		EmailVerified: true,
		FirstName:     "Existing",
		LastName:      "User",
	})

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create existing user
	testutil.CreateTestUser(t, app, email, password)

	reqBody := `{"credential": "valid-google-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["accessToken"]; !ok {
		t.Error("Expected 'accessToken' field in response")
	}
}

func TestOAuthEmailNotVerified(t *testing.T) {
	app, e := setupOAuthTest(t, &MockGoogleValidator{
		Email:         "unverified@example.com",
		EmailVerified: false,
	})
	_ = app

	reqBody := `{"credential": "valid-google-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["detail"] != "Email not verified" {
		t.Errorf("Expected 'Email not verified', got '%v'", response["detail"])
	}
}

func TestOAuthNoEmail(t *testing.T) {
	app, e := setupOAuthTest(t, &MockGoogleValidator{
		Email:         "",
		EmailVerified: true,
	})
	_ = app

	reqBody := `{"credential": "valid-google-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestOAuthGoogleValidationFails(t *testing.T) {
	app, e := setupOAuthTest(t, &MockGoogleValidator{
		ShouldFail: true,
		FailError:  errors.New("token expired"),
	})
	_ = app

	reqBody := `{"credential": "expired-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestOAuthInactiveUser(t *testing.T) {
	email := "oauth-inactive@example.com"
	password := "Test_password_123"

	app, e := setupOAuthTest(t, &MockGoogleValidator{
		Email:         email,
		EmailVerified: true,
	})

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	// Create user but don't activate
	_, err := app.AuthService.Register(email, password, "", "")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	reqBody := `{"credential": "valid-google-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/oauth/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}
