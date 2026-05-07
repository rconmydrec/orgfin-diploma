package accounts

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	accountsservice "github.com/go-budget/backend/internal/services/accounts"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// CustomValidator for echo
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// MockAccountsService implements AccountsService for testing
type MockAccountsService struct {
	CreateAccountFunc     func(userID int, input accountsservice.CreateAccountInput) (*accountsservice.Account, error)
	GetUserAccountsFunc   func(userID int, input accountsservice.GetAccountsInput) ([]*accountsservice.Account, error)
	GetAccountDetailsFunc func(accountID, userID int) (*accountsservice.Account, error)
	GetAccountTypesFunc   func() ([]*accountsservice.AccountType, error)
	UpdateAccountFunc     func(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error)
	DeleteAccountFunc     func(accountID, userID int) error
	SetArchiveStatusFunc  func(accountID int, isArchived bool, userID int) error
	AdjustBalanceFunc     func(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error)
}

func (m *MockAccountsService) CreateAccount(userID int, input accountsservice.CreateAccountInput) (*accountsservice.Account, error) {
	if m.CreateAccountFunc != nil {
		return m.CreateAccountFunc(userID, input)
	}
	return nil, nil
}

func (m *MockAccountsService) GetUserAccounts(userID int, input accountsservice.GetAccountsInput) ([]*accountsservice.Account, error) {
	if m.GetUserAccountsFunc != nil {
		return m.GetUserAccountsFunc(userID, input)
	}
	return nil, nil
}

func (m *MockAccountsService) GetAccountDetails(accountID, userID int) (*accountsservice.Account, error) {
	if m.GetAccountDetailsFunc != nil {
		return m.GetAccountDetailsFunc(accountID, userID)
	}
	return nil, nil
}

func (m *MockAccountsService) GetAccountTypes() ([]*accountsservice.AccountType, error) {
	if m.GetAccountTypesFunc != nil {
		return m.GetAccountTypesFunc()
	}
	return nil, nil
}

func (m *MockAccountsService) UpdateAccount(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error) {
	if m.UpdateAccountFunc != nil {
		return m.UpdateAccountFunc(accountID, userID, input)
	}
	return nil, nil
}

func (m *MockAccountsService) DeleteAccount(accountID, userID int) error {
	if m.DeleteAccountFunc != nil {
		return m.DeleteAccountFunc(accountID, userID)
	}
	return nil
}

func (m *MockAccountsService) SetArchiveStatus(accountID int, isArchived bool, userID int) error {
	if m.SetArchiveStatusFunc != nil {
		return m.SetArchiveStatusFunc(accountID, isArchived, userID)
	}
	return nil
}

func (m *MockAccountsService) AdjustBalance(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error) {
	if m.AdjustBalanceFunc != nil {
		return m.AdjustBalanceFunc(accountID, newBalance, notes, userID)
	}
	return nil, nil
}

func setupTestHandler(mockService *MockAccountsService) (*Handler, *echo.Echo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := New(mockService, logger)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	return handler, e
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CreateAccount Error Tests ====================

func TestCreateAccountDBError(t *testing.T) {
	mockService := &MockAccountsService{
		CreateAccountFunc: func(userID int, input accountsservice.CreateAccountInput) (*accountsservice.Account, error) {
			return nil, errors.New("database connection failed")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currencyId": 1, "accountTypeId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to create account")
}

func TestCreateAccountLimitExceeded(t *testing.T) {
	mockService := &MockAccountsService{
		CreateAccountFunc: func(userID int, input accountsservice.CreateAccountInput) (*accountsservice.Account, error) {
			return nil, accountsservice.ErrAccountLimitExceeded
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currencyId": 1, "accountTypeId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
	assert.Contains(t, rec.Body.String(), "Account limit exceeded")
}

// ==================== GetAccounts Error Tests ====================

func TestGetAccountsInvalidUser(t *testing.T) {
	mockService := &MockAccountsService{
		GetUserAccountsFunc: func(userID int, input accountsservice.GetAccountsInput) ([]*accountsservice.Account, error) {
			return nil, accountsservice.ErrInvalidUser
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetAccounts(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid user")
}

func TestGetAccountsDBError(t *testing.T) {
	mockService := &MockAccountsService{
		GetUserAccountsFunc: func(userID int, input accountsservice.GetAccountsInput) ([]*accountsservice.Account, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetAccounts(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get accounts")
}

// ==================== GetAccountTypes Error Tests ====================

func TestGetAccountTypesDBError(t *testing.T) {
	mockService := &MockAccountsService{
		GetAccountTypesFunc: func() ([]*accountsservice.AccountType, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/types/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetAccountTypes(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get account types")
}

// ==================== GetAccountDetails Error Tests ====================

func TestGetAccountDetailsDBError(t *testing.T) {
	mockService := &MockAccountsService{
		GetAccountDetailsFunc: func(accountID, userID int) (*accountsservice.Account, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetAccountDetails(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get account")
}

// ==================== UpdateAccount Error Tests ====================

func TestUpdateAccountInvalidDateFormat(t *testing.T) {
	mockService := &MockAccountsService{}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currency_id": 1, "account_type_id": 1, "opening_date": "invalid-date"}`
	req := httptest.NewRequest(http.MethodPut, "/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid opening_date format")
}

func TestUpdateAccountDBError(t *testing.T) {
	mockService := &MockAccountsService{
		UpdateAccountFunc: func(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currencyId": 1, "accountTypeId": 1}`
	req := httptest.NewRequest(http.MethodPut, "/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to update account")
}

func TestUpdateAccountInvalidCurrency(t *testing.T) {
	mockService := &MockAccountsService{
		UpdateAccountFunc: func(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error) {
			return nil, accountsservice.ErrInvalidCurrency
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currencyId": 999, "accountTypeId": 1}`
	req := httptest.NewRequest(http.MethodPut, "/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid currency")
}

func TestUpdateAccountInvalidAccountType(t *testing.T) {
	mockService := &MockAccountsService{
		UpdateAccountFunc: func(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error) {
			return nil, accountsservice.ErrInvalidAccountType
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"name": "Test", "currencyId": 1, "accountTypeId": 999}`
	req := httptest.NewRequest(http.MethodPut, "/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid account type")
}

// ==================== DeleteAccount Error Tests ====================

func TestDeleteAccountDBError(t *testing.T) {
	mockService := &MockAccountsService{
		DeleteAccountFunc: func(accountID, userID int) error {
			return errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to delete account")
}

// ==================== SetArchiveStatus Error Tests ====================

func TestSetArchiveStatusNotFound(t *testing.T) {
	mockService := &MockAccountsService{
		SetArchiveStatusFunc: func(accountID int, isArchived bool, userID int) error {
			return accountsservice.ErrAccountNotFound
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 999, "isArchived": true}`
	req := httptest.NewRequest(http.MethodPut, "/set-archive-status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.SetArchiveStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Account not found")
}

func TestSetArchiveStatusAccessDenied(t *testing.T) {
	mockService := &MockAccountsService{
		SetArchiveStatusFunc: func(accountID int, isArchived bool, userID int) error {
			return accountsservice.ErrAccessDenied
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 1, "isArchived": true}`
	req := httptest.NewRequest(http.MethodPut, "/set-archive-status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.SetArchiveStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Access denied")
}

func TestSetArchiveStatusDBError(t *testing.T) {
	mockService := &MockAccountsService{
		SetArchiveStatusFunc: func(accountID int, isArchived bool, userID int) error {
			return errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 1, "isArchived": true}`
	req := httptest.NewRequest(http.MethodPut, "/set-archive-status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.SetArchiveStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to set archive status")
}

// ==================== AdjustBalance Error Tests ====================

func TestAdjustBalanceUnchanged(t *testing.T) {
	mockService := &MockAccountsService{
		AdjustBalanceFunc: func(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error) {
			return nil, accountsservice.ErrBalanceUnchanged
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 1, "newBalance": 100}`
	req := httptest.NewRequest(http.MethodPost, "/adjust-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.AdjustBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Balance unchanged")
}

func TestAdjustBalanceDBError(t *testing.T) {
	mockService := &MockAccountsService{
		AdjustBalanceFunc: func(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error) {
			return nil, errors.New("database error")
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 1, "newBalance": 100}`
	req := httptest.NewRequest(http.MethodPost, "/adjust-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.AdjustBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to adjust balance")
}

func TestSetArchiveStatusValidationError(t *testing.T) {
	mockService := &MockAccountsService{}

	handler, e := setupTestHandler(mockService)

	// Missing required accountId field
	body := `{"isArchived": true}`
	req := httptest.NewRequest(http.MethodPut, "/set-archive-status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.SetArchiveStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Validation failed")
}

func TestAdjustBalanceValidationError(t *testing.T) {
	mockService := &MockAccountsService{}

	handler, e := setupTestHandler(mockService)

	// Missing required accountId field
	body := `{"newBalance": 100}`
	req := httptest.NewRequest(http.MethodPost, "/adjust-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.AdjustBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Validation failed")
}

func TestAdjustBalanceNotFound(t *testing.T) {
	mockService := &MockAccountsService{
		AdjustBalanceFunc: func(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error) {
			return nil, accountsservice.ErrAccountNotFound
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 999, "newBalance": 100}`
	req := httptest.NewRequest(http.MethodPost, "/adjust-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.AdjustBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Account not found")
}

func TestAdjustBalanceAccessDenied(t *testing.T) {
	mockService := &MockAccountsService{
		AdjustBalanceFunc: func(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error) {
			return nil, accountsservice.ErrAccessDenied
		},
	}

	handler, e := setupTestHandler(mockService)

	body := `{"accountId": 1, "newBalance": 100}`
	req := httptest.NewRequest(http.MethodPost, "/adjust-balance", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.AdjustBalance(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Access denied")
}

// ==================== UnmarshalJSON Tests ====================

func TestCreateAccountRequestSnakeCaseFields(t *testing.T) {
	payload := `{
		"name": "Snake Account",
		"account_type_id": 42,
		"currency_id": 7,
		"initial_balance": 1500.50,
		"balance": 2000,
		"credit_limit": 500.25,
		"is_hidden": true,
		"show_in_reports": false,
		"opening_date": "2026-02-24T10:30:00Z",
		"comment": "snake_case test"
	}`

	var req CreateAccountRequest
	err := json.Unmarshal([]byte(payload), &req)
	assert.NoError(t, err)

	assert.Equal(t, "Snake Account", req.Name)
	assert.Equal(t, 42, req.AccountTypeID)
	assert.Equal(t, 7, req.CurrencyID)
	assert.True(t, decimal.NewFromFloat(1500.50).Equal(req.InitialBalance))
	assert.True(t, decimal.NewFromFloat(2000).Equal(req.Balance))
	assert.True(t, decimal.NewFromFloat(500.25).Equal(req.CreditLimit))
	assert.True(t, req.IsHidden)
	assert.NotNil(t, req.ShowInReports)
	assert.False(t, *req.ShowInReports)
	assert.NotNil(t, req.OpeningDate)
	expected, _ := time.Parse(time.RFC3339, "2026-02-24T10:30:00Z")
	assert.True(t, expected.Equal(*req.OpeningDate))
	assert.Equal(t, "snake_case test", req.Comment)
}

func TestCreateAccountRequestMixedCaseFields(t *testing.T) {
	payload := `{
		"name": "Mixed Case Account",
		"currencyId": 3,
		"account_type_id": 10,
		"initialBalance": 100,
		"credit_limit": 200,
		"isHidden": false,
		"show_in_reports": true,
		"opening_date": "2026-01-15T09:00"
	}`

	var req CreateAccountRequest
	err := json.Unmarshal([]byte(payload), &req)
	assert.NoError(t, err)

	assert.Equal(t, "Mixed Case Account", req.Name)
	assert.Equal(t, 3, req.CurrencyID)
	assert.Equal(t, 10, req.AccountTypeID)
	assert.True(t, decimal.NewFromInt(100).Equal(req.InitialBalance))
	assert.True(t, decimal.NewFromInt(200).Equal(req.CreditLimit))
	assert.False(t, req.IsHidden)
	assert.NotNil(t, req.ShowInReports)
	assert.True(t, *req.ShowInReports)
	assert.NotNil(t, req.OpeningDate)
}

func TestCreateAccountRequestDecimalAsString(t *testing.T) {
	payload := `{
		"name": "String Decimal Account",
		"currencyId": 1,
		"accountTypeId": 1,
		"balance": "5000.75"
	}`

	var req CreateAccountRequest
	err := json.Unmarshal([]byte(payload), &req)
	assert.NoError(t, err)

	expected, _ := decimal.NewFromString("5000.75")
	assert.True(t, expected.Equal(req.Balance))
}

func TestCreateAccountRequestTimeFormats(t *testing.T) {
	tests := []struct {
		name         string
		openingDate  string
		expectNil    bool
		expectedTime string
	}{
		{
			name:         "RFC3339",
			openingDate:  `"2026-02-24T10:30:00Z"`,
			expectNil:    false,
			expectedTime: "2026-02-24T10:30:00Z",
		},
		{
			name:         "DateTime",
			openingDate:  `"2026-02-24T10:30:05"`,
			expectNil:    false,
			expectedTime: "2026-02-24T10:30:05",
		},
		{
			name:         "DateMinute",
			openingDate:  `"2026-02-24T10:30"`,
			expectNil:    false,
			expectedTime: "2026-02-24T10:30",
		},
		{
			name:        "EmptyString",
			openingDate: `""`,
			expectNil:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"name":"T","currencyId":1,"accountTypeId":1,"openingDate":` + tc.openingDate + `}`

			var req CreateAccountRequest
			err := json.Unmarshal([]byte(payload), &req)
			assert.NoError(t, err)

			if tc.expectNil {
				assert.Nil(t, req.OpeningDate)
			} else {
				assert.NotNil(t, req.OpeningDate)
				// Parse the expected time in the same way to compare
				formats := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"}
				var expected time.Time
				for _, f := range formats {
					parsed, parseErr := time.Parse(f, tc.expectedTime)
					if parseErr == nil {
						expected = parsed
						break
					}
				}
				assert.True(t, expected.Equal(*req.OpeningDate),
					"expected %v, got %v", expected, *req.OpeningDate)
			}
		})
	}
}

func TestCreateAccountRequestInvalidValues(t *testing.T) {
	t.Run("InvalidDecimal", func(t *testing.T) {
		payload := `{"name":"T","currencyId":1,"accountTypeId":1,"balance":"not-a-number"}`

		var req CreateAccountRequest
		err := json.Unmarshal([]byte(payload), &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "balance")
	})

	t.Run("InvalidTime", func(t *testing.T) {
		payload := `{"name":"T","currencyId":1,"accountTypeId":1,"openingDate":"invalid"}`

		var req CreateAccountRequest
		err := json.Unmarshal([]byte(payload), &req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "openingDate")
	})
}

func TestCreateAccountRequestCamelCasePrecedence(t *testing.T) {
	// When both camelCase and snake_case are present, camelCase should win
	// because the get() helper checks camelCase first.
	payload := `{
		"name": "Precedence Test",
		"currencyId": 99,
		"currency_id": 1,
		"accountTypeId": 88,
		"account_type_id": 2
	}`

	var req CreateAccountRequest
	err := json.Unmarshal([]byte(payload), &req)
	assert.NoError(t, err)

	assert.Equal(t, 99, req.CurrencyID, "camelCase currencyId should take precedence")
	assert.Equal(t, 88, req.AccountTypeID, "camelCase accountTypeId should take precedence")
}

// ==================== Validation Error Response Test ====================

func TestCreateAccountValidationErrorResponse(t *testing.T) {
	mockService := &MockAccountsService{}
	handler, e := setupTestHandler(mockService)

	// Send empty JSON — all required fields missing (Name, CurrencyID, AccountTypeID)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateAccount(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var resp ValidationErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, "Validation failed", resp.Detail)
	assert.NotEmpty(t, resp.Errors, "expected field-level validation errors")

	// Collect field names from the response
	fieldNames := make(map[string]string)
	for _, fe := range resp.Errors {
		fieldNames[fe.Field] = fe.Message
	}

	// Verify required fields are reported
	assert.Contains(t, fieldNames, "Name", "expected validation error for Name")
	assert.Contains(t, fieldNames, "CurrencyID", "expected validation error for CurrencyID")
	assert.Contains(t, fieldNames, "AccountTypeID", "expected validation error for AccountTypeID")

	// Verify the message is the validator tag
	assert.Equal(t, "required", fieldNames["Name"])
	assert.Equal(t, "required", fieldNames["CurrencyID"])
	assert.Equal(t, "required", fieldNames["AccountTypeID"])
}
