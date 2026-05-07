// Contract tests for the accounts service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package accounts_test

import (
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/accounts"
	currencyservice "github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real accounts service connected to the test DB.
func setupTestService(t *testing.T) (accounts.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	currencySvc := currencyservice.NewWithCurrencyRepo(app.ExchangeRateRepo, app.CurrencyRepo, logger)
	svc := accounts.New(
		app.AccountRepo,
		app.AccountTypeRepo,
		app.CurrencyRepo,
		app.UserRepo,
		app.TransactionRepo,
		currencySvc,
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

// defaultCreateInput returns a valid CreateAccountInput for testing.
func defaultCreateInput(t *testing.T, app *testutil.TestApp) accounts.CreateAccountInput {
	t.Helper()
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	accTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	return accounts.CreateAccountInput{
		Name:           "Test Account",
		CurrencyID:     usdID,
		AccountTypeID:  accTypeID,
		InitialBalance: decimal.NewFromFloat(100.00),
		Balance:        decimal.NewFromFloat(100.00),
		CreditLimit:    decimal.Zero,
		IsHidden:       false,
		ShowInReports:  true,
	}
}

// --- CreateAccount ---

func TestCreateAccount_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-create@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)

	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.Equal(t, "Test Account", acc.Name)
	assert.Equal(t, userID, acc.UserID)
	assert.True(t, acc.Balance.Equal(decimal.NewFromFloat(100.00)))
	assert.NotZero(t, acc.ID)
	assert.NotNil(t, acc.Currency)
	assert.NotNil(t, acc.AccountType)

	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })
}

func TestCreateAccount_InvalidCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-badcurr@example.com")
	input := defaultCreateInput(t, app)
	input.CurrencyID = 999999

	_, err := svc.CreateAccount(userID, input)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrInvalidCurrency)
}

func TestCreateAccount_InvalidAccountType(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-badtype@example.com")
	input := defaultCreateInput(t, app)
	input.AccountTypeID = 999999

	_, err := svc.CreateAccount(userID, input)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrInvalidAccountType)
}

func TestCreateAccount_InvalidUser(t *testing.T) {
	svc, app := setupTestService(t)
	input := defaultCreateInput(t, app)

	_, err := svc.CreateAccount(999999, input)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrInvalidUser)
}

func TestCreateAccount_UserNotActivated(t *testing.T) {
	svc, app := setupTestService(t)
	testutil.CleanupUserByEmail(t, app.DB, "accounts-contract-inactive@example.com")
	result, err := app.AuthService.Register("accounts-contract-inactive@example.com", "Password123!", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupUser(t, app.DB, result.User.ID) })
	// User is not activated after registration

	input := defaultCreateInput(t, app)
	_, err = svc.CreateAccount(result.User.ID, input)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrUserNotActivated)
}

// --- GetUserAccounts ---

func TestGetUserAccounts_ReturnsCreatedAccounts(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-list@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	result, err := svc.GetUserAccounts(userID, accounts.GetAccountsInput{
		IncludeHidden: true,
	})

	require.NoError(t, err)
	found := false
	for _, a := range result {
		if a.ID == acc.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created account should be in the list")
}

func TestGetUserAccounts_ExcludesHiddenByDefault(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-hidden@example.com")
	input := defaultCreateInput(t, app)
	input.IsHidden = true

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	result, err := svc.GetUserAccounts(userID, accounts.GetAccountsInput{
		IncludeHidden: false,
	})

	require.NoError(t, err)
	for _, a := range result {
		assert.NotEqual(t, acc.ID, a.ID, "hidden account should be excluded when IncludeHidden=false")
	}
}

func TestGetUserAccounts_InvalidUser(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetUserAccounts(999999, accounts.GetAccountsInput{})

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrInvalidUser)
}

// --- GetAccountDetails ---

func TestGetAccountDetails_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-details@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	result, err := svc.GetAccountDetails(acc.ID, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, acc.ID, result.ID)
	assert.Equal(t, "Test Account", result.Name)
}

func TestGetAccountDetails_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-detailsnf@example.com")

	_, err := svc.GetAccountDetails(999999, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestGetAccountDetails_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "accounts-contract-det1@example.com")
	user2 := createTestUser(t, app, "accounts-contract-det2@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(user1, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	_, err = svc.GetAccountDetails(acc.ID, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccessDenied)
}

// --- GetAccountTypes ---

func TestGetAccountTypes_ReturnsTypes(t *testing.T) {
	svc, _ := setupTestService(t)

	types, err := svc.GetAccountTypes()

	require.NoError(t, err)
	require.NotEmpty(t, types)
	for _, at := range types {
		assert.NotZero(t, at.ID)
		assert.NotEmpty(t, at.TypeName)
	}
}

// --- UpdateAccount ---

func TestUpdateAccount_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-upd@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	updated, err := svc.UpdateAccount(acc.ID, userID, accounts.UpdateAccountInput{
		Name:           "Updated Name",
		CurrencyID:     input.CurrencyID,
		AccountTypeID:  input.AccountTypeID,
		InitialBalance: decimal.NewFromFloat(200.00),
		CreditLimit:    decimal.Zero,
		IsHidden:       false,
		ShowInReports:  true,
		OpeningDate:    time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.True(t, updated.InitialBalance.Equal(decimal.NewFromFloat(200.00)))
}

func TestUpdateAccount_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-updnf@example.com")

	_, err := svc.UpdateAccount(999999, userID, accounts.UpdateAccountInput{
		Name:          "Nope",
		CurrencyID:    testutil.GetCurrencyID(t, app.DB, "USD"),
		AccountTypeID: testutil.GetAccountTypeID(t, app.DB, false),
		OpeningDate:   time.Now().UTC(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestUpdateAccount_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "accounts-contract-upd1@example.com")
	user2 := createTestUser(t, app, "accounts-contract-upd2@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(user1, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	_, err = svc.UpdateAccount(acc.ID, user2, accounts.UpdateAccountInput{
		Name:          "Stolen",
		CurrencyID:    input.CurrencyID,
		AccountTypeID: input.AccountTypeID,
		OpeningDate:   time.Now().UTC(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccessDenied)
}

func TestUpdateAccount_InvalidCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-updcurr@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	_, err = svc.UpdateAccount(acc.ID, userID, accounts.UpdateAccountInput{
		Name:          "Bad Currency",
		CurrencyID:    999999,
		AccountTypeID: input.AccountTypeID,
		OpeningDate:   time.Now().UTC(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrInvalidCurrency)
}

// --- DeleteAccount ---

func TestDeleteAccount_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-del@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)

	err = svc.DeleteAccount(acc.ID, userID)

	require.NoError(t, err)
}

func TestDeleteAccount_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-delnf@example.com")

	err := svc.DeleteAccount(999999, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestDeleteAccount_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "accounts-contract-del1@example.com")
	user2 := createTestUser(t, app, "accounts-contract-del2@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(user1, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	err = svc.DeleteAccount(acc.ID, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccessDenied)
}

// --- SetArchiveStatus ---

func TestSetArchiveStatus_Archive(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-archive@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	err = svc.SetArchiveStatus(acc.ID, true, userID)
	require.NoError(t, err)

	// Verify the account is archived
	details, err := svc.GetAccountDetails(acc.ID, userID)
	require.NoError(t, err)
	assert.True(t, details.IsArchived)
}

func TestSetArchiveStatus_Unarchive(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-unarchive@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	// Archive first
	err = svc.SetArchiveStatus(acc.ID, true, userID)
	require.NoError(t, err)

	// Unarchive
	err = svc.SetArchiveStatus(acc.ID, false, userID)
	require.NoError(t, err)

	details, err := svc.GetAccountDetails(acc.ID, userID)
	require.NoError(t, err)
	assert.False(t, details.IsArchived)
}

func TestSetArchiveStatus_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-arcnf@example.com")

	err := svc.SetArchiveStatus(999999, true, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestSetArchiveStatus_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "accounts-contract-arc1@example.com")
	user2 := createTestUser(t, app, "accounts-contract-arc2@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(user1, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	err = svc.SetArchiveStatus(acc.ID, true, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccessDenied)
}

// --- AdjustBalance ---

func TestAdjustBalance_IncreasesBalance(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-adjust@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	newBalance := decimal.NewFromFloat(200.00)
	notes := "test adjustment"
	tx, err := svc.AdjustBalance(acc.ID, newBalance, &notes, userID)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.IsAdjustment)
	assert.True(t, tx.IsIncome, "increasing balance should be income")

	// Verify account balance updated
	details, err := svc.GetAccountDetails(acc.ID, userID)
	require.NoError(t, err)
	assert.True(t, details.Balance.Equal(newBalance))
}

func TestAdjustBalance_DecreasesBalance(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-adjustdec@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	newBalance := decimal.NewFromFloat(50.00)
	tx, err := svc.AdjustBalance(acc.ID, newBalance, nil, userID)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.False(t, tx.IsIncome, "decreasing balance should not be income")
}

func TestAdjustBalance_Unchanged(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-adjustsame@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(userID, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	_, err = svc.AdjustBalance(acc.ID, acc.Balance, nil, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrBalanceUnchanged)
}

func TestAdjustBalance_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "accounts-contract-adjustnf@example.com")

	_, err := svc.AdjustBalance(999999, decimal.NewFromFloat(100.00), nil, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestAdjustBalance_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "accounts-contract-adj1@example.com")
	user2 := createTestUser(t, app, "accounts-contract-adj2@example.com")
	input := defaultCreateInput(t, app)

	acc, err := svc.CreateAccount(user1, input)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, acc.ID) })

	_, err = svc.AdjustBalance(acc.ID, decimal.NewFromFloat(200.00), nil, user2)

	require.Error(t, err)
	assert.ErrorIs(t, err, accounts.ErrAccessDenied)
}
