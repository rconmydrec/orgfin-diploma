package budgets

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockBudgetRepo struct {
	GetByUserIDFunc           func(userID int, include string) ([]*models.Budget, error)
	UpdateCollectedAmountFunc func(budgetID int, amount decimal.Decimal) error
	CreateFunc                func(budget *models.Budget) (*models.Budget, error)
	GetByIDFunc               func(id int) (*models.Budget, error)
	GetOutdatedBudgetsFunc    func() ([]*models.Budget, error)
	UpdateFunc                func(budget *models.Budget) error
	DeleteFunc                func(id int) error
	ArchiveFunc               func(id int) error
	UpdateCalls               []updateCall
}

type updateCall struct {
	BudgetID int
	Amount   decimal.Decimal
}

func (m *mockBudgetRepo) GetByUserID(userID int, include string) ([]*models.Budget, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID, include)
	}
	return nil, nil
}

func (m *mockBudgetRepo) UpdateCollectedAmount(budgetID int, amount decimal.Decimal) error {
	m.UpdateCalls = append(m.UpdateCalls, updateCall{BudgetID: budgetID, Amount: amount})
	if m.UpdateCollectedAmountFunc != nil {
		return m.UpdateCollectedAmountFunc(budgetID, amount)
	}
	return nil
}

func (m *mockBudgetRepo) Create(budget *models.Budget) (*models.Budget, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(budget)
	}
	created := *budget
	created.ID = 1
	return &created, nil
}

func (m *mockBudgetRepo) GetByID(id int) (*models.Budget, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Budget{ID: id, UserID: 1, Name: "Test"}, nil
}

func (m *mockBudgetRepo) GetOutdatedBudgets() ([]*models.Budget, error) {
	if m.GetOutdatedBudgetsFunc != nil {
		return m.GetOutdatedBudgetsFunc()
	}
	return []*models.Budget{}, nil
}

func (m *mockBudgetRepo) Update(budget *models.Budget) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(budget)
	}
	return nil
}

func (m *mockBudgetRepo) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *mockBudgetRepo) Archive(id int) error {
	if m.ArchiveFunc != nil {
		return m.ArchiveFunc(id)
	}
	return nil
}

type mockTransactionRepo struct {
	GetExpenseTransactionsForBudgetFunc func(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error)
}

func (m *mockTransactionRepo) GetExpenseTransactionsForBudget(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error) {
	if m.GetExpenseTransactionsForBudgetFunc != nil {
		return m.GetExpenseTransactionsForBudgetFunc(userID, startDate, endDate, categoryIDs)
	}
	return nil, nil
}

type mockCurrencyConverter struct {
	ConvertAmountFunc func(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal
}

func (m *mockCurrencyConverter) ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal {
	if m.ConvertAmountFunc != nil {
		return m.ConvertAmountFunc(amount, fromCurrency, toCurrency, date)
	}
	return amount
}

type mockCurrencyRepo struct {
	GetByIDFunc func(id int) (*models.Currency, error)
}

func (m *mockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Currency{ID: id, Code: "USD", Name: "US Dollar"}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newTestService creates a service with all mock dependencies.
func newTestService(budgetRepo *mockBudgetRepo, currencyRepo *mockCurrencyRepo) *Service {
	return New(budgetRepo, &mockTransactionRepo{}, currencyRepo, &mockCurrencyConverter{}, testLogger())
}

// --- RecalculateCollectedAmounts Tests (existing) ---

func TestRecalculateCollectedAmounts_NoBudgets(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			assert.Equal(t, 42, userID)
			assert.Equal(t, "active", include)
			return nil, nil
		},
	}
	svc := New(budgetRepo, &mockTransactionRepo{}, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)
	assert.Empty(t, budgetRepo.UpdateCalls)
}

func TestRecalculateCollectedAmounts_EmptyBudgetSlice(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{}, nil
		},
	}
	svc := New(budgetRepo, &mockTransactionRepo{}, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)
}

func TestRecalculateCollectedAmounts_GetBudgetsError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return nil, errors.New("database connection lost")
		},
	}
	svc := New(budgetRepo, &mockTransactionRepo{}, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get user budgets")
}

func TestRecalculateCollectedAmounts_Success_SameCurrency(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)
	endDate := now.AddDate(0, 0, 1)

	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:                 1,
					UserID:             42,
					Name:               "Groceries",
					StartDate:          startDate,
					EndDate:            endDate,
					IncludedCategories: "10,20",
					Currency:           &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"},
				},
			}, nil
		},
	}

	txDate := now.AddDate(0, 0, -5)
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			assert.Equal(t, 42, userID)
			assert.Equal(t, startDate, sd)
			assert.Equal(t, endDate, ed)
			assert.Equal(t, []int{10, 20}, catIDs)
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(100), CurrencyCode: "USD", DateTime: txDate},
				{Amount: decimal.NewFromInt(50), CurrencyCode: "USD", DateTime: txDate},
			}, nil
		},
	}

	// Same currency: converter returns amount as-is
	converter := &mockCurrencyConverter{
		ConvertAmountFunc: func(amount decimal.Decimal, from, to string, date time.Time) decimal.Decimal {
			assert.Equal(t, "USD", from)
			assert.Equal(t, "USD", to)
			return amount
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, converter, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)

	require.Len(t, budgetRepo.UpdateCalls, 1)
	assert.Equal(t, 1, budgetRepo.UpdateCalls[0].BudgetID)
	assert.True(t, decimal.NewFromInt(150).Equal(budgetRepo.UpdateCalls[0].Amount),
		"expected 150, got %s", budgetRepo.UpdateCalls[0].Amount)
}

func TestRecalculateCollectedAmounts_Success_CurrencyConversion(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)
	endDate := now.AddDate(0, 0, 1)

	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:                 1,
					UserID:             42,
					Name:               "Travel",
					StartDate:          startDate,
					EndDate:            endDate,
					IncludedCategories: "",
					Currency:           &models.Currency{ID: 2, Code: "EUR", Name: "Euro"},
				},
			}, nil
		},
	}

	txDate := now.AddDate(0, 0, -3)
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			assert.Nil(t, catIDs, "empty IncludedCategories should yield nil categoryIDs")
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(100), CurrencyCode: "USD", DateTime: txDate},
				{Amount: decimal.NewFromInt(200), CurrencyCode: "GBP", DateTime: txDate},
			}, nil
		},
	}

	// Simulate conversion: USD->EUR = amount * 0.9, GBP->EUR = amount * 1.15
	converter := &mockCurrencyConverter{
		ConvertAmountFunc: func(amount decimal.Decimal, from, to string, date time.Time) decimal.Decimal {
			assert.Equal(t, "EUR", to)
			switch from {
			case "USD":
				return amount.Mul(decimal.NewFromFloat(0.9))
			case "GBP":
				return amount.Mul(decimal.NewFromFloat(1.15))
			default:
				return amount
			}
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, converter, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)

	require.Len(t, budgetRepo.UpdateCalls, 1)
	// 100 * 0.9 + 200 * 1.15 = 90 + 230 = 320
	expected := decimal.NewFromInt(90).Add(decimal.NewFromInt(230))
	assert.True(t, expected.Equal(budgetRepo.UpdateCalls[0].Amount),
		"expected %s, got %s", expected, budgetRepo.UpdateCalls[0].Amount)
}

func TestRecalculateCollectedAmounts_MultipleBudgets(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:                 1,
					UserID:             42,
					Name:               "Budget1",
					StartDate:          now.AddDate(0, -1, 0),
					EndDate:            now,
					IncludedCategories: "5",
					Currency:           &models.Currency{ID: 1, Code: "USD"},
				},
				{
					ID:                 2,
					UserID:             42,
					Name:               "Budget2",
					StartDate:          now.AddDate(0, -2, 0),
					EndDate:            now,
					IncludedCategories: "",
					Currency:           &models.Currency{ID: 1, Code: "USD"},
				},
			}, nil
		},
	}

	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(75), CurrencyCode: "USD", DateTime: now},
			}, nil
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)

	require.Len(t, budgetRepo.UpdateCalls, 2)
	assert.Equal(t, 1, budgetRepo.UpdateCalls[0].BudgetID)
	assert.Equal(t, 2, budgetRepo.UpdateCalls[1].BudgetID)
}

func TestRecalculateCollectedAmounts_NoCurrency(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:       1,
					UserID:   42,
					Name:     "NoCurrency",
					Currency: nil, // no currency loaded
				},
			}, nil
		},
	}

	svc := New(budgetRepo, &mockTransactionRepo{}, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no currency information")
}

func TestRecalculateCollectedAmounts_EmptyCurrencyCode(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:       1,
					UserID:   42,
					Name:     "EmptyCode",
					Currency: &models.Currency{ID: 1, Code: "", Name: "Unknown"},
				},
			}, nil
		},
	}

	svc := New(budgetRepo, &mockTransactionRepo{}, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no currency information")
}

func TestRecalculateCollectedAmounts_TransactionQueryError(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    42,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					Currency:  &models.Currency{ID: 1, Code: "USD"},
				},
			}, nil
		},
	}

	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return nil, errors.New("query timeout")
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query transactions")
}

func TestRecalculateCollectedAmounts_UpdateError(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    42,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					Currency:  &models.Currency{ID: 1, Code: "USD"},
				},
			}, nil
		},
		UpdateCollectedAmountFunc: func(budgetID int, amount decimal.Decimal) error {
			return errors.New("db write error")
		},
	}

	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(50), CurrencyCode: "USD", DateTime: now},
			}, nil
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update collected amount")
}

func TestRecalculateCollectedAmounts_PartialError(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    42,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					Currency:  &models.Currency{ID: 1, Code: "USD"},
				},
				{
					ID:       2,
					UserID:   42,
					Currency: nil, // will fail: no currency
				},
				{
					ID:        3,
					UserID:    42,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					Currency:  &models.Currency{ID: 1, Code: "USD"},
				},
			}, nil
		},
	}

	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return nil, nil
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recalculate budget 2")
	// Budget 1 and 3 should still have been updated
	require.Len(t, budgetRepo.UpdateCalls, 2)
	assert.Equal(t, 1, budgetRepo.UpdateCalls[0].BudgetID)
	assert.Equal(t, 3, budgetRepo.UpdateCalls[1].BudgetID)
}

func TestRecalculateCollectedAmounts_NoTransactions(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    42,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					Currency:  &models.Currency{ID: 1, Code: "USD"},
				},
			}, nil
		},
	}

	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return nil, nil // no transactions
		},
	}

	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())
	err := svc.RecalculateCollectedAmounts(42)
	assert.NoError(t, err)

	require.Len(t, budgetRepo.UpdateCalls, 1)
	assert.True(t, decimal.Zero.Equal(budgetRepo.UpdateCalls[0].Amount),
		"expected 0, got %s", budgetRepo.UpdateCalls[0].Amount)
}

// --- parseCategoryIDs tests ---

func TestParseCategoryIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{"empty string", "", nil},
		{"whitespace only", "   ", nil},
		{"single ID", "5", []int{5}},
		{"multiple IDs", "1,2,3", []int{1, 2, 3}},
		{"with spaces", " 10 , 20 , 30 ", []int{10, 20, 30}},
		{"with invalid entries", "1,abc,3", []int{1, 3}},
		{"trailing comma", "1,2,", []int{1, 2}},
		{"leading comma", ",1,2", []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCategoryIDs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- categoriesToCSV tests ---

func TestCategoriesToCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected string
	}{
		{"empty slice", []int{}, ""},
		{"nil slice", nil, ""},
		{"single ID", []int{5}, "5"},
		{"multiple IDs", []int{1, 2, 3}, "1,2,3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categoriesToCSV(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Create Tests ---

func TestCreate_Success_WithCategories(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, 1, budget.UserID)
			assert.Equal(t, "Test Budget", budget.Name)
			assert.Equal(t, 1, budget.CurrencyID)
			assert.Equal(t, "MONTHLY", budget.Period)
			assert.Equal(t, "1,2", budget.IncludedCategories)
			created := *budget
			created.ID = 10
			return &created, nil
		},
	}
	currencyRepo := &mockCurrencyRepo{}
	svc := newTestService(budgetRepo, currencyRepo)

	comment := "test comment"
	result, err := svc.Create(1, CreateParams{
		Name:         "Test Budget",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(1000),
		Period:       "monthly",
		Repeat:       true,
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		Categories:   []int{1, 2},
		Comment:      &comment,
	})

	assert.NoError(t, err)
	assert.Equal(t, 10, result.ID)
	assert.Equal(t, "Test Budget", result.Name)
	require.NotNil(t, result.Currency)
	assert.Equal(t, "USD", result.Currency.Code)
}

func TestCreate_Success_WithoutCategories(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, "", budget.IncludedCategories)
			created := *budget
			created.ID = 10
			return &created, nil
		},
	}
	currencyRepo := &mockCurrencyRepo{}
	svc := newTestService(budgetRepo, currencyRepo)

	result, err := svc.Create(1, CreateParams{
		Name:         "Test Budget",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(1000),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		Categories:   []int{},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc := newTestService(&mockBudgetRepo{}, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "not-a-date",
		EndDate:      "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrInvalidStartDate)
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc := newTestService(&mockBudgetRepo{}, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "not-a-date",
	})

	assert.ErrorIs(t, err, ErrInvalidEndDate)
}

func TestCreate_InvalidCurrency(t *testing.T) {
	currencyRepo := &mockCurrencyRepo{
		GetByIDFunc: func(id int) (*models.Currency, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(&mockBudgetRepo{}, currencyRepo)

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   999,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestCreate_DBError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidStartDate)
	assert.NotErrorIs(t, err, ErrInvalidEndDate)
	assert.NotErrorIs(t, err, ErrInvalidCurrency)
}

func TestCreate_PeriodNormalizedToUppercase(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, "MONTHLY", budget.Period)
			created := *budget
			created.ID = 1
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	assert.NoError(t, err)
}

func TestCreate_EndDatePlus1Day(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			// End date "2024-01-31" should become 2024-02-01 (31 + 1 day)
			assert.Equal(t, 2024, budget.EndDate.Year())
			assert.Equal(t, time.February, budget.EndDate.Month())
			assert.Equal(t, 1, budget.EndDate.Day())
			created := *budget
			created.ID = 1
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	assert.NoError(t, err)
}

func TestCreate_DateOnlyFormat(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, 2024, budget.StartDate.Year())
			assert.Equal(t, time.January, budget.StartDate.Month())
			assert.Equal(t, 1, budget.StartDate.Day())
			created := *budget
			created.ID = 1
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	assert.NoError(t, err)
}

func TestCreate_RFC3339Format(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, 2024, budget.StartDate.Year())
			assert.Equal(t, time.January, budget.StartDate.Month())
			assert.Equal(t, 1, budget.StartDate.Day())
			created := *budget
			created.ID = 1
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Create(1, CreateParams{
		Name:         "Test",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(100),
		Period:       "monthly",
		StartDate:    "2024-01-01T00:00:00Z",
		EndDate:      "2024-12-31T00:00:00Z",
	})

	assert.NoError(t, err)
}

// TestCreate_RecalculatesCollectedAmount verifies that after persistence the
// service recalculates collected_amount from existing matching transactions
// and the returned domain object reflects the new value.
func TestCreate_RecalculatesCollectedAmount(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			created := *budget
			created.ID = 42
			return &created, nil
		},
	}
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			assert.Equal(t, 1, userID)
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(120), CurrencyCode: "USD", DateTime: sd},
				{Amount: decimal.NewFromInt(80), CurrencyCode: "USD", DateTime: sd},
			}, nil
		},
	}
	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	result, err := svc.Create(1, CreateParams{
		Name:         "Recalc Budget",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(1000),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		Categories:   []int{1, 2},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromInt(200).Equal(result.CollectedAmount),
		"expected collectedAmount=200, got %s", result.CollectedAmount)
	require.Len(t, budgetRepo.UpdateCalls, 1)
	assert.Equal(t, 42, budgetRepo.UpdateCalls[0].BudgetID)
	assert.True(t, decimal.NewFromInt(200).Equal(budgetRepo.UpdateCalls[0].Amount))
}

// TestCreate_RecalcFailureReturns200 verifies that when persistence succeeds
// but recalculation fails, the service still returns the persisted budget
// without an error (the failure is logged, not propagated).
func TestCreate_RecalcFailureReturns200(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			created := *budget
			created.ID = 7
			return &created, nil
		},
		UpdateCollectedAmountFunc: func(budgetID int, amount decimal.Decimal) error {
			return errors.New("simulated update failure")
		},
	}
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(50), CurrencyCode: "USD", DateTime: sd},
			}, nil
		},
	}
	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	result, err := svc.Create(1, CreateParams{
		Name:         "Failing Recalc",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(500),
		Period:       "monthly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
	})

	require.NoError(t, err, "create must succeed even when recalc fails")
	require.NotNil(t, result)
	assert.Equal(t, 7, result.ID)
	assert.Equal(t, "Failing Recalc", result.Name)
	// CollectedAmount stays at the persisted (zero) value because recalc failed
	// before the in-memory assignment.
	assert.True(t, result.CollectedAmount.IsZero())
}

// --- GetByUserID Tests ---

func TestGetByUserID_Success(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			assert.Equal(t, 1, userID)
			assert.Equal(t, "active", include)
			return []*models.Budget{
				{ID: 1, UserID: 1, Name: "Budget1", StartDate: now, EndDate: now.AddDate(0, 1, 0)},
			}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.GetByUserID(1, "active")
	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Budget1", result[0].Name)
}

func TestGetByUserID_EmptyResult(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return []*models.Budget{}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.GetByUserID(1, "all")
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetByUserID_DefaultInclude(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			assert.Equal(t, "all", include)
			return []*models.Budget{}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.GetByUserID(1, "")
	assert.NoError(t, err)
}

func TestGetByUserID_DBError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByUserIDFunc: func(userID int, include string) ([]*models.Budget, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.GetByUserID(1, "all")
	assert.Error(t, err)
}

// --- Update Tests ---

func TestUpdate_Success(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1, Name: "Old Name"}, nil
		},
		UpdateFunc: func(budget *models.Budget) error {
			assert.Equal(t, "New Name", budget.Name)
			assert.Equal(t, "WEEKLY", budget.Period)
			return nil
		},
	}
	currencyRepo := &mockCurrencyRepo{}
	svc := newTestService(budgetRepo, currencyRepo)

	result, err := svc.Update(1, 10, UpdateParams{
		Name:         "New Name",
		CurrencyID:   1,
		TargetAmount: decimal.NewFromInt(500),
		Period:       "weekly",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		Categories:   []int{1},
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
	require.NotNil(t, result.Currency)
}

func TestUpdate_InvalidStartDate(t *testing.T) {
	svc := newTestService(&mockBudgetRepo{}, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:      "Test",
		StartDate: "bad-date",
		EndDate:   "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrInvalidStartDate)
}

func TestUpdate_InvalidEndDate(t *testing.T) {
	svc := newTestService(&mockBudgetRepo{}, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:      "Test",
		StartDate: "2024-01-01",
		EndDate:   "bad-date",
	})

	assert.ErrorIs(t, err, ErrInvalidEndDate)
}

func TestUpdate_NotFound(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:      "Test",
		StartDate: "2024-01-01",
		EndDate:   "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_GetByIDDBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, dbErr
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:      "Test",
		StartDate: "2024-01-01",
		EndDate:   "2024-01-31",
	})

	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestUpdate_AccessDenied(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 999}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:      "Test",
		StartDate: "2024-01-01",
		EndDate:   "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestUpdate_InvalidCurrency(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
	}
	currencyRepo := &mockCurrencyRepo{
		GetByIDFunc: func(id int) (*models.Currency, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(budgetRepo, currencyRepo)

	_, err := svc.Update(1, 10, UpdateParams{
		Name:       "Test",
		CurrencyID: 999,
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
	})

	assert.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestUpdate_DBError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		UpdateFunc: func(budget *models.Budget) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:       "Test",
		CurrencyID: 1,
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
	})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrAccessDenied)
	assert.NotErrorIs(t, err, ErrInvalidCurrency)
}

func TestUpdate_PeriodNormalized_EndDatePlus1Day(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		UpdateFunc: func(budget *models.Budget) error {
			assert.Equal(t, "MONTHLY", budget.Period)
			// End date "2024-01-31" + 1 day = 2024-02-01
			assert.Equal(t, 2024, budget.EndDate.Year())
			assert.Equal(t, time.February, budget.EndDate.Month())
			assert.Equal(t, 1, budget.EndDate.Day())
			return nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.Update(1, 10, UpdateParams{
		Name:       "Test",
		CurrencyID: 1,
		Period:     "monthly",
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
	})

	assert.NoError(t, err)
}

// TestUpdate_RecalculatesCollectedAmount verifies that after persistence the
// service recalculates collected_amount against the (possibly changed) date
// window, categories, and currency, and the returned domain object reflects
// the new value.
func TestUpdate_RecalculatesCollectedAmount(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			// Pre-existing collected amount that should be overwritten by recalc.
			return &models.Budget{
				ID:              id,
				UserID:          1,
				CollectedAmount: decimal.NewFromInt(999),
			}, nil
		},
	}
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(60), CurrencyCode: "USD", DateTime: sd},
				{Amount: decimal.NewFromInt(40), CurrencyCode: "USD", DateTime: sd},
			}, nil
		},
	}
	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	result, err := svc.Update(1, 10, UpdateParams{
		Name:       "Updated",
		CurrencyID: 1,
		Period:     "monthly",
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
		Categories: []int{1, 2},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromInt(100).Equal(result.CollectedAmount),
		"expected recalculated collectedAmount=100, got %s", result.CollectedAmount)
	require.Len(t, budgetRepo.UpdateCalls, 1)
	assert.Equal(t, 10, budgetRepo.UpdateCalls[0].BudgetID)
}

// TestUpdate_RecalcFailureReturns200 verifies that when persistence succeeds
// but recalculation fails, the service still returns the persisted budget
// without an error (the failure is logged, not propagated).
func TestUpdate_RecalcFailureReturns200(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{
				ID:              id,
				UserID:          1,
				CollectedAmount: decimal.NewFromInt(500),
			}, nil
		},
		UpdateCollectedAmountFunc: func(budgetID int, amount decimal.Decimal) error {
			return errors.New("simulated update failure")
		},
	}
	txRepo := &mockTransactionRepo{
		GetExpenseTransactionsForBudgetFunc: func(userID int, sd, ed time.Time, catIDs []int) ([]types.BudgetTransactionRow, error) {
			return []types.BudgetTransactionRow{
				{Amount: decimal.NewFromInt(123), CurrencyCode: "USD", DateTime: sd},
			}, nil
		},
	}
	svc := New(budgetRepo, txRepo, &mockCurrencyRepo{}, &mockCurrencyConverter{}, testLogger())

	result, err := svc.Update(1, 10, UpdateParams{
		Name:       "Updated",
		CurrencyID: 1,
		Period:     "monthly",
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
	})

	require.NoError(t, err, "update must succeed even when recalc fails")
	require.NotNil(t, result)
	assert.Equal(t, 10, result.ID)
	assert.Equal(t, "Updated", result.Name)
	// CollectedAmount stays at the pre-update value because recalc failed
	// before the in-memory assignment.
	assert.True(t, decimal.NewFromInt(500).Equal(result.CollectedAmount))
}

// --- Delete Tests ---

func TestDelete_Success(t *testing.T) {
	deleteCalled := false
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		DeleteFunc: func(id int) error {
			deleteCalled = true
			assert.Equal(t, 10, id)
			return nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Delete(1, 10)
	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDelete_NotFound(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Delete(1, 10)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_GetByIDDBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, dbErr
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Delete(1, 10)
	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestDelete_AccessDenied(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 999}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Delete(1, 10)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestDelete_DBError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		DeleteFunc: func(id int) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Delete(1, 10)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrAccessDenied)
}

// --- Archive Tests ---

func TestArchive_Success(t *testing.T) {
	archiveCalled := false
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		ArchiveFunc: func(id int) error {
			archiveCalled = true
			assert.Equal(t, 10, id)
			return nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Archive(1, 10)
	assert.NoError(t, err)
	assert.True(t, archiveCalled)
}

func TestArchive_NotFound(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Archive(1, 10)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestArchive_GetByIDDBError(t *testing.T) {
	dbErr := errors.New("connection refused")
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return nil, dbErr
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Archive(1, 10)
	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestArchive_AccessDenied(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 999}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Archive(1, 10)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestArchive_DBError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetByIDFunc: func(id int) (*models.Budget, error) {
			return &models.Budget{ID: id, UserID: 1}, nil
		},
		ArchiveFunc: func(id int) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	err := svc.Archive(1, 10)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrAccessDenied)
}

// --- DailyProcessing Tests ---

func TestDailyProcessing_GetOutdatedError(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.DailyProcessing()
	assert.Error(t, err)
}

func TestDailyProcessing_NoOutdatedBudgets(t *testing.T) {
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{}, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
}

func TestDailyProcessing_NonRepeating(t *testing.T) {
	now := time.Now()
	archiveCalled := false
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Budget",
					Period:    "MONTHLY",
					Repeat:    false,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		ArchiveFunc: func(id int) error {
			archiveCalled = true
			return nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.True(t, archiveCalled)
}

func TestDailyProcessing_RepeatMonthly(t *testing.T) {
	now := time.Now()
	createCalled := false
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:           1,
					UserID:       1,
					Name:         "Monthly Budget",
					CurrencyID:   1,
					TargetAmount: decimal.NewFromInt(1000),
					Period:       "MONTHLY",
					Repeat:       true,
					StartDate:    now.AddDate(0, -1, 0),
					EndDate:      now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			createCalled = true
			assert.Contains(t, budget.Name, "(copy)")
			assert.True(t, budget.CollectedAmount.IsZero())
			assert.Equal(t, 1, budget.CurrencyID)
			assert.Equal(t, decimal.NewFromInt(1000), budget.TargetAmount)
			assert.Equal(t, "MONTHLY", budget.Period)
			assert.True(t, budget.Repeat)

			// Verify monthly date shift
			expectedStart := now.AddDate(0, -1, 0).AddDate(0, 1, 0)
			expectedEnd := now.AddDate(0, 0, -1).AddDate(0, 1, 0)
			assert.Equal(t, expectedStart, budget.StartDate)
			assert.Equal(t, expectedEnd, budget.EndDate)

			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.True(t, createCalled)
}

func TestDailyProcessing_RepeatDaily(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Daily Budget",
					Period:    "DAILY",
					Repeat:    true,
					StartDate: now.AddDate(0, 0, -2),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			expectedStart := now.AddDate(0, 0, -2).AddDate(0, 0, 1)
			expectedEnd := now.AddDate(0, 0, -1).AddDate(0, 0, 1)
			assert.Equal(t, expectedStart, budget.StartDate)
			assert.Equal(t, expectedEnd, budget.EndDate)
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
}

func TestDailyProcessing_RepeatWeekly(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Weekly Budget",
					Period:    "WEEKLY",
					Repeat:    true,
					StartDate: now.AddDate(0, 0, -14),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			expectedStart := now.AddDate(0, 0, -14).AddDate(0, 0, 7)
			expectedEnd := now.AddDate(0, 0, -1).AddDate(0, 0, 7)
			assert.Equal(t, expectedStart, budget.StartDate)
			assert.Equal(t, expectedEnd, budget.EndDate)
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
}

func TestDailyProcessing_RepeatYearly(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Yearly Budget",
					Period:    "YEARLY",
					Repeat:    true,
					StartDate: now.AddDate(-1, 0, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			expectedStart := now.AddDate(-1, 0, 0).AddDate(1, 0, 0)
			expectedEnd := now.AddDate(0, 0, -1).AddDate(1, 0, 0)
			assert.Equal(t, expectedStart, budget.StartDate)
			assert.Equal(t, expectedEnd, budget.EndDate)
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
}

func TestDailyProcessing_CustomPeriod(t *testing.T) {
	now := time.Now()
	start := now.AddDate(0, 0, -10)
	end := now.AddDate(0, 0, -1)
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Custom Budget",
					Period:    "custom",
					Repeat:    true,
					StartDate: start,
					EndDate:   end,
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			duration := end.Sub(start)
			assert.Equal(t, end, budget.StartDate)
			assert.Equal(t, end.Add(duration), budget.EndDate)
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
}

func TestDailyProcessing_CopyName(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "My Budget",
					Period:    "MONTHLY",
					Repeat:    true,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.Equal(t, "My Budget (copy)", budget.Name)
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.DailyProcessing()
	assert.NoError(t, err)
}

func TestDailyProcessing_CopyZeroCollected(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:              1,
					UserID:          1,
					Name:            "Budget",
					Period:          "MONTHLY",
					Repeat:          true,
					CollectedAmount: decimal.NewFromInt(500),
					StartDate:       now.AddDate(0, -1, 0),
					EndDate:         now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			assert.True(t, budget.CollectedAmount.IsZero())
			created := *budget
			created.ID = 2
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	_, err := svc.DailyProcessing()
	assert.NoError(t, err)
}

func TestDailyProcessing_CreateError(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Budget",
					Period:    "MONTHLY",
					Repeat:    true,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
}

func TestDailyProcessing_ArchiveError_NoCopy(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Budget",
					Period:    "MONTHLY",
					Repeat:    false,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		ArchiveFunc: func(id int) error {
			return errors.New("archive error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
}

func TestDailyProcessing_ArchiveError_AfterCopy(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Budget",
					Period:    "MONTHLY",
					Repeat:    true,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			created := *budget
			created.ID = 2
			return &created, nil
		},
		ArchiveFunc: func(id int) error {
			return errors.New("archive error")
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
}

func TestDailyProcessing_MultipleBudgets(t *testing.T) {
	now := time.Now()
	budgetRepo := &mockBudgetRepo{
		GetOutdatedBudgetsFunc: func() ([]*models.Budget, error) {
			return []*models.Budget{
				{
					ID:        1,
					UserID:    1,
					Name:      "Budget1",
					Period:    "MONTHLY",
					Repeat:    false,
					StartDate: now.AddDate(0, -1, 0),
					EndDate:   now.AddDate(0, 0, -1),
				},
				{
					ID:        2,
					UserID:    1,
					Name:      "Budget2",
					Period:    "WEEKLY",
					Repeat:    true,
					StartDate: now.AddDate(0, 0, -7),
					EndDate:   now.AddDate(0, 0, -1),
				},
				{
					ID:        3,
					UserID:    2,
					Name:      "Budget3",
					Period:    "DAILY",
					Repeat:    false,
					StartDate: now.AddDate(0, 0, -2),
					EndDate:   now.AddDate(0, 0, -1),
				},
			}, nil
		},
		CreateFunc: func(budget *models.Budget) (*models.Budget, error) {
			created := *budget
			created.ID = 100
			return &created, nil
		},
	}
	svc := newTestService(budgetRepo, &mockCurrencyRepo{})

	result, err := svc.DailyProcessing()
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Processed)
}
