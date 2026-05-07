package transactions_test

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
	testEmail    = "tx-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Create Transaction Tests ====================

func TestCreateExpenseTransaction(t *testing.T) {
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

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 50.00,
		"label": "Groceries",
		"isIncome": false
	}`, accountID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["id"])
	assert.Equal(t, float64(accountID), response["accountId"])
	assert.Equal(t, float64(50), response["amount"])
	assert.Equal(t, false, response["isIncome"])
	assert.Equal(t, "Groceries", response["label"])
}

func TestCreateIncomeTransaction(t *testing.T) {
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

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Salary", true)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 5000.00,
		"label": "Monthly Salary",
		"isIncome": true
	}`, accountID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isIncome"])
	assert.Equal(t, float64(5000), response["amount"])
}

func TestCreateTransferTransaction(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	sourceAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Source Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, sourceAccountID)

	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Target Wallet", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"label": "Transfer to savings",
		"isTransfer": true
	}`, sourceAccountID, targetAccountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isTransfer"])
	assert.NotNil(t, response["linkedTransactionId"])
}

func TestCreateTransferDifferentCurrencies(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	usdCurrencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	eurCurrencyID := testutil.GetCurrencyID(t, app.DB, "EUR")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	usdAccountID := testutil.CreateTestAccount(t, app.DB, userID, "USD Account", usdCurrencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, usdAccountID)

	eurAccountID := testutil.CreateTestAccount(t, app.DB, userID, "EUR Account", eurCurrencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, eurAccountID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"targetAmount": 92.00,
		"label": "USD to EUR transfer",
		"isTransfer": true
	}`, usdAccountID, eurAccountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isTransfer"])
	assert.Equal(t, float64(100), response["amount"])
}

func TestCreateTransactionWithNotes(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 75.50,
		"label": "Restaurant",
		"notes": "Dinner with friends at Italian place",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Dinner with friends at Italian place", response["notes"])
}

func TestCreateTransactionWithDateTime(t *testing.T) {
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

	specificDate := "2024-06-15T14:30:00Z"
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "Past expense",
		"dateTime": "%s",
		"isIncome": false
	}`, accountID, specificDate)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["dateTime"])
}

func TestCreateTransactionExcludeFromReports(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 50.00,
		"label": "Hidden expense",
		"isIncome": false,
		"excludeFromReports": true
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["excludeFromReports"])
}

func TestCreateTransactionWithTemplate(t *testing.T) {
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

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Utilities", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 100.00,
		"label": "Electric Bill",
		"isIncome": false,
		"isTemplate": true
	}`, accountID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify template was created
	templatesReq := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	templatesReq.Header.Set("auth-token", token)
	templatesRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(templatesRec, templatesReq)

	assert.Equal(t, http.StatusOK, templatesRec.Code)

	var templates []map[string]interface{}
	err := json.Unmarshal(templatesRec.Body.Bytes(), &templates)
	require.NoError(t, err)

	found := false
	for _, tmpl := range templates {
		if tmpl["label"] == "Electric Bill" {
			found = true
			break
		}
	}
	assert.True(t, found, "Template should be created")
}

func TestCreateTransactionWithoutCategory(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 25.00,
		"label": "No category expense",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Nil(t, response["categoryId"])
}

func TestCreateTransactionInvalidAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"accountId": 999999,
		"amount": 50.00,
		"label": "Test",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransactionOtherUserAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with account
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID1, "User1 Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create second user
	otherEmail := "other-create@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	// Try to create transaction on first user's account
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 50.00,
		"label": "Hacked transaction",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should not be allowed
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestCreateTransactionMissingAmount(t *testing.T) {
	// Note: In Go, decimal.Decimal defaults to 0 when not provided in JSON
	// This is expected behavior - missing amount creates a zero-amount transaction
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"label": "No amount",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Missing amount defaults to 0 which is valid
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify amount is 0
	assert.Equal(t, float64(0), response["amount"])
}

func TestCreateTransactionMissingAccountId(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"amount": 50.00,
		"label": "No account",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransactionUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"accountId": 1,
		"amount": 50.00,
		"label": "Test",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateTransactionZeroAmount(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 0,
		"label": "Zero amount",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Zero amount might be allowed or not depending on business rules
	// Just verify it doesn't crash
	assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateTransactionDecimalAmount(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 123.456789,
		"label": "Decimal amount",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCreateTransactionCategoryIdAsString(t *testing.T) {
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

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// categoryId as string (FlexInt should handle this)
	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": "%d",
		"amount": 50.00,
		"label": "Test FlexInt",
		"isIncome": false
	}`, accountID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Get Transactions Tests ====================

func TestGetTransactionsList(t *testing.T) {
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

	txID1 := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID1)

	txID2 := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, true)
	defer testutil.DeleteTestTransaction(t, app.DB, txID2)

	req := httptest.NewRequest(http.MethodGet, "/transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(transactions), 2)
}

func TestGetTransactionsEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "empty-tx@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	assert.Equal(t, 0, len(transactions))
}

func TestGetTransactionsPagination(t *testing.T) {
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

	var txIDs []int
	for i := 0; i < 10; i++ {
		txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, float64(10*(i+1)), false)
		txIDs = append(txIDs, txID)
	}
	defer func() {
		for _, txID := range txIDs {
			testutil.DeleteTestTransaction(t, app.DB, txID)
		}
	}()

	// First page
	req := httptest.NewRequest(http.MethodGet, "/transactions/?page=1&per_page=5", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var page1 []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &page1)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(page1), 5)

	// Second page
	req2 := httptest.NewRequest(http.MethodGet, "/transactions/?page=2&per_page=5", nil)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestGetTransactionsFilterByAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	account1ID := testutil.CreateTestAccount(t, app.DB, userID, "Account 1", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, account1ID)

	account2ID := testutil.CreateTestAccount(t, app.DB, userID, "Account 2", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, account2ID)

	tx1ID := testutil.CreateTestTransaction(t, app.DB, userID, account1ID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx1ID)

	tx2ID := testutil.CreateTestTransaction(t, app.DB, userID, account2ID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx2ID)

	// Filter by account1
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/?accounts=%d", account1ID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	for _, tx := range transactions {
		assert.Equal(t, float64(account1ID), tx["accountId"])
	}
}

func TestGetTransactionsFilterByCategory(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	cat1ID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat1ID)

	cat2ID := testutil.CreateTestCategory(t, app.DB, userID, "Transport", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat2ID)

	tx1ID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat1ID, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx1ID)

	tx2ID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat2ID, 30, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx2ID)

	// Filter by category1
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/?categories=%d", cat1ID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	for _, tx := range transactions {
		if tx["categoryId"] != nil {
			assert.Equal(t, float64(cat1ID), tx["categoryId"])
		}
	}
}

func TestGetTransactionsFilterByDateRange(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Filter by date range
	fromDate := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	toDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/?from_date=%s&to_date=%s", fromDate, toDate), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(transactions), 1)
}

func TestGetTransactionsFilterByType(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create expense
	expenseID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, expenseID)

	// Create income
	incomeID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, true)
	defer testutil.DeleteTestTransaction(t, app.DB, incomeID)

	// Filter by type=expense
	req := httptest.NewRequest(http.MethodGet, "/transactions/?types=expense", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	for _, tx := range transactions {
		assert.Equal(t, false, tx["isIncome"])
	}
}

func TestGetTransactionsFilterByMultipleAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	account1ID := testutil.CreateTestAccount(t, app.DB, userID, "Account 1", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, account1ID)

	account2ID := testutil.CreateTestAccount(t, app.DB, userID, "Account 2", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, account2ID)

	account3ID := testutil.CreateTestAccount(t, app.DB, userID, "Account 3", currencyID, accountTypeID, 300)
	defer testutil.DeleteTestAccount(t, app.DB, account3ID)

	tx1ID := testutil.CreateTestTransaction(t, app.DB, userID, account1ID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx1ID)

	tx2ID := testutil.CreateTestTransaction(t, app.DB, userID, account2ID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx2ID)

	tx3ID := testutil.CreateTestTransaction(t, app.DB, userID, account3ID, nil, 75, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx3ID)

	// Filter by account1 and account2
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/?accounts=%d,%d", account1ID, account2ID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var transactions []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &transactions)
	require.NoError(t, err)

	for _, tx := range transactions {
		accID := int(tx["accountId"].(float64))
		assert.True(t, accID == account1ID || accID == account2ID)
	}
}

func TestGetTransactionsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/transactions/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Transaction Details Tests ====================

func TestGetTransactionDetails(t *testing.T) {
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

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 75.50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(txID), response["id"])
	assert.Equal(t, float64(accountID), response["accountId"])
	assert.Equal(t, 75.50, response["amount"])
}

func TestGetTransactionDetailsWithAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Detailed Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should include account details
	if account, ok := response["account"].(map[string]interface{}); ok {
		assert.Equal(t, "Detailed Wallet", account["name"])
	}
}

func TestGetTransactionDetailsNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/transactions/999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetTransactionDetailsOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID1, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID1, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Create second user
	otherEmail := "other-details@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetTransactionDetailsInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/transactions/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetTransactionDetailsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/transactions/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Transaction Tests ====================

func TestUpdateTransaction(t *testing.T) {
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

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 75.00,
		"label": "Updated label",
		"isIncome": false
	}`, txID, accountID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(75), response["amount"])
	assert.Equal(t, "Updated label", response["label"])
}

func TestUpdateTransactionChangeCategory(t *testing.T) {
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

	cat1ID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat1ID)

	cat2ID := testutil.CreateTestCategory(t, app.DB, userID, "Transport", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat2ID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat1ID, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"categoryId": %d,
		"amount": 50.00,
		"label": "Changed category",
		"isIncome": false
	}`, txID, accountID, cat2ID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(cat2ID), response["categoryId"])
}

func TestUpdateTransactionAddNotes(t *testing.T) {
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

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 50.00,
		"label": "Test",
		"notes": "Added notes after creation",
		"isIncome": false
	}`, txID, accountID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Added notes after creation", response["notes"])
}

func TestUpdateTransactionNotFound(t *testing.T) {
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

	body := fmt.Sprintf(`{
		"id": 999999,
		"accountId": %d,
		"amount": 75.00,
		"label": "Test",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTransactionOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID1, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID1, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Create second user
	otherEmail := "other-update@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	account2ID := testutil.CreateTestAccount(t, app.DB, userID2, "Other Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, account2ID)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 999.00,
		"label": "Hacked",
		"isIncome": false
	}`, txID, account2ID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should not be allowed
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestUpdateTransactionUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"id": 1,
		"accountId": 1,
		"amount": 75.00,
		"label": "Test",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateTransactionMissingID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"accountId": 1,
		"amount": 75.00,
		"label": "No ID",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== Delete Transaction Tests ====================

func TestDeleteTransaction(t *testing.T) {
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

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify deletion
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	getReq.Header.Set("auth-token", token)
	getRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(getRec, getReq)

	assert.Equal(t, http.StatusNotFound, getRec.Code)
}

func TestDeleteTransactionNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteTransactionOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID1, "Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID1, accountID, nil, 50, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Create second user
	otherEmail := "other-delete@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteTransactionUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteTransactionInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ==================== Template Tests ====================

func TestGetTemplates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Bills", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Monthly Rent", &categoryID)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	req := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var templates []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &templates)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(templates), 1)

	found := false
	for _, tmpl := range templates {
		if tmpl["label"] == "Monthly Rent" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGetTemplatesEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "empty-templates@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var templates []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &templates)
	require.NoError(t, err)

	assert.Equal(t, 0, len(templates))
}

func TestGetTemplatesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateTemplate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Bills", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Old Label", &categoryID)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	body := fmt.Sprintf(`{
		"id": %d,
		"label": "New Label",
		"categoryId": %d
	}`, templateID, categoryID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "New Label", response["label"])
}

func TestUpdateTemplateNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"id": 999999,
		"label": "Not Found"
	}`

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTemplateUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"id": 1,
		"label": "Test"
	}`

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteTemplates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	templateID1 := testutil.CreateTestTemplate(t, app.DB, userID, "Template 1", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID1)

	templateID2 := testutil.CreateTestTemplate(t, app.DB, userID, "Template 2", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID2)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/templates?ids=%d,%d", templateID1, templateID2), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteTemplatesSingle(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Single Template", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/templates?ids=%d", templateID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteTemplatesMissingIds(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDeleteTemplatesInvalidIds(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates?ids=abc,def", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDeleteTemplatesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates?ids=1,2", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Balance Impact Tests ====================

func TestExpenseReducesBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Balance Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Get initial balance
	var initialBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&initialBalance)
	require.NoError(t, err)

	// Create expense
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "Expense",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check balance decreased
	var newBalance float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&newBalance)
	require.NoError(t, err)

	assert.Equal(t, initialBalance-100, newBalance)
}

func TestIncomeIncreasesBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Balance Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Get initial balance
	var initialBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&initialBalance)
	require.NoError(t, err)

	// Create income
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 500.00,
		"label": "Income",
		"isIncome": true
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check balance increased
	var newBalance float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&newBalance)
	require.NoError(t, err)

	assert.Equal(t, initialBalance+500, newBalance)
}

func TestTransferMovesBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	sourceAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Source", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, sourceAccountID)

	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Target", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	// Get initial balances
	var sourceInitial, targetInitial float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", sourceAccountID).Scan(&sourceInitial)
	require.NoError(t, err)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", targetAccountID).Scan(&targetInitial)
	require.NoError(t, err)

	// Create transfer
	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"label": "Transfer",
		"isTransfer": true
	}`, sourceAccountID, targetAccountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check balances
	var sourceNew, targetNew float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", sourceAccountID).Scan(&sourceNew)
	require.NoError(t, err)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", targetAccountID).Scan(&targetNew)
	require.NoError(t, err)

	assert.Equal(t, sourceInitial-200, sourceNew)
	assert.Equal(t, targetInitial+200, targetNew)
}

func TestDeleteRestoresBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Balance Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Get initial balance
	var initialBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&initialBalance)
	require.NoError(t, err)

	// Create expense via API
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 150.00,
		"label": "To Delete",
		"isIncome": false
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var createResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	txID := int(createResponse["id"].(float64))

	// Verify balance decreased
	var afterCreate float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&afterCreate)
	require.NoError(t, err)
	assert.Equal(t, initialBalance-150, afterCreate)

	// Delete the transaction
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	delReq.Header.Set("auth-token", token)
	delRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(delRec, delReq)

	assert.Equal(t, http.StatusOK, delRec.Code)

	// Check balance restored
	var afterDelete float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&afterDelete)
	require.NoError(t, err)

	assert.Equal(t, initialBalance, afterDelete)
}

func TestUpdateTransactionChangesBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Edit Balance Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create expense of 100 via API
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "Original Expense",
		"isIncome": false
	}`, accountID)

	createReq := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createReq.Header.Set("auth-token", token)
	createRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var createResponse map[string]interface{}
	err := json.Unmarshal(createRec.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	txID := int(createResponse["id"].(float64))

	// Verify balance is 900 (1000 - 100)
	var afterCreate float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&afterCreate)
	require.NoError(t, err)
	assert.Equal(t, float64(900), afterCreate)

	// Update expense to 150
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 150.00,
		"label": "Updated Expense",
		"isIncome": false
	}`, txID, accountID)

	updateReq := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateReq.Header.Set("auth-token", token)
	updateRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Verify balance is now 850 (1000 - 150)
	var afterUpdate float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&afterUpdate)
	require.NoError(t, err)
	assert.Equal(t, float64(850), afterUpdate)
}

func TestUpdateTransactionRecalculatesSubsequentBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Recalc Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create first expense of 100
	body1 := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "First Expense",
		"isIncome": false
	}`, accountID)

	req1 := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	var resp1 map[string]interface{}
	json.Unmarshal(rec1.Body.Bytes(), &resp1)
	txID1 := int(resp1["id"].(float64))

	// Small delay to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create second expense of 200
	body2 := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 200.00,
		"label": "Second Expense",
		"isIncome": false
	}`, accountID)

	req2 := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)

	var resp2 map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &resp2)
	_ = int(resp2["id"].(float64)) // txID2 for potential future use

	// Verify account balance is 700 (1000 - 100 - 200)
	var balance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(700), balance)

	// Update first transaction to 50 (reduce by 50)
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 50.00,
		"label": "First Expense Updated",
		"isIncome": false
	}`, txID1, accountID)

	updateReq := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateReq.Header.Set("auth-token", token)
	updateRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Verify account balance is now 750 (1000 - 50 - 200)
	// This is the key assertion - the account balance must be correct after update
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(750), balance)
}

func TestDeleteTransactionRecalculatesSubsequentBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Delete Recalc Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create first expense of 100
	body1 := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "First Expense",
		"isIncome": false
	}`, accountID)

	req1 := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	var resp1 map[string]interface{}
	json.Unmarshal(rec1.Body.Bytes(), &resp1)
	txID1 := int(resp1["id"].(float64))

	time.Sleep(10 * time.Millisecond)

	// Create second expense of 200
	body2 := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 200.00,
		"label": "Second Expense",
		"isIncome": false
	}`, accountID)

	req2 := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)

	var resp2 map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &resp2)
	txID2 := int(resp2["id"].(float64))

	// Verify balance is 700 (1000 - 100 - 200)
	var balance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(700), balance)

	// Delete first transaction
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID1), nil)
	delReq.Header.Set("auth-token", token)
	delRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(delRec, delReq)
	assert.Equal(t, http.StatusOK, delRec.Code)

	// Verify balance is now 800 (1000 - 200, first tx deleted)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(800), balance)

	// Verify second transaction's new_balance was recalculated
	var tx2NewBalance float64
	err = app.DB.QueryRow("SELECT new_balance FROM transactions WHERE id = $1", txID2).Scan(&tx2NewBalance)
	require.NoError(t, err)
	// After second tx (200 expense) with no first tx: 800
	assert.Equal(t, float64(800), tx2NewBalance)
}

func TestIncomeToExpenseChange(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Type Change Test", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create income of 100
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"label": "Originally Income",
		"isIncome": true
	}`, accountID)

	createReq := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createReq.Header.Set("auth-token", token)
	createRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var createResp map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResp)
	txID := int(createResp["id"].(float64))

	// Verify balance is 1100 (1000 + 100)
	var balance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(1100), balance)

	// Update to expense
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 100.00,
		"label": "Now Expense",
		"isIncome": false
	}`, txID, accountID)

	updateReq := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateReq.Header.Set("auth-token", token)
	updateRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Verify balance is now 900 (1000 - 100)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, float64(900), balance)
}

func TestDeleteTransferRestoresSourceBalance(t *testing.T) {
	// Tests that deleting a transfer restores the source account balance.
	// Note: Target account balance recalculation has a known issue where the linked
	// transaction's AccountID cannot be retrieved after soft-delete. The source
	// account is correctly recalculated.
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	sourceAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Source Delete", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, sourceAccountID)

	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Target Delete", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	// Create transfer of 200
	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"label": "Transfer to delete",
		"isTransfer": true
	}`, sourceAccountID, targetAccountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	txID := int(resp["id"].(float64))

	// Verify balances after transfer
	var sourceBalance, targetBalance float64
	app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", sourceAccountID).Scan(&sourceBalance)
	app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", targetAccountID).Scan(&targetBalance)
	assert.Equal(t, float64(800), sourceBalance)
	assert.Equal(t, float64(700), targetBalance)

	// Delete the transfer
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	delReq.Header.Set("auth-token", token)
	delRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(delRec, delReq)
	assert.Equal(t, http.StatusOK, delRec.Code)

	// Verify source balance restored
	app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", sourceAccountID).Scan(&sourceBalance)
	assert.Equal(t, float64(1000), sourceBalance)

	// Verify linked transaction is soft-deleted
	var linkedDeleted bool
	err := app.DB.QueryRow("SELECT is_deleted FROM transactions WHERE linked_transaction_id = $1", txID).Scan(&linkedDeleted)
	require.NoError(t, err)
	assert.True(t, linkedDeleted, "Linked transaction should be soft-deleted")
}

// ==================== Inactive User Tests ====================

func TestCreateTransactionInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "createtx-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Inactive TX Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Inactive Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 50.00,
		"label": "Inactive user tx"
	}`, accountID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestGetTransactionsInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "gettxs-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	req := httptest.NewRequest(http.MethodGet, "/transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestGetTransactionDetailsInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "gettxdetails-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Inactive Detail Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Deactivate user after getting token and creating data
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestUpdateTransactionInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updatetx-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Inactive Update Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Deactivate user after getting token and creating data
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 75.00,
		"label": "Updated inactive"
	}`, txID, accountID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestDeleteTransactionInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deletetx-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Inactive Delete Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	txID := testutil.CreateTestTransaction(t, app.DB, userID, accountID, nil, 100, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	// Deactivate user after getting token and creating data
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestGetTemplatesInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "gettemplates-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Inactive Template", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	// Deactivate user after getting token and creating template
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	req := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestUpdateTemplateInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updatetpl-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Template to Update", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	// Deactivate user after getting token and creating template
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	body := fmt.Sprintf(`{"id": %d, "label": "Updated Label"}`, templateID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

func TestDeleteTemplatesInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deletetpls-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Template to Delete", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	// Deactivate user after getting token and creating template
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err, "Failed to deactivate user")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/templates?ids=%d", templateID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 for inactive user, got %d. Body: %s", rec.Code, rec.Body.String())
}

// ==================== Bug Fix Regression Tests ====================

// TestUpdateTransactionChangeAccount verifies that updating a transaction to a
// different account correctly persists the new account_id in the database.
func TestUpdateTransactionChangeAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "tx-change-account@example.com"
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	accountA := testutil.CreateTestAccount(t, app.DB, userID, "Account A", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountA)

	accountB := testutil.CreateTestAccount(t, app.DB, userID, "Account B", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, accountB)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Test Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Create a transaction on Account A
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 100.00,
		"label": "Original",
		"isIncome": false
	}`, accountA, categoryID)

	createReq := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createReq.Header.Set("auth-token", token)
	createRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code, "Create failed: %s", createRec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	assert.Equal(t, float64(accountA), createResp["accountId"], "Transaction should initially belong to Account A")

	// Update the transaction to Account B
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"categoryId": %d,
		"amount": 100.00,
		"label": "Moved",
		"isIncome": false
	}`, txID, accountB, categoryID)

	updateReq := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateReq.Header.Set("auth-token", token)
	updateRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(updateRec, updateReq)
	require.Equal(t, http.StatusOK, updateRec.Code, "Update failed: %s", updateRec.Body.String())

	// Fetch the transaction details and verify accountId is Account B
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions/%d", txID), nil)
	getReq.Header.Set("auth-token", token)
	getRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code, "Get failed: %s", getRec.Body.String())

	var getResp map[string]interface{}
	require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &getResp))
	assert.Equal(t, float64(accountB), getResp["accountId"], "Transaction should now belong to Account B after update")
}

// TestDeleteTransferRecalculatesTargetBalance verifies that deleting a transfer
// transaction correctly recalculates both source and target account balances.
func TestDeleteTransferRecalculatesTargetBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "tx-del-transfer-balance@example.com"
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create Account A with 1000 and Account B with 500
	accountA := testutil.CreateTestAccount(t, app.DB, userID, "Source Acct", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountA)

	accountB := testutil.CreateTestAccount(t, app.DB, userID, "Target Acct", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, accountB)

	// Create a transfer: A -> B, amount 200
	transferBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"label": "Transfer to B",
		"isTransfer": true
	}`, accountA, accountB)

	createReq := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(transferBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createReq.Header.Set("auth-token", token)
	createRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code, "Create transfer failed: %s", createRec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))

	// Verify intermediate balances: A should be 800, B should be 700
	getAccountA := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountA), nil)
	getAccountA.Header.Set("auth-token", token)
	recA := httptest.NewRecorder()
	app.Echo.ServeHTTP(recA, getAccountA)
	require.Equal(t, http.StatusOK, recA.Code)

	var respA map[string]interface{}
	require.NoError(t, json.Unmarshal(recA.Body.Bytes(), &respA))
	assert.Equal(t, float64(800), respA["balance"], "Account A balance should be 800 after transfer")

	getAccountB := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountB), nil)
	getAccountB.Header.Set("auth-token", token)
	recB := httptest.NewRecorder()
	app.Echo.ServeHTTP(recB, getAccountB)
	require.Equal(t, http.StatusOK, recB.Code)

	var respB map[string]interface{}
	require.NoError(t, json.Unmarshal(recB.Body.Bytes(), &respB))
	assert.Equal(t, float64(700), respB["balance"], "Account B balance should be 700 after transfer")

	// Delete the transfer transaction
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txID), nil)
	deleteReq.Header.Set("auth-token", token)
	deleteRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(deleteRec, deleteReq)
	require.Equal(t, http.StatusOK, deleteRec.Code, "Delete transfer failed: %s", deleteRec.Body.String())

	// Verify both balances are restored: A should be 1000, B should be 500
	getAccountA2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountA), nil)
	getAccountA2.Header.Set("auth-token", token)
	recA2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(recA2, getAccountA2)
	require.Equal(t, http.StatusOK, recA2.Code)

	var respA2 map[string]interface{}
	require.NoError(t, json.Unmarshal(recA2.Body.Bytes(), &respA2))
	assert.Equal(t, float64(1000), respA2["balance"], "Account A balance should be restored to 1000 after deleting transfer")

	getAccountB2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountB), nil)
	getAccountB2.Header.Set("auth-token", token)
	recB2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(recB2, getAccountB2)
	require.Equal(t, http.StatusOK, recB2.Code)

	var respB2 map[string]interface{}
	require.NoError(t, json.Unmarshal(recB2.Body.Bytes(), &respB2))
	assert.Equal(t, float64(500), respB2["balance"], "Account B balance should be restored to 500 after deleting transfer")
}

// TestUpdateTransactionChangeAccountRecalculatesBalances verifies that when a
// transaction is moved from Account A to Account B, both account balances are
// correctly recalculated: the old account's balance is restored and the new
// account's balance reflects the moved transaction.
func TestUpdateTransactionChangeAccountRecalculatesBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "tx-move-account-balance@example.com"
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create Account A (balance 1000) and Account B (balance 500)
	accountA := testutil.CreateTestAccount(t, app.DB, userID, "Account A", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountA)

	accountB := testutil.CreateTestAccount(t, app.DB, userID, "Account B", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, accountB)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Expense Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Step 1: Create an expense transaction of 200 on Account A
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": 200.00,
		"label": "Expense on A",
		"isIncome": false
	}`, accountA, categoryID)

	createReq := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createReq.Header.Set("auth-token", token)
	createRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code, "Create failed: %s", createRec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))

	// Step 2: Verify Account A balance is 800 (1000 - 200)
	var balanceA float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountA).Scan(&balanceA)
	require.NoError(t, err)
	assert.Equal(t, float64(800), balanceA, "Account A should be 800 after expense of 200")

	// Verify Account B balance is still 500
	var balanceB float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountB).Scan(&balanceB)
	require.NoError(t, err)
	assert.Equal(t, float64(500), balanceB, "Account B should still be 500")

	// Step 3: Update the transaction to move it to Account B
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"categoryId": %d,
		"amount": 200.00,
		"label": "Expense moved to B",
		"isIncome": false
	}`, txID, accountB, categoryID)

	updateReq := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateReq.Header.Set("auth-token", token)
	updateRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(updateRec, updateReq)
	require.Equal(t, http.StatusOK, updateRec.Code, "Update failed: %s", updateRec.Body.String())

	// Step 4: Verify Account A balance is back to 1000 (expense removed)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountA).Scan(&balanceA)
	require.NoError(t, err)
	assert.Equal(t, float64(1000), balanceA, "Account A should be restored to 1000 after moving expense away")

	// Step 5: Verify Account B balance is 300 (500 - 200, expense applied)
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountB).Scan(&balanceB)
	require.NoError(t, err)
	assert.Equal(t, float64(300), balanceB, "Account B should be 300 after expense of 200 is moved here")
}

// ==================== ValidateTemplateIDs Tests ====================

func TestValidateTemplateIDsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates/validate?ids=1,2,3", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var ids []int
	err := json.Unmarshal(rec.Body.Bytes(), &ids)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, ids)
}

func TestValidateTemplateIDsInvalidFormat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates/validate?ids=a,b,c", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid template IDs format", response["detail"])
}

func TestValidateTemplateIDsEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates/validate?ids=", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid template IDs format", response["detail"])
}

func TestValidateTemplateIDsNegative(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates/validate?ids=-1,0", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Invalid template IDs format", response["detail"])
}

func TestValidateTemplateIDsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/transactions/templates/validate?ids=1,2,3", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Transfer Template Tests ====================

func TestCreateTransferTransactionWithTemplate(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	sourceAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Source Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, sourceAccountID)

	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Target Savings", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"label": "Transfer to savings",
		"isIncome": false,
		"isTransfer": true,
		"isTemplate": true
	}`, sourceAccountID, targetAccountID)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify template was created with targetAccountId
	templatesReq := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	templatesReq.Header.Set("auth-token", token)
	templatesRec := httptest.NewRecorder()

	app.Echo.ServeHTTP(templatesRec, templatesReq)

	assert.Equal(t, http.StatusOK, templatesRec.Code)

	var templates []map[string]interface{}
	err := json.Unmarshal(templatesRec.Body.Bytes(), &templates)
	require.NoError(t, err)

	found := false
	for _, tmpl := range templates {
		if tmpl["label"] == "Transfer to savings" {
			found = true
			assert.Equal(t, float64(targetAccountID), tmpl["targetAccountId"])
			assert.Equal(t, "Target Savings", tmpl["targetAccountName"])
			break
		}
	}
	assert.True(t, found, "Transfer template should be created with targetAccountId")
}

func TestGetTemplatesWithTargetAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Savings Account", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	templateID := testutil.CreateTestTransferTemplate(t, app.DB, userID, "Monthly Transfer", targetAccountID)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	req := httptest.NewRequest(http.MethodGet, "/transactions/templates", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var templates []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &templates)
	require.NoError(t, err)

	found := false
	for _, tmpl := range templates {
		if tmpl["label"] == "Monthly Transfer" {
			found = true
			assert.Equal(t, float64(targetAccountID), tmpl["targetAccountId"])
			assert.Equal(t, "Savings Account", tmpl["targetAccountName"])
			// categoryId should be nil for transfer templates
			assert.Nil(t, tmpl["categoryId"])
			break
		}
	}
	assert.True(t, found, "Transfer template should appear in templates list")
}

func TestUpdateTemplateWithTargetAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Investment Account", currencyID, accountTypeID, 2000)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	templateID := testutil.CreateTestTemplate(t, app.DB, userID, "Old Transfer", nil)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	body := fmt.Sprintf(`{
		"id": %d,
		"label": "Updated Transfer",
		"targetAccountId": %d
	}`, templateID, targetAccountID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Transfer", response["label"])
	assert.Equal(t, float64(targetAccountID), response["targetAccountId"])
	assert.Equal(t, "Investment Account", response["targetAccountName"])
}

func TestUpdateTemplateRemoveTargetAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Old Target", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, targetAccountID)

	templateID := testutil.CreateTestTransferTemplate(t, app.DB, userID, "Was Transfer", targetAccountID)
	defer testutil.DeleteTestTemplate(t, app.DB, templateID)

	// Update without targetAccountId to remove it (send null explicitly)
	body := fmt.Sprintf(`{
		"id": %d,
		"label": "Now Regular"
	}`, templateID)

	req := httptest.NewRequest(http.MethodPut, "/transactions/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Now Regular", response["label"])
	assert.Nil(t, response["targetAccountId"])
	assert.Nil(t, response["targetAccountName"])
}

// ==================== Migration Smoke Test ====================

// TestTransactionsLabelColumnIsVarchar255 is a smoke check that asserts
// migration 00006 has been applied — i.e. `transactions.label` is
// `character varying(255)`. If this fails, fresh-DB setups are out of sync
// with the migrations.
func TestTransactionsLabelColumnIsVarchar255(t *testing.T) {
	app := testutil.NewTestApp(t)

	var dataType string
	var maxLen *int
	err := app.DB.QueryRow(`
		SELECT data_type, character_maximum_length
		  FROM information_schema.columns
		 WHERE table_name = 'transactions' AND column_name = 'label'
	`).Scan(&dataType, &maxLen)
	require.NoError(t, err)

	assert.Equal(t, "character varying", dataType)
	if assert.NotNil(t, maxLen, "label column must have a varchar length") {
		assert.Equal(t, 255, *maxLen, "label must be VARCHAR(255) — apply migration 00006")
	}
}

// ==================== Label / Notes Length Validation Tests ====================

// assertNoValidatorInternalsLeak verifies the response body never carries Go
// struct names, validator tag literals, or raw `validator.FieldError` strings
// — the API contract is `{ detail, errorCode, params }` only.
func assertNoValidatorInternalsLeak(t *testing.T, body string) {
	t.Helper()
	leakSubstrings := []string{
		"CreateTransactionRequest",
		"UpdateTransactionRequest",
		"Field validation",
		"failed on the",
		"Tag=",
		"Field=",
	}
	for _, s := range leakSubstrings {
		assert.NotContains(t, body, s, "validator internals must not leak in response body")
	}
}

func TestCreateTransactionLabel255Boundary(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Boundary Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	label255 := strings.Repeat("a", 255)
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 10.00,
		"label": "%s",
		"isIncome": false
	}`, accountID, label255)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "255-char label is the boundary and must be accepted")

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, label255, response["label"])
}

func TestCreateTransactionLabelTooLong(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "TooLong Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Count rows before to assert no insert happened on rejection.
	var beforeCount int
	require.NoError(t, app.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE user_id = $1", userID).Scan(&beforeCount))

	label256 := strings.Repeat("a", 256)
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 10.00,
		"label": "%s",
		"isIncome": false
	}`, accountID, label256)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	bodyStr := rec.Body.String()
	assertNoValidatorInternalsLeak(t, bodyStr)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, "errors.transaction.labelTooLong", response["errorCode"])
	assert.Equal(t, "Label must be at most 255 characters", response["detail"])
	if params, ok := response["params"].(map[string]interface{}); assert.True(t, ok, "params must be an object") {
		assert.Equal(t, float64(255), params["max"])
	}

	// No row inserted.
	var afterCount int
	require.NoError(t, app.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE user_id = $1", userID).Scan(&afterCount))
	assert.Equal(t, beforeCount, afterCount, "rejected request must not insert a transaction row")
}

func TestCreateTransactionNotesTooLong(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Notes Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	notes4001 := strings.Repeat("n", 4001)
	body := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 10.00,
		"label": "ok",
		"notes": "%s",
		"isIncome": false
	}`, accountID, notes4001)

	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	bodyStr := rec.Body.String()
	assertNoValidatorInternalsLeak(t, bodyStr)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, "errors.transaction.notesTooLong", response["errorCode"])
	assert.Equal(t, "Notes must be at most 4000 characters", response["detail"])
	if params, ok := response["params"].(map[string]interface{}); assert.True(t, ok, "params must be an object") {
		assert.Equal(t, float64(4000), params["max"])
	}
}
