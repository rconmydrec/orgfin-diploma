package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRequireInternalToken_ValidToken(t *testing.T) {
	mw := RequireInternalToken("secret-token-123")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireInternalToken_InvalidToken(t *testing.T) {
	mw := RequireInternalToken("secret-token-123")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRequireInternalToken_MissingHeader(t *testing.T) {
	mw := RequireInternalToken("secret-token-123")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRequireInternalToken_EmptyConfigToken(t *testing.T) {
	mw := RequireInternalToken("")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRequireInternalToken_MalformedHeader_NoBearer(t *testing.T) {
	mw := RequireInternalToken("secret-token-123")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Basic secret-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}

func TestRequireInternalToken_MalformedHeader_TokenOnly(t *testing.T) {
	mw := RequireInternalToken("secret-token-123")

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "secret-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unauthorized")
}
