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

// assertAccountBalance is a helper to check that an account has the expected balance.
func assertAccountBalance(t *testing.T, db *sql.DB, accountID int, expected float64) {
	t.Helper()
	var balance float64
	err := db.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, expected, balance, "Account %d balance mismatch", accountID)
}

// TestUpdateTransferAmountUpdatesBothBalances verifies that when a transfer's
// amount is updated, the linked transaction's amount also changes and both
// account balances are recalculated correctly.
//
// Scenario:
//  1. Create source (USD, 1000) and destination (USD, 500) accounts.
//  2. Transfer 200 from source to dest. Source=800, Dest=700.
//  3. Update the transfer amount to 300.
//  4. Verify source=700, dest=800 (recalculated from initial).
func TestUpdateTransferAmountUpdatesBothBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("update-transfer-amount-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src Account", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst Account", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Create transfer: 200 from src to dst
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Original transfer"
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	linkedTxID := int(createResp["linkedTransactionId"].(float64))

	// Verify intermediate balances
	assertAccountBalance(t, app.DB, srcAccountID, 800.0)
	assertAccountBalance(t, app.DB, dstAccountID, 700.0)

	// Update transfer amount to 300
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 300.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Updated transfer"
	}`, txID, srcAccountID, dstAccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "update transfer: %s", rec.Body.String())

	// Verify balances after update
	assertAccountBalance(t, app.DB, srcAccountID, 700.0)
	assertAccountBalance(t, app.DB, dstAccountID, 800.0)

	// Verify linked transaction amount was also updated
	var linkedAmount float64
	err := app.DB.QueryRow("SELECT amount FROM transactions WHERE id = $1 AND is_deleted = false", linkedTxID).Scan(&linkedAmount)
	require.NoError(t, err)
	assert.Equal(t, 300.0, linkedAmount, "Linked transaction amount should be updated to 300")
}

// TestUpdateTransferTargetAccountRecalculatesBalances verifies that changing the
// target account of a transfer correctly recalculates balances for the old target,
// the new target, and moves the linked transaction.
//
// Scenario:
//  1. Create src (1000), dst1 (500), dst2 (200).
//  2. Transfer 100 from src to dst1. src=900, dst1=600.
//  3. Update transfer to target dst2 instead.
//  4. Verify src=900, dst1=500 (restored), dst2=300.
func TestUpdateTransferTargetAccountRecalculatesBalances(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("update-transfer-target-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dst1AccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst1", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dst1AccountID)

	dst2AccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst2", usdID, accTypeID, 200)
	defer testutil.DeleteTestAccount(t, app.DB, dst2AccountID)

	// Create transfer: 100 from src to dst1
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "To dst1"
	}`, srcAccountID, dst1AccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	linkedTxID := int(createResp["linkedTransactionId"].(float64))

	// Verify intermediate balances
	assertAccountBalance(t, app.DB, srcAccountID, 900.0)
	assertAccountBalance(t, app.DB, dst1AccountID, 600.0)
	assertAccountBalance(t, app.DB, dst2AccountID, 200.0)

	// Update transfer: change target to dst2
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "To dst2 now"
	}`, txID, srcAccountID, dst2AccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "update transfer target: %s", rec.Body.String())

	// Verify balances: src unchanged, dst1 restored, dst2 credited
	assertAccountBalance(t, app.DB, srcAccountID, 900.0)
	assertAccountBalance(t, app.DB, dst1AccountID, 500.0)
	assertAccountBalance(t, app.DB, dst2AccountID, 300.0)

	// Verify linked transaction moved to dst2
	var linkedAccountID int
	err := app.DB.QueryRow("SELECT account_id FROM transactions WHERE id = $1 AND is_deleted = false", linkedTxID).Scan(&linkedAccountID)
	require.NoError(t, err)
	assert.Equal(t, dst2AccountID, linkedAccountID, "Linked transaction should belong to dst2")
}

// TestConvertTransferToRegularTransaction verifies that converting a transfer
// to a regular transaction soft-deletes the linked tx and recalculates the
// target account balance.
//
// Scenario:
//  1. Create src (1000), dst (500).
//  2. Transfer 200 from src to dst. src=800, dst=700.
//  3. Update: change isTransfer to false.
//  4. Verify dst=500 (restored), linked tx is deleted.
func TestConvertTransferToRegularTransaction(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("convert-transfer-to-regular-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Create transfer: 200 from src to dst
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Transfer"
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	linkedTxID := int(createResp["linkedTransactionId"].(float64))

	// Verify intermediate balances
	assertAccountBalance(t, app.DB, srcAccountID, 800.0)
	assertAccountBalance(t, app.DB, dstAccountID, 700.0)

	// Convert to regular transaction (isTransfer = false)
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"amount": 200.00,
		"isIncome": false,
		"isTransfer": false,
		"label": "Now regular expense"
	}`, txID, srcAccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "convert to regular: %s", rec.Body.String())

	var updateResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updateResp))
	assert.Nil(t, updateResp["linkedTransactionId"], "Source should have no linked transaction after converting to regular")

	// Verify balances: src still 800 (same expense), dst restored to 500
	assertAccountBalance(t, app.DB, srcAccountID, 800.0)
	assertAccountBalance(t, app.DB, dstAccountID, 500.0)

	// Verify linked transaction is soft-deleted
	var isDeleted bool
	err := app.DB.QueryRow("SELECT is_deleted FROM transactions WHERE id = $1", linkedTxID).Scan(&isDeleted)
	require.NoError(t, err)
	assert.True(t, isDeleted, "Linked transaction should be soft-deleted")
}

// TestConvertRegularToTransfer verifies that converting a regular transaction
// into a transfer creates a new linked transaction and adjusts balances.
//
// Scenario:
//  1. Create src (1000), dst (500).
//  2. Create regular expense of 100 on src. src=900.
//  3. Update: change isTransfer to true, set targetAccountId to dst.
//  4. Verify dst=600 (received 100), linked tx created.
func TestConvertRegularToTransfer(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("convert-regular-to-transfer-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Create regular expense: 100 on src
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": false,
		"label": "Regular expense"
	}`, srcAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create expense: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))

	// Verify intermediate balances
	assertAccountBalance(t, app.DB, srcAccountID, 900.0)
	assertAccountBalance(t, app.DB, dstAccountID, 500.0)

	// Convert to transfer
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Now a transfer"
	}`, txID, srcAccountID, dstAccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "convert to transfer: %s", rec.Body.String())

	var updateResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updateResp))
	assert.NotNil(t, updateResp["linkedTransactionId"], "Source should have a linked transaction after converting to transfer")
	assert.Equal(t, true, updateResp["isTransfer"])

	// Verify balances: src=900 (still debited), dst=600 (received 100)
	assertAccountBalance(t, app.DB, srcAccountID, 900.0)
	assertAccountBalance(t, app.DB, dstAccountID, 600.0)

	// Verify linked transaction exists and is correct
	linkedTxID := int(updateResp["linkedTransactionId"].(float64))
	var linkedIsIncome, linkedIsTransfer bool
	var linkedAccountID int
	var linkedAmount float64
	err := app.DB.QueryRow(
		"SELECT account_id, amount, is_income, is_transfer FROM transactions WHERE id = $1 AND is_deleted = false",
		linkedTxID,
	).Scan(&linkedAccountID, &linkedAmount, &linkedIsIncome, &linkedIsTransfer)
	require.NoError(t, err)
	assert.Equal(t, dstAccountID, linkedAccountID)
	assert.Equal(t, 100.0, linkedAmount)
	assert.True(t, linkedIsIncome, "Linked transaction should be income")
	assert.True(t, linkedIsTransfer, "Linked transaction should be a transfer")

	// Verify the linked transaction's linked_transaction_id points back to the source
	var linkedLinkedTxID int
	err = app.DB.QueryRow("SELECT linked_transaction_id FROM transactions WHERE id = $1", linkedTxID).Scan(&linkedLinkedTxID)
	require.NoError(t, err)
	assert.Equal(t, txID, linkedLinkedTxID, "Linked transaction should point back to source")
}

// TestUpdateTransferLabelNotesDateTime verifies that updating a transfer's
// label, notes, datetime, excludeFromReports, and categoryId mirrors the
// changes on the linked transaction.
func TestUpdateTransferLabelNotesDateTime(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("update-transfer-fields-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Dst", usdID, accTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Create a test category to verify categoryId mirroring
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Transfer Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Create transfer
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Original label",
		"notes": "Original notes"
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	linkedTxID := int(createResp["linkedTransactionId"].(float64))

	// Update label, notes, datetime, excludeFromReports, and categoryId
	newDateTime := "2025-06-15T10:30:00Z"
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Updated label",
		"notes": "Updated notes",
		"dateTime": "%s",
		"excludeFromReports": true,
		"categoryId": %d
	}`, txID, srcAccountID, dstAccountID, newDateTime, categoryID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "update transfer fields: %s", rec.Body.String())

	// Verify linked transaction mirrors the changes
	var linkedLabel, linkedNotes string
	var linkedDateTime time.Time
	var linkedExcludeFromReports bool
	var linkedCategoryID *int
	err := app.DB.QueryRow(
		"SELECT label, notes, date_time, exclude_from_reports, category_id FROM transactions WHERE id = $1 AND is_deleted = false",
		linkedTxID,
	).Scan(&linkedLabel, &linkedNotes, &linkedDateTime, &linkedExcludeFromReports, &linkedCategoryID)
	require.NoError(t, err)
	assert.Equal(t, "Updated label", linkedLabel, "Linked tx label should be updated")
	assert.Equal(t, "Updated notes", linkedNotes, "Linked tx notes should be updated")
	expectedTime, _ := time.Parse(time.RFC3339, newDateTime)
	assert.True(t, linkedDateTime.Equal(expectedTime), "Linked tx datetime should be updated, got %v expected %v", linkedDateTime, expectedTime)
	assert.True(t, linkedExcludeFromReports, "Linked tx excludeFromReports should be true")
	require.NotNil(t, linkedCategoryID, "Linked tx categoryId should not be nil")
	assert.Equal(t, categoryID, *linkedCategoryID, "Linked tx categoryId should match the source")
}

// TestCrossCurrencyTransferUpdateWithTargetAmount verifies that when updating
// a cross-currency transfer with a TargetAmount, the linked transaction uses
// the TargetAmount and the destination balance is correct.
//
// Scenario:
//  1. Create src (USD, 1000), dst (EUR, 0).
//  2. Transfer 100 USD -> 92 EUR. src=900, dst=92.
//  3. Update: change to 200 USD -> 180 EUR.
//  4. Verify src=800, dst=180.
func TestCrossCurrencyTransferUpdateWithTargetAmount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("update-transfer-fx-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "USD Account", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	dstAccountID := testutil.CreateTestAccount(t, app.DB, userID, "EUR Account", eurID, accTypeID, 0)
	defer testutil.DeleteTestAccount(t, app.DB, dstAccountID)

	// Create transfer: 100 USD -> 92 EUR
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"targetAmount": 92.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "FX transfer"
	}`, srcAccountID, dstAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create FX transfer: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))
	linkedTxID := int(createResp["linkedTransactionId"].(float64))

	// Verify intermediate balances
	assertAccountBalance(t, app.DB, srcAccountID, 900.0)
	assertAccountBalance(t, app.DB, dstAccountID, 92.0)

	// Update: 200 USD -> 180 EUR
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 200.00,
		"targetAmount": 180.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Updated FX transfer"
	}`, txID, srcAccountID, dstAccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "update FX transfer: %s", rec.Body.String())

	// Verify balances after update
	assertAccountBalance(t, app.DB, srcAccountID, 800.0)
	assertAccountBalance(t, app.DB, dstAccountID, 180.0)

	// Verify linked transaction uses TargetAmount
	var linkedAmount float64
	err := app.DB.QueryRow("SELECT amount FROM transactions WHERE id = $1 AND is_deleted = false", linkedTxID).Scan(&linkedAmount)
	require.NoError(t, err)
	assert.Equal(t, 180.0, linkedAmount, "Linked transaction should use targetAmount (180 EUR)")
}

// TestSelfTransferRejected verifies that updating a transfer where
// source and target account are the same returns an error.
func TestSelfTransferRejected(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("self-transfer-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	srcAccountID := testutil.CreateTestAccount(t, app.DB, userID, "Src", usdID, accTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, srcAccountID)

	// Create a regular expense first
	createBody := fmt.Sprintf(`{
		"accountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": false,
		"label": "Expense"
	}`, srcAccountID)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", createBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "create expense: %s", rec.Body.String())

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	txID := int(createResp["id"].(float64))

	// Attempt to convert to self-transfer (same source and target)
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"accountId": %d,
		"targetAccountId": %d,
		"amount": 100.00,
		"isIncome": false,
		"isTransfer": true,
		"label": "Self transfer"
	}`, txID, srcAccountID, srcAccountID)

	rec = doRequest(t, app, http.MethodPut, "/transactions/", updateBody, token)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "self-transfer should be rejected")

	var errResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp["detail"], "Source and target accounts must be different")
}
