package reports_test

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
	testEmail    = "reports-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Cash Flow Report Tests ====================

func TestCashFlowReportSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["currency"])
}

func TestCashFlowReportDailyPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "daily"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCashFlowReportUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01",
		"endDate": "2024-12-31",
		"period": "monthly"
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Balance Report Tests ====================

func TestBalanceReportSuccess(t *testing.T) {
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

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d],
		"balanceDate": "%s"
	}`, accountID, balanceDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(response), 1)
}

func TestBalanceReportAllAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [],
		"balanceDate": "%s"
	}`, balanceDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceReportUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"accountIds": [],
		"balanceDate": "2024-01-01"
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Balance Report Non-Hidden Tests ====================

func TestBalanceReportNonHiddenSuccess(t *testing.T) {
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

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d],
		"balanceDate": "%s"
	}`, accountID, balanceDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceReportNonHiddenUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"accountIds": [],
		"balanceDate": "2024-01-01"
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Expenses by Categories Tests ====================

func TestExpensesByCategoriesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"hideEmptyCategories": false
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
}

func TestExpensesByCategoriesHideEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"hideEmptyCategories": true
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01",
		"endDate": "2024-12-31",
		"hideEmptyCategories": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Diagram Tests ====================

func TestDiagramSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reports/diagram/pie/%s/%s", startDate, endDate), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDiagramUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/reports/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Expenses Data Tests ====================

func TestExpensesDataSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"hideEmptyCategories": false
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesDataUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"startDate": "2024-01-01",
		"endDate": "2024-12-31",
		"hideEmptyCategories": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Report Tests ====================

func TestCashFlowReportWithTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account and transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["totalExpenses"])
}

func TestCashFlowReportMissingPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceReportMissingDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"accountIds": []
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestExpensesByCategoriesMissingDates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"hideEmptyCategories": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestExpensesDataMissingDates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"hideEmptyCategories": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDiagramWithTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create category, account and transaction
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &categoryID, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reports/diagram/pie/%s/%s", startDate, endDate), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["labels"])
	assert.NotNil(t, response["data"])
}

func TestBalanceReportWithMultipleAccounts(t *testing.T) {
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

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d, %d],
		"balanceDate": "%s"
	}`, account1ID, account2ID, balanceDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, len(response))
}

func TestCashFlowReportInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceReportInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== Additional Coverage Tests ====================

func TestExpensesByCategoriesInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestExpensesDataInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceReportNonHiddenInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceReportNonHiddenMissingDate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"accountIds": []}`

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCashFlowReportWithIncomeTransaction(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account and income transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create income transaction (is_income = true)
	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 500, true)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["totalIncome"])
}

func TestCashFlowReportDefaultPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().Format(time.RFC3339)

	// Use unknown period - should default to monthly
	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "unknown"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesWithTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create category, account and expense transaction
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Shopping", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create expense transaction
	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &categoryID, 75, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"hideEmptyCategories": false
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have at least one category with expenses
	assert.GreaterOrEqual(t, len(response), 1)
}

func TestExpensesDataWithTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create category, account and expense transaction
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Utilities", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create expense transaction
	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &categoryID, 120, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"hideEmptyCategories": false
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should have at least one expense item
	assert.GreaterOrEqual(t, len(response), 1)
}

func TestBalanceReportExcludesHiddenAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create visible account
	visibleAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Visible Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, visibleAccountID)

	// Create hidden account
	hiddenAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Hidden Wallet", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, hiddenAccountID)

	// Set account as hidden
	_, err := app.DB.Exec("UPDATE accounts SET is_hidden = true WHERE id = $1", hiddenAccountID)
	require.NoError(t, err)

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [],
		"balanceDate": "%s"
	}`, balanceDate)

	// Non-hidden endpoint should exclude hidden accounts
	req := httptest.NewRequest(http.MethodPost, "/reports/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should only have the visible account
	for _, acc := range response {
		assert.NotEqual(t, "Hidden Wallet", acc["accountName"])
	}
}

func TestBalanceReportIncludesHiddenAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create hidden account
	hiddenAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Hidden Savings", currencyID, accountTypeID, 5000)
	defer testutil.DeleteTestAccount(t, app.DB, hiddenAccountID)

	// Set account as hidden
	_, err := app.DB.Exec("UPDATE accounts SET is_hidden = true WHERE id = $1", hiddenAccountID)
	require.NoError(t, err)

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d],
		"balanceDate": "%s"
	}`, hiddenAccountID, balanceDate)

	// Regular balance endpoint should include hidden accounts
	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should include the hidden account
	assert.Equal(t, 1, len(response))
	assert.Equal(t, "Hidden Savings", response[0]["accountName"])
}

func TestCashFlowReportNoDates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Request without dates (should still work, gets all transactions)
	body := `{"period": "monthly"}`

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesWithSpecificCategories(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create categories
	cat1ID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat1ID)

	cat2ID := testutil.CreateTestCategory(t, app.DB, userID, "Transport", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat2ID)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	// Request with specific categories filter
	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d],
		"hideEmptyCategories": false
	}`, startDate, endDate, cat1ID)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDiagramBarChart(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	// Test with bar chart type
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reports/diagram/bar/%s/%s", startDate, endDate), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["currency"])
}

func TestCashFlowReportWithMixedTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create income transaction
	incomeTxID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 1000, true)
	defer testutil.DeleteTestTransaction(t, app.DB, incomeTxID)

	// Create expense transaction
	expenseTxID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 250, false)
	defer testutil.DeleteTestTransaction(t, app.DB, expenseTxID)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["totalIncome"])
	assert.NotNil(t, response["totalExpenses"])
	assert.NotNil(t, response["netFlow"])
}

func TestBalanceReportResponseStructure(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Checking", currencyID, accountTypeID, 2500)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	balanceDate := time.Now().Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d],
		"balanceDate": "%s"
	}`, accountID, balanceDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	require.Equal(t, 1, len(response))

	// Check all required fields in response
	item := response[0]
	assert.NotNil(t, item["accountId"])
	assert.NotNil(t, item["accountName"])
	assert.NotNil(t, item["accountTypeId"])
	assert.NotNil(t, item["currencyCode"])
	assert.NotNil(t, item["balance"])
	assert.NotNil(t, item["baseCurrencyBalance"])
	assert.NotNil(t, item["baseCurrencyCode"])
	assert.NotNil(t, item["reportDate"])
}

// ==================== Bug 3: Deleted Account Exclusion Tests ====================

func TestCashFlowReportExcludesDeletedAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account and transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Deleted Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Soft-delete the account
	_, err := app.DB.Exec("UPDATE accounts SET is_deleted = true WHERE id = $1", accountID)
	require.NoError(t, err)

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// totalExpenses should be empty since the only account is deleted
	expenses, ok := response["totalExpenses"].(map[string]interface{})
	assert.True(t, ok)
	assert.Empty(t, expenses)
}

// ==================== Bug 4: endDate Inclusive Integration Tests ====================

func TestCashFlowReportEndDateInclusive(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create transaction at 15:00 today
	today := time.Now()
	txTime := time.Date(today.Year(), today.Month(), today.Day(), 15, 0, 0, 0, today.Location())
	var txID int
	err := app.DB.QueryRow(`
		INSERT INTO transactions (user_id, account_id, amount, is_income, is_transfer, is_deleted, date_time, new_balance)
		VALUES ($1, $2, $3, false, false, false, $4, $3)
		RETURNING id
	`, userID, accountID, 200, txTime).Scan(&txID)
	require.NoError(t, err)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Use today as the endDate
	todayStr := today.Format("2006-01-02")
	startDate := today.AddDate(0, -1, 0).Format("2006-01-02")

	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"period": "monthly"
	}`, startDate, todayStr)

	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// The transaction at 15:00 today should be included
	expenses, ok := response["totalExpenses"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, expenses, "Transaction at 15:00 on endDate should be included")
}

// ==================== Bug 6: Historical Balance Integration Test ====================

func TestBalanceReportHistoricalBalanceIntegration(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account with balance 800
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Wallet", currencyID, accountTypeID, 800)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Add expense of 200 yesterday (so historical balance 2 days ago should be 800+200=1000)
	yesterday := time.Now().AddDate(0, 0, -1)
	var txID int
	err := app.DB.QueryRow(`
		INSERT INTO transactions (user_id, account_id, amount, is_income, is_transfer, is_deleted, date_time, new_balance)
		VALUES ($1, $2, $3, false, false, false, $4, $3)
		RETURNING id
	`, userID, accountID, 200, yesterday).Scan(&txID)
	require.NoError(t, err)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Request balance for 2 days ago
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

	body := fmt.Sprintf(`{
		"accountIds": [%d],
		"balanceDate": "%s"
	}`, accountID, twoDaysAgo)

	req := httptest.NewRequest(http.MethodPost, "/reports/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response))

	// Balance at 2 days ago should be 800 + 200 = 1000 (reversing the expense)
	balance := response[0]["balance"].(float64)
	assert.Equal(t, 1000.0, balance)
}

// ==================== Bug 8: Categories Filter Integration Test ====================

func TestExpensesByCategoriesFilterIntegration(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create two categories
	cat1ID := testutil.CreateTestCategory(t, app.DB, userID, "CategoryA", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat1ID)
	cat2ID := testutil.CreateTestCategory(t, app.DB, userID, "CategoryB", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat2ID)

	// Create account and transactions
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	tx1ID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat1ID, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx1ID)
	tx2ID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat2ID, 200, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx2ID)

	startDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	// Request only cat1ID
	body := fmt.Sprintf(`{
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d],
		"hideEmptyCategories": true
	}`, startDate, endDate, cat1ID)

	req := httptest.NewRequest(http.MethodPost, "/reports/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Only CategoryA should be present (not CategoryB)
	for _, item := range response {
		assert.NotEqual(t, "CategoryB", item["name"])
	}
}

// ==================== Bug 1: Default Date Range Integration Test ====================

func TestCashFlowReportNoDatesWithDefaultRange(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create account and transaction within default window
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Request without dates -- should apply default range and still return 200
	body := `{"period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/reports/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// The transaction created "now" should be within the default 12-month window
	expenses, ok := response["totalExpenses"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, expenses, "Transaction should be included in the default date range")
}
