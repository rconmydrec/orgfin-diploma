package management_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// Test user credentials
const (
	testEmail    = "management-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Backup Tests ====================

func TestBackupUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBackupNonAdmin(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Regular user should be forbidden
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestBackupInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	req.Header.Set("auth-token", "invalid-token-123")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBackupEmptyToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	req.Header.Set("auth-token", "")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Empty token should fail
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestBackupMultipleNonAdminUsers(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "management-test1@example.com"
	email2 := "management-test2@example.com"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	userID1 := testutil.CreateTestUser(t, app, email1, testPassword)
	userID2 := testutil.CreateTestUser(t, app, email2, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token1 := testutil.GetAuthToken(t, app, email1, testPassword)
	token2 := testutil.GetAuthToken(t, app, email2, testPassword)

	// First user
	req1 := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	req1.Header.Set("auth-token", token1)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusForbidden, rec1.Code)

	// Second user
	req2 := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	req2.Header.Set("auth-token", token2)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusForbidden, rec2.Code)
}

func TestBackupRequiresAuth(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Test without any auth header
	req := httptest.NewRequest(http.MethodGet, "/management/backup/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBackupMethodNotAllowed(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Try POST instead of GET
	req := httptest.NewRequest(http.MethodPost, "/management/backup/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should fail - either method not allowed or not found
	assert.NotEqual(t, http.StatusOK, rec.Code)
}
