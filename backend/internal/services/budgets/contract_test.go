// Contract tests for the budgets service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package budgets_test

import (
	"testing"
	"time"

	currencyservice "github.com/go-budget/backend/internal/services/currency"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/services/budgets"
	"github.com/go-budget/backend/internal/testutil"
)

// setupTestService creates a real budgets service connected to the test DB.
func setupTestService(t *testing.T) (budgets.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	currencySvc := currencyservice.NewWithCurrencyRepo(app.ExchangeRateRepo, app.CurrencyRepo, logger)
	svc := budgets.New(app.BudgetRepo, app.BudgetRepo, app.CurrencyRepo, currencySvc, logger)
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

// defaultCreateParams returns valid CreateParams for testing.
func defaultCreateParams(t *testing.T, app *testutil.TestApp) budgets.CreateParams {
	t.Helper()
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	now := time.Now()
	start := now.Format("2006-01-02")
	end := now.Add(30 * 24 * time.Hour).Format("2006-01-02")
	return budgets.CreateParams{
		Name:         "Test Budget",
		CurrencyID:   usdID,
		TargetAmount: decimal.NewFromFloat(1000.00),
		Period:       "MONTHLY",
		Repeat:       false,
		StartDate:    start,
		EndDate:      end,
		Categories:   []int{},
	}
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-create@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(userID, params)

	require.NoError(t, err)
	require.NotNil(t, budget)
	assert.Equal(t, "Test Budget", budget.Name)
	assert.Equal(t, userID, budget.UserID)
	assert.True(t, budget.TargetAmount.Equal(decimal.NewFromFloat(1000.00)))
	assert.Equal(t, "MONTHLY", budget.Period)
	assert.NotZero(t, budget.ID)

	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })
}

func TestCreate_InvalidCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-badcurr@example.com")
	params := defaultCreateParams(t, app)
	params.CurrencyID = 999999

	_, err := svc.Create(userID, params)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrInvalidCurrency)
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-badstart@example.com")
	params := defaultCreateParams(t, app)
	params.StartDate = "not-a-date"

	_, err := svc.Create(userID, params)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrInvalidStartDate)
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-badend@example.com")
	params := defaultCreateParams(t, app)
	params.EndDate = "not-a-date"

	_, err := svc.Create(userID, params)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrInvalidEndDate)
}

func TestCreate_PeriodNormalizedToUppercase(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-period@example.com")
	params := defaultCreateParams(t, app)
	params.Period = "monthly"

	budget, err := svc.Create(userID, params)

	require.NoError(t, err)
	assert.Equal(t, "MONTHLY", budget.Period)

	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })
}

// --- GetByUserID ---

func TestGetByUserID_ReturnsCreatedBudgets(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-list@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(userID, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	result, err := svc.GetByUserID(userID, "active")

	require.NoError(t, err)
	found := false
	for _, b := range result {
		if b.ID == budget.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created budget should appear in the list")
}

func TestGetByUserID_EmptyForNonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	result, err := svc.GetByUserID(999999, "active")

	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-update@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(userID, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	updated, err := svc.Update(userID, budget.ID, budgets.UpdateParams{
		Name:         "Updated Budget",
		CurrencyID:   params.CurrencyID,
		TargetAmount: decimal.NewFromFloat(2000.00),
		Period:       "WEEKLY",
		Repeat:       true,
		StartDate:    params.StartDate,
		EndDate:      params.EndDate,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Budget", updated.Name)
	assert.True(t, updated.TargetAmount.Equal(decimal.NewFromFloat(2000.00)))
}

func TestUpdate_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-updnf@example.com")

	_, err := svc.Update(userID, 999999, budgets.UpdateParams{
		Name:       "Nope",
		CurrencyID: testutil.GetCurrencyID(t, app.DB, "USD"),
		StartDate:  time.Now().Format("2006-01-02"),
		EndDate:    time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrNotFound)
}

func TestUpdate_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "budgets-contract-upd1@example.com")
	user2 := createTestUser(t, app, "budgets-contract-upd2@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(user1, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	_, err = svc.Update(user2, budget.ID, budgets.UpdateParams{
		Name:       "Stolen",
		CurrencyID: params.CurrencyID,
		StartDate:  params.StartDate,
		EndDate:    params.EndDate,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrAccessDenied)
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-delete@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(userID, params)
	require.NoError(t, err)

	err = svc.Delete(userID, budget.ID)

	require.NoError(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-delnf@example.com")

	err := svc.Delete(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrNotFound)
}

func TestDelete_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "budgets-contract-del1@example.com")
	user2 := createTestUser(t, app, "budgets-contract-del2@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(user1, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	err = svc.Delete(user2, budget.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrAccessDenied)
}

// --- Archive ---

func TestArchive_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-archive@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(userID, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	err = svc.Archive(userID, budget.ID)

	require.NoError(t, err)
}

func TestArchive_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-arcnf@example.com")

	err := svc.Archive(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrNotFound)
}

func TestArchive_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "budgets-contract-arc1@example.com")
	user2 := createTestUser(t, app, "budgets-contract-arc2@example.com")
	params := defaultCreateParams(t, app)

	budget, err := svc.Create(user1, params)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budget.ID) })

	err = svc.Archive(user2, budget.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, budgets.ErrAccessDenied)
}

// --- DailyProcessing ---

func TestDailyProcessing_RunsWithoutError(t *testing.T) {
	svc, _ := setupTestService(t)

	result, err := svc.DailyProcessing()

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Processed, 0)
}

// --- RecalculateCollectedAmounts ---

func TestRecalculateCollectedAmounts_RunsWithoutError(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "budgets-contract-recalc@example.com")

	err := svc.RecalculateCollectedAmounts(userID)

	require.NoError(t, err)
}
