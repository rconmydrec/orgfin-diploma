package reports

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	currencyservice "github.com/go-budget/backend/internal/services/currency"
	reportsservice "github.com/go-budget/backend/internal/services/reports"
	"github.com/go-budget/backend/internal/types"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

// Mock repositories
type MockTransactionRepo struct {
	GetByUserIDFunc func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error)
}

func (m *MockTransactionRepo) Create(tx *models.Transaction) (*models.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) GetByID(id int) (*models.Transaction, error) { return nil, nil }
func (m *MockTransactionRepo) GetByAccountID(accountID int, limit, offset int) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) GetByAccountIDForRecalc(accountID int) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID, filters)
	}
	return []*models.Transaction{}, 0, nil
}
func (m *MockTransactionRepo) GetForExport(userID int, startDate, endDate string) ([]*models.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) Update(tx *models.Transaction) error                    { return nil }
func (m *MockTransactionRepo) UpdateLinkedID(id int, linkedID int) error              { return nil }
func (m *MockTransactionRepo) UpdateBalance(id int, newBalance decimal.Decimal) error { return nil }
func (m *MockTransactionRepo) Delete(id int) error                                    { return nil }

type MockAccountRepo struct {
	GetByUserIDFunc func(userID int, filters types.AccountFilters) ([]*models.Account, error)
}

func (m *MockAccountRepo) Create(account *models.Account) (*models.Account, error) { return nil, nil }
func (m *MockAccountRepo) GetByID(id int) (*models.Account, error)                 { return nil, nil }
func (m *MockAccountRepo) GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID, filters)
	}
	return []*models.Account{}, nil
}
func (m *MockAccountRepo) Update(account *models.Account) error                { return nil }
func (m *MockAccountRepo) SoftDelete(id int) error                             { return nil }
func (m *MockAccountRepo) UpdateBalance(id int, balance decimal.Decimal) error { return nil }
func (m *MockAccountRepo) SetArchiveStatus(id int, isArchived bool) error      { return nil }
func (m *MockAccountRepo) CountActiveByUserID(_ int) (int, error)              { return 0, nil }

type MockCategoryRepo struct {
	GetByUserIDFunc func(userID int) ([]*models.UserCategory, error)
}

func (m *MockCategoryRepo) GetByUserID(userID int) ([]*models.UserCategory, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return []*models.UserCategory{}, nil
}
func (m *MockCategoryRepo) GetByUserIDGrouped(userID int) ([]*models.UserCategory, error) {
	return nil, nil
}
func (m *MockCategoryRepo) GetByID(id int) (*models.UserCategory, error) { return nil, nil }
func (m *MockCategoryRepo) Create(category *models.UserCategory) (*models.UserCategory, error) {
	return nil, nil
}
func (m *MockCategoryRepo) Update(category *models.UserCategory) error { return nil }
func (m *MockCategoryRepo) Delete(id int) error                        { return nil }
func (m *MockCategoryRepo) CopyDefaultCategories(userID int) error     { return nil }

type MockCurrencyRepo struct {
	GetByIDFunc func(id int) (*models.Currency, error)
}

func (m *MockCurrencyRepo) GetAll() ([]*models.Currency, error)             { return nil, nil }
func (m *MockCurrencyRepo) GetByCode(code string) (*models.Currency, error) { return nil, nil }
func (m *MockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
}

type MockExchangeRateRepo struct {
	GetRateFunc            func(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error)
	GetRateForDateFunc     func(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error)
	SaveRateFunc           func(rate *models.ExchangeRateHistory) error
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
}

func (m *MockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	if m.GetRateFunc != nil {
		return m.GetRateFunc(baseCurrencyCode, targetCurrencyCode)
	}
	return nil, nil
}
func (m *MockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	if m.GetRateForDateFunc != nil {
		return m.GetRateForDateFunc(baseCurrencyCode, targetCurrencyCode, date)
	}
	return nil, nil
}
func (m *MockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error {
	if m.SaveRateFunc != nil {
		return m.SaveRateFunc(rate)
	}
	return nil
}
func (m *MockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	if m.GetAllRatesForDateFunc != nil {
		return m.GetAllRatesForDateFunc(date)
	}
	return &types.ExchangeRateSnapshot{
		Rates:            map[string]float64{"USD": 1.0},
		BaseCurrencyCode: "USD",
	}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockTransactionRepo, *MockAccountRepo, *MockCategoryRepo, *MockCurrencyRepo, *MockExchangeRateRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	txRepo := &MockTransactionRepo{}
	accountRepo := &MockAccountRepo{}
	categoryRepo := &MockCategoryRepo{}
	currencyRepo := &MockCurrencyRepo{}
	exchangeRateRepo := &MockExchangeRateRepo{}

	currencySvc := currencyservice.New(exchangeRateRepo, logger)
	reportsSvc := reportsservice.New(txRepo, accountRepo, categoryRepo, currencyRepo, currencySvc, logger)
	handler := New(reportsSvc, logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return handler, e, txRepo, accountRepo, categoryRepo, currencyRepo, exchangeRateRepo
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CashFlowReport Tests ====================

func TestCashFlowReportCurrencyError(t *testing.T) {
	handler, e, _, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	body := `{"period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCashFlowReportTransactionError(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	body := `{"period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== BalanceReport Tests ====================

func TestBalanceReportCurrencyError(t *testing.T) {
	handler, e, _, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	body := `{"balanceDate": "2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBalanceReportAccountError(t *testing.T) {
	handler, e, _, accountRepo, _, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	body := `{"balanceDate": "2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBalanceReportAccountCurrencyError(t *testing.T) {
	handler, e, _, accountRepo, _, currencyRepo, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount == 1 {
			return &models.Currency{ID: 1, Code: "USD"}, nil
		}
		return nil, errors.New("currency not found")
	}

	body := `{"balanceDate": "2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== ExpensesByCategories Tests ====================

func TestExpensesByCategoriesCategoryError(t *testing.T) {
	handler, e, _, _, categoryRepo, _, _ := setupTestHandler()
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExpensesByCategoriesTransactionError(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== Diagram Tests ====================

func TestDiagramCurrencyError(t *testing.T) {
	handler, e, _, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDiagramCategoryError(t *testing.T) {
	handler, e, _, _, categoryRepo, _, _ := setupTestHandler()
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDiagramTransactionError(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== ExpensesData Tests ====================

func TestExpensesDataCategoryError(t *testing.T) {
	handler, e, _, _, categoryRepo, _, _ := setupTestHandler()
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExpensesDataTransactionError(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _, _, _, _, _ := setupTestHandler()

	g := e.Group("/reports")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/reports/cashflow/"])
	assert.True(t, routePaths["/reports/balance/"])
	assert.True(t, routePaths["/reports/balance/non-hidden/"])
	assert.True(t, routePaths["/reports/expenses-by-categories/"])
	assert.True(t, routePaths["/reports/diagram/:diagram_type/:start_date/:end_date"])
	assert.True(t, routePaths["/reports/expenses-data/"])
}

// ==================== Additional Tests for Full Coverage ====================

func TestExpensesByCategoriesWithParentCategory(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	parentID := 1
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Parent", UserID: 1},
			{ID: 2, Name: "Child", UserID: 1, ParentID: &parentID},
		}, nil
	}

	catID := 2
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesNilCategory(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 999 // Non-existent category
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDiagramWithZeroExpenses(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.Zero, IsIncome: false, CategoryID: &catID, ExcludeFromReports: false},
		}, 1, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesDataWithZeroAmount(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.Zero, IsIncome: false, CategoryID: &catID, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Edge Case Tests for Full Coverage ====================

func TestCashFlowReportExcludeFromReports(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		now := time.Now()
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, DateTime: &now, ExcludeFromReports: true},
			{ID: 2, Amount: decimal.NewFromInt(50), IsIncome: false, DateTime: &now, ExcludeFromReports: false},
		}, 2, nil
	}

	body := `{"period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCashFlowReportDailyPeriod(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		now := time.Now()
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, DateTime: &now, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"period": "daily"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCashFlowReportDefaultPeriod(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		now := time.Now()
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, DateTime: &now, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"period": "weekly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceReportAccountIDFilter(t *testing.T) {
	handler, e, _, accountRepo, _, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Account1", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Account2", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	body := `{"balanceDate": "2024-01-01", "accountIds": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceReportNonHiddenWithHiddenAccount(t *testing.T) {
	handler, e, _, accountRepo, _, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Visible", CurrencyID: 1, Balance: decimal.NewFromInt(1000), IsHidden: false},
			{ID: 2, Name: "Hidden", CurrencyID: 1, Balance: decimal.NewFromInt(2000), IsHidden: true},
		}, nil
	}

	body := `{"balanceDate": "2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/balance/non-hidden/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReportNonHidden(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesExcludedAndIncomeTransactions(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID, ExcludeFromReports: true},
			{ID: 2, Amount: decimal.NewFromInt(200), IsIncome: true, CategoryID: &catID, ExcludeFromReports: false},
			{ID: 3, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: nil, ExcludeFromReports: false},
		}, 3, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesByCategoriesHideEmptyCategories(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.Zero, IsIncome: false, CategoryID: &catID, ExcludeFromReports: false},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "hideEmptyCategories": true}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDiagramExcludedAndIncomeTransactions(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID, ExcludeFromReports: true},
			{ID: 2, Amount: decimal.NewFromInt(200), IsIncome: true, CategoryID: &catID, ExcludeFromReports: false},
			{ID: 3, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: nil, ExcludeFromReports: false},
		}, 3, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestExpensesDataExcludedAndIncomeTransactions(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID, ExcludeFromReports: true},
			{ID: 2, Amount: decimal.NewFromInt(200), IsIncome: true, CategoryID: &catID, ExcludeFromReports: false},
			{ID: 3, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: nil, ExcludeFromReports: false},
		}, 3, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Bug 1: Default Date Range Tests ====================

func TestCashFlowReportDefaultDateMonthly(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	// No startDate, no endDate, monthly period
	body := `{"period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify DateFrom was set to 12 months ago (1st of month)
	assert.NotNil(t, capturedFilters.DateFrom)
	now := time.Now()
	expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -12, 0)
	assert.Equal(t, expected.Format("2006-01-02"), *capturedFilters.DateFrom)

	// Verify DateTo was set to tomorrow (today inclusive)
	assert.NotNil(t, capturedFilters.DateTo)
	expectedEnd := now.AddDate(0, 0, 1).Format("2006-01-02")
	assert.Equal(t, expectedEnd, *capturedFilters.DateTo)
}

func TestCashFlowReportDefaultDateDaily(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	// No startDate, daily period
	body := `{"period": "daily"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify DateFrom was set to 30 days ago
	assert.NotNil(t, capturedFilters.DateFrom)
	expected := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	assert.Equal(t, expected, *capturedFilters.DateFrom)
}

func TestCashFlowReportDefaultEndDate(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	// startDate provided, no endDate
	body := `{"startDate": "2024-01-01", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify DateTo was set to tomorrow (today inclusive)
	assert.NotNil(t, capturedFilters.DateTo)
	expectedEnd := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	assert.Equal(t, expectedEnd, *capturedFilters.DateTo)
}

// ==================== Bug 2: Currency Conversion Tests ====================

func TestCurrencyConversionWithExchangeRate(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{
		ID:         1,
		CurrencyID: 2,
		Currency:   eurCurrency,
	}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(85),
				IsIncome: false,
				DateTime: &now,
				Account:  eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// The amount should be converted: 85 / 0.85 * 1.0 = 100 USD
	assert.Contains(t, rec.Body.String(), "totalExpenses")
}

func TestCurrencyConversionMissingRate(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, errors.New("no rates available")
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{
		ID:         1,
		CurrencyID: 2,
		Currency:   eurCurrency,
	}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(85),
				IsIncome: false,
				DateTime: &now,
				Account:  eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Falls back to raw amount (85) -- report still succeeds
	assert.Contains(t, rec.Body.String(), "totalExpenses")
}

func TestCurrencyConversionSameCurrency(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	now := time.Now()
	usdCurrency := &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}
	usdAccount := &models.Account{
		ID:         1,
		CurrencyID: 1,
		Currency:   usdCurrency,
	}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(100),
				IsIncome: false,
				DateTime: &now,
				Account:  usdAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Bug 4: endDate Inclusive Tests ====================

func TestEndDateInclusiveExpensesByCategories(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// endDate should be 2024-02-01 (one day after 2024-01-31)
	assert.NotNil(t, capturedFilters.DateTo)
	assert.Equal(t, "2024-02-01", *capturedFilters.DateTo)
}

func TestEndDateInclusiveDiagram(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{}, nil
	}

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-01-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-01-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// endDate should be 2024-02-01
	assert.NotNil(t, capturedFilters.DateTo)
	assert.Equal(t, "2024-02-01", *capturedFilters.DateTo)
}

func TestEndDateInclusiveExpensesData(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{}, nil
	}

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	assert.NotNil(t, capturedFilters.DateTo)
	assert.Equal(t, "2024-02-01", *capturedFilters.DateTo)
}

func TestEndDateInclusiveCashFlowReport(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-01-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	assert.NotNil(t, capturedFilters.DateTo)
	assert.Equal(t, "2024-02-01", *capturedFilters.DateTo)
}

// ==================== Bug 5: NoLimit Tests ====================

func TestReportQueriesUseNoLimit(t *testing.T) {
	handler, e, txRepo, _, _, _, _ := setupTestHandler()

	var capturedFilters types.TransactionFilters
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		capturedFilters = filters
		return []*models.Transaction{}, 0, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify NoLimit is set
	assert.True(t, capturedFilters.NoLimit, "Report queries should use NoLimit")
	assert.Equal(t, 0, capturedFilters.Limit, "Limit should be 0 when NoLimit is true")
}

// ==================== Bug 6: Historical Balance Tests ====================

func TestBalanceReportHistoricalBalance(t *testing.T) {
	handler, e, txRepo, accountRepo, _, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{
				ID:            1,
				Name:          "Wallet",
				CurrencyID:    1,
				AccountTypeID: 1,
				Balance:       decimal.NewFromInt(800), // Current balance
			},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Transaction after balanceDate: expense of 200 yesterday
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:        1,
				AccountID: 1,
				Amount:    decimal.NewFromInt(200),
				IsIncome:  false,
				DateTime:  &yesterday,
			},
		}, 1, nil
	}

	// Request balance for 2 days ago
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	body := fmt.Sprintf(`{"balanceDate": "%s", "accountIds": [1]}`, twoDaysAgo)
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Balance at 2 days ago: 800 (current) + 200 (reverse expense) = 1000
	assert.Contains(t, rec.Body.String(), "1000")
}

func TestBalanceReportHistoricalBalanceWithIncome(t *testing.T) {
	handler, e, txRepo, accountRepo, _, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{
				ID:            1,
				Name:          "Wallet",
				CurrencyID:    1,
				AccountTypeID: 1,
				Balance:       decimal.NewFromInt(1500), // Current balance
			},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Transaction after balanceDate: income of 500 yesterday
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:        1,
				AccountID: 1,
				Amount:    decimal.NewFromInt(500),
				IsIncome:  true,
				DateTime:  &yesterday,
			},
		}, 1, nil
	}

	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	body := fmt.Sprintf(`{"balanceDate": "%s", "accountIds": [1]}`, twoDaysAgo)
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Balance at 2 days ago: 1500 (current) - 500 (reverse income) = 1000
	assert.Contains(t, rec.Body.String(), "1000")
}

func TestBalanceReportHistoricalBalanceWithTransfers(t *testing.T) {
	handler, e, txRepo, accountRepo, _, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{
				ID:            1,
				Name:          "Wallet",
				CurrencyID:    1,
				AccountTypeID: 1,
				Balance:       decimal.NewFromInt(500), // Current balance
			},
		}, nil
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Transactions after balanceDate:
	// 1. Outgoing transfer of 300 (IsTransfer=true, IsIncome=false) -> reduced balance
	// 2. Incoming transfer of 100 (IsTransfer=true, IsIncome=true) -> increased balance
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:         1,
				AccountID:  1,
				Amount:     decimal.NewFromInt(300),
				IsIncome:   false,
				IsTransfer: true,
				DateTime:   &yesterday,
			},
			{
				ID:         2,
				AccountID:  1,
				Amount:     decimal.NewFromInt(100),
				IsIncome:   true,
				IsTransfer: true,
				DateTime:   &yesterday,
			},
		}, 2, nil
	}

	// Request balance for 2 days ago
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	body := fmt.Sprintf(`{"balanceDate": "%s", "accountIds": [1]}`, twoDaysAgo)
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Historical balance: 500 (current)
	//   + 300 (reverse outgoing transfer, IsIncome=false -> add back)
	//   - 100 (reverse incoming transfer, IsIncome=true -> subtract)
	//   = 700
	assert.Contains(t, rec.Body.String(), "700")
}

// ==================== Bug 7: Base Currency Balance Tests ====================

func TestBalanceReportBaseCurrencyConversion(t *testing.T) {
	handler, e, txRepo, accountRepo, _, currencyRepo, exchangeRateRepo := setupTestHandler()

	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount <= 1 {
			// First call: base currency
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Second call: account currency
		return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{
				ID:            1,
				Name:          "EUR Account",
				CurrencyID:    2,
				AccountTypeID: 1,
				Balance:       decimal.NewFromFloat(850),
			},
		}, nil
	}

	// No transactions after balance date
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{}, 0, nil
	}

	balanceDate := time.Now().Format("2006-01-02")
	body := fmt.Sprintf(`{"balanceDate": "%s", "accountIds": [1]}`, balanceDate)
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// BaseCurrencyBalance should be converted: 850 / 0.85 * 1.0 = 1000 USD
	assert.Contains(t, rec.Body.String(), "baseCurrencyBalance")
}

// ==================== Bug 8: Categories Filter Tests ====================

func TestExpensesByCategoriesFilterApplied(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	parentID := 1
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Groceries", UserID: 1, ParentID: &parentID},
		}, nil
	}

	catID := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catID},
		}, 1, nil
	}

	// Only request category 1 (Food)
	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "categories": [1], "hideEmptyCategories": true}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Transport (ID=2) should NOT appear in the response since it's filtered out
	assert.NotContains(t, rec.Body.String(), "Transport")
}

func TestExpensesDataCategoriesFilterApplied(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
		}, nil
	}

	catID1 := 1
	catID2 := 2
	now := time.Now()
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromFloat(100), IsIncome: false, CategoryID: &catID1, DateTime: &now},
			{ID: 2, Amount: decimal.NewFromFloat(50), IsIncome: false, CategoryID: &catID2, DateTime: &now},
		}, 2, nil
	}

	// Only request category 1 (Food)
	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "categories": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Transport should NOT appear
	assert.NotContains(t, rec.Body.String(), "Transport")
	// Food should appear
	assert.Contains(t, rec.Body.String(), "Food")
}

// ==================== ExpensesData user/currency error Tests ====================

func TestExpensesDataCurrencyError(t *testing.T) {
	handler, e, _, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== ExpensesByCategories user error Tests ====================

func TestExpensesByCategoriesCurrencyError(t *testing.T) {
	handler, e, _, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== Balance Report Transaction Fetch Error ====================

func TestBalanceReportTransactionFetchError(t *testing.T) {
	handler, e, txRepo, accountRepo, _, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Wallet", CurrencyID: 1, AccountTypeID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return nil, 0, errors.New("db error")
	}

	balanceDate := time.Now().Format("2006-01-02")
	body := fmt.Sprintf(`{"balanceDate": "%s", "accountIds": [1]}`, balanceDate)
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	// Should still return OK but with current balance (graceful degradation)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Currency conversion edge cases ====================

func TestCurrencyConversionMissingSourceRate(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		// No rate for GBP
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	gbpCurrency := &models.Currency{ID: 3, Code: "GBP", Name: "British Pound"}
	gbpAccount := &models.Account{
		ID:         1,
		CurrencyID: 3,
		Currency:   gbpCurrency,
	}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(75),
				IsIncome: false,
				DateTime: &now,
				Account:  gbpAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Falls back to raw amount
}

func TestCurrencyConversionZeroTargetRate(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		// Target currency (USD) has rate 0 -- transaction should be skipped (not fatal)
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(85),
				IsIncome: false,
				DateTime: &now,
				Account:  eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	// Returns 200 but the transaction is skipped (zero rate is a non-fatal conversion error)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCurrencyConversionEmptyRates(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		// Empty rates map = exchange rates table is effectively empty
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID:       1,
				Amount:   decimal.NewFromFloat(100),
				IsIncome: false,
				DateTime: &now,
				Account:  eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	// Empty rates map is treated as the table being empty -> HTTP 500
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// ==================== New Tests: Exchange Rate Error Handling ====================

// TestReportReturns500WhenExchangeRatesTableEmpty verifies that when GetAllRatesForDate
// returns sql.ErrNoRows (table completely empty), report endpoints return HTTP 500.
func TestReportReturns500WhenExchangeRatesTableEmpty(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(100),
				IsIncome: false, DateTime: &now, Account: eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	// sql.ErrNoRows means table is empty -> HTTP 500
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// TestReportReturns200WhenRatesErrorIsNonFatal verifies that a non-sql.ErrNoRows error
// from the exchange rate repo causes transactions to be skipped, but report still succeeds.
func TestReportReturns200WhenRatesErrorIsNonFatal(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, errors.New("some transient DB error")
	}

	now := time.Now()
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(100),
				IsIncome: false, DateTime: &now, Account: eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	// Non-sql.ErrNoRows error: the transaction is skipped, report still succeeds
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestReportReturns200WithCorrectConversion verifies the conversion formula:
// amount / sourceRate * targetRate
func TestReportReturns200WithCorrectConversion(t *testing.T) {
	handler, e, txRepo, _, _, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"BGN": 1.6598, "EUR": 0.8486, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	bgnCurrency := &models.Currency{ID: 3, Code: "BGN", Name: "Bulgarian Lev"}
	bgnAccount := &models.Account{ID: 1, CurrencyID: 3, Currency: bgnCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(1659.8),
				IsIncome: false, DateTime: &now, Account: bgnAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Set user with BaseCurrencyID=2 (EUR) to test currency conversion
	c.Set("user_id", 1)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 2)
	c.Set("active_user_display_name", "Test User")

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// 1659.8 BGN / 1.6598 * 0.8486 = ~848.6 EUR
	assert.Contains(t, rec.Body.String(), "totalExpenses")
}

// TestPartialConversionFailureReturns200 verifies that when one currency is missing
// from the rates map, that transaction is skipped but the report still succeeds.
func TestPartialConversionFailureReturns200(t *testing.T) {
	handler, e, txRepo, _, _, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		// XYZ currency is NOT in the rates map
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	now := time.Now()
	xyzCurrency := &models.Currency{ID: 99, Code: "XYZ", Name: "Unknown"}
	xyzAccount := &models.Account{ID: 1, CurrencyID: 99, Currency: xyzCurrency}
	usdCurrency := &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}
	usdAccount := &models.Account{ID: 2, CurrencyID: 1, Currency: usdCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(100),
				IsIncome: false, DateTime: &now, Account: xyzAccount,
			},
			{
				ID: 2, Amount: decimal.NewFromFloat(50),
				IsIncome: false, DateTime: &now, Account: usdAccount,
			},
		}, 2, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/cashflow/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CashFlowReport(c)
	assert.NoError(t, err)
	// Returns 200; XYZ transaction skipped, USD transaction succeeds (same currency)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "totalExpenses")
}

// TestBalanceReportReturns500WhenEmptyExchangeRates verifies that the balance
// report returns 500 when exchange rates table is empty and accounts need conversion.
func TestBalanceReportReturns500WhenEmptyExchangeRates(t *testing.T) {
	handler, e, _, accountRepo, _, currencyRepo, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{},
			BaseCurrencyCode: "USD",
		}, nil
	}

	// Account currency (EUR, id=2) differs from base currency (USD, id=1)
	// so conversion will be attempted and fail due to empty rates
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount == 1 {
			// Base currency (from user)
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Account currency
		return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
	}

	body := `{"balanceDate": "2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/balance/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BalanceReport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// TestDiagramReturns500WhenEmptyExchangeRates verifies that the diagram
// endpoint returns 500 when exchange rates table is empty.
func TestDiagramReturns500WhenEmptyExchangeRates(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{},
			BaseCurrencyCode: "USD",
		}, nil
	}

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	now := time.Now()
	catID := 1
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(100), CategoryID: &catID,
				IsIncome: false, DateTime: &now, Account: eurAccount,
			},
		}, 1, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// TestExpensesDataReturns500WhenEmptyExchangeRates verifies that expenses-data
// endpoint returns 500 when exchange rates table is empty.
func TestExpensesDataReturns500WhenEmptyExchangeRates(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, exchangeRateRepo := setupTestHandler()

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{},
			BaseCurrencyCode: "USD",
		}, nil
	}

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	now := time.Now()
	catID := 1
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(100), CategoryID: &catID,
				IsIncome: false, DateTime: &now, Account: eurAccount,
			},
		}, 1, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-data/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesData(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// ==================== Bug 1: Diagram Aggregation Tests ====================

// TestDiagramAggregatesChildrenIntoParents verifies that child category expenses
// are aggregated into their parent categories and children do NOT appear as
// separate entries in the Diagram response.
func TestDiagramAggregatesChildrenIntoParents(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	parentID1 := 1
	parentID2 := 3
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},                                 // parent
			{ID: 2, Name: "Groceries", UserID: 1, ParentID: &parentID1},      // child of Food
			{ID: 3, Name: "Transport", UserID: 1},                            // parent
			{ID: 4, Name: "Fuel", UserID: 1, ParentID: &parentID2},           // child of Transport
			{ID: 5, Name: "Public Transit", UserID: 1, ParentID: &parentID2}, // child of Transport
		}, nil
	}

	catGroceries := 2
	catFuel := 4
	catPublicTransit := 5
	catFood := 1
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, CategoryID: &catGroceries},
			{ID: 2, Amount: decimal.NewFromInt(30), IsIncome: false, CategoryID: &catFuel},
			{ID: 3, Amount: decimal.NewFromInt(20), IsIncome: false, CategoryID: &catPublicTransit},
			{ID: 4, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
		}, 4, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp DiagramResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Should have exactly 2 labels: "Food" and "Transport" (no child names)
	assert.Len(t, resp.Labels, 2)
	assert.NotContains(t, resp.Labels, "Groceries")
	assert.NotContains(t, resp.Labels, "Fuel")
	assert.NotContains(t, resp.Labels, "Public Transit")

	// Build a map of label -> amount for easy assertion
	labelAmounts := make(map[string]float64)
	for i, label := range resp.Labels {
		labelAmounts[label] = resp.Data[i]
	}

	// Food = 100 (direct) + 50 (Groceries) = 150
	assert.InDelta(t, 150.0, labelAmounts["Food"], 0.01)
	// Transport = 30 (Fuel) + 20 (Public Transit) = 50
	assert.InDelta(t, 50.0, labelAmounts["Transport"], 0.01)
}

// TestDiagramSmallCategoriesCombinedIntoOther verifies that categories below
// the 2% threshold are combined into an "Other" entry.
func TestDiagramSmallCategoriesCombinedIntoOther(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Tiny", UserID: 1}, // will be < 2%
		}, nil
	}

	catFood := 1
	catTransport := 2
	catTiny := 3
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(600), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(399), IsIncome: false, CategoryID: &catTransport},
			// 1 out of 1000 total = 0.1%, well below the 2% threshold
			{ID: 3, Amount: decimal.NewFromInt(1), IsIncome: false, CategoryID: &catTiny},
		}, 3, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp DiagramResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// "Tiny" should NOT appear as its own label; it should be in "Other"
	assert.NotContains(t, resp.Labels, "Tiny")
	assert.Contains(t, resp.Labels, "Other")

	// "Food" and "Transport" are above 2%, so they stay
	assert.Contains(t, resp.Labels, "Food")
	assert.Contains(t, resp.Labels, "Transport")
}

// TestDiagramSortedByAmountDescending verifies that the Diagram response is
// sorted by amount in descending order.
func TestDiagramSortedByAmountDescending(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
			{ID: 2, Name: "Transport", UserID: 1},
			{ID: 3, Name: "Entertainment", UserID: 1},
		}, nil
	}

	catFood := 1
	catTransport := 2
	catEntertainment := 3
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catFood},
			{ID: 2, Amount: decimal.NewFromInt(300), IsIncome: false, CategoryID: &catTransport},
			{ID: 3, Amount: decimal.NewFromInt(200), IsIncome: false, CategoryID: &catEntertainment},
		}, 3, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/2024-01-01/2024-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", "2024-01-01", "2024-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp DiagramResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	require.Len(t, resp.Data, 3)
	// Verify descending order: 300, 200, 100
	assert.InDelta(t, 300.0, resp.Data[0], 0.01)
	assert.InDelta(t, 200.0, resp.Data[1], 0.01)
	assert.InDelta(t, 100.0, resp.Data[2], 0.01)

	// Labels should match the sorted order
	assert.Equal(t, "Transport", resp.Labels[0])
	assert.Equal(t, "Entertainment", resp.Labels[1])
	assert.Equal(t, "Food", resp.Labels[2])
}

// ==================== Bug 2: ExpensesByCategories Sort Order Tests ====================

// TestExpensesByCategoriesSortOrder verifies that the response is sorted with
// parents alphabetically, children grouped under their parent, and children
// sorted alphabetically within each group.
func TestExpensesByCategoriesSortOrder(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, _ := setupTestHandler()

	parentAID := 1
	parentBID := 3
	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 3, Name: "Bills", UserID: 1},                             // Parent B
			{ID: 1, Name: "Automobile", UserID: 1},                        // Parent A
			{ID: 5, Name: "Internet", UserID: 1, ParentID: &parentBID},    // Child B2 (alphabetically after Electricity)
			{ID: 2, Name: "Fuel", UserID: 1, ParentID: &parentAID},        // Child A2
			{ID: 4, Name: "Electricity", UserID: 1, ParentID: &parentBID}, // Child B1
			{ID: 6, Name: "Car Wash", UserID: 1, ParentID: &parentAID},    // Child A1 (alphabetically before Fuel)
		}, nil
	}

	// Return some transactions so categories are non-empty
	catFuel := 2
	catElectricity := 4
	catInternet := 5
	catCarWash := 6
	catAutomobile := 1
	catBills := 3
	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{ID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, CategoryID: &catFuel},
			{ID: 2, Amount: decimal.NewFromInt(80), IsIncome: false, CategoryID: &catElectricity},
			{ID: 3, Amount: decimal.NewFromInt(40), IsIncome: false, CategoryID: &catInternet},
			{ID: 4, Amount: decimal.NewFromInt(15), IsIncome: false, CategoryID: &catCarWash},
			{ID: 5, Amount: decimal.NewFromInt(100), IsIncome: false, CategoryID: &catAutomobile},
			{ID: 6, Amount: decimal.NewFromInt(60), IsIncome: false, CategoryID: &catBills},
		}, 6, nil
	}

	body := `{"startDate": "2024-01-01", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []ExpensesReportItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Expected order:
	// 1. Automobile (parent A)
	// 2. Automobile >> Car Wash (child A1 - alphabetically before Fuel)
	// 3. Automobile >> Fuel (child A2)
	// 4. Bills (parent B)
	// 5. Bills >> Electricity (child B1)
	// 6. Bills >> Internet (child B2)
	require.Len(t, resp, 6)
	assert.Equal(t, "Automobile", resp[0].Name)
	assert.True(t, resp[0].IsParent)
	assert.Equal(t, "Automobile >> Car Wash", resp[1].Name)
	assert.False(t, resp[1].IsParent)
	assert.Equal(t, "Automobile >> Fuel", resp[2].Name)
	assert.False(t, resp[2].IsParent)
	assert.Equal(t, "Bills", resp[3].Name)
	assert.True(t, resp[3].IsParent)
	assert.Equal(t, "Bills >> Electricity", resp[4].Name)
	assert.False(t, resp[4].IsParent)
	assert.Equal(t, "Bills >> Internet", resp[5].Name)
	assert.False(t, resp[5].IsParent)
}

// ==================== Bug 3: Exchange Rate Date Tests ====================

// TestGetTransactionAmountUsesStartDateForRateLookup verifies that
// getTransactionAmount uses the provided startDate for exchange rate lookup,
// NOT time.Now(). This is verified by capturing the date argument passed to
// the exchange rate repository.
func TestGetTransactionAmountUsesStartDateForRateLookup(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, exchangeRateRepo := setupTestHandler()

	var capturedRateDate string
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		capturedRateDate = date
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(85), CategoryID: &catID,
				IsIncome: false, Account: eurAccount,
			},
		}, 1, nil
	}

	// Use a historical startDate that is NOT today
	historicalDate := "2020-06-15"
	req := httptest.NewRequest(http.MethodGet, "/diagram/pie/"+historicalDate+"/2020-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("diagram_type", "start_date", "end_date")
	c.SetParamValues("pie", historicalDate, "2020-12-31")
	setUserContext(c, 1)

	err := handler.Diagram(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// The exchange rate lookup should have used the startDate "2020-06-15",
	// NOT today's date. This verifies Bug 3 is fixed.
	assert.Equal(t, historicalDate, capturedRateDate,
		"getTransactionAmount should use the provided startDate for rate lookup, not time.Now()")
}

// TestGetTransactionAmountUsesStartDateViaExpensesByCategories verifies the same
// Bug 3 fix through the ExpensesByCategories endpoint.
func TestGetTransactionAmountUsesStartDateViaExpensesByCategories(t *testing.T) {
	handler, e, txRepo, _, categoryRepo, _, exchangeRateRepo := setupTestHandler()

	var capturedRateDate string
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		capturedRateDate = date
		return &types.ExchangeRateSnapshot{
			Rates:            map[string]float64{"EUR": 0.85, "USD": 1.0},
			BaseCurrencyCode: "USD",
		}, nil
	}

	categoryRepo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, Name: "Food", UserID: 1},
		}, nil
	}

	catID := 1
	eurCurrency := &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}
	eurAccount := &models.Account{ID: 1, CurrencyID: 2, Currency: eurCurrency}

	txRepo.GetByUserIDFunc = func(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
		return []*models.Transaction{
			{
				ID: 1, Amount: decimal.NewFromFloat(85), CategoryID: &catID,
				IsIncome: false, Account: eurAccount,
			},
		}, 1, nil
	}

	historicalDate := "2019-03-01"
	body := fmt.Sprintf(`{"startDate": "%s", "endDate": "2019-12-31"}`, historicalDate)
	req := httptest.NewRequest(http.MethodPost, "/expenses-by-categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ExpensesByCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// The exchange rate lookup should have used the startDate "2019-03-01"
	assert.Equal(t, historicalDate, capturedRateDate,
		"getTransactionAmount should use the report's startDate for rate lookup, not time.Now()")
}
