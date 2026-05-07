package financial_planning

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/services/planned_transactions"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mock services and repositories ====================

type mockPlannedTxSvc struct {
	getUpcomingFunc func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error)
}

func (m *mockPlannedTxSvc) GetUpcoming(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
	if m.getUpcomingFunc != nil {
		return m.getUpcomingFunc(userID, days, includeInactive)
	}
	return nil, nil
}

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
	return nil, nil
}
func (m *mockAccountRepo) Update(account *models.Account) error                { return nil }
func (m *mockAccountRepo) SoftDelete(id int) error                             { return nil }
func (m *mockAccountRepo) UpdateBalance(id int, balance decimal.Decimal) error { return nil }
func (m *mockAccountRepo) SetArchiveStatus(id int, isArchived bool) error      { return nil }
func (m *mockAccountRepo) CountActiveByUserID(_ int) (int, error)              { return 0, nil }

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
		ID:               1,
		Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
		ActualDate:       date,
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

func setupTestService() (*Service, *mockPlannedTxSvc, *mockAccountRepo, *mockCurrencyRepo, *mockExchangeRateRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	plannedTxSvc := &mockPlannedTxSvc{}
	accountRepo := &mockAccountRepo{}
	currencyRepo := &mockCurrencyRepo{}
	exchangeRateRepo := &mockExchangeRateRepo{}

	currencySvc := &mockCurrencyService{exchangeRateRepo: exchangeRateRepo, logger: logger}
	svc := New(plannedTxSvc, accountRepo, currencyRepo, currencySvc, logger)

	return svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo
}

// ==================== CalculateFutureBalance Tests ====================

func TestCalculateFutureBalanceBaseCurrencyError(t *testing.T) {
	svc, _, _, currencyRepo, _ := setupTestService()
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestCalculateFutureBalanceAccountFetchError(t *testing.T) {
	svc, _, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestCalculateFutureBalancePlannedTxFetchError(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestCalculateFutureBalanceFutureDate(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		assert.Greater(t, days, 0, "future date should produce positive days")
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(1000), result.TotalCurrentBalance)
}

func TestCalculateFutureBalancePastDate(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		assert.Equal(t, 0, days, "past date should produce 0 days")
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(-1, 0, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(1000), result.TotalCurrentBalance)
}

func TestCalculateFutureBalancePlannedTxCurrencyLookupFailure(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestService()

	callCount := 0
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if id == 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return nil, errors.New("currency not found")
	}

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 999, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Only the USD tx (50) should be counted; bad currency tx is skipped
	assert.Equal(t, float64(50), result.TotalPlannedIncome)
	assert.Equal(t, 1, result.IncomeCount)
}

func TestCalculateFutureBalancePlannedTxErrNoExchangeRates(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, currency.ErrNoExchangeRates)
}

func TestCalculateFutureBalancePlannedTxNonFatalConversionError(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 99:
			return &models.Currency{ID: 99, Code: "XYZ", Name: "Unknown Currency"}, nil
		}
		return nil, errors.New("currency not found")
	}
	// Rates include USD but NOT XYZ
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 99, Amount: decimal.NewFromInt(500), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Only USD tx (200) counted, XYZ tx skipped
	assert.Equal(t, float64(200), result.TotalPlannedIncome)
	assert.Equal(t, 1, result.IncomeCount)
	assert.Equal(t, float64(1200), result.TotalProjectedBalance)
}

func TestCalculateFutureBalanceAccountCurrencyLookupFailure(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestService()

	callCount := 0
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount == 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return nil, errors.New("currency not found")
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Account is skipped from projections due to currency lookup failure
	assert.Empty(t, result.Accounts)
}

func TestCalculateFutureBalanceAccountErrNoExchangeRates(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(850)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, currency.ErrNoExchangeRates)
}

func TestCalculateFutureBalanceAccountFallbackToRawBalance(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 4:
			return &models.Currency{ID: 4, Code: "CHF", Name: "Swiss Franc"}, nil
		}
		return nil, errors.New("currency not found")
	}
	// Rates include USD but NOT CHF
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "CHF Account", CurrencyID: 4, Balance: decimal.NewFromInt(750)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(750), result.TotalCurrentBalance)
	assert.Equal(t, float64(750), result.TotalProjectedBalance)
}

func TestCalculateFutureBalanceMultiCurrencyConversion(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		case 3:
			return &models.Currency{ID: 3, Code: "GBP", Name: "British Pound"}, nil
		}
		return nil, errors.New("currency not found")
	}
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(850)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 3, Amount: decimal.NewFromInt(73), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)

	// EUR 850 / 0.85 = 1000 USD
	assert.Equal(t, float64(1000), result.TotalCurrentBalance)
	// GBP 73 / 0.73 = 100 USD
	assert.Equal(t, float64(100), result.TotalPlannedIncome)
	// Projected = 1000 + 100 = 1100
	assert.Equal(t, float64(1100), result.TotalProjectedBalance)
	// Per-account balance in own currency
	require.Len(t, result.Accounts, 1)
	assert.Equal(t, float64(850), result.Accounts[0].CurrentBalance)
	assert.Equal(t, "EUR", result.Accounts[0].CurrencyCode)
}

func TestCalculateFutureBalanceAccountFilterByIDs(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		AccountIDs:     []int{1},
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)

	// Only account 1 in projections
	require.Len(t, result.Accounts, 1)
	assert.Equal(t, 1, result.Accounts[0].AccountID)
	// Total current = only account 1 = 1000
	assert.Equal(t, float64(1000), result.TotalCurrentBalance)
}

func TestCalculateFutureBalanceIncomeExpenseCounting(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 4, CurrencyID: 1, Amount: decimal.NewFromInt(75), IsIncome: false, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, result.IncomeCount)
	assert.Equal(t, 2, result.ExpensesCount)
	assert.Equal(t, float64(300), result.TotalPlannedIncome)
	assert.Equal(t, float64(125), result.TotalPlannedExpenses)
	// Total current = 1000 + 2000 = 3000
	// Projected = 3000 + 300 - 125 = 3175
	assert.Equal(t, float64(3000), result.TotalCurrentBalance)
	assert.Equal(t, float64(3175), result.TotalProjectedBalance)
}

func TestCalculateFutureBalancePerAccountZeroPlannedAmounts(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	result, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)

	require.Len(t, result.Accounts, 1)
	assert.Equal(t, float64(0), result.Accounts[0].TotalPlannedIncome)
	assert.Equal(t, float64(0), result.Accounts[0].TotalPlannedExpenses)
	assert.Equal(t, float64(1000), result.Accounts[0].ProjectedBalance)
}

// ==================== GetBalanceProjection Tests ====================

func TestGetBalanceProjectionDefaultStartDate(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		// StartDate is zero
		EndDate:        time.Now().AddDate(0, 0, 3),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// StartDate should be defaulted to today-ish
	assert.False(t, result.StartDate.IsZero())
}

func TestGetBalanceProjectionDefaultPeriod(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "daily", result.Period)
}

func TestGetBalanceProjectionBaseCurrencyError(t *testing.T) {
	svc, _, _, currencyRepo, _ := setupTestService()
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestGetBalanceProjectionAccountFetchError(t *testing.T) {
	svc, _, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestGetBalanceProjectionEndDateBeforeStart(t *testing.T) {
	svc, _, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEndDateBeforeStart)
}

func TestGetBalanceProjectionPlannedTxFetchError(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
}

func TestGetBalanceProjectionAccountCurrencyFallbackToRaw(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestService()

	callCount := 0
	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount <= 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return nil, errors.New("currency not found")
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(500)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Raw balance used (500)
	require.Len(t, result.ProjectionPoints, 3)
	assert.Equal(t, float64(500), result.ProjectionPoints[0].Balance)
}

func TestGetBalanceProjectionAccountErrNoExchangeRates(t *testing.T) {
	svc, _, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, currency.ErrNoExchangeRates)
}

func TestGetBalanceProjectionCurrentBalanceMultipleAccounts(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Account1", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Account2", CurrencyID: 1, Balance: decimal.NewFromInt(500)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// 1000 + 500 = 1500
	require.Len(t, result.ProjectionPoints, 1)
	assert.Equal(t, float64(1500), result.ProjectionPoints[0].Balance)
}

func TestGetBalanceProjectionDailyPoints(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	assert.Len(t, result.ProjectionPoints, 5)
}

func TestGetBalanceProjectionWeeklyPoints(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), // Wednesday
		EndDate:        time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		Period:         "weekly",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Points: Mar 4, Mar 15, Mar 22, Mar 25
	assert.Len(t, result.ProjectionPoints, 4)
}

func TestGetBalanceProjectionMonthlyPoints(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		Period:         "monthly",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	// Points: Jan 15, Feb 28, Mar 31, Apr 10
	assert.Len(t, result.ProjectionPoints, 4)
}

func TestGetBalanceProjectionPeriodGrouping(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(30), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	require.Len(t, result.ProjectionPoints, 5)

	// Day 1: no tx, balance = 1000
	assert.Equal(t, float64(0), result.ProjectionPoints[0].Income)
	assert.Equal(t, float64(0), result.ProjectionPoints[0].Expenses)
	assert.Equal(t, float64(1000), result.ProjectionPoints[0].Balance)

	// Day 2: +100 income, -30 expense, balance = 1070
	assert.Equal(t, float64(100), result.ProjectionPoints[1].Income)
	assert.Equal(t, float64(30), result.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1070), result.ProjectionPoints[1].Balance)

	// Day 3: no tx, balance = 1070
	assert.Equal(t, float64(1070), result.ProjectionPoints[2].Balance)

	// Day 4: -50, balance = 1020
	assert.Equal(t, float64(50), result.ProjectionPoints[3].Expenses)
	assert.Equal(t, float64(1020), result.ProjectionPoints[3].Balance)

	// Day 5: no tx, balance = 1020
	assert.Equal(t, float64(1020), result.ProjectionPoints[4].Balance)
}

func TestGetBalanceProjectionPlannedTxCurrencyLookupFailureInPeriodLoop(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		if id == 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		return nil, errors.New("currency not found")
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	occDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
			{PlannedTransactionID: 2, CurrencyID: 88, Amount: decimal.NewFromInt(300), IsIncome: true, OccurrenceDate: occDate},
		}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	require.Len(t, result.ProjectionPoints, 3)

	// Day 2: only valid USD tx (100) counted, bad currency skipped
	assert.Equal(t, float64(100), result.ProjectionPoints[1].Income)
	assert.Equal(t, float64(1100), result.ProjectionPoints[1].Balance)
}

func TestGetBalanceProjectionPlannedTxErrNoExchangeRatesInPeriodLoop(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	occDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
		}, nil
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, currency.ErrNoExchangeRates)
}

func TestGetBalanceProjectionPlannedTxNonFatalMissingRateInPeriodLoop(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 99:
			return &models.Currency{ID: 99, Code: "XYZ", Name: "Unknown Currency"}, nil
		}
		return nil, errors.New("currency not found")
	}
	// Rates include USD but NOT XYZ
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	occDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
			{PlannedTransactionID: 2, CurrencyID: 99, Amount: decimal.NewFromInt(500), IsIncome: false, OccurrenceDate: occDate},
		}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	require.Len(t, result.ProjectionPoints, 3)

	// Day 2: USD income (100) counted, XYZ expense skipped
	assert.Equal(t, float64(100), result.ProjectionPoints[1].Income)
	assert.Equal(t, float64(0), result.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1100), result.ProjectionPoints[1].Balance)
}

func TestGetBalanceProjectionRunningBalanceTracking(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	// Weekly grouping with transactions in different periods
	startDate := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // Wednesday
	endDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      startDate,
		EndDate:        endDate,
		Period:         "weekly",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)

	// Points: Mar 4, Mar 15, Mar 22, Mar 25
	require.Len(t, result.ProjectionPoints, 4)

	// Point 0 (Mar 4): no tx on this day, balance = 1000
	assert.Equal(t, float64(0), result.ProjectionPoints[0].Income)
	assert.Equal(t, float64(0), result.ProjectionPoints[0].Expenses)
	assert.Equal(t, float64(1000), result.ProjectionPoints[0].Balance)

	// Point 1 (Mar 5-15): income=100, expense=50, balance = 1050
	assert.Equal(t, float64(100), result.ProjectionPoints[1].Income)
	assert.Equal(t, float64(50), result.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1050), result.ProjectionPoints[1].Balance)

	// Point 2 (Mar 16-22): income=200, balance = 1250
	assert.Equal(t, float64(200), result.ProjectionPoints[2].Income)
	assert.Equal(t, float64(0), result.ProjectionPoints[2].Expenses)
	assert.Equal(t, float64(1250), result.ProjectionPoints[2].Balance)

	// Point 3 (Mar 23-25): no tx, balance = 1250
	assert.Equal(t, float64(0), result.ProjectionPoints[3].Income)
	assert.Equal(t, float64(0), result.ProjectionPoints[3].Expenses)
	assert.Equal(t, float64(1250), result.ProjectionPoints[3].Balance)
}

func TestGetBalanceProjectionAccountFallbackToRawBalance(t *testing.T) {
	svc, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestService()

	currencyRepo.getByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 4:
			return &models.Currency{ID: 4, Code: "CHF", Name: "Swiss Franc"}, nil
		}
		return nil, errors.New("currency not found")
	}
	// Rates do not include CHF
	exchangeRateRepo.getAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "CHF Account", CurrencyID: 4, Balance: decimal.NewFromInt(500)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	result, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		Period:         "daily",
		BaseCurrencyID: 1,
	})
	require.NoError(t, err)
	require.Len(t, result.ProjectionPoints, 3)
	assert.Equal(t, float64(500), result.ProjectionPoints[0].Balance)
	assert.Equal(t, float64(500), result.ProjectionPoints[1].Balance)
	assert.Equal(t, float64(500), result.ProjectionPoints[2].Balance)
}

// ==================== AccountFilters Tests ====================

func TestCalculateFutureBalanceAccountFiltersIncludeHiddenTrue(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	var capturedFilters types.AccountFilters
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
		IncludeHidden:  true,
	})
	require.NoError(t, err)
	assert.True(t, capturedFilters.IncludeHidden, "IncludeHidden should be true when params.IncludeHidden is true")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestCalculateFutureBalanceAccountFiltersIncludeHiddenFalse(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	var capturedFilters types.AccountFilters
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	_, err := svc.CalculateFutureBalance(1, FutureBalanceParams{
		TargetDate:     time.Now().AddDate(0, 1, 0),
		BaseCurrencyID: 1,
		IncludeHidden:  false,
	})
	require.NoError(t, err)
	assert.False(t, capturedFilters.IncludeHidden, "IncludeHidden should be false when params.IncludeHidden is false")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestGetBalanceProjectionAccountFiltersIncludeHiddenTrue(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	var capturedFilters types.AccountFilters
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
		IncludeHidden:  true,
	})
	require.NoError(t, err)
	assert.True(t, capturedFilters.IncludeHidden, "IncludeHidden should be true when params.IncludeHidden is true")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestGetBalanceProjectionAccountFiltersIncludeHiddenFalse(t *testing.T) {
	svc, plannedTxSvc, accountRepo, _, _ := setupTestService()

	var capturedFilters types.AccountFilters
	accountRepo.getByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.getUpcomingFunc = func(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error) {
		return []planned_transactions.Occurrence{}, nil
	}

	_, err := svc.GetBalanceProjection(1, BalanceProjectionParams{
		StartDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		BaseCurrencyID: 1,
		IncludeHidden:  false,
	})
	require.NoError(t, err)
	assert.False(t, capturedFilters.IncludeHidden, "IncludeHidden should be false when params.IncludeHidden is false")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

// ==================== generateDatePoints Tests ====================

func TestGenerateDatePointsDaily(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	points := generateDatePoints(start, end, "daily")

	require.Len(t, points, 5)
	assert.Equal(t, "2026-03-01", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-02", points[1].Format("2006-01-02"))
	assert.Equal(t, "2026-03-03", points[2].Format("2006-01-02"))
	assert.Equal(t, "2026-03-04", points[3].Format("2006-01-02"))
	assert.Equal(t, "2026-03-05", points[4].Format("2006-01-02"))
}

func TestGenerateDatePointsWeekly(t *testing.T) {
	// Start on Wednesday March 4, 2026 (Wednesday)
	start := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	points := generateDatePoints(start, end, "weekly")

	// First point: Mar 4 (start)
	// Next Monday: Mar 9, so first Sunday: Mar 15
	// Next Sunday: Mar 22
	// Next Sunday: Mar 29 > end, so cap at Mar 25
	require.Len(t, points, 4)
	assert.Equal(t, "2026-03-04", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-15", points[1].Format("2006-01-02"))
	assert.Equal(t, "2026-03-22", points[2].Format("2006-01-02"))
	assert.Equal(t, "2026-03-25", points[3].Format("2006-01-02"))
}

func TestGenerateDatePointsWeeklyStartOnMonday(t *testing.T) {
	// Start on Monday March 2, 2026
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

	points := generateDatePoints(start, end, "weekly")

	// First point: Mar 2 (start, Monday)
	// Next Monday after start: Mar 9
	// Sunday from Mar 9: Mar 15
	// Next Monday: Mar 16, Sunday: Mar 22
	require.Len(t, points, 3)
	assert.Equal(t, "2026-03-02", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-15", points[1].Format("2006-01-02"))
	assert.Equal(t, "2026-03-22", points[2].Format("2006-01-02"))
}

func TestGenerateDatePointsWeeklyShortRange(t *testing.T) {
	// Range is only 3 days (shorter than a week)
	start := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // Wednesday
	end := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)   // Friday

	points := generateDatePoints(start, end, "weekly")

	// First point: Mar 4
	// Next Monday: Mar 9, Sunday: Mar 15 > end, cap at Mar 6
	require.Len(t, points, 2)
	assert.Equal(t, "2026-03-04", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-06", points[1].Format("2006-01-02"))
}

func TestGenerateDatePointsMonthly(t *testing.T) {
	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	points := generateDatePoints(start, end, "monthly")

	// First point: Jan 15
	// Start from Feb 1: end of Feb = Feb 28
	// Mar 1: end of Mar = Mar 31
	// Apr 1: end of Apr = Apr 30 > end, cap at Apr 10
	require.Len(t, points, 4)
	assert.Equal(t, "2026-01-15", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-02-28", points[1].Format("2006-01-02"))
	assert.Equal(t, "2026-03-31", points[2].Format("2006-01-02"))
	assert.Equal(t, "2026-04-10", points[3].Format("2006-01-02"))
}

func TestGenerateDatePointsMonthlySameDay(t *testing.T) {
	// Start on Jan 31, end on May 1
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	points := generateDatePoints(start, end, "monthly")

	// Points: [Jan 31, Feb 28, Mar 31, Apr 30, May 1]
	require.Len(t, points, 5)
	assert.Equal(t, "2026-01-31", points[0].Format("2006-01-02"))
	assert.Equal(t, "2026-02-28", points[1].Format("2006-01-02"))
	assert.Equal(t, "2026-03-31", points[2].Format("2006-01-02"))
	assert.Equal(t, "2026-04-30", points[3].Format("2006-01-02"))
	assert.Equal(t, "2026-05-01", points[4].Format("2006-01-02"))
}
