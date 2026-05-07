package financial_planning_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail    = "fin-planning-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Future Balance Tests ====================

func TestFutureBalanceSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"targetDate": "%s"
	}`, targetDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["targetDate"])
	assert.NotNil(t, response["baseCurrencyCode"])
}

func TestFutureBalanceWithAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create an account
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"targetDate": "%s",
		"accountIds": [%d]
	}`, targetDate, accountID)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["accounts"])
}

func TestFutureBalanceUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"targetDate": "2024-12-31T00:00:00Z"
	}`

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Balance Projection Tests ====================

func TestBalanceProjectionSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "daily"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["startDate"])
	assert.NotNil(t, response["endDate"])
	assert.NotNil(t, response["period"])
	assert.NotNil(t, response["projectionPoints"])
}

func TestBalanceProjectionWeekly(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 2, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "weekly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "weekly", response["period"])
}

func TestBalanceProjectionMonthly(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 6, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "monthly", response["period"])
}

func TestBalanceProjectionInvalidDates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// End date before start date
	startDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "daily"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBalanceProjectionUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"endDate": "2024-12-31T00:00:00Z",
		"period": "daily"
	}`

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Financial Planning Tests ====================

func TestFutureBalanceMissingDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestFutureBalanceInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceProjectionInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceProjectionMissingEndDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"period": "daily"
	}`, startDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestFutureBalanceWithMultipleAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	account1ID := testutil.CreateTestAccount(t, app.DB, userID, "Wallet 1", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, account1ID)

	account2ID := testutil.CreateTestAccount(t, app.DB, userID, "Wallet 2", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, account2ID)

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"targetDate": "%s",
		"accountIds": [%d, %d]
	}`, targetDate, account1ID, account2ID)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	accounts := response["accounts"].([]interface{})
	assert.Equal(t, 2, len(accounts))
}

func TestBalanceProjectionWithAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 14).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "daily",
		"accountIds": [%d]
	}`, startDate, endDate, accountID)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	points := response["projectionPoints"].([]interface{})
	assert.GreaterOrEqual(t, len(points), 1)
}

func TestFutureBalanceWithIncludeInactive(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"targetDate": "%s",
		"includeInactive": true
	}`, targetDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceProjectionDefaultPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"endDate": "%s"
	}`, endDate)

	req := httptest.NewRequest(http.MethodPost, "/financial-planning/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Default should be daily
	assert.Equal(t, "daily", response["period"])
}
