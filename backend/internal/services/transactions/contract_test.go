// Contract tests for the transactions service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package transactions_test

import (
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/transactions"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real transactions service connected to the test DB.
func setupTestService(t *testing.T) (transactions.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	svc := transactions.New(
		app.TransactionRepo,
		app.AccountRepo,
		app.CategoryRepo,
		app.UserRepo,
		app.TemplateRepo,
		nil, // currencyService: not needed for core CRUD contract tests
		nil, // enqueuer: budget recalculation enqueuing is best-effort
		logger,
	)
	return svc, app
}

// createTestUser creates a user and schedules cleanup.
func createTestUser(t *testing.T, app *testutil.TestApp, email string) int {
	t.Helper()
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, "Password123!")
	t.Cleanup(func() { testutil.CleanupUser(t, app.DB, userID) })
	return userID
}

// createAccount creates a test account for a user and schedules cleanup.
func createAccount(t *testing.T, app *testutil.TestApp, userID int) int {
	t.Helper()
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accID := testutil.CreateTestAccount(t, app.DB, userID, "Test Account", usdID, accTypeID, 1000.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, accID) })
	return accID
}

// --- CreateTransaction ---

func TestCreateTransaction_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-create@example.com")
	accID := createAccount(t, app, userID)

	now := time.Now().UTC()
	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(50.00),
		Label:     "Groceries",
		IsIncome:  false,
		DateTime:  &now,
	})

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, userID, tx.UserID)
	assert.Equal(t, accID, tx.AccountID)
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(50.00)))
	assert.False(t, tx.IsIncome)
	assert.NotZero(t, tx.ID)
}

func TestCreateTransaction_Income(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-income@example.com")
	accID := createAccount(t, app, userID)

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(200.00),
		Label:     "Salary",
		IsIncome:  true,
	})

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.IsIncome)
}

func TestCreateTransaction_InvalidAccount(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-badacc@example.com")

	_, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: 999999,
		Amount:    decimal.NewFromFloat(50.00),
		Label:     "Nope",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrInvalidAccount)
}

func TestCreateTransaction_InvalidUser(t *testing.T) {
	svc, app := setupTestService(t)
	// Create an account owned by user1, then try creating a transaction as non-existent user
	userID := createTestUser(t, app, "txn-contract-baduser@example.com")
	accID := createAccount(t, app, userID)

	_, err := svc.CreateTransaction(999999, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(50.00),
		Label:     "Bad User",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrInvalidUser)
}

func TestCreateTransaction_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "txn-contract-crd1@example.com")
	user2 := createTestUser(t, app, "txn-contract-crd2@example.com")
	accID := createAccount(t, app, user1)

	_, err := svc.CreateTransaction(user2, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(50.00),
		Label:     "Stolen",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrAccessDenied)
}

func TestCreateTransaction_SelfTransferRejected(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-selftx@example.com")
	accID := createAccount(t, app, userID)

	_, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID:       accID,
		TargetAccountID: &accID,
		Amount:          decimal.NewFromFloat(100.00),
		Label:           "Self Transfer",
		IsTransfer:      true,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrSelfTransfer)
}

func TestCreateTransaction_Transfer(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-transfer@example.com")
	accID1 := createAccount(t, app, userID)
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accID2 := testutil.CreateTestAccount(t, app.DB, userID, "Second Account", usdID, accTypeID, 500.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, accID2) })

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID:       accID1,
		TargetAccountID: &accID2,
		Amount:          decimal.NewFromFloat(100.00),
		Label:           "Transfer",
		IsTransfer:      true,
	})

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.IsTransfer)
	assert.NotNil(t, tx.LinkedTransactionID, "transfer should create a linked transaction")
}

// --- GetTransactions ---

func TestGetTransactions_ReturnsCreated(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-list@example.com")
	accID := createAccount(t, app, userID)

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(25.00),
		Label:     "Coffee",
	})
	require.NoError(t, err)

	result, err := svc.GetTransactions(userID, transactions.GetTransactionsInput{
		Page:    1,
		PerPage: 50,
	})

	require.NoError(t, err)
	found := false
	for _, r := range result {
		if r.ID == tx.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created transaction should appear in the list")
}

func TestGetTransactions_InvalidUser(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetTransactions(999999, transactions.GetTransactionsInput{
		Page:    1,
		PerPage: 50,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrInvalidUser)
}

// --- GetTransactionDetails ---

func TestGetTransactionDetails_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-details@example.com")
	accID := createAccount(t, app, userID)

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "Lunch",
	})
	require.NoError(t, err)

	result, err := svc.GetTransactionDetails(tx.ID, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, tx.ID, result.ID)
}

func TestGetTransactionDetails_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-detnf@example.com")

	_, err := svc.GetTransactionDetails(999999, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrTransactionNotFound)
}

func TestGetTransactionDetails_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "txn-contract-det1@example.com")
	user2 := createTestUser(t, app, "txn-contract-det2@example.com")
	accID := createAccount(t, app, user1)

	tx, err := svc.CreateTransaction(user1, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "Private",
	})
	require.NoError(t, err)

	_, err = svc.GetTransactionDetails(tx.ID, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrAccessDenied)
}

// --- UpdateTransaction ---

func TestUpdateTransaction_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-update@example.com")
	accID := createAccount(t, app, userID)

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "Original",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(45.00),
		Label:     "Updated",
	}, tx.ID)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, updated.Amount.Equal(decimal.NewFromFloat(45.00)))
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updnf@example.com")
	accID := createAccount(t, app, userID)

	_, err := svc.UpdateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "Nope",
	}, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrTransactionNotFound)
}

// --- DeleteTransaction ---

func TestDeleteTransaction_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-delete@example.com")
	accID := createAccount(t, app, userID)

	tx, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "ToDelete",
	})
	require.NoError(t, err)

	deleted, err := svc.DeleteTransaction(tx.ID, userID)

	require.NoError(t, err)
	require.NotNil(t, deleted)
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-delnf@example.com")

	_, err := svc.DeleteTransaction(999999, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrTransactionNotFound)
}

func TestDeleteTransaction_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "txn-contract-del1@example.com")
	user2 := createTestUser(t, app, "txn-contract-del2@example.com")
	accID := createAccount(t, app, user1)

	tx, err := svc.CreateTransaction(user1, transactions.CreateTransactionInput{
		AccountID: accID,
		Amount:    decimal.NewFromFloat(30.00),
		Label:     "Private",
	})
	require.NoError(t, err)

	_, err = svc.DeleteTransaction(tx.ID, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrAccessDenied)
}

// --- GetTemplates ---

func TestGetTemplates_ReturnsTemplates(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-tmpl@example.com")
	accID := createAccount(t, app, userID)

	// Create a transaction with IsTemplate=true to generate a template
	_, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID:  accID,
		Amount:     decimal.NewFromFloat(30.00),
		Label:      "Template Transaction",
		IsTemplate: true,
	})
	require.NoError(t, err)

	templates, err := svc.GetTemplates(userID)

	require.NoError(t, err)
	// Should have at least one template
	found := false
	for _, tmpl := range templates {
		if tmpl.Label == "Template Transaction" {
			found = true
			break
		}
	}
	assert.True(t, found, "template should be created for IsTemplate=true transaction")
}

func TestGetTemplates_EmptyForNewUser(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-tmplempty@example.com")

	templates, err := svc.GetTemplates(userID)

	require.NoError(t, err)
	assert.Empty(t, templates)
}

// --- UpdateTemplate ---

func TestUpdateTemplate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmpl@example.com")

	// Create a template directly
	tmplID := testutil.CreateTestTemplate(t, app.DB, userID, "Original Label", nil)
	t.Cleanup(func() { testutil.DeleteTestTemplate(t, app.DB, tmplID) })

	updated, err := svc.UpdateTemplate(userID, tmplID, "New Label", nil, nil)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "New Label", updated.Label)
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmplnf@example.com")

	_, err := svc.UpdateTemplate(userID, 999999, "Nope", nil, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrTemplateNotFound)
}

// --- DeleteTemplates ---

func TestDeleteTemplates_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-deltmpl@example.com")

	tmplID := testutil.CreateTestTemplate(t, app.DB, userID, "To Delete", nil)

	// DeleteTemplates silently skips non-owned/missing templates and returns
	// the remaining templates for the user after deletion.
	remaining, err := svc.DeleteTemplates(userID, []int{tmplID})

	require.NoError(t, err)
	// After deleting the only template, remaining should be empty
	assert.Empty(t, remaining)

	// Verify template was actually deleted
	templates, err := svc.GetTemplates(userID)
	require.NoError(t, err)
	for _, tmpl := range templates {
		assert.NotEqual(t, tmplID, tmpl.ID, "deleted template should not appear")
	}
}

func TestDeleteTemplates_NonExistentIDSkipped(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-deltmplnf@example.com")

	// DeleteTemplates silently skips IDs that don't exist or belong to other users.
	// It does not return an error for missing templates.
	remaining, err := svc.DeleteTemplates(userID, []int{999999})

	require.NoError(t, err)
	assert.Empty(t, remaining)
}

// --- Transfer Template Tests ---

func TestCreateTransaction_TransferTemplate(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-xfertmpl@example.com")
	sourceAccID := createAccount(t, app, userID)

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccID := testutil.CreateTestAccount(t, app.DB, userID, "Savings", usdID, accTypeID, 500.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, targetAccID) })

	_, err := svc.CreateTransaction(userID, transactions.CreateTransactionInput{
		AccountID:       sourceAccID,
		TargetAccountID: &targetAccID,
		Amount:          decimal.NewFromFloat(100.00),
		Label:           "Transfer Template Test",
		IsTransfer:      true,
		IsTemplate:      true,
	})
	require.NoError(t, err)

	templates, err := svc.GetTemplates(userID)
	require.NoError(t, err)

	found := false
	for _, tmpl := range templates {
		if tmpl.Label == "Transfer Template Test" {
			found = true
			require.NotNil(t, tmpl.TargetAccountID)
			assert.Equal(t, targetAccID, *tmpl.TargetAccountID)
			require.NotNil(t, tmpl.TargetAccount)
			assert.Equal(t, "Savings", tmpl.TargetAccount.Name)
			break
		}
	}
	assert.True(t, found, "transfer template should be created with TargetAccountID")
}

func TestUpdateTemplate_WithTargetAccountID(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmplta@example.com")

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccID := testutil.CreateTestAccount(t, app.DB, userID, "Investment", usdID, accTypeID, 2000.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, targetAccID) })

	tmplID := testutil.CreateTestTemplate(t, app.DB, userID, "Regular Label", nil)
	t.Cleanup(func() { testutil.DeleteTestTemplate(t, app.DB, tmplID) })

	updated, err := svc.UpdateTemplate(userID, tmplID, "Transfer Label", nil, &targetAccID)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Transfer Label", updated.Label)
	require.NotNil(t, updated.TargetAccountID)
	assert.Equal(t, targetAccID, *updated.TargetAccountID)
	require.NotNil(t, updated.TargetAccount)
	assert.Equal(t, "Investment", updated.TargetAccount.Name)
}

func TestUpdateTemplate_OtherUsersAccountDenied(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmplidor@example.com")
	otherUserID := createTestUser(t, app, "txn-contract-updtmplidor-other@example.com")

	// Create account owned by otherUser
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	otherAccID := testutil.CreateTestAccount(t, app.DB, otherUserID, "Other User Account", usdID, accTypeID, 500.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, otherAccID) })

	tmplID := testutil.CreateTestTemplate(t, app.DB, userID, "My Template", nil)
	t.Cleanup(func() { testutil.DeleteTestTemplate(t, app.DB, tmplID) })

	// Try to set targetAccountID to another user's account
	_, err := svc.UpdateTemplate(userID, tmplID, "My Template", nil, &otherAccID)

	require.Error(t, err)
	assert.ErrorIs(t, err, transactions.ErrAccessDenied)
}

func TestUpdateTemplate_DeletedTargetAccountSetsNull(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmpldelacct@example.com")

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccID := testutil.CreateTestAccount(t, app.DB, userID, "Temp Account", usdID, accTypeID, 500.00)

	tmplID := testutil.CreateTestTransferTemplate(t, app.DB, userID, "Transfer Label", targetAccID)
	t.Cleanup(func() { testutil.DeleteTestTemplate(t, app.DB, tmplID) })

	// Delete the target account — ON DELETE SET NULL should nullify the FK
	testutil.DeleteTestAccount(t, app.DB, targetAccID)

	// Template should survive with nil TargetAccountID
	templates, err := svc.GetTemplates(userID)
	require.NoError(t, err)

	found := false
	for _, tmpl := range templates {
		if tmpl.ID == tmplID {
			found = true
			assert.Nil(t, tmpl.TargetAccountID, "targetAccountId should be null after account deletion")
			assert.Nil(t, tmpl.TargetAccount, "targetAccount relation should be nil after account deletion")
			break
		}
	}
	assert.True(t, found, "template should survive after target account deletion")
}

func TestUpdateTemplate_RemoveTargetAccountID(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "txn-contract-updtmplrmta@example.com")

	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	targetAccID := testutil.CreateTestAccount(t, app.DB, userID, "Target", usdID, accTypeID, 500.00)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, targetAccID) })

	tmplID := testutil.CreateTestTransferTemplate(t, app.DB, userID, "Transfer Label", targetAccID)
	t.Cleanup(func() { testutil.DeleteTestTemplate(t, app.DB, tmplID) })

	// Update with nil targetAccountID to remove it
	updated, err := svc.UpdateTemplate(userID, tmplID, "Regular Label", nil, nil)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Regular Label", updated.Label)
	assert.Nil(t, updated.TargetAccountID)
	assert.Nil(t, updated.TargetAccount)
}
