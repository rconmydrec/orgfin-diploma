package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRequireAdmin_AdminUserProceeds(t *testing.T) {
	mw := RequireAdmin([]string{"admin@example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", "admin@example.com")

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAdmin_NonAdminUserReturns403(t *testing.T) {
	mw := RequireAdmin([]string{"admin@example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", "user@example.com")

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin access required")
}

func TestRequireAdmin_NoActiveUserInContext(t *testing.T) {
	mw := RequireAdmin([]string{"admin@example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Do not set "active_user_email" in context

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin access required")
}

func TestRequireAdmin_WrongTypeForActiveUserEmail(t *testing.T) {
	mw := RequireAdmin([]string{"admin@example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", 12345) // wrong type

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin access required")
}

func TestRequireAdmin_EmptyAdminEmailsList(t *testing.T) {
	mw := RequireAdmin([]string{})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", "admin@example.com")

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin access required")
}

func TestRequireAdmin_MultipleAdminEmails(t *testing.T) {
	mw := RequireAdmin([]string{"admin1@example.com", "admin2@example.com", "admin3@example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", "admin2@example.com")

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAdmin_CaseSensitive(t *testing.T) {
	mw := RequireAdmin([]string{"Admin@Example.com"})

	handlerCalled := false
	handler := mw(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("active_user_email", "admin@example.com")

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin access required")
}
