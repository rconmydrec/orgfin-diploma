package reports

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mock repositories ====================

type mockTransactionRepo struct {
	getByUserIDFunc func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error)
}

func (m *mockTransactionRepo) Create(tx *models.Transaction) (*models.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepo) GetByID(id int) (*models.Transaction, error) { return nil, nil }
func (m *mockTransactionRepo) GetByAccountID(accountID int, limit, offset int) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepo) GetByAccountIDForRecalc(accountID int) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepo) GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID, filters)
	}
	return []*models.Transaction{}, 0, nil
}
func (m *mockTransactionRepo) GetForExport(userID int, startDate, endDate string) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionRepo) Update(tx *models.Transaction) error                    { return nil }
func (m *mockTransactionRepo) UpdateLinkedID(id int, linkedID int) error              { return nil }
func (m *mockTransactionRepo) UpdateBalance(id int, newBalance decimal.Decimal) error { return nil }
func (m *mockTransactionRepo) Delete(id int) error                                    { return nil }

type mockAccountRepo struct {
	getByUserIDFunc func(userID int, filters types.AccountFilters) ([]*models.Account, error)
}

func (m *mockAccountRepo) Create(account *models.Account) (*models.Account, error) {
	return nil, nil
}
func (m *mockAccountRepo) GetByID(id int) (*models.Account, error) { return nil, nil }
func (m *mockAccountRepo) GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID, filters)
	}
	return []*models.Account{}, nil
}
func (m *mockAccountRepo) Update(account *models.Account) error                { return nil }
func (m *mockAccountRepo) SoftDelete(id int) error                             { return nil }
func (m *mockAccountRepo) UpdateBalance(id int, balance decimal.Decimal) error { return nil }
func (m *mockAccountRepo) SetArchiveStatus(id int, isArchived bool) error      { return nil }
func (m *mockAccountRepo) CountActiveByUserID(_ int) (int, error)              { return 0, nil }

type mockCategoryRepo struct {
	getByUserIDFunc func(userID int) ([]*models.UserCategory, error)
}

func (m *mockCategoryRepo) GetByUserID(userID int) ([]*models.UserCategory, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID)
	}
	return []*models.UserCategory{}, nil
}
func (m *mockCategoryRepo) GetByUserIDGrouped(userID int) ([]*models.UserCategory, error) {
	return nil, nil
}
func (m *mockCategoryRepo) GetByID(id int) (*models.UserCategory, error) { return nil, nil }
func (m *mockCategoryRepo) Create(category *models.UserCategory) (*models.UserCategory, error) {
	return nil, nil
}
func (m *mockCategoryRepo) Update(category *models.UserCategory) error { return nil }
func (m *mockCategoryRepo) Delete(id int) error                        { return nil }
func (m *mockCategoryRepo) CopyDefaultCategories(userID int) error     { return nil }

type mockCurrencyRepo struct {
	getByIDFunc func(id int) (*models.Currency, error)
}

func (m *mockCurrencyRepo) GetAll() ([]*models.Currency, error)             { return nil, nil }
func (m *mockCurrencyRepo) GetByCode(code string) (*models.Currency, error) { return nil, nil }
func (m *mockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
}

type mockExchangeRateRepo struct {
	getAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
}

func (m *mockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *mockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *mockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error { return nil }
func (m *mockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	if m.getAllRatesForDateFunc != nil {
		return m.getAllRatesForDateFunc(date)
	}
	return &types.ExchangeRateSnapshot{
		Rates:            map[string]float64{"USD": 1.0},
		BaseCurrencyCode: "USD",
	}, nil
}

// mockCurrencyService implements CurrencyService using a mockExchangeRateRepo.
// This avoids importing services/currency while preserving identical conversion logic.
type mockCurrencyService struct {
	exchangeRateRepo *mockExchangeRateRepo
	logger           *slog.Logger
}

func (m *mockCurrencyService) NewRateCache() *currency.RateCache {
	return currency.NewRateCache(m.exchangeRateRepo)
}

func (m *mockCurrencyService) ConvertToBaseCurrency(
	amount decimal.Decimal,
	sourceCurrencyCode, baseCurrencyCode string,
	dateStr string,
	rc *currency.RateCache,
) (decimal.Decimal, error) {
	if sourceCurrencyCode == baseCurrencyCode {
		return amount, nil
	}

	rates, err := rc.GetRatesForDate(dateStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, currency.ErrNoExchangeRates
		}
		return decimal.Zero, err
	}

	if rates == nil || len(rates.Rates) == 0 {
		return decimal.Zero, currency.ErrNoExchangeRates
	}

	sourceRate, sourceOK := rates.Rates[sourceCurrencyCode]
	targetRate, targetOK := rates.Rates[baseCurrencyCode]

	if !sourceOK || !targetOK || sourceRate == 0 || targetRate == 0 {
		return decimal.Zero, errors.New("currency rate missing")
	}

	sourceRateDec := decimal.NewFromFloat(sourceRate)
	targetRateDec := decimal.NewFromFloat(targetRate)
	return amount.Div(sourceRateDec).Mul(targetRateDec), nil
}

// ==================== Test helpers ====================

func newTestService() (*Service, *mockTransactionRepo, *mockAccountRepo, *mockCategoryRepo, *mockCurrencyRepo, *mockExchangeRateRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	txRepo := &mockTransactionRepo{}
	accountRepo := &mockAccountRepo{}
	categoryRepo := &mockCategoryRepo{}
	currencyRepo := &mockCurrencyRepo{}
	exchangeRateRepo := &mockExchangeRateRepo{}

	currencySvc := &mockCurrencyService{exchangeRateRepo: exchangeRateRepo, logger: logger}
	svc := New(txRepo, accountRepo, categoryRepo, currencyRepo, currencySvc, logger)

	return svc, txRepo, accountRepo, categoryRepo, currencyRepo, exchangeRateRepo
}

// ==================== getTransactionAmount Tests ====================

func TestGetTransactionAmount_NilAccount(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	tx := &models.Transaction{
		ID:     1,
		Amount: decimal.NewFromFloat(100),
	}

	rc := svc.currencySvc.NewRateCache()
	result, err := svc.getTransactionAmount(tx, "USD", "2024-01-01", rc)
	assert.NoError(t, err)
	assert.True(t, result.Equal(decimal.NewFromFloat(100)))
}

func TestGetTransactionAmount_NilCurrency(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	tx := &models.Transaction{
		ID:      1,
		Amount:  decimal.NewFromFloat(100),
		Account: &models.Account{ID: 1},
	}

	rc := svc.currencySvc.NewRateCache()
	result, err := svc.getTransactionAmount(tx, "USD", "2024-01-01", rc)
	assert.NoError(t, err)
	assert.True(t, result.Equal(decimal.NewFromFloat(100)))
}

func TestGetTransactionAmount_EmptyCurrencyCode(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	tx := &models.Transaction{
		ID:     1,
		Amount: decimal.NewFromFloat(100),
		Account: &models.Account{
			ID:       1,
			Currency: &models.Currency{Code: ""},
		},
	}

	rc := svc.currencySvc.NewRateCache()
	result, err := svc.getTransactionAmount(tx, "USD", "2024-01-01", rc)
	assert.NoError(t, err)
	assert.True(t, result.Equal(decimal.NewFromFloat(100)))
}

func TestGetTransactionAmount_SameCurrency(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	tx := &models.Transaction{
		ID:     1,
		Amount: decimal.NewFromFloat(100),
		Account: &models.Account{
			ID:       1,
			Currency: &models.Currency{Code: "USD"},
		},
	}

	rc := svc.currencySvc.NewRateCache()
	result, err := svc.getTransactionAmount(tx, "USD", "2024-01-01", rc)
	assert.NoError(t, err)
	assert.True(t, result.Equal(decimal.NewFromFloat(100)))
}

func TestGetTransactionAmount_DifferentCurrency(t *testing.T) {
	svc, _, _, _, _, exchangeRateRepo := newTestService()

	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	tx := &models.Transaction{
		ID:     1,
		Amount: decimal.NewFromFloat(85),
		Account: &models.Account{
			ID:       1,
			Currency: &models.Currency{Code: "EUR"},
		},
	}

	rc := svc.currencySvc.NewRateCache()
	result, err := svc.getTransactionAmount(tx, "USD", "2024-01-01", rc)
	assert.NoError(t, err)
	// 85 / 0.85 * 1.0 = 100
	f, _ := result.Float64()
	assert.InDelta(t, 100.0, f, 0.01)
}

// ==================== CashFlowReport Tests ====================

func TestCashFlowReport_DefaultDatesMonthly(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	var capturedFilters types.TransactionFilters
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify DateFrom was set to 12 months ago (1st of month)
	assert.NotNil(t, capturedFilters.DateFrom)
	now := time.Now()
	expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -12, 0)
	assert.Equal(t, expected.Format("2006-01-02"), *capturedFilters.DateFrom)

	// Verify DateTo was set to tomorrow
	assert.NotNil(t, capturedFilters.DateTo)
	expectedEnd := now.AddDate(0, 0, 1).Format("2006-01-02")
	assert.Equal(t, expectedEnd, *capturedFilters.DateTo)
	assert.True(t, capturedFilters.NoLimit)
}

func TestCashFlowReport_DefaultDatesDaily(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	var capturedFilters types.TransactionFilters
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	params := CashFlowParams{
		Period:         "daily",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify DateFrom was set to 30 days ago
	assert.NotNil(t, capturedFilters.DateFrom)
	expected := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	assert.Equal(t, expected, *capturedFilters.DateFrom)
}

func TestCashFlowReport_ProvidedDates(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	var capturedFilters types.TransactionFilters
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	params := CashFlowParams{
		StartDate:      &start,
		EndDate:        &end,
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "2024-01-01", *capturedFilters.DateFrom)
	// EndDate inclusive: 2024-01-31 + 1 day = 2024-02-01
	assert.Equal(t, "2024-02-01", *capturedFilters.DateTo)
}

func TestCashFlowReport_IncomeAndExpenseAggregation(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	now := time.Now()
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, DateTime: &now},
			{ID: 2, Amount: decimal.NewFromInt(50), IsIncome: false, DateTime: &now},
			{ID: 3, Amount: decimal.NewFromInt(200), IsIncome: true, DateTime: &now},
		}, 3, nil
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	periodKey := now.Format("2006-01")
	assert.InDelta(t, 300.0, result.TotalIncome[periodKey], 0.01)
	assert.InDelta(t, 50.0, result.TotalExpenses[periodKey], 0.01)
	assert.InDelta(t, 250.0, result.NetFlow[periodKey], 0.01)
}

func TestCashFlowReport_ExcludedAndTransferTransactions(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	now := time.Now()
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, DateTime: &now, ExcludeFromReports: true},
			{ID: 2, Amount: decimal.NewFromInt(50), IsIncome: false, DateTime: &now, IsTransfer: true},
			{ID: 3, Amount: decimal.NewFromInt(200), IsIncome: false, DateTime: &now},
		}, 3, nil
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)

	// Only transaction ID=3 should be counted
	periodKey := now.Format("2006-01")
	assert.InDelta(t, 0.0, result.TotalIncome[periodKey], 0.01)
	assert.InDelta(t, 200.0, result.TotalExpenses[periodKey], 0.01)
}

func TestCashFlowReport_DailyPeriodKey(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	now := time.Now()
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, DateTime: &now},
		}, 1, nil
	}

	params := CashFlowParams{
		Period:         "daily",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)

	dailyKey := now.Format("2006-01-02")
	assert.InDelta(t, 100.0, result.TotalExpenses[dailyKey], 0.01)
}

func TestCashFlowReport_FatalCurrencyError(t *testing.T) {
	svc, txRepo, _, _, _, exchangeRateRepo := newTestService()

	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromFloat(100), IsIncome: false, DateTime: &now, Account: eurAccount},
		}, 1, nil
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, currency.ErrNoExchangeRates))
}

func TestCashFlowReport_NonFatalCurrencyError(t *testing.T) {
	svc, txRepo, _, _, _, exchangeRateRepo := newTestService()

	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, errors.New("some transient error")
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromFloat(100), IsIncome: false, DateTime: &now, Account: eurAccount},
		}, 1, nil
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Transaction skipped, so maps should be empty
	assert.Empty(t, result.TotalExpenses)
}

func TestCashFlowReport_BaseCurrencyError(t *testing.T) {
	svc, _, _, _, currencyRepo, _ := newTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestCashFlowReport_TransactionFetchError(t *testing.T) {
	svc, txRepo, _, _, _, _ := newTestService()

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	params := CashFlowParams{
		Period:         "monthly",
		BaseCurrencyID: 1,
	}

	result, err := svc.CashFlowReport(1, params)
	assert.Nil(t, result)
	assert.Error(t, err)
}

// ==================== BalanceReport Tests ====================

func TestBalanceReport_AllAccounts(t *testing.T) {
	svc, _, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Bank", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	params := BalanceParams{
		BalanceDate:    "2024-01-01",
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestBalanceReport_FilterByIDs(t *testing.T) {
	svc, _, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Bank", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    "2024-01-01",
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, 1, items[0].AccountID)
}

func TestBalanceReport_ExcludeHidden(t *testing.T) {
	svc, _, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Visible", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1000), IsHidden: false},
			{ID: 2, Name: "Hidden", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(2000), IsHidden: true},
		}, nil
	}

	params := BalanceParams{
		BalanceDate:    "2024-01-01",
		ExcludeHidden:  true,
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Visible", items[0].AccountName)
}

func TestBalanceReport_HistoricalBalance_ExpenseReversal(t *testing.T) {
	svc, txRepo, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(800)},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, AccountID: 1, Amount: decimal.NewFromInt(200), IsIncome: false, DateTime: &yesterday},
		}, 1, nil
	}

	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    twoDaysAgo,
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// 800 + 200 (reverse expense) = 1000
	assert.InDelta(t, 1000.0, items[0].Balance, 0.01)
}

func TestBalanceReport_HistoricalBalance_IncomeReversal(t *testing.T) {
	svc, txRepo, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1500)},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, AccountID: 1, Amount: decimal.NewFromInt(500), IsIncome: true, DateTime: &yesterday},
		}, 1, nil
	}

	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    twoDaysAgo,
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// 1500 - 500 (reverse income) = 1000
	assert.InDelta(t, 1000.0, items[0].Balance, 0.01)
}

func TestBalanceReport_HistoricalBalance_TransferReversal(t *testing.T) {
	svc, txRepo, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, AccountID: 1, Amount: decimal.NewFromInt(300), IsIncome: false, IsTransfer: true, DateTime: &yesterday},
			{ID: 2, AccountID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, IsTransfer: true, DateTime: &yesterday},
		}, 2, nil
	}

	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    twoDaysAgo,
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// 500 + 300 (reverse outgoing) - 100 (reverse incoming) = 700
	assert.InDelta(t, 700.0, items[0].Balance, 0.01)
}

func TestBalanceReport_CurrencyConversion(t *testing.T) {
	svc, txRepo, accountRepo, _, currencyRepo, exchangeRateRepo := newTestService()

	callCount := 0
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount <= 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
	}

	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, AccountTypeID: 1, Balance: decimal.NewFromFloat(850)},
		}, nil
	}

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{}, 0, nil
	}

	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    time.Now().Format("2006-01-02"),
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// 850 / 0.85 * 1.0 = 1000 USD
	assert.InDelta(t, 1000.0, items[0].BaseCurrencyBalance, 0.01)
}

func TestBalanceReport_ConversionFallback(t *testing.T) {
	svc, _, accountRepo, _, currencyRepo, exchangeRateRepo := newTestService()

	callCount := 0
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount <= 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return &models.Currency{ID: 3, Code: "GBP", Name: "British Pound"}, nil
	}

	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		// GBP is missing from rates
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "GBP Account", CurrencyID: 3, AccountTypeID: 1, Balance: decimal.NewFromFloat(500)},
		}, nil
	}

	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    "2024-01-01",
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// Falls back to raw balance
	assert.InDelta(t, 500.0, items[0].BaseCurrencyBalance, 0.01)
}

func TestBalanceReport_TransactionFetchError(t *testing.T) {
	svc, txRepo, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	params := BalanceParams{
		AccountIDs:     []int{1},
		BalanceDate:    time.Now().Format("2006-01-02"),
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 1)
	// Uses current balance (graceful degradation)
	assert.InDelta(t, 1000.0, items[0].Balance, 0.01)
}

func TestBalanceReport_BaseCurrencyError(t *testing.T) {
	svc, _, _, _, currencyRepo, _ := newTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	params := BalanceParams{
		BalanceDate:    "2024-01-01",
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.Nil(t, items)
	assert.Error(t, err)
}

func TestBalanceReport_AccountFetchError(t *testing.T) {
	svc, _, accountRepo, _, _, _ := newTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	params := BalanceParams{
		BalanceDate:    "2024-01-01",
		BaseCurrencyID: 1,
	}

	items, err := svc.BalanceReport(1, params)
	assert.Nil(t, items)
	assert.Error(t, err)
}

// ==================== ExpensesByCategories Tests ====================

func TestExpensesByCategories_ParentAndChildCategories(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	parentID := 1
	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Groceries", UserID: 1, ParentID: &parentID},
		}, nil
	}

	catID := 2
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID},
		}, 1, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	items, err := svc.ExpensesByCategories(1, params)
	assert.NoError(t, err)
	assert.Len(t, items, 2) // parent + child

	// Verify child has display name with >>
	var childItem *ExpensesCategoryItem
	for i := range items {
		if !items[i].IsParent {
			childItem = &items[i]
		}
	}
	require.NotNil(t, childItem)
	assert.Equal(t, "Food >> Groceries", childItem.Name)
}

func TestExpensesByCategories_CategoryFilter(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	parentID := 1
	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Groceries", UserID: 1, ParentID: &parentID},
		}, nil
	}

	catID := 1
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID},
		}, 1, nil
	}

	params := ExpensesParams{
		StartDate:           "2024-01-01",
		EndDate:             "2024-12-31",
		Categories:          []int{1},
		HideEmptyCategories: true,
		BaseCurrencyID:      1,
	}

	items, err := svc.ExpensesByCategories(1, params)
	assert.NoError(t, err)

	// Transport (ID=2) should NOT appear
	for _, item := range items {
		assert.NotEqual(t, "Transport", item.Name)
	}
}

func TestExpensesByCategories_IncomeCategoriesSkipped(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Salary", UserID: 1, IsIncome: true},
		}, nil
	}

	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{}, 0, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	items, err := svc.ExpensesByCategories(1, params)
	assert.NoError(t, err)

	// Salary should not be present
	for _, item := range items {
		assert.NotEqual(t, "Salary", item.Name)
	}
}

func TestExpensesByCategories_HideEmptyCategories(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID},
		}, 1, nil
	}

	params := ExpensesParams{
		StartDate:           "2024-01-01",
		EndDate:             "2024-12-31",
		HideEmptyCategories: true,
		BaseCurrencyID:      1,
	}

	items, err := svc.ExpensesByCategories(1, params)
	assert.NoError(t, err)

	// Transport has no expenses and should be hidden
	for _, item := range items {
		assert.NotEqual(t, "Transport", item.Name)
	}
}

func TestExpensesByCategories_SortOrder(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	parentAID := 1
	parentBID := 3
	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 3, Name: "Bills", UserID: 1},
			{ID: 1, Name: "Automobile", UserID: 1},
			{ID: 5, Name: "Internet", UserID: 1, ParentID: &parentBID},
			{ID: 2, Name: "Fuel", UserID: 1, ParentID: &parentAID},
			{ID: 4, Name: "Electricity", UserID: 1, ParentID: &parentBID},
			{ID: 6, Name: "Car Wash", UserID: 1, ParentID: &parentAID},
		}, nil
	}

	catFuel := 2
	catElectricity := 4
	catInternet := 5
	catCarWash := 6
	catAutomobile := 1
	catBills := 3
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, CategoryID: &catFuel},
			{ID: 2, Amount: decimal.NewFromInt(80), IsIncome: false, CategoryID: &catElectricity},
			{ID: 3, Amount: decimal.NewFromInt(40), IsIncome: false, CategoryID: &catInternet},
			{ID: 4, Amount: decimal.NewFromInt(15), IsIncome: false, CategoryID: &catCarWash},
			{ID: 5, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catAutomobile},
			{ID: 6, Amount: decimal.NewFromInt(60), IsIncome: false, CategoryID: &catBills},
		}, 6, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	items, err := svc.ExpensesByCategories(1, params)
	assert.NoError(t, err)
	require.Len(t, items, 6)

	// Expected order:
	// 1. Automobile (parent A)
	// 2. Automobile >> Car Wash
	// 3. Automobile >> Fuel
	// 4. Bills (parent B)
	// 5. Bills >> Electricity
	// 6. Bills >> Internet
	assert.Equal(t, "Automobile", items[0].Name)
	assert.True(t, items[0].IsParent)
	assert.Equal(t, "Automobile >> Car Wash", items[1].Name)
	assert.Equal(t, "Automobile >> Fuel", items[2].Name)
	assert.Equal(t, "Bills", items[3].Name)
	assert.True(t, items[3].IsParent)
	assert.Equal(t, "Bills >> Electricity", items[4].Name)
	assert.Equal(t, "Bills >> Internet", items[5].Name)
}

// ==================== DiagramData Tests ====================

func TestDiagramData_ParentAggregation(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	parentID1 := 1
	parentID2 := 3
	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Groceries", UserID: 1, ParentID: &parentID1},
			{ID: 3, Name: "Transport", UserID: 1},
			{ID: 4, Name: "Fuel", UserID: 1, ParentID: &parentID2},
		}, nil
	}

	catGroceries := 2
	catFuel := 4
	catFood := 1
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, CategoryID: &catGroceries},
			{ID: 2, Amount: decimal.NewFromInt(30), IsIncome: false, CategoryID: &catFuel},
			{ID: 3, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
		}, 3, nil
	}

	params := DiagramParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		DiagramType:    "pie",
		BaseCurrencyID: 1,
	}

	result, err := svc.DiagramData(1, params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should have 2 labels: Food and Transport
	assert.Len(t, result.Labels, 2)

	labelAmounts := make(map[string]float64)
	for i, label := range result.Labels {
		labelAmounts[label] = result.Data[i]
	}

	// Food = 100 (direct) + 50 (Groceries) = 150
	assert.InDelta(t, 150.0, labelAmounts["Food"], 0.01)
	// Transport = 30 (Fuel)
	assert.InDelta(t, 30.0, labelAmounts["Transport"], 0.01)
}

func TestDiagramData_ZeroTotalsSkipped(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.Zero, IsIncome: false, CategoryID: &catID},
		}, 1, nil
	}

	params := DiagramParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		DiagramType:    "pie",
		BaseCurrencyID: 1,
	}

	result, err := svc.DiagramData(1, params)
	assert.NoError(t, err)
	assert.Empty(t, result.Labels)
	assert.Empty(t, result.Data)
}

func TestDiagramData_OtherThreshold(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Tiny", UserID: 1},
		}, nil
	}

	catFood := 1
	catTransport := 2
	catTiny := 3
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(600), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(399), IsIncome: false, CategoryID: &catTransport},
			{ID: 3, Amount: decimal.NewFromInt(1), IsIncome: false, CategoryID: &catTiny}, // 0.1%
		}, 3, nil
	}

	params := DiagramParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		DiagramType:    "pie",
		BaseCurrencyID: 1,
	}

	result, err := svc.DiagramData(1, params)
	assert.NoError(t, err)

	assert.NotContains(t, result.Labels, "Tiny")
	assert.Contains(t, result.Labels, "Other")
	assert.Contains(t, result.Labels, "Food")
	assert.Contains(t, result.Labels, "Transport")
}

func TestDiagramData_SortByAmountDescending(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Entertainment", UserID: 1},
		}, nil
	}

	catFood := 1
	catTransport := 2
	catEntertainment := 3
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: &catTransport},
			{ID: 3, Amount: decimal.NewFromInt(200), IsIncome: false, CategoryID: &catEntertainment},
		}, 3, nil
	}

	params := DiagramParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		DiagramType:    "pie",
		BaseCurrencyID: 1,
	}

	result, err := svc.DiagramData(1, params)
	assert.NoError(t, err)

	require.Len(t, result.Data, 3)
	assert.InDelta(t, 300.0, result.Data[0], 0.01)
	assert.InDelta(t, 200.0, result.Data[1], 0.01)
	assert.InDelta(t, 100.0, result.Data[2], 0.01)

	assert.Equal(t, "Transport", result.Labels[0])
	assert.Equal(t, "Entertainment", result.Labels[1])
	assert.Equal(t, "Food", result.Labels[2])
}

// ==================== ExpensesData Tests ====================

func TestExpensesData_CategoryFilter(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
		}, nil
	}

	catID1 := 1
	catID2 := 2
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromFloat(100), IsIncome: false, CategoryID: &catID1},
			{ID: 2, Amount: decimal.NewFromFloat(50), IsIncome: false, CategoryID: &catID2},
		}, 2, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		Categories:     []int{1},
		BaseCurrencyID: 1,
	}

	entries, err := svc.ExpensesData(1, params)
	assert.NoError(t, err)

	// Only Food should appear
	for _, entry := range entries {
		assert.NotEqual(t, "Transport", entry.CategoryName)
	}
	require.Len(t, entries, 1)
	assert.Equal(t, "Food", entries[0].CategoryName)
}

func TestExpensesData_ParentAggregation(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	parentID := 1
	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Groceries", UserID: 1, ParentID: &parentID},
		}, nil
	}

	catFood := 1
	catGroceries := 2
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(50), IsIncome: false, CategoryID: &catGroceries},
		}, 2, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	entries, err := svc.ExpensesData(1, params)
	assert.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "Food", entries[0].CategoryName)
	assert.Equal(t, 1, entries[0].CategoryID)
	// Food = 100 + 50 = 150
	assert.InDelta(t, 150.0, entries[0].Amount, 0.01)
}

func TestExpensesData_OtherThreshold(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Tiny", UserID: 1},
		}, nil
	}

	catFood := 1
	catTiny := 2
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(1000), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(1), IsIncome: false, CategoryID: &catTiny},
		}, 2, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	entries, err := svc.ExpensesData(1, params)
	assert.NoError(t, err)

	// Tiny should be combined into Other
	var hasOther bool
	for _, entry := range entries {
		if entry.CategoryName == "Other" {
			hasOther = true
		}
		assert.NotEqual(t, "Tiny", entry.CategoryName)
	}
	assert.True(t, hasOther)
}

func TestExpensesData_SortByAmountDescending(t *testing.T) {
	svc, txRepo, _, categoryRepo, _, _ := newTestService()

	categoryRepo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Entertainment", UserID: 1},
		}, nil
	}

	catFood := 1
	catTransport := 2
	catEntertainment := 3
	txRepo.getByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: &catTransport},
			{ID: 3, Amount: decimal.NewFromInt(200), IsIncome: false, CategoryID: &catEntertainment},
		}, 3, nil
	}

	params := ExpensesParams{
		StartDate:      "2024-01-01",
		EndDate:        "2024-12-31",
		BaseCurrencyID: 1,
	}

	entries, err := svc.ExpensesData(1, params)
	assert.NoError(t, err)

	require.Len(t, entries, 3)
	assert.InDelta(t, 300.0, entries[0].Amount, 0.01)
	assert.InDelta(t, 200.0, entries[1].Amount, 0.01)
	assert.InDelta(t, 100.0, entries[2].Amount, 0.01)
}

// ==================== aggregateChildrenIntoParents Tests ====================

func TestAggregateChildrenIntoParents(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	parentID := 1
	categoryMap := map[int]*catInfo{
		1: {ID: 1, Name: "Food"},
		2: {ID: 2, Name: "Groceries", ParentID: &parentID},
		3: {ID: 3, Name: "Transport"},
	}

	categoryExpenses := map[int]decimal.Decimal{
		1: decimal.NewFromInt(100),
		2: decimal.NewFromInt(50),
		3: decimal.NewFromInt(30),
	}

	result := svc.aggregateChildrenIntoParents(categoryExpenses, categoryMap)

	// Food = 100 (direct) + 50 (Groceries) = 150
	assert.True(t, result[1].Equal(decimal.NewFromInt(150)))
	// Transport = 30
	assert.True(t, result[3].Equal(decimal.NewFromInt(30)))
	// Category 2 should not exist as a separate entry
	_, hasCat2 := result[2]
	assert.False(t, hasCat2)
}

// ==================== applyOtherThreshold Tests ====================

func TestApplyOtherThreshold_BelowThreshold(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	entries := []expenseEntry{
		{Label: "Food", Amount: 600, CategoryID: 1, CategoryName: "Food"},
		{Label: "Transport", Amount: 399, CategoryID: 2, CategoryName: "Transport"},
		{Label: "Tiny", Amount: 1, CategoryID: 3, CategoryName: "Tiny"},
	}

	result := svc.applyOtherThreshold(entries, 0.02)

	// Tiny should be combined into Other
	var found bool
	for _, entry := range result {
		if entry.Label == "Other" {
			found = true
			assert.InDelta(t, 1.0, entry.Amount, 0.01)
		}
		assert.NotEqual(t, "Tiny", entry.Label)
	}
	assert.True(t, found)

	// Sorted by amount descending
	for i := 0; i < len(result)-1; i++ {
		assert.GreaterOrEqual(t, result[i].Amount, result[i+1].Amount)
	}
}

func TestApplyOtherThreshold_AllAboveThreshold(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	entries := []expenseEntry{
		{Label: "Food", Amount: 500, CategoryID: 1, CategoryName: "Food"},
		{Label: "Transport", Amount: 500, CategoryID: 2, CategoryName: "Transport"},
	}

	result := svc.applyOtherThreshold(entries, 0.02)

	// No "Other" entry
	for _, entry := range result {
		assert.NotEqual(t, "Other", entry.Label)
	}
	assert.Len(t, result, 2)
}

func TestApplyOtherThreshold_ZeroTotal(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	entries := []expenseEntry{
		{Label: "Food", Amount: 0, CategoryID: 1, CategoryName: "Food"},
	}

	result := svc.applyOtherThreshold(entries, 0.02)
	assert.Len(t, result, 1)
}

// ==================== getBaseCurrency Tests ====================

func TestGetBaseCurrency_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	curr, err := svc.getBaseCurrency(1)
	assert.NoError(t, err)
	assert.Equal(t, "USD", curr.Code)
}

func TestGetBaseCurrency_Error(t *testing.T) {
	svc, _, _, _, currencyRepo, _ := newTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("not found")
	}

	curr, err := svc.getBaseCurrency(1)
	assert.Error(t, err)
	assert.Nil(t, curr)
}
