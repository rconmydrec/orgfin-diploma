package scenarios_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdjustmentUpdatesAccountBalance verifies that a balance adjustment
// correctly sets the account to the target balance.
func TestAdjustmentUpdatesAccountBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Adjustment Test Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Adjust balance to 2500.50.
	body := fmt.Sprintf(`{"accountId": %d, "newBalance": 2500.50}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust balance: %s", rec.Body.String())

	// Verify via GET /accounts/:id.
	rec = doRequest(t, app, http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var accountResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &accountResp))
	assert.Equal(t, 2500.5, accountResp["balance"])
}

// TestAdjustmentExcludedFromCashFlowReport verifies that adjustment transactions
// do not affect the cash flow report totals.
func TestAdjustmentExcludedFromCashFlowReport(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-cf-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "CF Adjustment Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	startDate := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	endDate := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	// Get initial cash flow.
	cfBody := fmt.Sprintf(`{"startDate": "%s", "endDate": "%s", "period": "monthly"}`, startDate, endDate)
	initialRec := doRequest(t, app, http.MethodPost, "/reports/cashflow/", cfBody, token)
	require.Equal(t, http.StatusOK, initialRec.Code, "initial cashflow: %s", initialRec.Body.String())

	var initialCF map[string]interface{}
	require.NoError(t, json.Unmarshal(initialRec.Body.Bytes(), &initialCF))

	// Create an adjustment (+1000).
	adjustBody := fmt.Sprintf(`{"accountId": %d, "newBalance": 2000}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", adjustBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust: %s", rec.Body.String())

	// Get cash flow again.
	afterRec := doRequest(t, app, http.MethodPost, "/reports/cashflow/", cfBody, token)
	require.Equal(t, http.StatusOK, afterRec.Code)

	var afterCF map[string]interface{}
	require.NoError(t, json.Unmarshal(afterRec.Body.Bytes(), &afterCF))

	// Cash flow totals should not change because adjustments are excluded.
	assert.Equal(t, initialCF["totalIncome"], afterCF["totalIncome"],
		"totalIncome should not change after adjustment")
	assert.Equal(t, initialCF["totalExpenses"], afterCF["totalExpenses"],
		"totalExpenses should not change after adjustment")
}

// TestAdjustmentAppearsInTransactionHistory verifies that the adjustment
// transaction is visible when listing transactions for the account.
func TestAdjustmentAppearsInTransactionHistory(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-hist-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "History Adjustment Account", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create adjustment.
	adjustBody := fmt.Sprintf(`{"accountId": %d, "newBalance": 1000, "notes": "Test visibility in history"}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", adjustBody, token)
	require.Equal(t, http.StatusOK, rec.Code)

	var adjustResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &adjustResp))
	adjustmentID := int(adjustResp["id"].(float64))

	// List transactions for the account.
	txListRec := doRequest(t, app, http.MethodGet, fmt.Sprintf("/transactions/?accounts=%d", accountID), "", token)
	require.Equal(t, http.StatusOK, txListRec.Code)

	var txList []map[string]interface{}
	require.NoError(t, json.Unmarshal(txListRec.Body.Bytes(), &txList))

	// Find the adjustment in the list.
	found := false
	for _, tx := range txList {
		if int(tx["id"].(float64)) == adjustmentID {
			found = true
			assert.Equal(t, true, tx["isAdjustment"], "Transaction should be marked as adjustment")
			break
		}
	}
	assert.True(t, found, "Adjustment should appear in transaction history")
}

// TestMultipleAdjustmentsInSequence verifies that consecutive adjustments
// correctly update the balance each time and all appear in history.
func TestMultipleAdjustmentsInSequence(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-multi-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Multi Adjustment Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// First adjustment: increase to 2000.
	body := fmt.Sprintf(`{"accountId": %d, "newBalance": 2000}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust 1: %s", rec.Body.String())

	// Second adjustment: decrease to 1500.
	body = fmt.Sprintf(`{"accountId": %d, "newBalance": 1500}`, accountID)
	rec = doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust 2: %s", rec.Body.String())

	// Third adjustment: increase to 3000.
	body = fmt.Sprintf(`{"accountId": %d, "newBalance": 3000}`, accountID)
	rec = doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust 3: %s", rec.Body.String())

	// Verify final balance.
	rec = doRequest(t, app, http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var accountResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &accountResp))
	assert.Equal(t, float64(3000), accountResp["balance"])

	// Verify all three adjustments exist in history.
	txListRec := doRequest(t, app, http.MethodGet, fmt.Sprintf("/transactions/?accounts=%d", accountID), "", token)
	require.Equal(t, http.StatusOK, txListRec.Code)

	var txList []map[string]interface{}
	require.NoError(t, json.Unmarshal(txListRec.Body.Bytes(), &txList))

	adjustmentCount := 0
	for _, tx := range txList {
		if tx["isAdjustment"] == true {
			adjustmentCount++
		}
	}
	assert.Equal(t, 3, adjustmentCount, "All 3 adjustments should appear in history")
}

// TestAdjustmentWithRegularTransactions verifies that adjustments work correctly
// alongside regular income/expense transactions.
//
// Scenario:
//  1. Account starts with 1000.
//  2. Regular expense -100 => balance = 900.
//  3. Adjustment sets balance to 1500.
//  4. Regular income +200 => balance = 1700.
func TestAdjustmentWithRegularTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-mixed-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Mixed Transactions Account", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	expenseCategoryID := testutil.CreateTestCategory(t, app.DB, userID, "Mixed Expense", false)
	defer testutil.DeleteTestCategory(t, app.DB, expenseCategoryID)

	incomeCategoryID := testutil.CreateTestCategory(t, app.DB, userID, "Mixed Income", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCategoryID)

	// Step 2: Regular expense -100.
	createTx(t, app, token, accountID, expenseCategoryID, 100, false)

	// Step 3: Adjustment to 1500.
	adjustBody := fmt.Sprintf(`{"accountId": %d, "newBalance": 1500}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", adjustBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjustment: %s", rec.Body.String())

	// Step 4: Regular income +200.
	createTx(t, app, token, accountID, incomeCategoryID, 200, true)

	// Final balance should be 1700.
	rec = doRequest(t, app, http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var accountResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &accountResp))
	assert.Equal(t, float64(1700), accountResp["balance"],
		"Balance should be 1500 (from adjustment) + 200 (income) = 1700")
}

// TestAdjustmentDownward verifies that a downward adjustment works correctly.
func TestAdjustmentDownward(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-adjust-down-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Downward Adjustment Account", currencyID, accountTypeID, 5000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Adjust balance down to 2000.
	body := fmt.Sprintf(`{"accountId": %d, "newBalance": 2000}`, accountID)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust down: %s", rec.Body.String())

	var txResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &txResp))
	assert.Equal(t, false, txResp["isIncome"], "Downward adjustment should not be income")
	assert.Equal(t, true, txResp["isAdjustment"])
	assert.Equal(t, true, txResp["excludeFromReports"])

	// Verify balance.
	var dbBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&dbBalance)
	require.NoError(t, err)
	assert.Equal(t, 2000.0, dbBalance)
}
