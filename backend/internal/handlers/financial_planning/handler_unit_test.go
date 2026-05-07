package financial_planning

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	currencyservice "github.com/go-budget/backend/internal/services/currency"
	financialplanningservice "github.com/go-budget/backend/internal/services/financial_planning"
	plannedtransactionsservice "github.com/go-budget/backend/internal/services/planned_transactions"
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

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock planned transactions service
type MockPlannedTxSvc struct {
	GetUpcomingFunc func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error)
}

func (m *MockPlannedTxSvc) GetUpcoming(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
	if m.GetUpcomingFunc != nil {
		return m.GetUpcomingFunc(userID, days, includeInactive)
	}
	return nil, nil
}

type MockAccountRepo struct {
	GetByUserIDFunc func(userID int, filters types.AccountFilters) ([]*models.Account, error)
}

func (m *MockAccountRepo) Create(account *models.Account) (*models.Account, error) { return nil, nil }
func (m *MockAccountRepo) GetByID(id int) (*models.Account, error)                 { return nil, nil }
func (m *MockAccountRepo) GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID, filters)
	}
	return nil, nil
}
func (m *MockAccountRepo) Update(account *models.Account) error                { return nil }
func (m *MockAccountRepo) SoftDelete(id int) error                             { return nil }
func (m *MockAccountRepo) UpdateBalance(id int, balance decimal.Decimal) error { return nil }
func (m *MockAccountRepo) SetArchiveStatus(id int, isArchived bool) error      { return nil }
func (m *MockAccountRepo) CountActiveByUserID(_ int) (int, error)              { return 0, nil }

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
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
}

func (m *MockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *MockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *MockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error { return nil }
func (m *MockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	if m.GetAllRatesForDateFunc != nil {
		return m.GetAllRatesForDateFunc(date)
	}
	// Default: return rates where USD=1.0 (same-currency conversion is a no-op)
	return &types.ExchangeRateSnapshot{
		ID:               1,
		Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
		ActualDate:       date,
		BaseCurrencyCode: "USD",
	}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockPlannedTxSvc, *MockAccountRepo, *MockCurrencyRepo, *MockExchangeRateRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	plannedTxSvc := &MockPlannedTxSvc{}
	accountRepo := &MockAccountRepo{}
	currencyRepo := &MockCurrencyRepo{}
	exchangeRateRepo := &MockExchangeRateRepo{}

	currencySvc := currencyservice.New(exchangeRateRepo, logger)
	svc := financialplanningservice.New(plannedTxSvc, accountRepo, currencyRepo, currencySvc, logger)
	handler := New(svc, logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CalculateFutureBalance Tests ====================

func TestCalculateFutureBalanceBindError(t *testing.T) {
	handler, e, _, _, _, _ := setupTestHandler()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCalculateFutureBalanceValidationError(t *testing.T) {
	handler, e, _, _, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCalculateFutureBalanceCurrencyError(t *testing.T) {
	handler, e, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCalculateFutureBalanceAccountError(t *testing.T) {
	handler, e, _, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCalculateFutureBalancePlannedTxError(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return nil, errors.New("db error")
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCalculateFutureBalancePastDate(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	// Use past date
	targetDate := time.Now().AddDate(-1, 0, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== GetBalanceProjection Tests ====================

func TestGetBalanceProjectionBindError(t *testing.T) {
	handler, e, _, _, _, _ := setupTestHandler()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetBalanceProjectionValidationError(t *testing.T) {
	handler, e, _, _, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetBalanceProjectionCurrencyError(t *testing.T) {
	handler, e, _, _, currencyRepo, _ := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, errors.New("db error")
	}

	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetBalanceProjectionAccountError(t *testing.T) {
	handler, e, _, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return nil, errors.New("db error")
	}

	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetBalanceProjectionEndDateBeforeStart(t *testing.T) {
	handler, e, _, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}

	startDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	endDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	body := `{"startDate": "` + startDate + `", "endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "End date must be after start date")
}

func TestGetBalanceProjectionPlannedTxError(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return nil, errors.New("db error")
	}

	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetBalanceProjectionWeeklyPeriod(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `", "period": "weekly"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetBalanceProjectionMonthlyPeriod(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 3, 0).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `", "period": "monthly"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _, _, _, _ := setupTestHandler()

	g := e.Group("/financial-planning")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/financial-planning/future-balance"])
	assert.True(t, routePaths["/financial-planning/projection"])
}

func TestCalculateFutureBalanceWithTransactions(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestHandler()

	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// Total current = 1000, planned income = 100, planned expenses = 50
	// Projected = 1000 + 100 - 50 = 1050
	assert.Equal(t, float64(1050), resp.TotalProjectedBalance)
	assert.Equal(t, float64(100), resp.TotalPlannedIncome)
	assert.Equal(t, float64(50), resp.TotalPlannedExpenses)
	assert.Equal(t, 1, resp.IncomeCount)
	assert.Equal(t, 1, resp.ExpensesCount)
	// Per-account projections have zero planned income/expenses
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, float64(0), resp.Accounts[0].TotalPlannedIncome)
	assert.Equal(t, float64(0), resp.Accounts[0].TotalPlannedExpenses)
	assert.Equal(t, float64(1000), resp.Accounts[0].ProjectedBalance)
}

func TestCalculateFutureBalanceWithAccountFilter(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test1", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Test2", CurrencyID: 1, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `", "accountIds": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCalculateFutureBalanceCurrencyErrorInLoop(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestHandler()

	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount == 1 {
			// First call is for base currency
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Second call is for account currency - return error
		return nil, errors.New("currency not found")
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetBalanceProjectionWithAccountFilter(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test1", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Test2", CurrencyID: 1, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `", "accountIds": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetBalanceProjectionWithTransactions(t *testing.T) {
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	occDate := time.Now().AddDate(0, 0, 1)
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: occDate},
		}, nil
	}

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Bug 1 Fix: Account-Specific Planned Transactions ====================

func TestCalculateFutureBalanceMultipleAccounts(t *testing.T) {
	// All planned transactions contribute to global totals only.
	// Per-account projected balances remain unchanged (no planned tx distribution).
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
			{ID: 3, Name: "Credit", CurrencyID: 1, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// All 3 accounts present, each with zero planned income/expenses
	require.Len(t, resp.Accounts, 3)
	for _, acc := range resp.Accounts {
		assert.Equal(t, float64(0), acc.TotalPlannedIncome)
		assert.Equal(t, float64(0), acc.TotalPlannedExpenses)
		assert.Equal(t, acc.CurrentBalance, acc.ProjectedBalance)
	}

	// Global totals: income = 100 + 200 = 300, expense = 50
	// Total current = 1000 + 2000 + 500 = 3500
	// Total projected = 3500 + 300 - 50 = 3750
	assert.Equal(t, float64(3500), resp.TotalCurrentBalance)
	assert.Equal(t, float64(300), resp.TotalPlannedIncome)
	assert.Equal(t, float64(50), resp.TotalPlannedExpenses)
	assert.Equal(t, float64(3750), resp.TotalProjectedBalance)

	assert.Equal(t, 2, resp.IncomeCount)
	assert.Equal(t, 1, resp.ExpensesCount)
}

func TestCalculateFutureBalanceGlobalTotals(t *testing.T) {
	// All planned transactions contribute to global totals only.
	// Per-account projections remain unchanged.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(300), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: false, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Per-account projections: balance unchanged (zero planned per-account)
	require.Len(t, resp.Accounts, 2)
	assert.Equal(t, float64(1000), resp.Accounts[0].ProjectedBalance)
	assert.Equal(t, float64(0), resp.Accounts[0].TotalPlannedIncome)
	assert.Equal(t, float64(0), resp.Accounts[0].TotalPlannedExpenses)
	assert.Equal(t, float64(2000), resp.Accounts[1].ProjectedBalance)
	assert.Equal(t, float64(0), resp.Accounts[1].TotalPlannedIncome)
	assert.Equal(t, float64(0), resp.Accounts[1].TotalPlannedExpenses)

	// Global totals
	// Total current = 1000 + 2000 = 3000
	// Income = 300, expense = 100
	// Total projected = 3000 + 300 - 100 = 3200
	assert.Equal(t, float64(3000), resp.TotalCurrentBalance)
	assert.Equal(t, float64(300), resp.TotalPlannedIncome)
	assert.Equal(t, float64(100), resp.TotalPlannedExpenses)
	assert.Equal(t, float64(3200), resp.TotalProjectedBalance)

	assert.Equal(t, 1, resp.IncomeCount)
	assert.Equal(t, 1, resp.ExpensesCount)
}

func TestCalculateFutureBalanceMultiplePlannedTx(t *testing.T) {
	// Multiple planned transactions all contribute to global totals.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 4, CurrencyID: 1, Amount: decimal.NewFromInt(75), IsIncome: false, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	require.Len(t, resp.Accounts, 2)

	// Per-account: balance unchanged
	assert.Equal(t, float64(1000), resp.Accounts[0].ProjectedBalance)
	assert.Equal(t, float64(2000), resp.Accounts[1].ProjectedBalance)

	// Global totals: income = 100 + 200 = 300, expense = 50 + 75 = 125
	// Total current = 3000, projected = 3000 + 300 - 125 = 3175
	assert.Equal(t, float64(3000), resp.TotalCurrentBalance)
	assert.Equal(t, float64(300), resp.TotalPlannedIncome)
	assert.Equal(t, float64(125), resp.TotalPlannedExpenses)
	assert.Equal(t, float64(3175), resp.TotalProjectedBalance)

	assert.Equal(t, 2, resp.IncomeCount)
	assert.Equal(t, 2, resp.ExpensesCount)
}

func TestCalculateFutureBalanceWithAccountFilter2(t *testing.T) {
	// When account filter is applied, only filtered accounts appear in projections.
	// Planned transactions still contribute to global totals (not per-account).
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Checking", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
			{ID: 2, Name: "Savings", CurrencyID: 1, Balance: decimal.NewFromInt(2000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(999), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `", "accountIds": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Only account 1 should be in projections
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, 1, resp.Accounts[0].AccountID)
	// Per-account projected balance is unchanged (no per-account planned tx)
	assert.Equal(t, float64(1000), resp.Accounts[0].ProjectedBalance)

	// Global totals include all planned transactions
	assert.Equal(t, float64(1099), resp.TotalPlannedIncome)
	assert.Equal(t, float64(0), resp.TotalPlannedExpenses)
	assert.Equal(t, 2, resp.IncomeCount)
	assert.Equal(t, 0, resp.ExpensesCount)
}

// ==================== Currency Conversion Tests ====================

func TestCalculateFutureBalanceMultiCurrencyConversion(t *testing.T) {
	// Test that account balances and planned tx amounts are converted to base currency.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	// Base currency is USD (id=1), account has EUR (id=2), planned tx has GBP (id=3)
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
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

	// Exchange rates: USD=1.0, EUR=0.85, GBP=0.73
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(850)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 3, Amount: decimal.NewFromInt(73), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// EUR balance 850 / 0.85 * 1.0 = 1000 USD
	assert.Equal(t, float64(1000), resp.TotalCurrentBalance)

	// GBP income 73 / 0.73 * 1.0 = 100 USD
	assert.Equal(t, float64(100), resp.TotalPlannedIncome)

	// Projected = 1000 + 100 = 1100
	assert.Equal(t, float64(1100), resp.TotalProjectedBalance)

	// Per-account balance stays in its own currency
	require.Len(t, resp.Accounts, 1)
	assert.Equal(t, float64(850), resp.Accounts[0].CurrentBalance)
	assert.Equal(t, "EUR", resp.Accounts[0].CurrencyCode)
}

func TestCalculateFutureBalanceExchangeRateEmpty(t *testing.T) {
	// When exchange rates table is empty, return 500
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(850)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

func TestCalculateFutureBalancePlannedTxCurrencyError(t *testing.T) {
	// When a planned tx currency lookup fails, it should be skipped
	handler, e, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestHandler()

	currencyCallCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		currencyCallCount++
		if id == 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Currency ID 999 not found
		return nil, errors.New("currency not found")
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 999, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Only the USD tx (50) should be counted, the one with bad currency is skipped
	assert.Equal(t, float64(50), resp.TotalPlannedIncome)
	assert.Equal(t, 1, resp.IncomeCount)
}

func TestGetBalanceProjectionMultiCurrencyConversion(t *testing.T) {
	// Test projection with multi-currency conversion
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.5},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	// Transaction on day 2 in EUR
	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	occDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// EUR 500 / 0.5 * 1.0 = 1000 USD initial balance
	// EUR 100 / 0.5 * 1.0 = 200 USD income on day 3
	// 5 daily points: Mar 1-5
	require.Len(t, resp.ProjectionPoints, 5)

	// Day 1 (Mar 1): balance = 1000 (no tx)
	assert.Equal(t, float64(1000), resp.ProjectionPoints[0].Balance)
	// Day 3 (Mar 3): +200 income
	assert.Equal(t, float64(200), resp.ProjectionPoints[2].Income)
	assert.Equal(t, float64(1200), resp.ProjectionPoints[2].Balance)
	// Day 5: still 1200
	assert.Equal(t, float64(1200), resp.ProjectionPoints[4].Balance)
}

func TestGetBalanceProjectionExchangeRateEmpty(t *testing.T) {
	// When exchange rates table is empty and accounts need conversion, return 500
	handler, e, _, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// generateDatePoints tests have been moved to services/financial_planning/service_test.go
// since the function is now private in the service package.

// ==================== Projection period-based grouping tests ====================

func TestGetBalanceProjectionPerPeriodGrouping(t *testing.T) {
	// Verify that transactions within a period are summed correctly
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	// Two transactions on day 2, one on day 4
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(30), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	require.Len(t, resp.ProjectionPoints, 5)

	// Day 1: no tx, balance = 1000
	assert.Equal(t, float64(0), resp.ProjectionPoints[0].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[0].Expenses)
	assert.Equal(t, float64(1000), resp.ProjectionPoints[0].Balance)

	// Day 2: +100 income, -30 expense, balance = 1000 + 100 - 30 = 1070
	assert.Equal(t, float64(100), resp.ProjectionPoints[1].Income)
	assert.Equal(t, float64(30), resp.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1070), resp.ProjectionPoints[1].Balance)

	// Day 3: no tx, balance = 1070
	assert.Equal(t, float64(0), resp.ProjectionPoints[2].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[2].Expenses)
	assert.Equal(t, float64(1070), resp.ProjectionPoints[2].Balance)

	// Day 4: -50, balance = 1020
	assert.Equal(t, float64(0), resp.ProjectionPoints[3].Income)
	assert.Equal(t, float64(50), resp.ProjectionPoints[3].Expenses)
	assert.Equal(t, float64(1020), resp.ProjectionPoints[3].Balance)

	// Day 5: no tx, balance = 1020
	assert.Equal(t, float64(1020), resp.ProjectionPoints[4].Balance)
}

func TestGetBalanceProjectionWeeklyGrouping(t *testing.T) {
	// Test that weekly periods group transactions correctly
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	// Wednesday start, 3-week range
	startDate := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // Wednesday
	endDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	// Transactions spread across weeks
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// Week 1 (Mar 4-15): two transactions
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
			// Week 2 (Mar 16-22): one transaction
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `", "period": "weekly"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Points: Mar 4, Mar 15, Mar 22, Mar 25
	require.Len(t, resp.ProjectionPoints, 4)
	assert.Equal(t, "2026-03-04", resp.ProjectionPoints[0].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-15", resp.ProjectionPoints[1].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-22", resp.ProjectionPoints[2].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-25", resp.ProjectionPoints[3].Date.Format("2006-01-02"))
}

func TestGetBalanceProjectionCurrencyErrorInAccountLoop(t *testing.T) {
	// When currency lookup fails for an account in projection, raw balance should be used
	handler, e, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestHandler()

	callCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		callCount++
		if callCount <= 1 {
			// First call for base currency
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Account currency lookup fails
		return nil, errors.New("currency not found")
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 2, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 0, 3).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Currency conversion tests (via shared currency service) ====================
// The convertToBaseCurrency and RateCache logic has been extracted to
// services/currency/rate_cache.go and is thoroughly tested there.
// The following tests verify the handler correctly integrates with the service.

func TestCalculateFutureBalancePlannedTxExchangeRateEmpty(t *testing.T) {
	// When exchange rates are empty and planned tx needs conversion, return 500
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		// All accounts in base currency (no conversion needed for accounts)
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

func TestGetBalanceProjectionPlannedTxExchangeRateEmpty(t *testing.T) {
	// When exchange rates are empty and projection needs conversion, return 500.
	// Use a non-base-currency account so the balance conversion triggers the error
	// before even reaching planned tx conversion.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// All exchange rate calls fail (empty table)
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			// EUR account requires conversion, which will fail
			{ID: 1, Name: "EUR Account", CurrencyID: 2, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

// ==================== QA Issue 1: Planned tx skipped on non-fatal exchange rate error in CalculateFutureBalance ====================

func TestCalculateFutureBalancePlannedTxSkippedOnMissingRate(t *testing.T) {
	// When a planned tx's currency rate is missing from the rates map (but the
	// exchange_rates table is NOT empty), the tx should be skipped and the
	// handler should continue processing other transactions.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	// Base currency USD (id=1), valid planned tx in USD, invalid one in XYZ (id=99)
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 99:
			return &models.Currency{ID: 99, Code: "XYZ", Name: "Unknown Currency"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// Rates include USD but NOT XYZ
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// Valid USD tx (same as base currency, no conversion needed)
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Now()},
			// XYZ tx: rate is missing from the map, should be skipped
			{PlannedTransactionID: 2, CurrencyID: 99, Amount: decimal.NewFromInt(500), IsIncome: true, OccurrenceDate: time.Now()},
		}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Only the valid USD tx (200) should be counted; the XYZ one is skipped
	assert.Equal(t, float64(200), resp.TotalPlannedIncome)
	assert.Equal(t, 1, resp.IncomeCount)
	// Total current = 1000, projected = 1000 + 200 = 1200
	assert.Equal(t, float64(1000), resp.TotalCurrentBalance)
	assert.Equal(t, float64(1200), resp.TotalProjectedBalance)
}

// ==================== QA Issue 2: Account balance falls back to raw value in CalculateFutureBalance ====================

func TestCalculateFutureBalanceAccountFallbackToRawBalance(t *testing.T) {
	// When an account's currency exists in currencyRepo but its rate is NOT in
	// the exchange rates map (table is not empty), the handler should fall back
	// to using the raw balance for the total.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 4:
			return &models.Currency{ID: 4, Code: "CHF", Name: "Swiss Franc"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// Rates include USD, EUR, GBP but NOT CHF
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "CHF Account", CurrencyID: 4, Balance: decimal.NewFromInt(750)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp FutureBalanceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// CHF rate is missing, so the handler falls back to raw balance (750)
	assert.Equal(t, float64(750), resp.TotalCurrentBalance)
	assert.Equal(t, float64(750), resp.TotalProjectedBalance)
}

// ==================== QA Issue 3: Account balance falls back to raw value in GetBalanceProjection ====================

func TestGetBalanceProjectionAccountFallbackToRawBalance(t *testing.T) {
	// Same as issue 2 but for the GetBalanceProjection endpoint.
	// When an account's currency is found in currencyRepo but the rate is not
	// in the exchange rates map, the raw balance should be used.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 4:
			return &models.Currency{ID: 4, Code: "CHF", Name: "Swiss Franc"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// Rates include USD, EUR, GBP but NOT CHF
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "CHF Account", CurrencyID: 4, Balance: decimal.NewFromInt(500)},
		}, nil
	}

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// CHF rate is missing, raw balance (500) should be used as the initial balance.
	// No planned transactions, so all projection points should have balance = 500.
	require.Len(t, resp.ProjectionPoints, 3)
	assert.Equal(t, float64(500), resp.ProjectionPoints[0].Balance)
	assert.Equal(t, float64(500), resp.ProjectionPoints[1].Balance)
	assert.Equal(t, float64(500), resp.ProjectionPoints[2].Balance)
}

// ==================== QA Issue 4: Planned tx conversion errors inside period loop of GetBalanceProjection ====================

func TestGetBalanceProjectionPlannedTxCurrencyLookupFailure(t *testing.T) {
	// Issue 4a: Currency lookup failure for a planned tx inside the period loop.
	// Account is in USD (same as base, no conversion needed for accounts),
	// but a planned tx has a currency ID that makes currencyRepo.GetByID fail.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, _ := setupTestHandler()

	currencyCallCount := 0
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		currencyCallCount++
		if id == 1 {
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		}
		// Currency ID 88 lookup fails
		return nil, errors.New("currency not found")
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			// USD account: same as base, no conversion needed
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	occDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// Valid USD tx
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
			// Invalid currency lookup (id=88 will fail)
			{PlannedTransactionID: 2, CurrencyID: 88, Amount: decimal.NewFromInt(300), IsIncome: true, OccurrenceDate: occDate},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// 3 daily points: Mar 1, 2, 3
	require.Len(t, resp.ProjectionPoints, 3)

	// Day 1: no tx, balance = 1000
	assert.Equal(t, float64(1000), resp.ProjectionPoints[0].Balance)
	// Day 2: only the valid USD tx (100) is counted; the bad-currency one is skipped
	assert.Equal(t, float64(100), resp.ProjectionPoints[1].Income)
	assert.Equal(t, float64(1100), resp.ProjectionPoints[1].Balance)
	// Day 3: no tx, balance = 1100
	assert.Equal(t, float64(1100), resp.ProjectionPoints[2].Balance)
}

func TestGetBalanceProjectionPlannedTxFatalNoExchangeRates(t *testing.T) {
	// Issue 4b: Fatal errNoExchangeRates for a planned tx inside the period loop.
	// Account is in USD (same as base, no conversion needed), but a planned tx
	// in a non-base currency triggers the exchange rate lookup, which returns
	// sql.ErrNoRows (empty table).
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 2:
			return &models.Currency{ID: 2, Code: "EUR", Name: "Euro"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// Exchange rate table is empty (sql.ErrNoRows)
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, sql.ErrNoRows
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			// USD account: same as base, no conversion needed for account balance
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	occDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// EUR tx requires conversion, which will fail with errNoExchangeRates
			{PlannedTransactionID: 1, CurrencyID: 2, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rates unavailable")
}

func TestGetBalanceProjectionPlannedTxNonFatalMissingRate(t *testing.T) {
	// Issue 4c: Non-fatal rate missing for a planned tx inside the period loop.
	// Account is in USD (same as base, no conversion needed), but a planned tx
	// has a currency code (XYZ) that is not in the rates map.
	handler, e, plannedTxSvc, accountRepo, currencyRepo, exchangeRateRepo := setupTestHandler()

	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		switch id {
		case 1:
			return &models.Currency{ID: 1, Code: "USD", Name: "US Dollar"}, nil
		case 99:
			return &models.Currency{ID: 99, Code: "XYZ", Name: "Unknown Currency"}, nil
		}
		return nil, errors.New("currency not found")
	}

	// Rates include USD but NOT XYZ
	exchangeRateRepo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return &types.ExchangeRateSnapshot{
			ID:               1,
			Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
			ActualDate:       date,
			BaseCurrencyCode: "USD",
		}, nil
	}

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			// USD account: same as base, no conversion needed
			{ID: 1, Name: "USD Account", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	occDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// Valid USD tx
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: occDate},
			// XYZ tx: rate missing from map, should be skipped
			{PlannedTransactionID: 2, CurrencyID: 99, Amount: decimal.NewFromInt(500), IsIncome: false, OccurrenceDate: occDate},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// 3 daily points: Mar 1, 2, 3
	require.Len(t, resp.ProjectionPoints, 3)

	// Day 1: no tx, balance = 1000
	assert.Equal(t, float64(1000), resp.ProjectionPoints[0].Balance)
	// Day 2: only the valid USD tx (100) income counted, XYZ expense skipped
	assert.Equal(t, float64(100), resp.ProjectionPoints[1].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1100), resp.ProjectionPoints[1].Balance)
	// Day 3: no tx, balance = 1100
	assert.Equal(t, float64(1100), resp.ProjectionPoints[2].Balance)
}

// ==================== QA Suggestion 1: Weekly grouping test with amount assertions ====================

func TestGetBalanceProjectionWeeklyGroupingWithAmounts(t *testing.T) {
	// Verify that weekly periods group transactions correctly with full
	// income/expense/balance assertions on each projection point.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}

	// Wednesday start, 3-week range
	startDate := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // Wednesday
	endDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	// Points: Mar 4, Mar 15, Mar 22, Mar 25
	// Period 0 (Mar 4): start point, transactions on Mar 4
	// Period 1 (Mar 5 - Mar 15): transactions on Mar 5, Mar 10
	// Period 2 (Mar 16 - Mar 22): transaction on Mar 18
	// Period 3 (Mar 23 - Mar 25): no transactions
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{
			// Period 1 (Mar 5 - Mar 15)
			{PlannedTransactionID: 1, CurrencyID: 1, Amount: decimal.NewFromInt(100), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)},
			{PlannedTransactionID: 2, CurrencyID: 1, Amount: decimal.NewFromInt(50), IsIncome: false, OccurrenceDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
			// Period 2 (Mar 16 - Mar 22)
			{PlannedTransactionID: 3, CurrencyID: 1, Amount: decimal.NewFromInt(200), IsIncome: true, OccurrenceDate: time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	body := `{"startDate": "` + startDate.Format(time.RFC3339) + `", "endDate": "` + endDate.Format(time.RFC3339) + `", "period": "weekly"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BalanceProjectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Points: Mar 4, Mar 15, Mar 22, Mar 25
	require.Len(t, resp.ProjectionPoints, 4)
	assert.Equal(t, "2026-03-04", resp.ProjectionPoints[0].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-15", resp.ProjectionPoints[1].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-22", resp.ProjectionPoints[2].Date.Format("2006-01-02"))
	assert.Equal(t, "2026-03-25", resp.ProjectionPoints[3].Date.Format("2006-01-02"))

	// Point 0 (Mar 4): no transactions on this day, balance = 1000
	assert.Equal(t, float64(0), resp.ProjectionPoints[0].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[0].Expenses)
	assert.Equal(t, float64(1000), resp.ProjectionPoints[0].Balance)

	// Point 1 (Mar 5 - Mar 15): income=100, expense=50, balance = 1000 + 100 - 50 = 1050
	assert.Equal(t, float64(100), resp.ProjectionPoints[1].Income)
	assert.Equal(t, float64(50), resp.ProjectionPoints[1].Expenses)
	assert.Equal(t, float64(1050), resp.ProjectionPoints[1].Balance)

	// Point 2 (Mar 16 - Mar 22): income=200, expense=0, balance = 1050 + 200 = 1250
	assert.Equal(t, float64(200), resp.ProjectionPoints[2].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[2].Expenses)
	assert.Equal(t, float64(1250), resp.ProjectionPoints[2].Balance)

	// Point 3 (Mar 23 - Mar 25): no transactions, balance = 1250
	assert.Equal(t, float64(0), resp.ProjectionPoints[3].Income)
	assert.Equal(t, float64(0), resp.ProjectionPoints[3].Expenses)
	assert.Equal(t, float64(1250), resp.ProjectionPoints[3].Balance)
}

// ==================== IncludeHidden filter tests ====================

func TestCalculateFutureBalanceIncludeHiddenTruePassedToRepo(t *testing.T) {
	// Verify that when includeHidden: true is sent in the request body,
	// the repository receives filters.IncludeHidden == true,
	// and OnlyShowInReports and IncludeArchived are always set correctly.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	var capturedFilters types.AccountFilters
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `", "includeHidden": true}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, capturedFilters.IncludeHidden, "IncludeHidden should be true when request has includeHidden: true")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestCalculateFutureBalanceIncludeHiddenFalsePassedToRepo(t *testing.T) {
	// Verify that when includeHidden is omitted (defaults to false),
	// the repository receives filters.IncludeHidden == false.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	var capturedFilters types.AccountFilters
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	targetDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	body := `{"targetDate": "` + targetDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/future-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CalculateFutureBalance(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, capturedFilters.IncludeHidden, "IncludeHidden should be false when request omits includeHidden")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestGetBalanceProjectionIncludeHiddenTruePassedToRepo(t *testing.T) {
	// Verify that when includeHidden: true is sent in the projection request,
	// the repository receives filters.IncludeHidden == true.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	var capturedFilters types.AccountFilters
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `", "includeHidden": true}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, capturedFilters.IncludeHidden, "IncludeHidden should be true when request has includeHidden: true")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

func TestGetBalanceProjectionIncludeHiddenFalsePassedToRepo(t *testing.T) {
	// Verify that when includeHidden is omitted (defaults to false),
	// the repository receives filters.IncludeHidden == false.
	handler, e, plannedTxSvc, accountRepo, _, _ := setupTestHandler()

	var capturedFilters types.AccountFilters
	accountRepo.GetByUserIDFunc = func(userID int, filters types.AccountFilters) ([]*models.Account, error) {
		capturedFilters = filters
		return []*models.Account{
			{ID: 1, Name: "Test", CurrencyID: 1, Balance: decimal.NewFromInt(1000)},
		}, nil
	}
	plannedTxSvc.GetUpcomingFunc = func(userID int, days int, includeInactive bool) ([]plannedtransactionsservice.Occurrence, error) {
		return []plannedtransactionsservice.Occurrence{}, nil
	}

	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)
	body := `{"endDate": "` + endDate + `"}`
	req := httptest.NewRequest(http.MethodPost, "/projection", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBalanceProjection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, capturedFilters.IncludeHidden, "IncludeHidden should be false when request omits includeHidden")
	assert.True(t, capturedFilters.OnlyShowInReports, "OnlyShowInReports should always be true")
	assert.False(t, capturedFilters.IncludeArchived, "IncludeArchived should always be false")
}

// TestConvertToBaseCurrencyMissingTargetRate has been moved to
// services/currency/rate_cache_test.go as TestConvertToBaseCurrencyMissingTargetRate.
