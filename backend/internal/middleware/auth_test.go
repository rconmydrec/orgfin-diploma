package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-budget/backend/internal/auth"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockTokenGenerator implements TokenGenerator for testing.
// It delegates to a real JWTService for validation, but allows
// overriding GenerateToken to simulate failures.
type mockTokenGenerator struct {
	realService      *auth.JWTService
	generateTokenErr error
}

func (m *mockTokenGenerator) GenerateToken(userID int, email string) (string, error) {
	if m.generateTokenErr != nil {
		return "", m.generateTokenErr
	}
	return m.realService.GenerateToken(userID, email)
}

func (m *mockTokenGenerator) ValidateToken(tokenString string) (*auth.Claims, error) {
	return m.realService.ValidateToken(tokenString)
}

func setupAuthMiddleware() (*AuthMiddleware, *auth.JWTService) {
	jwtService := auth.NewJWTService("test-secret-key-for-middleware", 60)
	mw := NewAuthMiddleware(jwtService, testLogger())
	return mw, jwtService
}

func TestRequireAuth_MissingToken(t *testing.T) {
	mw, _ := setupAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Missing authorization header")
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	mw, _ := setupAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid token")
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	// Create a JWT service with 0 expiration to generate an expired token
	expiredJWTService := auth.NewJWTService("test-secret-key-for-middleware", -1)
	token, err := expiredJWTService.GenerateToken(1, "test@example.com")
	assert.NoError(t, err)

	mw, _ := setupAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Token has expired")
}

func TestRequireAuth_ValidTokenSetsNewAccessTokenHeader(t *testing.T) {
	mw, jwtService := setupAuthMiddleware()

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		// Verify context values are set
		assert.Equal(t, 42, c.Get("user_id"))
		assert.Equal(t, "user@example.com", c.Get("email"))
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify new_access_token header is set on 2xx response
	newToken := rec.Header().Get("new_access_token")
	assert.NotEmpty(t, newToken, "Expected new_access_token header to be set on 2xx response")

	// Verify the new token is valid
	claims, err := jwtService.ValidateToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, 42, claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
}

func TestRequireAuth_NoRefreshOnErrorResponse(t *testing.T) {
	mw, jwtService := setupAuthMiddleware()

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.RequireAuth(func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{"detail": "not found"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Verify new_access_token header is NOT set on 4xx response
	newToken := rec.Header().Get("new_access_token")
	assert.Empty(t, newToken, "Expected new_access_token header NOT to be set on 4xx response")
}

func TestRequireAuth_NoRefreshOn500Response(t *testing.T) {
	mw, jwtService := setupAuthMiddleware()

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.RequireAuth(func(c echo.Context) error {
		return c.JSON(http.StatusInternalServerError, map[string]string{"detail": "server error"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify new_access_token header is NOT set on 5xx response
	newToken := rec.Header().Get("new_access_token")
	assert.Empty(t, newToken, "Expected new_access_token header NOT to be set on 5xx response")
}

func TestRequireAuth_NoRefreshWhenHandlerReturnsError(t *testing.T) {
	mw, jwtService := setupAuthMiddleware()

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.RequireAuth(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	err = handler(c)
	// Handler returned an error, so the middleware should propagate it
	assert.Error(t, err)

	// Verify new_access_token header is NOT set when handler returns error
	newToken := rec.Header().Get("new_access_token")
	assert.Empty(t, newToken, "Expected new_access_token header NOT to be set when handler returns error")
}

func TestRequireAuth_RefreshOnCreatedResponse(t *testing.T) {
	mw, jwtService := setupAuthMiddleware()

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := mw.RequireAuth(func(c echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{"created": "true"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify new_access_token header IS set on 201 response (2xx range)
	newToken := rec.Header().Get("new_access_token")
	assert.NotEmpty(t, newToken, "Expected new_access_token header to be set on 201 response")
}

func TestRequireAuth_GracefulDegradationOnTokenGenerationFailure(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-for-middleware", 60)
	mock := &mockTokenGenerator{
		realService:      jwtService,
		generateTokenErr: errors.New("signing key unavailable"),
	}
	mw := NewAuthMiddleware(mock, testLogger())

	token, err := jwtService.GenerateToken(42, "user@example.com")
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)

	// When GenerateToken fails, the response should still succeed (2xx)
	// but the new_access_token header should NOT be set.
	newToken := rec.Header().Get("new_access_token")
	assert.Empty(t, newToken, "Expected new_access_token header NOT to be set when token generation fails")
}

func TestRequireAuth_TokenWithDifferentSecret(t *testing.T) {
	// Generate a token with a different secret
	otherJWTService := auth.NewJWTService("different-secret", 60)
	token, err := otherJWTService.GenerateToken(1, "test@example.com")
	assert.NoError(t, err)

	mw, _ := setupAuthMiddleware()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := mw.RequireAuth(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid token")
}
