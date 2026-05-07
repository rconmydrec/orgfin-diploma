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

// TestRunningBalanceAfterIncomesAndExpenses verifies that the account balance
// is correctly updated across a sequence of income and expense transactions.
//
// Scenario:
//  1. Create an account with 0 initial balance.
//  2. Add +100 income  -> balance should be 100.
//  3. Add +200 income  -> balance should be 300.
//  4. Add -50  expense -> balance should be 250.
//  5. Verify the final account balance through the GET endpoint and the DB.
func TestRunningBalanceAfterIncomesAndExpenses(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-balance-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Step 1: Create account with zero balance via API.
	createAccountBody := fmt.Sprintf(`{
		"name": "Scenario Checking USD",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 0,
		"balance": 0
	}`, currencyID, accountTypeID)

	rec := doRequest(t, app, http.MethodPost, "/accounts/", createAccountBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create account: %s", rec.Body.String())

	var accountResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &accountResp))
	accountID := int(accountResp["id"].(float64))

	// We need an income and an expense category.
	incomeCategoryID := testutil.CreateTestCategory(t, app.DB, userID, "Scenario Income", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCategoryID)

	expenseCategoryID := testutil.CreateTestCategory(t, app.DB, userID, "Scenario Expense", false)
	defer testutil.DeleteTestCategory(t, app.DB, expenseCategoryID)

	// Step 2: +100 income
	createTx(t, app, token, accountID, incomeCategoryID, 100, true)

	// Step 3: +200 income
	createTx(t, app, token, accountID, incomeCategoryID, 200, true)

	// Step 4: -50 expense
	createTx(t, app, token, accountID, expenseCategoryID, 50, false)

	// Step 5: Verify final account balance == 250 via API.
	rec = doRequest(t, app, http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var detailResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &detailResp))
	assert.Equal(t, float64(250), detailResp["balance"], "Expected account balance to be 250 after +100 +200 -50")

	// Also verify via direct DB query.
	var dbBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&dbBalance)
	require.NoError(t, err)
	assert.Equal(t, 250.0, dbBalance, "DB balance should match 250")
}
