package scenarios_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the recalculateBalances service method indirectly,
// by triggering it through the standard HTTP edit/delete paths
// (PUT /transactions/, DELETE /transactions/:id). Recalc fires whenever a
// transaction's amount, account, or transfer pairing changes.
//
// The pivotal regression case is "adjustment in the middle": before the fix,
// recalc overwrote the running balance with tx.Amount (the *delta*) on
// adjustment rows instead of *tx.NewBalance (the *target*). Editing any
// regular transaction that sat after an adjustment in the account history
// would silently rewrite accounts.balance to a wrong value.

// adjustBalance is a small wrapper around POST /accounts/adjust-balance.
func adjustBalance(t *testing.T, app *testutil.TestApp, token string, accountID int, newBalance float64) {
	t.Helper()
	body := fmt.Sprintf(`{"accountId": %d, "newBalance": %.2f}`, accountID, newBalance)
	rec := doRequest(t, app, http.MethodPost, "/accounts/adjust-balance", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "adjust balance: %s", rec.Body.String())
}

// createTxReturningID is a variant of createTx that also returns the new
// transaction's ID so the test can edit it later.
func createTxReturningID(t *testing.T, app *testutil.TestApp, token string, accountID, categoryID int, amount float64, isIncome bool) int {
	t.Helper()
	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": %.2f,
		"isIncome": %t,
		"isTransfer": false
	}`, accountID, categoryID, amount, isIncome)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "create tx: %s", rec.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return int(resp["id"].(float64))
}

// updateTxAmount edits an existing regular transaction's amount and forces a
// recalc on the source account.
func updateTxAmount(t *testing.T, app *testutil.TestApp, token string, txID, accountID, categoryID int, amount float64, isIncome bool) {
	t.Helper()
	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"categoryId": %d,
		"amount": %.2f,
		"isIncome": %t,
		"isTransfer": false,
		"label": "Updated"
	}`, txID, accountID, categoryID, amount, isIncome)

	rec := doRequest(t, app, http.MethodPut, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "update tx: %s", rec.Body.String())
}

// updateTxAmountWithDateTime edits an existing regular transaction's amount
// while pinning its date_time so chronological ordering vs. other rows is
// preserved. The service replaces date_time with time.Now().UTC() if no
// dateTime is supplied in the update payload — which means a naive amount
// edit on an old transaction would jump it to the present and reshuffle the
// recalc order. Tests that need stable ordering must use this helper.
func updateTxAmountWithDateTime(t *testing.T, app *testutil.TestApp, token string, txID, accountID, categoryID int, amount float64, isIncome bool, dateTime time.Time) {
	t.Helper()
	body := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"categoryId": %d,
		"amount": %.2f,
		"isIncome": %t,
		"isTransfer": false,
		"label": "Updated",
		"dateTime": "%s"
	}`, txID, accountID, categoryID, amount, isIncome, dateTime.UTC().Format(time.RFC3339))

	rec := doRequest(t, app, http.MethodPut, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "update tx: %s", rec.Body.String())
}

// dbBalance fetches the stored accounts.balance for the given account.
func dbBalance(t *testing.T, db *sql.DB, accountID int) float64 {
	t.Helper()
	var b float64
	err := db.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&b)
	require.NoError(t, err)
	return b
}

// TestRecalcNoAdjustmentsBaseline guards the unchanged behavior path: an
// account with only regular transactions must produce the naive
// initial + sum(income) - sum(expense) balance after recalc.
func TestRecalcNoAdjustmentsBaseline(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-baseline-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Recalc Baseline", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	// 1000 + 100 - 50 + 200 = 1250
	createTxReturningID(t, app, token, accountID, incomeCat, 100, true)
	expID := createTxReturningID(t, app, token, accountID, expCat, 50, false)
	createTxReturningID(t, app, token, accountID, incomeCat, 200, true)

	// Trigger a recalc by editing the expense (changing its amount).
	updateTxAmount(t, app, token, expID, accountID, expCat, 75, false)

	// 1000 + 100 - 75 + 200 = 1225
	assert.Equal(t, 1225.0, dbBalance(t, app.DB, accountID),
		"baseline recalc must be naive initial + sum(income) - sum(expense)")
}

// TestRecalcAdjustmentInTheMiddle is the pivotal regression test for the bug.
//
// Scenario:
//  1. Initial balance 1000.
//  2. Expense 100 -> 900.
//  3. Adjust to 1500.
//  4. Income 200 -> 1700.
//  5. Edit the income amount to 250 -> recalc must produce 1500 + 250 = 1750.
//
// Pre-fix code: balance = tx.Amount on adjustment row, so the running
// balance after the adjustment becomes 600 (the delta), final = 600 + 250 = 850.
// Post-fix code: balance = *tx.NewBalance, so running balance becomes 1500,
// final = 1500 + 250 = 1750.
func TestRecalcAdjustmentInTheMiddle(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-mid-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Mid Adj", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	// Step 2: expense 100 -> 900
	createTxReturningID(t, app, token, accountID, expCat, 100, false)
	// Step 3: adjustment to 1500
	adjustBalance(t, app, token, accountID, 1500)
	// Step 4: income 200 -> 1700
	incomeID := createTxReturningID(t, app, token, accountID, incomeCat, 200, true)

	// Sanity: balance is currently 1700.
	require.Equal(t, 1700.0, dbBalance(t, app.DB, accountID))

	// Step 5: edit income amount to 250, which triggers a recalc.
	updateTxAmount(t, app, token, incomeID, accountID, incomeCat, 250, true)

	// Expected: 1500 + 250 = 1750.
	assert.Equal(t, 1750.0, dbBalance(t, app.DB, accountID),
		"recalc after adjustment must base off adjustment.new_balance, not adjustment.amount (delta)")
}

// TestRecalcDownwardAdjustment guards the same logic but with a downward
// adjustment.
//
// Scenario:
//  1. 5000 initial.
//  2. Expense 200 -> 4800.
//  3. Adjust DOWN to 2000.
//  4. Income 100 -> 2100.
//  5. Expense 50 -> 2050.
//  6. Edit the last expense to 50 (no-op edit on amount; still triggers
//     recalc because the service updates the row regardless). Final must be 2050.
func TestRecalcDownwardAdjustment(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-down-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Down Adj", usdID, accTypeID, 5000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	createTxReturningID(t, app, token, accountID, expCat, 200, false)
	adjustBalance(t, app, token, accountID, 2000)
	createTxReturningID(t, app, token, accountID, incomeCat, 100, true)
	lastExpID := createTxReturningID(t, app, token, accountID, expCat, 50, false)

	// Sanity: 2000 + 100 - 50 = 2050 by the live update path.
	require.Equal(t, 2050.0, dbBalance(t, app.DB, accountID))

	// Trigger a recalc by re-editing the last expense to the same amount.
	updateTxAmount(t, app, token, lastExpID, accountID, expCat, 50, false)

	assert.Equal(t, 2050.0, dbBalance(t, app.DB, accountID))
}

// TestRecalcAdjustmentAsLastTransaction verifies that when the account's
// last transaction is an adjustment and a prior regular transaction is
// edited, the final balance equals the adjustment's target (no later
// regular transactions to add on top).
func TestRecalcAdjustmentAsLastTransaction(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-last-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Last Adj", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	// Pin explicit dates so the trailing adjustment stays trailing even after
	// the earlier expense is edited. The service rewrites date_time to "now"
	// on update unless one is supplied, which would otherwise reshuffle the
	// chronological recalc order.
	now := time.Now().UTC()
	pastExpenseDate := now.Add(-2 * time.Hour)
	createTxReturningID(t, app, token, accountID, incomeCat, 50, true)
	expID := createTxReturningID(t, app, token, accountID, expCat, 30, false)
	// Final adjustment to 1234.56 — this happens "now" so it sorts last.
	adjustBalance(t, app, token, accountID, 1234.56)

	require.InDelta(t, 1234.56, dbBalance(t, app.DB, accountID), 0.0001)

	// Edit the earlier expense, keeping its date_time in the past so the
	// adjustment remains the trailing transaction.
	updateTxAmountWithDateTime(t, app, token, expID, accountID, expCat, 9999, false, pastExpenseDate)

	// Adjustment is still last, so final balance must equal the target.
	assert.InDelta(t, 1234.56, dbBalance(t, app.DB, accountID), 0.0001,
		"recalc must land on the trailing adjustment's target regardless of earlier activity")
}

// TestRecalcAdjustmentAsFirstTransaction verifies that an adjustment as the
// very first transaction sets the running base, and subsequent regular
// transactions accumulate on top of the adjustment target.
func TestRecalcAdjustmentAsFirstTransaction(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-first-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	// Account opened with balance 0.
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "First Adj", usdID, accTypeID, 0)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	// First: adjustment 0 -> 500.
	adjustBalance(t, app, token, accountID, 500)
	// Then: income 100, expense 50.
	createTxReturningID(t, app, token, accountID, incomeCat, 100, true)
	expID := createTxReturningID(t, app, token, accountID, expCat, 50, false)

	require.Equal(t, 550.0, dbBalance(t, app.DB, accountID))

	// Trigger recalc by re-editing the expense.
	updateTxAmount(t, app, token, expID, accountID, expCat, 70, false)

	// 500 + 100 - 70 = 530
	assert.Equal(t, 530.0, dbBalance(t, app.DB, accountID))
}

// TestRecalcMultipleAdjustments verifies that with two or more adjustments
// in the same account, recalc keys off the LAST adjustment's new_balance
// plus subsequent regular activity.
func TestRecalcMultipleAdjustments(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-multi-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Multi Adj", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	incomeCat := testutil.CreateTestCategory(t, app.DB, userID, "Inc", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeCat)
	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	createTxReturningID(t, app, token, accountID, incomeCat, 100, true)                // 1100
	adjustBalance(t, app, token, accountID, 2000)                                      // 2000
	createTxReturningID(t, app, token, accountID, expCat, 300, false)                  // 1700
	adjustBalance(t, app, token, accountID, 750)                                       // 750
	lastIncomeID := createTxReturningID(t, app, token, accountID, incomeCat, 50, true) // 800

	require.Equal(t, 800.0, dbBalance(t, app.DB, accountID))

	// Edit the last income to 75 -> trigger recalc.
	updateTxAmount(t, app, token, lastIncomeID, accountID, incomeCat, 75, true)

	// 750 (last adjustment target) + 75 = 825.
	assert.Equal(t, 825.0, dbBalance(t, app.DB, accountID),
		"recalc must base off the last adjustment's new_balance plus subsequent regular activity")
}

// TestRecalcCrossCurrencyTransferEditOverAdjustment is the user's actual
// reproducer. An EUR account has a legacy adjustment in its history; a
// cross-currency transfer is created from it; editing that transfer
// triggers recalc. Pre-fix: source balance lands on a wildly wrong value.
// Post-fix: source balance equals
//
//	adjustment.new_balance + sum(post-adjustment regular transactions on A)
//	- new_transfer_amount
func TestRecalcCrossCurrencyTransferEditOverAdjustment(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("recalc-fx-over-adj-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Source: EUR account, init 1000.
	srcID := testutil.CreateTestAccount(t, app.DB, userID, "EUR Src", eurID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcID)
	// Destination: USD account, init 0.
	dstID := testutil.CreateTestAccount(t, app.DB, userID, "USD Dst", usdID, accTypeID, 0)
	defer testutil.DeleteTestAccount(t, app.DB, dstID)

	expCat := testutil.CreateTestCategory(t, app.DB, userID, "Exp", false)
	defer testutil.DeleteTestCategory(t, app.DB, expCat)

	// Legacy regular expense before the adjustment.
	createTxReturningID(t, app, token, srcID, expCat, 100, false) // 900
	// Legacy adjustment up to 5000 EUR.
	adjustBalance(t, app, token, srcID, 5000)
	// One small regular expense after the adjustment.
	createTxReturningID(t, app, token, srcID, expCat, 50, false) // 4950

	// Now create the cross-currency transfer: 200 EUR -> 220 USD.
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"targetAmount": 220.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "FX transfer over adjustment"
	}`, srcID, dstID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create FX transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	transferID := int(createResp["id"].(float64))

	// At this point src = 4950 - 200 = 4750, dst = 0 + 220 = 220 (live path).
	require.Equal(t, 4750.0, dbBalance(t, app.DB, srcID))
	require.Equal(t, 220.0, dbBalance(t, app.DB, dstID))

	// Edit the transfer: change source amount to 350 EUR -> 380 USD.
	// This forces recalculateBalances on both accounts.
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 350.00,
		"targetAmount": 380.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Updated FX transfer"
	}`, transferID, srcID, dstID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "update FX transfer: %s", rec.Body.String())

	// Expected source balance after recalc:
	//   adjustment.new_balance(5000) - post_adj_expense(50) - new_transfer(350) = 4600
	assert.Equal(t, 4600.0, dbBalance(t, app.DB, srcID),
		"post-fix recalc must use adjustment target as the running base")

	// Destination is straightforward (no adjustments): 0 + 380 = 380.
	assert.Equal(t, 380.0, dbBalance(t, app.DB, dstID))
}
