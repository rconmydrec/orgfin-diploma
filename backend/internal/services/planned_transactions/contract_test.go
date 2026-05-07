// Contract tests for the planned_transactions service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package planned_transactions_test

import (
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/planned_transactions"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real planned_transactions service connected to the test DB.
func setupTestService(t *testing.T) (planned_transactions.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	svc := planned_transactions.New(app.PlannedTxRepo, testutil.TestLogger())
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

// getBaseCurrencyID returns the user's base currency ID.
func getBaseCurrencyID(t *testing.T, app *testutil.TestApp, userID int) int {
	t.Helper()
	var currencyID int
	err := app.DB.QueryRow("SELECT base_currency_id FROM users WHERE id = $1", userID).Scan(&currencyID)
	require.NoError(t, err)
	return currencyID
}

// cleanupPlannedTx removes a planned transaction from the test database.
func cleanupPlannedTx(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM planned_transactions WHERE id = $1", id)
}

// defaultCreateParams returns valid CreateParams for testing.
func defaultCreateParams() planned_transactions.CreateParams {
	return planned_transactions.CreateParams{
		Amount:      decimal.NewFromFloat(50.00),
		Label:       "Monthly Rent",
		IsIncome:    false,
		PlannedDate: time.Now().Add(7 * 24 * time.Hour),
		IsRecurring: false,
	}
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-create@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)
	params := defaultCreateParams()

	ptx, err := svc.Create(userID, baseCurrencyID, params)

	require.NoError(t, err)
	require.NotNil(t, ptx)
	assert.Equal(t, "Monthly Rent", ptx.Label)
	assert.Equal(t, userID, ptx.UserID)
	assert.True(t, ptx.Amount.Equal(decimal.NewFromFloat(50.00)))
	assert.False(t, ptx.IsIncome)
	assert.False(t, ptx.IsRecurring)
	assert.NotZero(t, ptx.ID)

	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })
}

func TestCreate_WithRecurrenceRule(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-recurring@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	interval := 1
	dayOfMonth := 15
	params := planned_transactions.CreateParams{
		Amount:      decimal.NewFromFloat(1200.00),
		Label:       "Salary",
		IsIncome:    true,
		PlannedDate: time.Now().Add(7 * 24 * time.Hour),
		IsRecurring: true,
		RecurrenceRule: &planned_transactions.RecurrenceRuleParams{
			Frequency:  "MONTHLY",
			Interval:   interval,
			DayOfMonth: &dayOfMonth,
		},
	}

	ptx, err := svc.Create(userID, baseCurrencyID, params)

	require.NoError(t, err)
	require.NotNil(t, ptx)
	assert.True(t, ptx.IsRecurring)
	assert.NotNil(t, ptx.RecurrenceRule)
	assert.True(t, ptx.IsIncome)

	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })
}

func TestCreate_BaseCurrencyNotSet(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-nocurr@example.com")
	params := defaultCreateParams()

	_, err := svc.Create(userID, 0, params)

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrBaseCurrencyNotSet)
}

// --- List ---

func TestList_ReturnsCreatedTransactions(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-list@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)
	params := defaultCreateParams()

	ptx, err := svc.Create(userID, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	result, err := svc.List(userID, planned_transactions.ListFilters{})

	require.NoError(t, err)
	found := false
	for _, p := range result {
		if p.ID == ptx.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created planned transaction should appear in the list")
}

func TestList_EmptyForNonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	result, err := svc.List(999999, planned_transactions.ListFilters{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- GetUpcoming ---

func TestGetUpcoming_ReturnsOccurrences(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-upcoming@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	params := planned_transactions.CreateParams{
		Amount:      decimal.NewFromFloat(100.00),
		Label:       "Upcoming Payment",
		IsIncome:    false,
		PlannedDate: time.Now().Add(5 * 24 * time.Hour),
		IsRecurring: false,
	}

	ptx, err := svc.Create(userID, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	result, err := svc.GetUpcoming(userID, 30, false)

	require.NoError(t, err)
	// Should include the planned transaction's occurrence within the 30-day window
	found := false
	for _, occ := range result {
		if occ.PlannedTransactionID == ptx.ID {
			found = true
			assert.Equal(t, "Upcoming Payment", occ.Label)
			assert.True(t, occ.Amount.Equal(decimal.NewFromFloat(100.00)))
			break
		}
	}
	assert.True(t, found, "planned transaction should generate an upcoming occurrence")
}

func TestGetUpcoming_EmptyForNonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	result, err := svc.GetUpcoming(999999, 30, false)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- GetByID ---

func TestGetByID_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-getbyid@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)
	params := defaultCreateParams()

	ptx, err := svc.Create(userID, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	result, err := svc.GetByID(userID, ptx.ID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ptx.ID, result.ID)
	assert.Equal(t, "Monthly Rent", result.Label)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-getnf@example.com")

	_, err := svc.GetByID(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrNotFound)
}

func TestGetByID_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "ptx-contract-get1@example.com")
	user2 := createTestUser(t, app, "ptx-contract-get2@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, user1)
	params := defaultCreateParams()

	ptx, err := svc.Create(user1, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	_, err = svc.GetByID(user2, ptx.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrAccessDenied)
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-update@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)
	params := defaultCreateParams()

	ptx, err := svc.Create(userID, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	updated, err := svc.Update(userID, ptx.ID, planned_transactions.UpdateParams{
		Amount:      decimal.NewFromFloat(75.00),
		Label:       "Updated Rent",
		IsIncome:    false,
		PlannedDate: ptx.PlannedDate,
		IsRecurring: false,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Rent", updated.Label)
	assert.True(t, updated.Amount.Equal(decimal.NewFromFloat(75.00)))
}

func TestUpdate_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-updnf@example.com")

	_, err := svc.Update(userID, 999999, planned_transactions.UpdateParams{
		Label:       "Nope",
		PlannedDate: time.Now(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrNotFound)
}

func TestUpdate_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "ptx-contract-upd1@example.com")
	user2 := createTestUser(t, app, "ptx-contract-upd2@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, user1)
	params := defaultCreateParams()

	ptx, err := svc.Create(user1, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	_, err = svc.Update(user2, ptx.ID, planned_transactions.UpdateParams{
		Label:       "Stolen",
		PlannedDate: time.Now(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrAccessDenied)
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-delete@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)
	params := defaultCreateParams()

	ptx, err := svc.Create(userID, baseCurrencyID, params)
	require.NoError(t, err)

	err = svc.Delete(userID, ptx.ID)

	require.NoError(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "ptx-contract-delnf@example.com")

	err := svc.Delete(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrNotFound)
}

func TestDelete_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "ptx-contract-del1@example.com")
	user2 := createTestUser(t, app, "ptx-contract-del2@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, user1)
	params := defaultCreateParams()

	ptx, err := svc.Create(user1, baseCurrencyID, params)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupPlannedTx(t, app, ptx.ID) })

	err = svc.Delete(user2, ptx.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, planned_transactions.ErrAccessDenied)
}
