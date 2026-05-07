// Contract tests for the financial_planning service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package financial_planning_test

import (
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/services/financial_planning"
	"github.com/go-budget/backend/internal/services/planned_transactions"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real financial_planning service connected to the test DB.
// It injects the real planned_transactions service and currency service.
func setupTestService(t *testing.T) (financial_planning.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()

	plannedTxSvc := planned_transactions.New(app.PlannedTxRepo, logger)
	currencySvc := currency.NewWithCurrencyRepo(app.ExchangeRateRepo, app.CurrencyRepo, logger)

	svc := financial_planning.New(
		plannedTxSvc,
		app.AccountRepo,
		app.CurrencyRepo,
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

// getBaseCurrencyID returns the user's base currency ID.
func getBaseCurrencyID(t *testing.T, app *testutil.TestApp, userID int) int {
	t.Helper()
	var currencyID int
	err := app.DB.QueryRow("SELECT base_currency_id FROM users WHERE id = $1", userID).Scan(&currencyID)
	require.NoError(t, err)
	return currencyID
}

// seedExchangeRates inserts exchange rates for a given date.
func seedExchangeRates(t *testing.T, app *testutil.TestApp, date string) {
	t.Helper()
	parsedDate, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)

	rate := &models.ExchangeRateHistory{
		Rates: map[string]float64{
			"USD": 1.0,
			"EUR": 0.92,
			"GBP": 0.79,
			"UAH": 41.5,
		},
		ActualDate:       parsedDate,
		BaseCurrencyCode: "USD",
		ServiceName:      "test-fp",
	}
	err = app.ExchangeRateRepo.SaveRate(rate)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = app.DB.Exec("DELETE FROM exchange_rates WHERE service_name = 'test-fp' AND actual_date = $1", date)
	})
}

// createTestAccountWithCurrency creates a test account with the given currency and returns
// the account ID.
func createTestAccountWithCurrency(t *testing.T, app *testutil.TestApp, userID int, name string, currencyCode string, balance float64) int {
	t.Helper()
	currencyID := testutil.GetCurrencyID(t, app.DB, currencyCode)
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, name, currencyID, accountTypeID, balance)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, accountID) })
	return accountID
}

// createPlannedTransaction creates a planned transaction via direct SQL insert.
func createPlannedTransaction(t *testing.T, app *testutil.TestApp, userID int, currencyCode string, amount float64, isIncome bool, plannedDate time.Time) int {
	t.Helper()
	currencyID := testutil.GetCurrencyID(t, app.DB, currencyCode)

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO planned_transactions (user_id, currency_id, amount, label, is_income, planned_date, is_recurring, is_active, is_deleted, is_executed)
		VALUES ($1, $2, $3, $4, $5, $6, false, true, false, false)
		RETURNING id
	`, userID, currencyID, amount, "Test Planned TX", isIncome, plannedDate).Scan(&id)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = app.DB.Exec("DELETE FROM planned_transactions WHERE id = $1", id)
	})
	return id
}

// --- CalculateFutureBalance ---

func TestCalculateFutureBalance_NoAccounts(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-noaccounts@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	// Seed exchange rates for today so the service does not fail on empty table
	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0.0, result.TotalCurrentBalance)
	assert.Equal(t, 0.0, result.TotalProjectedBalance)
	assert.Empty(t, result.Accounts)
}

func TestCalculateFutureBalance_WithAccountsAndRates(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-balance@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	// Seed exchange rates for today
	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	// Create a USD account with balance 1000
	createTestAccountWithCurrency(t, app, userID, "Checking USD", "USD", 1000.0)

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Accounts)
	// The account should appear in projections
	found := false
	for _, acct := range result.Accounts {
		if acct.AccountName == "Checking USD" {
			found = true
			assert.Equal(t, 1000.0, acct.CurrentBalance)
			break
		}
	}
	assert.True(t, found, "created account should appear in projections")
}

func TestCalculateFutureBalance_WithPlannedTransactions(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-planned@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	// Seed exchange rates for today and the planned date
	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	futureDate := time.Now().Add(10 * 24 * time.Hour)
	futureDateStr := futureDate.Format("2006-01-02")
	seedExchangeRates(t, app, futureDateStr)

	// Create an account
	createTestAccountWithCurrency(t, app, userID, "Main Account", "USD", 500.0)

	// Create a planned income transaction 10 days from now
	createPlannedTransaction(t, app, userID, "USD", 200.0, true, futureDate)

	// Create a planned expense transaction 10 days from now
	createPlannedTransaction(t, app, userID, "USD", 50.0, false, futureDate)

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	// Invariant: all amounts converted to base currency
	assert.NotEmpty(t, result.BaseCurrencyCode)

	// Should have planned income and expenses
	assert.Greater(t, result.TotalPlannedIncome, 0.0)
	assert.Greater(t, result.TotalPlannedExpenses, 0.0)
	assert.Greater(t, result.IncomeCount, 0)
	assert.Greater(t, result.ExpensesCount, 0)

	// Projected balance = current + income - expenses
	expectedProjected := result.TotalCurrentBalance + result.TotalPlannedIncome - result.TotalPlannedExpenses
	assert.InDelta(t, expectedProjected, result.TotalProjectedBalance, 0.01)
}

func TestCalculateFutureBalance_InvalidBaseCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-badcurr@example.com")

	_, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: 999999, // non-existent currency
	})

	require.Error(t, err)
}

func TestCalculateFutureBalance_OnlyShowInReportsAccounts(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-showreports@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	// Create an account with show_in_reports = true (default from helper)
	createTestAccountWithCurrency(t, app, userID, "Visible Account", "USD", 1000.0)

	// Create an account with show_in_reports = false
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	var hiddenID int
	err := app.DB.QueryRow(`
		INSERT INTO accounts (user_id, name, currency_id, account_type_id, initial_balance, balance, is_hidden, show_in_reports, is_deleted, is_archived, opening_date)
		VALUES ($1, 'Hidden Account', $2, $3, 2000, 2000, false, false, false, false, NOW())
		RETURNING id
	`, userID, currencyID, accountTypeID).Scan(&hiddenID)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestAccount(t, app.DB, hiddenID) })

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Invariant: Only accounts with ShowInReports=true are included
	for _, acct := range result.Accounts {
		assert.NotEqual(t, "Hidden Account", acct.AccountName,
			"account with show_in_reports=false should not be included")
	}
}

func TestCalculateFutureBalance_FilterByAccountIDs(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-filter@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	acct1 := createTestAccountWithCurrency(t, app, userID, "Account A", "USD", 500.0)
	createTestAccountWithCurrency(t, app, userID, "Account B", "USD", 300.0)

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
		AccountIDs:     []int{acct1},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Only Account A should be in the result
	assert.Len(t, result.Accounts, 1)
	assert.Equal(t, "Account A", result.Accounts[0].AccountName)
}

func TestCalculateFutureBalance_AccountProjectionsShowOwnCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-owncurr@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	createTestAccountWithCurrency(t, app, userID, "EUR Account", "EUR", 500.0)

	result, err := svc.CalculateFutureBalance(userID, financial_planning.FutureBalanceParams{
		TargetDate:     time.Now().Add(30 * 24 * time.Hour),
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Invariant: account projections show balances in account's own currency
	for _, acct := range result.Accounts {
		if acct.AccountName == "EUR Account" {
			assert.Equal(t, "EUR", acct.CurrencyCode)
			assert.Equal(t, 500.0, acct.CurrentBalance,
				"account projection should show balance in account's own currency, not converted")
		}
	}
}

// --- GetBalanceProjection ---

func TestGetBalanceProjection_DailyPeriod(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-daily@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	createTestAccountWithCurrency(t, app, userID, "Daily Account", "USD", 1000.0)

	startDate := time.Now()
	endDate := startDate.Add(7 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "daily", result.Period)
	assert.NotEmpty(t, result.BaseCurrencyCode)
	// Daily period for 7 days should produce 8 points (start day + 7 days)
	assert.GreaterOrEqual(t, len(result.ProjectionPoints), 7)
}

func TestGetBalanceProjection_WeeklyPeriod(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-weekly@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	createTestAccountWithCurrency(t, app, userID, "Weekly Account", "USD", 1000.0)

	startDate := time.Now()
	endDate := startDate.Add(30 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "weekly",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "weekly", result.Period)
	assert.NotEmpty(t, result.ProjectionPoints)
}

func TestGetBalanceProjection_MonthlyPeriod(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-monthly@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	createTestAccountWithCurrency(t, app, userID, "Monthly Account", "USD", 1000.0)

	startDate := time.Now()
	endDate := startDate.Add(90 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "monthly",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "monthly", result.Period)
	assert.NotEmpty(t, result.ProjectionPoints)
}

func TestGetBalanceProjection_DefaultPeriodIsDaily(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-defperiod@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	startDate := time.Now()
	endDate := startDate.Add(3 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "", // empty should default to daily
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "daily", result.Period)
}

func TestGetBalanceProjection_ErrEndDateBeforeStart(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-baddate@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	startDate := time.Now()
	endDate := startDate.Add(-7 * 24 * time.Hour) // end before start

	_, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: baseCurrencyID,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, financial_planning.ErrEndDateBeforeStart)
}

func TestGetBalanceProjection_InvalidBaseCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-badcurr2@example.com")

	startDate := time.Now()
	endDate := startDate.Add(7 * 24 * time.Hour)

	_, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: 999999, // non-existent currency
	})

	require.Error(t, err)
}

func TestGetBalanceProjection_WithPlannedTransactions(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-projplan@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	futureDate := time.Now().Add(3 * 24 * time.Hour)
	futureDateStr := futureDate.Format("2006-01-02")
	seedExchangeRates(t, app, futureDateStr)

	createTestAccountWithCurrency(t, app, userID, "Proj Account", "USD", 1000.0)
	createPlannedTransaction(t, app, userID, "USD", 300.0, true, futureDate)

	startDate := time.Now()
	endDate := startDate.Add(7 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.ProjectionPoints)

	// The running balance should increase after the planned income date
	hasIncome := false
	for _, point := range result.ProjectionPoints {
		if point.Income > 0 {
			hasIncome = true
			break
		}
	}
	assert.True(t, hasIncome, "projection should include planned income")
}

func TestGetBalanceProjection_NoAccounts(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-noacct-proj@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	startDate := time.Now()
	endDate := startDate.Add(7 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	// All projection points should have zero balance
	for _, point := range result.ProjectionPoints {
		assert.Equal(t, 0.0, point.Balance)
	}
}

func TestGetBalanceProjection_AllAmountsInBaseCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "fp-contract-basecurr-proj@example.com")
	baseCurrencyID := getBaseCurrencyID(t, app, userID)

	today := time.Now().Format("2006-01-02")
	seedExchangeRates(t, app, today)

	// Create EUR account — balances should be converted to USD (base) in the projection
	createTestAccountWithCurrency(t, app, userID, "EUR Proj Account", "EUR", 100.0)

	startDate := time.Now()
	endDate := startDate.Add(3 * 24 * time.Hour)

	result, err := svc.GetBalanceProjection(userID, financial_planning.BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "daily",
		BaseCurrencyID: baseCurrencyID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Invariant: all amounts are converted to the user's base currency
	// The base currency code should be in the result
	assert.NotEmpty(t, result.BaseCurrencyCode)

	// EUR 100 converted to USD (rate: EUR=0.92, USD=1.0) should be ~108.70
	// The first projection point balance should reflect converted amounts
	require.NotEmpty(t, result.ProjectionPoints)
	firstPoint := result.ProjectionPoints[0]
	assert.Greater(t, firstPoint.Balance, 100.0,
		"EUR 100 converted to USD should be > 100 (EUR is worth more than USD)")
}
