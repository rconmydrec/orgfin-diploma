package exchange_rates_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail      = "exchange-rates-test@example.com"
	testAdminEmail = testutil.AdminTestEmail
	testPassword   = "TestPass123!"
)

// ==================== Get Rates Tests ====================

func TestGetRatesAuthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// The endpoint should respond (200 or 500 if table doesn't exist)
	// Main thing is we're not getting 401 Unauthorized
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestGetRatesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Rates Tests ====================

func TestUpdateRatesEnqueuesTask(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// The endpoint now enqueues a task and returns 200
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "Exchange rate update task enqueued")
}

func TestUpdateRatesNonAdmin(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Non-admin user should be forbidden
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateRatesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Rates Range Tests ====================

func TestUpdateRatesRangeNoAPIKey(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/"+startDate+"/to/"+endDate, nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Without a CurrencyBeacon API key configured, this returns 500
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateRatesRangeNonAdmin(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/"+startDate+"/to/"+endDate, nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Non-admin user should be forbidden
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateRatesRangeInvalidStartDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/invalid-date/to/2024-12-31", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateRatesRangeInvalidEndDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/2024-01-01/to/invalid-date", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateRatesRangeUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/2024-01-01/to/2024-12-31", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Exchange Rate Tests ====================

func TestGetRatesWithToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should get OK or error (depending on table existence)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateRatesRangeInvalidDateFormat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	// Wrong date format (DD-MM-YYYY instead of YYYY-MM-DD)
	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/01-01-2024/to/31-12-2024", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateRatesRangeEmptyDates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from//to//", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should fail with 404 (route doesn't match) or 400
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestGetRatesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/", nil)
	req.Header.Set("auth-token", "invalid-token-123")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateRatesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update", nil)
	req.Header.Set("auth-token", "invalid-token-123")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateRatesRangeEndBeforeStartIntegration(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/2024-12-31/to/2024-01-01", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["detail"], "end_date must not be before start_date")
}

func TestUpdateRatesRangeExceedsMaxDaysIntegration(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testAdminEmail)
	userID := testutil.CreateTestUser(t, app, testAdminEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testAdminEmail, testPassword)

	// 366 days exceeds the 365-day limit
	req := httptest.NewRequest(http.MethodGet, "/exchange-rates/update/from/2023-01-01/to/2024-01-01", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["detail"], "Date range must not exceed 365 days")
}
