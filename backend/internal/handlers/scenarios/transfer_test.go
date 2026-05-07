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

// TestTransferSameCurrencyUpdatesBothBalances verifies that a transfer between
// two USD accounts correctly debits the source and credits the destination.
//
// Scenario:
//  1. Create source account (USD, balance 1000).
//  2. Create destination account (USD, balance 500).
//  3. Transfer 200 from source to destination.
//  4. Verify source balance == 800, destination balance == 700.
//  5. Verify two linked transactions exist.
func TestTransferSameCurrencyUpdatesBothBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-transfer-same-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Scenario USD Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Scenario USD Dst", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Perform transfer.
	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"targetAmount": 200.00,
		"isIncome": false,
		"isTransfer": true
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "transfer: %s", rec.Body.String())

	var txResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &txResp))
	assert.Equal(t, true, txResp["isTransfer"])
	assert.NotNil(t, txResp["linkedTransactionId"], "Transfer should have a linked transaction")

	// Verify source balance.
	var srcBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", srcAccountID).Scan(&srcBalance)
	require.NoError(t, err)
	assert.Equal(t, 800.0, srcBalance, "Source balance should be 800 after transferring 200")

	// Verify destination balance.
	var dstBalance float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", dstAccountID).Scan(&dstBalance)
	require.NoError(t, err)
	assert.Equal(t, 700.0, dstBalance, "Destination balance should be 700 after receiving 200")

	// Verify two transactions exist and they are linked.
	var txCount int
	err = app.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND is_transfer = true", userID).Scan(&txCount)
	require.NoError(t, err)
	assert.Equal(t, 2, txCount, "There should be exactly 2 transfer transactions")
}

// TestTransferDifferentCurrenciesUsesTargetAmount verifies that a cross-currency
// transfer uses the targetAmount for the destination account.
//
// Scenario:
//  1. Create source account (USD, balance 1000).
//  2. Create destination account (EUR, balance 0).
//  3. Transfer 100 USD -> 92 EUR.
//  4. Verify source balance == 900, destination balance == 92.
func TestTransferDifferentCurrenciesUsesTargetAmount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-transfer-fx-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Scenario USD Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Scenario EUR Dst", eurID, accTypeID, 0)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	body := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"targetAmount": 92.00,
		"isIncome": false,
		"isTransfer": true
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "fx transfer: %s", rec.Body.String())

	// Verify source balance.
	var srcBalance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", srcAccountID).Scan(&srcBalance)
	require.NoError(t, err)
	assert.Equal(t, 900.0, srcBalance, "Source balance should be 900 after sending 100 USD")

	// Verify destination balance.
	var dstBalance float64
	err = app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", dstAccountID).Scan(&dstBalance)
	require.NoError(t, err)
	assert.Equal(t, 92.0, dstBalance, "Destination balance should be 92 EUR")
}
