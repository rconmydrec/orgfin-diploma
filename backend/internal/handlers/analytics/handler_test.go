package analytics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Test user credentials
const (
	testEmail    = "analytics-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Spending Trends Tests ====================

func TestSpendingTrendsNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := `{
		"startDate": "` + startDate + `",
		"endDate": "` + endDate + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// This endpoint is currently disabled and returns 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSpendingTrendsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T00:00:00Z"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Expense Categorization Tests ====================

func TestExpenseCategorizationNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := `{
		"startDate": "` + startDate + `",
		"endDate": "` + endDate + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// This endpoint is currently disabled and returns 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExpenseCategorizationUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T00:00:00Z"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Analytics Tests ====================

func TestSpendingTrendsInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Even with invalid JSON, endpoint is disabled so returns 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExpenseCategorizationInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Even with invalid JSON, endpoint is disabled so returns 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSpendingTrendsInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T00:00:00Z"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestExpenseCategorizationInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T00:00:00Z"
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSpendingTrendsEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := ``

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Endpoint is disabled
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExpenseCategorizationEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := ``

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Endpoint is disabled
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSpendingTrendsWithAccountID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := `{
		"startDate": "` + startDate + `",
		"endDate": "` + endDate + `",
		"accountId": 1
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/spending-trends", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Endpoint is disabled
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExpenseCategorizationWithAccountID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := `{
		"startDate": "` + startDate + `",
		"endDate": "` + endDate + `",
		"accountId": 1
	}`

	req := httptest.NewRequest(http.MethodPost, "/analytics/expense-categorization", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Endpoint is disabled
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
