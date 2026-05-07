package transactions

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/middleware"
	txservice "github.com/go-budget/backend/internal/services/transactions"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Mock service
type MockTxService struct {
	CreateTransactionFunc     func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error)
	GetTransactionsFunc       func(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error)
	GetTransactionDetailsFunc func(transactionID, userID int) (*txservice.Transaction, error)
	UpdateTransactionFunc     func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error)
	DeleteTransactionFunc     func(transactionID, userID int) (*txservice.Transaction, error)
	GetTemplatesFunc          func(userID int) ([]*txservice.TransactionTemplate, error)
	UpdateTemplateFunc        func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error)
	DeleteTemplatesFunc       func(userID int, ids []int) ([]*txservice.TransactionTemplate, error)
}

func (m *MockTxService) CreateTransaction(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
	if m.CreateTransactionFunc != nil {
		return m.CreateTransactionFunc(userID, input)
	}
	now := time.Now()
	return &txservice.Transaction{ID: 1, Amount: decimal.NewFromInt(100), DateTime: &now}, nil
}

func (m *MockTxService) GetTransactions(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error) {
	if m.GetTransactionsFunc != nil {
		return m.GetTransactionsFunc(userID, input)
	}
	return []*txservice.Transaction{}, nil
}

func (m *MockTxService) GetTransactionDetails(transactionID, userID int) (*txservice.Transaction, error) {
	if m.GetTransactionDetailsFunc != nil {
		return m.GetTransactionDetailsFunc(transactionID, userID)
	}
	now := time.Now()
	return &txservice.Transaction{ID: transactionID, Amount: decimal.NewFromInt(100), DateTime: &now}, nil
}

func (m *MockTxService) UpdateTransaction(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
	if m.UpdateTransactionFunc != nil {
		return m.UpdateTransactionFunc(userID, input, txID)
	}
	now := time.Now()
	return &txservice.Transaction{ID: txID, Amount: decimal.NewFromInt(100), DateTime: &now}, nil
}

func (m *MockTxService) DeleteTransaction(transactionID, userID int) (*txservice.Transaction, error) {
	if m.DeleteTransactionFunc != nil {
		return m.DeleteTransactionFunc(transactionID, userID)
	}
	now := time.Now()
	return &txservice.Transaction{ID: transactionID, Amount: decimal.NewFromInt(100), DateTime: &now}, nil
}

func (m *MockTxService) GetTemplates(userID int) ([]*txservice.TransactionTemplate, error) {
	if m.GetTemplatesFunc != nil {
		return m.GetTemplatesFunc(userID)
	}
	return []*txservice.TransactionTemplate{}, nil
}

func (m *MockTxService) UpdateTemplate(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
	if m.UpdateTemplateFunc != nil {
		return m.UpdateTemplateFunc(userID, templateID, label, categoryID, targetAccountID)
	}
	return &txservice.TransactionTemplate{ID: templateID, Label: label}, nil
}

func (m *MockTxService) DeleteTemplates(userID int, ids []int) ([]*txservice.TransactionTemplate, error) {
	if m.DeleteTemplatesFunc != nil {
		return m.DeleteTemplatesFunc(userID, ids)
	}
	return []*txservice.TransactionTemplate{}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockTxService) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockService := &MockTxService{}
	handler := New(mockService, logger)
	e := echo.New()
	e.Validator = middleware.NewValidator()
	return handler, e, mockService
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CreateTransaction Tests ====================

func TestCreateTransactionInvalidAccount(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidAccount
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid account")
}

func TestCreateTransactionInvalidCategory(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidCategory
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid category")
}

func TestCreateTransactionAccessDenied(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrAccessDenied
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateTransactionUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrUserNotActivated
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestCreateTransactionInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidUser
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestCreateTransactionSelfTransferError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, txservice.ErrSelfTransfer
	}

	body := `{"accountId": 10, "targetAccountId": 10, "amount": 100, "isTransfer": true, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Source and target accounts must be different")
}

func TestCreateTransactionDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.CreateTransactionFunc = func(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error) {
		return nil, errors.New("db error")
	}

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== GetTransactions Tests ====================

func TestGetTransactionsUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionsFunc = func(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error) {
		return nil, txservice.ErrUserNotActivated
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTransactions(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestGetTransactionsInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionsFunc = func(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidUser
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTransactions(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestGetTransactionsDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionsFunc = func(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTransactions(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== GetTransactionDetails Tests ====================

func TestGetTransactionDetailsUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrUserNotActivated
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestGetTransactionDetailsInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidUser
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestGetTransactionDetailsNotFound(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrTransactionNotFound
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetTransactionDetailsAccessDenied(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrAccessDenied
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetTransactionDetailsDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== UpdateTransaction Tests ====================

func TestUpdateTransactionUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrUserNotActivated
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestUpdateTransactionInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidUser
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestUpdateTransactionNotFound(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrTransactionNotFound
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTransactionInvalidAccount(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidAccount
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateTransactionAccessDenied(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrAccessDenied
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateTransactionDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, errors.New("db error")
	}

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== DeleteTransaction Tests ====================

func TestDeleteTransactionUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTransactionFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrUserNotActivated
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestDeleteTransactionInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTransactionFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidUser
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestDeleteTransactionNotFound(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTransactionFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrTransactionNotFound
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteTransactionAccessDenied(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTransactionFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrAccessDenied
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteTransactionDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTransactionFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== GetTemplates Tests ====================

func TestGetTemplatesUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTemplatesFunc = func(userID int) ([]*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrUserNotActivated
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestGetTemplatesInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTemplatesFunc = func(userID int) ([]*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrInvalidUser
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestGetTemplatesDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTemplatesFunc = func(userID int) ([]*txservice.TransactionTemplate, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== UpdateTemplate Tests ====================

func TestUpdateTemplateUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTemplateFunc = func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrUserNotActivated
	}

	body := `{"id": 1, "label": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestUpdateTemplateInvalidUser(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTemplateFunc = func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrInvalidUser
	}

	body := `{"id": 1, "label": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestUpdateTemplateNotFound(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTemplateFunc = func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrTemplateNotFound
	}

	body := `{"id": 1, "label": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTemplateAccessDenied(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTemplateFunc = func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrAccessDenied
	}

	body := `{"id": 1, "label": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateTemplateDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTemplateFunc = func(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error) {
		return nil, errors.New("db error")
	}

	body := `{"id": 1, "label": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== DeleteTemplates Tests ====================

func TestDeleteTemplatesUserNotActivated(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTemplatesFunc = func(userID int, ids []int) ([]*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrUserNotActivated
	}

	req := httptest.NewRequest(http.MethodDelete, "/templates?ids=1,2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestDeleteTemplatesInvalidUserUnit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTemplatesFunc = func(userID int, ids []int) ([]*txservice.TransactionTemplate, error) {
		return nil, txservice.ErrInvalidUser
	}

	req := httptest.NewRequest(http.MethodDelete, "/templates?ids=1,2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestDeleteTemplatesDBError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTemplatesFunc = func(userID int, ids []int) ([]*txservice.TransactionTemplate, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/templates?ids=1,2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDeleteTemplatesInvalidIDs(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/templates?ids=abc,xyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid IDs")
}

// ==================== Response Conversion Tests ====================

func TestToTransactionResponseWithCategory(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	now := time.Now()
	childCat := &txservice.TransactionCategory{ID: 2, Name: "Subcategory", UserID: 1, Children: []*txservice.TransactionCategory{}}
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return &txservice.Transaction{
			ID:       1,
			Amount:   decimal.NewFromInt(100),
			DateTime: &now,
			Category: &txservice.TransactionCategory{
				ID:       1,
				Name:     "Food",
				UserID:   1,
				Children: []*txservice.TransactionCategory{childCat},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Food")
}

func TestToAccountDTOWithCreditLimit(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	now := time.Now()
	creditLimit := decimal.NewFromInt(5000)
	balanceInBase := decimal.NewFromFloat(1234.56)
	mockService.GetTransactionDetailsFunc = func(transactionID, userID int) (*txservice.Transaction, error) {
		return &txservice.Transaction{
			ID:       1,
			Amount:   decimal.NewFromInt(100),
			DateTime: &now,
			Account: &txservice.TransactionAccount{
				ID:                    1,
				Name:                  "Credit Card",
				Balance:               decimal.NewFromInt(500),
				InitialBalance:        decimal.Zero,
				CreditLimit:           &creditLimit,
				BalanceInBaseCurrency: &balanceInBase,
				Currency:              &txservice.TransactionCurrency{ID: 1, Code: "USD", Name: "US Dollar"},
				AccountType:           &txservice.TransactionAccountType{ID: 2, TypeName: "Credit", IsCredit: true},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Credit Card")
}

func TestGetTemplatesWithCategory(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.GetTemplatesFunc = func(userID int) ([]*txservice.TransactionTemplate, error) {
		return []*txservice.TransactionTemplate{
			{
				ID:       1,
				Label:    "Groceries",
				Category: &txservice.TransactionCategory{ID: 1, Name: "Food", UserID: 1, Children: []*txservice.TransactionCategory{}},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Groceries")
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _ := setupTestHandler()

	g := e.Group("/transactions")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/transactions/"])
	assert.True(t, routePaths["/transactions/templates"])
	assert.True(t, routePaths["/transactions/:id"])
}

// ==================== Bind/Validate Error Tests ====================

func TestCreateTransactionBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransactionValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateTransactionBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateTransactionValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateTemplateBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateTemplateValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTemplate(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDeleteTemplatesMissingIDs(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "ids parameter required")
}

func TestGetTransactionDetailsInvalidID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.GetTransactionDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteTransactionInvalidID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.DeleteTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTransactionWithCategoryID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": 5}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateTransactionWithCategoryID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{"id": 1, "accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": 5}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== FlexInt Tests ====================

func TestFlexIntUnmarshalJSONString(t *testing.T) {
	handler, e, _ := setupTestHandler()

	// categoryId as string
	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": "42"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFlexIntUnmarshalJSONEmptyString(t *testing.T) {
	handler, e, _ := setupTestHandler()

	// categoryId as empty string
	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": ""}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFlexIntUnmarshalJSONInvalidString(t *testing.T) {
	handler, e, _ := setupTestHandler()

	// categoryId as invalid string
	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": "abc"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestFlexIntToIntPtrZero(t *testing.T) {
	handler, e, _ := setupTestHandler()

	// categoryId as zero
	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": 0}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteTemplatesSuccess(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.DeleteTemplatesFunc = func(userID int, ids []int) ([]*txservice.TransactionTemplate, error) {
		return []*txservice.TransactionTemplate{
			{ID: 1, Label: "Template1"},
			{ID: 2, Label: "Template2"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/templates?ids=1,2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DeleteTemplates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Template1")
	assert.Contains(t, rec.Body.String(), "Template2")
}

// ==================== Update Transfer Error Path Tests ====================

func TestUpdateTransactionSelfTransferError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrSelfTransfer
	}

	body := `{"id": 1, "accountId": 10, "targetAccountId": 10, "amount": 100, "isTransfer": true}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Source and target accounts must be different")
}

func TestUpdateTransactionInvalidTargetAccountError(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrInvalidAccount
	}

	body := `{"id": 1, "accountId": 10, "targetAccountId": 9999, "amount": 100, "isTransfer": true}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid account")
}

func TestUpdateTransactionAccessDeniedOnTargetAccount(t *testing.T) {
	handler, e, mockService := setupTestHandler()
	mockService.UpdateTransactionFunc = func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
		return nil, txservice.ErrAccessDenied
	}

	body := `{"id": 1, "accountId": 10, "targetAccountId": 20, "amount": 100, "isTransfer": true}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Access denied")
}

func TestFlexIntUnmarshalJSONNull(t *testing.T) {
	handler, e, _ := setupTestHandler()

	// categoryId as null
	body := `{"accountId": 1, "amount": 100, "dateTime": "2024-01-01T00:00:00Z", "categoryId": null}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestCreateTransactionValidationLogEnriched asserts that on a validation
// failure the structured log line carries `user_id`, `errorCode`, and the
// failed JSON field name(s). The field name is OK in logs (server-side) but
// must NEVER appear in the API response.
func TestCreateTransactionValidationLogEnriched(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mockService := &MockTxService{}
	handler := New(mockService, logger)
	e := echo.New()
	e.Validator = middleware.NewValidator()

	label256 := strings.Repeat("a", 256)
	body := `{"accountId": 1, "amount": 100, "label": "` + label256 + `", "isIncome": false}`
	req := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/transactions/")
	setUserContext(c, 42)

	err := handler.CreateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	logLine := buf.String()
	assert.Contains(t, logLine, `"user_id":42`, "validation log must include user_id")
	assert.Contains(t, logLine, `"errorCode":"errors.transaction.labelTooLong"`, "validation log must include errorCode")
	assert.Contains(t, logLine, `"label"`, "validation log must include the failed JSON field name")
	assert.Contains(t, logLine, `"handler":"transactions.CreateTransaction"`)
	assert.Contains(t, logLine, `"path":"/transactions/"`)

	// The response body must NOT carry validator internals or the failed field name.
	respBody := rec.Body.String()
	for _, leak := range []string{"CreateTransactionRequest", "Field validation", "failed on the", "Tag=", "Field="} {
		assert.NotContains(t, respBody, leak)
	}
}

// TestUpdateTransactionUnknownErrorLogsLengths asserts the unknown-DB-error
// branch of UpdateTransaction logs `label_len` and `notes_len` so operators
// can spot oversized payloads without leaking the raw user content.
func TestUpdateTransactionUnknownErrorLogsLengths(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mockService := &MockTxService{
		UpdateTransactionFunc: func(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error) {
			return nil, errors.New("pq: value too long for type character varying(50)")
		},
	}
	handler := New(mockService, logger)
	e := echo.New()
	e.Validator = middleware.NewValidator()

	notes := strings.Repeat("n", 30)
	body := `{"id": 7, "accountId": 1, "amount": 50, "label": "hello", "notes": "` + notes + `", "isIncome": false}`
	req := httptest.NewRequest(http.MethodPut, "/transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/transactions/")
	setUserContext(c, 42)

	err := handler.UpdateTransaction(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	logLine := buf.String()
	assert.Contains(t, logLine, `"user_id":42`)
	assert.Contains(t, logLine, `"label_len":5`)
	assert.Contains(t, logLine, `"notes_len":30`)
	assert.Contains(t, logLine, `"handler":"transactions.UpdateTransaction"`)
}
