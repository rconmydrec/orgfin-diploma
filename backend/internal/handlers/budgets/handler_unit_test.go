package budgets

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	budgetsservice "github.com/go-budget/backend/internal/services/budgets"
	"github.com/go-budget/backend/internal/types"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock repositories used to build the real service for handler tests.
type MockBudgetRepo struct {
	CreateFunc                          func(budget *models.Budget) (*models.Budget, error)
	GetByUserIDFunc                     func(userID int, include string) ([]*models.Budget, error)
	GetByIDFunc                         func(id int) (*models.Budget, error)
	GetOutdatedBudgetsFunc              func() ([]*models.Budget, error)
	UpdateFunc                          func(budget *models.Budget) error
	DeleteFunc                          func(id int) error
	ArchiveFunc                         func(id int) error
	UpdateCollectedAmountFunc           func(budgetID int, amount decimal.Decimal) error
	GetExpenseTransactionsForBudgetFunc func(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error)
}

func (m *MockBudgetRepo) Create(budget *models.Budget) (*models.Budget, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(budget)
	}
	return &models.Budget{ID: 1, Name: budget.Name}, nil
}

func (m *MockBudgetRepo) GetByUserID(userID int, include string) ([]*models.Budget, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID, include)
	}
	return []*models.Budget{}, nil
}

func (m *MockBudgetRepo) GetByID(id int) (*models.Budget, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Budget{ID: id, UserID: 1, Name: "Test"}, nil
}

func (m *MockBudgetRepo) GetOutdatedBudgets() ([]*models.Budget, error) {
	if m.GetOutdatedBudgetsFunc != nil {
		return m.GetOutdatedBudgetsFunc()
	}
	return []*models.Budget{}, nil
}

func (m *MockBudgetRepo) Update(budget *models.Budget) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(budget)
	}
	return nil
}

func (m *MockBudgetRepo) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockBudgetRepo) Archive(id int) error {
	if m.ArchiveFunc != nil {
		return m.ArchiveFunc(id)
	}
	return nil
}

func (m *MockBudgetRepo) UpdateCollectedAmount(budgetID int, amount decimal.Decimal) error {
	if m.UpdateCollectedAmountFunc != nil {
		return m.UpdateCollectedAmountFunc(budgetID, amount)
	}
	return nil
}

func (m *MockBudgetRepo) GetExpenseTransactionsForBudget(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error) {
	if m.GetExpenseTransactionsForBudgetFunc != nil {
		return m.GetExpenseTransactionsForBudgetFunc(userID, startDate, endDate, categoryIDs)
	}
	return nil, nil
}

type MockCurrencyRepo struct {
	GetByIDFunc func(id int) (*models.Currency, error)
}

func (m *MockCurrencyRepo) GetByID(id int) (*models.Currency, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.Currency{ID: id, Code: "USD", Name: "US Dollar"}, nil
}

type mockCurrencyConverter struct{}

func (m *mockCurrencyConverter) ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal {
	return amount
}

func setupTestHandler() (*Handler, *echo.Echo, *MockBudgetRepo, *MockCurrencyRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	budgetRepo := &MockBudgetRepo{}
	currencyRepo := &MockCurrencyRepo{}
	svc := budgetsservice.New(budgetRepo, budgetRepo, currencyRepo, &mockCurrencyConverter{}, logger)
	handler := New(svc, logger)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return handler, e, budgetRepo, currencyRepo
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CreateBudget Tests ====================

func TestCreateBudgetBindError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateBudgetValidateError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateBudgetInvalidCurrency(t *testing.T) {
	handler, e, _, currencyRepo := setupTestHandler()
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, sql.ErrNoRows
	}

	body := `{"name": "Test", "currencyId": 999, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateBudgetDBError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return nil, errors.New("db error")
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateBudgetSuccess(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{
			ID:           1,
			Name:         budget.Name,
			TargetAmount: budget.TargetAmount,
			StartDate:    budget.StartDate,
			EndDate:      budget.EndDate,
		}, nil
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z", "categories": [1, 2]}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== GetBudgets Tests ====================

func TestGetBudgetsDBError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByUserIDFunc = func(userID int, include string) ([]*models.Budget, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBudgets(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetBudgetsSuccess(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetByUserIDFunc = func(userID int, include string) ([]*models.Budget, error) {
		return []*models.Budget{
			{ID: 1, Name: "Budget1", TargetAmount: decimal.NewFromInt(100), StartDate: now, EndDate: now.AddDate(0, 1, 0), Currency: &models.Currency{ID: 1, Code: "USD"}},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/?include=active", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetBudgets(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Budget1")
}

// ==================== UpdateBudget Tests ====================

func TestUpdateBudgetInvalidID(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/abc/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateBudgetBindError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateBudgetValidateError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateBudgetNotFound(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return nil, sql.ErrNoRows
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateBudgetAccessDenied(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 999}, nil // Different user
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateBudgetInvalidCurrency(t *testing.T) {
	handler, e, budgetRepo, currencyRepo := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}
	currencyRepo.GetByIDFunc = func(id int) (*models.Currency, error) {
		return nil, sql.ErrNoRows
	}

	body := `{"name": "Test", "currencyId": 999, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateBudgetDBError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}
	budgetRepo.UpdateFunc = func(budget *models.Budget) error {
		return errors.New("db error")
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateBudgetSuccess(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1, StartDate: now, EndDate: now.AddDate(0, 1, 0)}, nil
	}

	body := `{"name": "Updated", "currencyId": 1, "targetAmount": 200, "period": "monthly", "startDate": "2024-01-01T00:00:00Z", "endDate": "2024-12-31T00:00:00Z", "categories": [1]}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== DeleteBudget Tests ====================

func TestDeleteBudgetInvalidID(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/abc/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.DeleteBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteBudgetNotFound(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return nil, sql.ErrNoRows
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteBudgetAccessDenied(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 999}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteBudgetDBError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}
	budgetRepo.DeleteFunc = func(id int) error {
		return errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDeleteBudgetSuccess(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== ArchiveBudget Tests ====================

func TestArchiveBudgetInvalidID(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodPut, "/abc/archive/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.ArchiveBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestArchiveBudgetNotFound(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return nil, sql.ErrNoRows
	}

	req := httptest.NewRequest(http.MethodPut, "/1/archive/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.ArchiveBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestArchiveBudgetAccessDenied(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 999}, nil
	}

	req := httptest.NewRequest(http.MethodPut, "/1/archive/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.ArchiveBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestArchiveBudgetDBError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}
	budgetRepo.ArchiveFunc = func(id int) error {
		return errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodPut, "/1/archive/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.ArchiveBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestArchiveBudgetSuccess(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1}, nil
	}

	req := httptest.NewRequest(http.MethodPut, "/1/archive/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.ArchiveBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	g := e.Group("/budgets")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/budgets/add/"])
	assert.True(t, routePaths["/budgets/"])
	assert.True(t, routePaths["/budgets/:id/"])
	assert.True(t, routePaths["/budgets/:id/archive/"])
}

// ==================== Date Format Tests ====================

func TestCreateBudgetDateOnlyFormat(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		assert.Equal(t, 2024, budget.StartDate.Year())
		assert.Equal(t, time.January, budget.StartDate.Month())
		assert.Equal(t, 1, budget.StartDate.Day())
		return &models.Budget{
			ID:        1,
			Name:      budget.Name,
			StartDate: budget.StartDate,
			EndDate:   budget.EndDate,
		}, nil
	}

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01", "endDate": "2024-12-31", "categories": [1]}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCreateBudgetInvalidStartDateFormat(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "not-a-date", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid start date format")
}

func TestCreateBudgetInvalidEndDateFormat(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01", "endDate": "not-a-date"}`
	req := httptest.NewRequest(http.MethodPost, "/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid end date format")
}

func TestUpdateBudgetDateOnlyFormat(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetByIDFunc = func(id int) (*models.Budget, error) {
		return &models.Budget{ID: id, UserID: 1, StartDate: now, EndDate: now.AddDate(0, 1, 0)}, nil
	}

	body := `{"name": "Updated", "currencyId": 1, "targetAmount": 200, "period": "monthly", "startDate": "2024-01-01", "endDate": "2024-12-31", "categories": [1]}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateBudgetInvalidStartDateFormat(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "bad-date", "endDate": "2024-12-31"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid start date format")
}

func TestUpdateBudgetInvalidEndDateFormat(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"name": "Test", "currencyId": 1, "targetAmount": 100, "period": "monthly", "startDate": "2024-01-01", "endDate": "bad-date"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateBudget(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid end date format")
}

// ==================== DailyProcessing Tests ====================

func TestDailyProcessingGetOutdatedError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDailyProcessingNoOutdatedBudgets(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return []*models.Budget{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Daily processing completed")
	assert.Contains(t, rec.Body.String(), `"processed":0`)
}

func TestDailyProcessingArchiveNonRepeating(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return []*models.Budget{
			{
				ID:        1,
				UserID:    1,
				Name:      "Monthly Budget",
				Period:    "MONTHLY",
				Repeat:    false,
				StartDate: now.AddDate(0, -1, 0),
				EndDate:   now.AddDate(0, 0, -1),
			},
		}, nil
	}

	archiveCalled := false
	budgetRepo.ArchiveFunc = func(id int) error {
		archiveCalled = true
		assert.Equal(t, 1, id)
		return nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, archiveCalled)
	assert.Contains(t, rec.Body.String(), `"processed":1`)
}

func TestDailyProcessingRepeatMonthly(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
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
	}

	createCalled := false
	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		createCalled = true
		assert.Contains(t, budget.Name, "(copy)")
		assert.True(t, budget.CollectedAmount.IsZero())
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, createCalled)
	assert.Contains(t, rec.Body.String(), `"processed":1`)
}

func TestDailyProcessingRepeatDaily(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
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
	}

	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDailyProcessingRepeatWeekly(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
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
	}

	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDailyProcessingRepeatYearly(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
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
	}

	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDailyProcessingRepeatCustomPeriod(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return []*models.Budget{
			{
				ID:        1,
				UserID:    1,
				Name:      "Custom Budget",
				Period:    "CUSTOM",
				Repeat:    true,
				StartDate: now.AddDate(0, 0, -10),
				EndDate:   now.AddDate(0, 0, -1),
			},
		}, nil
	}

	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDailyProcessingCreateError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return []*models.Budget{
			{
				ID:        1,
				UserID:    1,
				Name:      "Monthly Budget",
				Period:    "MONTHLY",
				Repeat:    true,
				StartDate: now.AddDate(0, -1, 0),
				EndDate:   now.AddDate(0, 0, -1),
			},
		}, nil
	}

	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Budget creation failed, so it should not be counted as processed
	assert.Contains(t, rec.Body.String(), `"processed":0`)
}

func TestDailyProcessingArchiveError(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
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
	}

	budgetRepo.ArchiveFunc = func(id int) error {
		return errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Archive failed, so it should not be counted
	assert.Contains(t, rec.Body.String(), `"processed":0`)
}

// ==================== DailyProcessing Archive Error After Copy ====================

func TestDailyProcessingArchiveErrorAfterCopy(t *testing.T) {
	handler, e, budgetRepo, _ := setupTestHandler()
	now := time.Now()
	budgetRepo.GetOutdatedBudgetsFunc = func() ([]*models.Budget, error) {
		return []*models.Budget{
			{
				ID:        1,
				UserID:    1,
				Name:      "Monthly Budget",
				Period:    "MONTHLY",
				Repeat:    true,
				StartDate: now.AddDate(0, -1, 0),
				EndDate:   now.AddDate(0, 0, -1),
			},
		}, nil
	}
	budgetRepo.CreateFunc = func(budget *models.Budget) (*models.Budget, error) {
		return &models.Budget{ID: 2, Name: budget.Name}, nil
	}
	budgetRepo.ArchiveFunc = func(id int) error {
		return errors.New("archive failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/daily-processing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DailyProcessing(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"processed":0`)
}
