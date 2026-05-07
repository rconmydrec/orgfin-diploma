package accounts_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
)

// TestCreateAccountEndpoint tests for POST /accounts/
func TestCreateAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "createaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false) // Regular account

	reqBody := fmt.Sprintf(`{
		"name": "New Checking Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 5000.00,
		"balance": 5000.00,
		"isHidden": false,
		"comment": "Test account"
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "New Checking Account" {
		t.Errorf("Expected name 'New Checking Account', got '%v'", response["name"])
	}
	if response["balance"].(float64) != 5000.00 {
		t.Errorf("Expected balance 5000.00, got '%v'", response["balance"])
	}
	if int(response["currencyId"].(float64)) != currencyID {
		t.Errorf("Expected currencyId %d, got '%v'", currencyID, response["currencyId"])
	}
}

func TestCreateAccountWithSnakeCaseFields(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "snakecaseaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"account_type_id": %d,
		"currency_id": %d,
		"name": "Snake Case Account",
		"balance": 1000,
		"credit_limit": 0,
		"is_hidden": false,
		"opening_date": "2026-02-24T08:39",
		"comment": "test"
	}`, accountTypeID, currencyID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "Snake Case Account" {
		t.Errorf("Expected name 'Snake Case Account', got '%v'", response["name"])
	}
	if response["balance"].(float64) != 1000 {
		t.Errorf("Expected balance 1000, got '%v'", response["balance"])
	}
	if int(response["currencyId"].(float64)) != currencyID {
		t.Errorf("Expected currencyId %d, got '%v'", currencyID, response["currencyId"])
	}
	if int(response["accountTypeId"].(float64)) != accountTypeID {
		t.Errorf("Expected accountTypeId %d, got '%v'", accountTypeID, response["accountTypeId"])
	}
	if response["comment"] != "test" {
		t.Errorf("Expected comment 'test', got '%v'", response["comment"])
	}
	if response["openingDate"] == nil {
		t.Error("Expected openingDate to be set")
	}
}

func TestCreateCreditAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "creditaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	// Try to get credit account type
	var creditTypeID int
	err := app.DB.QueryRow("SELECT id FROM account_types WHERE is_credit = true LIMIT 1").Scan(&creditTypeID)
	if err != nil {
		t.Skip("No credit account type found")
	}

	reqBody := fmt.Sprintf(`{
		"name": "Credit Card",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 0,
		"balance": 0,
		"creditLimit": 10000.00,
		"isHidden": false
	}`, currencyID, creditTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "Credit Card" {
		t.Errorf("Expected name 'Credit Card', got '%v'", response["name"])
	}
	if response["creditLimit"].(float64) != 10000.00 {
		t.Errorf("Expected creditLimit 10000.00, got '%v'", response["creditLimit"])
	}
}

func TestCreateAccountInvalidCurrency(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "invalidcurrency@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Invalid Currency Account",
		"currencyId": 999999,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000
	}`, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountInvalidType(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "invalidtype@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	reqBody := fmt.Sprintf(`{
		"name": "Invalid Type Account",
		"currencyId": %d,
		"accountTypeId": 999999,
		"initialBalance": 1000,
		"balance": 1000
	}`, currencyID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Unauthorized Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Without token - should be rejected
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateAccountMissingRequiredFields(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "missingfields@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"name": "Incomplete Account"
	}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}

	// Verify the response contains field-level validation errors
	var resp struct {
		Detail string `json:"detail"`
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse validation response: %v", err)
	}

	if resp.Detail != "Validation failed" {
		t.Errorf("Expected detail 'Validation failed', got '%s'", resp.Detail)
	}
	if len(resp.Errors) == 0 {
		t.Fatal("Expected field-level validation errors, got empty array")
	}

	// Collect field names from the errors
	fields := make(map[string]string)
	for _, fe := range resp.Errors {
		fields[fe.Field] = fe.Message
	}

	// CurrencyID and AccountTypeID are required and missing
	if _, ok := fields["CurrencyID"]; !ok {
		t.Error("Expected validation error for CurrencyID")
	}
	if _, ok := fields["AccountTypeID"]; !ok {
		t.Error("Expected validation error for AccountTypeID")
	}
}

func TestCreateHiddenAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "hiddenaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Hidden Savings",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 10000,
		"balance": 10000,
		"isHidden": true
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["isHidden"] != true {
		t.Errorf("Expected isHidden to be true, got '%v'", response["isHidden"])
	}
}

// TestGetAccountsEndpoint tests for GET /accounts/
func TestGetAccountsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "getaccounts@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Create a test account
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Test Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) < 1 {
		t.Error("Expected at least 1 account in response")
	}

	// Check that our test account is in the list
	found := false
	for _, acc := range response {
		if int(acc["id"].(float64)) == accountID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find account %d in response", accountID)
	}
}

func TestGetAccountsEmptyList(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "noaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected empty list, got %d accounts", len(response))
	}
}

func TestGetAccountsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// TestGetAccountTypesEndpoint tests for GET /accounts/types/
func TestGetAccountTypesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "accounttypes@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodGet, "/accounts/types/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response) == 0 {
		t.Error("Expected at least one account type")
	}

	// Check structure
	for _, accType := range response {
		if _, ok := accType["id"]; !ok {
			t.Error("Expected 'id' field in account type")
		}
		if _, ok := accType["type_name"]; !ok {
			t.Error("Expected 'type_name' field in account type")
		}
		if _, ok := accType["is_credit"]; !ok {
			t.Error("Expected 'is_credit' field in account type")
		}
	}
}

// TestGetAccountDetailsEndpoint tests for GET /accounts/{account_id}
func TestGetAccountDetailsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "accountdetails@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Details Account", currencyID, accountTypeID, 2500.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if int(response["id"].(float64)) != accountID {
		t.Errorf("Expected id %d, got '%v'", accountID, response["id"])
	}
	if response["name"] != "Details Account" {
		t.Errorf("Expected name 'Details Account', got '%v'", response["name"])
	}
	if _, ok := response["balance"]; !ok {
		t.Error("Expected 'balance' field in response")
	}
	if _, ok := response["currencyId"]; !ok {
		t.Error("Expected 'currencyId' field in response")
	}
}

func TestGetAccountDetailsNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "notfoundaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodGet, "/accounts/999999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestGetAccountDetailsOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "owner@example.com"
	email2 := "other@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	defer testutil.CleanupUserByEmail(t, app.DB, email1)
	defer testutil.CleanupUserByEmail(t, app.DB, email2)

	// Create owner and their account
	ownerID := testutil.CreateTestUser(t, app, email1, password)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, ownerID, "Owner Account", currencyID, accountTypeID, 5000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Create other user and get their token
	testutil.CreateTestUser(t, app, email2, password)
	otherToken := testutil.GetAuthToken(t, app, email2, password)

	// Try to access owner's account with other user's token
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", otherToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should be denied
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 400 or 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// TestDeleteAccountEndpoint tests for DELETE /accounts/{account_id}
func TestDeleteAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deleteaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Be Deleted", currencyID, accountTypeID, 100.00)
	// Note: cleanup will be handled automatically since we're deleting it

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["deleted"] != true {
		t.Error("Expected 'deleted' to be true")
	}

	// Verify account is soft deleted
	var isDeleted bool
	err := app.DB.QueryRow("SELECT is_deleted FROM accounts WHERE id = $1", accountID).Scan(&isDeleted)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}
	if !isDeleted {
		t.Error("Expected account to be marked as deleted")
	}
}

func TestDeleteAccountUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/accounts/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deletenf@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodDelete, "/accounts/999999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "deleteowner@example.com"
	email2 := "deletehacker@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	defer testutil.CleanupUserByEmail(t, app.DB, email1)
	defer testutil.CleanupUserByEmail(t, app.DB, email2)

	ownerID := testutil.CreateTestUser(t, app, email1, password)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, ownerID, "Owner Account", currencyID, accountTypeID, 5000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	testutil.CreateTestUser(t, app, email2, password)
	hackerToken := testutil.GetAuthToken(t, app, email2, password)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", hackerToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 400 or 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Update Account Tests ====================

func TestUpdateAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updateaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Original Name", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{
		"name": "Updated Account Name",
		"currency_id": %d,
		"account_type_id": %d,
		"initial_balance": 1000,
		"balance": 1000,
		"is_hidden": false,
		"show_in_reports": true,
		"opening_date": "2024-01-01T00:00:00",
		"comment": "Updated comment"
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d", accountID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "Updated Account Name" {
		t.Errorf("Expected name 'Updated Account Name', got '%v'", response["name"])
	}
	if response["comment"] != "Updated comment" {
		t.Errorf("Expected comment 'Updated comment', got '%v'", response["comment"])
	}
}

func TestUpdateAccountNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updatenf@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Nonexistent Account",
		"currency_id": %d,
		"account_type_id": %d,
		"initial_balance": 1000,
		"balance": 1000,
		"is_hidden": false,
		"show_in_reports": true,
		"opening_date": "2024-01-01T00:00:00"
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/999999999", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestUpdateAccountOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "updateowner@example.com"
	email2 := "updatehacker@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	defer testutil.CleanupUserByEmail(t, app.DB, email1)
	defer testutil.CleanupUserByEmail(t, app.DB, email2)

	ownerID := testutil.CreateTestUser(t, app, email1, password)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, ownerID, "Owner Account", currencyID, accountTypeID, 5000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	testutil.CreateTestUser(t, app, email2, password)
	hackerToken := testutil.GetAuthToken(t, app, email2, password)

	reqBody := fmt.Sprintf(`{
		"name": "Hacked Account",
		"currency_id": %d,
		"account_type_id": %d,
		"initial_balance": 1000000,
		"balance": 1000000
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d", accountID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", hackerToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 400 or 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAccountUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	reqBody := `{
		"name": "Unauthorized Update",
		"currencyId": 1,
		"accountTypeId": 1,
		"initialBalance": 1000,
		"balance": 1000
	}`

	req := httptest.NewRequest(http.MethodPut, "/accounts/1", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Archive Status Tests ====================

func TestArchiveAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "archiveaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Archive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{"accountId": %d, "isArchived": true}`, accountID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify account is archived
	var isArchived bool
	err := app.DB.QueryRow("SELECT is_archived FROM accounts WHERE id = $1", accountID).Scan(&isArchived)
	if err != nil {
		t.Fatalf("Failed to verify archive status: %v", err)
	}
	if !isArchived {
		t.Error("Expected account to be archived")
	}
}

func TestUnarchiveAccountSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "unarchiveaccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Unarchive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// First archive it
	_, _ = app.DB.Exec("UPDATE accounts SET is_archived = true WHERE id = $1", accountID)

	reqBody := fmt.Sprintf(`{"accountId": %d, "isArchived": false}`, accountID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify account is unarchived
	var isArchived bool
	err := app.DB.QueryRow("SELECT is_archived FROM accounts WHERE id = $1", accountID).Scan(&isArchived)
	if err != nil {
		t.Fatalf("Failed to verify archive status: %v", err)
	}
	if isArchived {
		t.Error("Expected account to be unarchived")
	}
}

func TestArchiveAccountOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "archiveowner@example.com"
	email2 := "archivehacker@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	defer testutil.CleanupUserByEmail(t, app.DB, email1)
	defer testutil.CleanupUserByEmail(t, app.DB, email2)

	ownerID := testutil.CreateTestUser(t, app, email1, password)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, ownerID, "Owner Account", currencyID, accountTypeID, 5000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	testutil.CreateTestUser(t, app, email2, password)
	hackerToken := testutil.GetAuthToken(t, app, email2, password)

	reqBody := fmt.Sprintf(`{"accountId": %d, "isArchived": true}`, accountID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", hackerToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestArchiveAccountUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	reqBody := `{"accountId": 1, "isArchived": true}`

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ==================== Adjust Balance Tests ====================

func TestAdjustBalanceIncreaseSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustincrease@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Adjust Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": 1100.50,
		"notes": "Test adjustment increase"
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["isAdjustment"] != true {
		t.Error("Expected isAdjustment to be true")
	}
	if response["excludeFromReports"] != true {
		t.Error("Expected excludeFromReports to be true")
	}
	if response["isIncome"] != true {
		t.Error("Expected isIncome to be true for increase")
	}
	if response["notes"] != "Test adjustment increase" {
		t.Errorf("Expected notes 'Test adjustment increase', got '%v'", response["notes"])
	}
}

func TestAdjustBalanceDecreaseSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustdecrease@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Adjust Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": 949.75
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["isAdjustment"] != true {
		t.Error("Expected isAdjustment to be true")
	}
	if response["isIncome"] != false {
		t.Error("Expected isIncome to be false for decrease")
	}
}

func TestAdjustBalanceSameBalanceFails(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustsame@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Same Balance Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": 1000.00
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceAccountNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustnf@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{
		"accountId": 999999999,
		"newBalance": 1000.00
	}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email1 := "adjustowner@example.com"
	email2 := "adjusthacker@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email1)
	testutil.CleanupUserByEmail(t, app.DB, email2)
	defer testutil.CleanupUserByEmail(t, app.DB, email1)
	defer testutil.CleanupUserByEmail(t, app.DB, email2)

	ownerID := testutil.CreateTestUser(t, app, email1, password)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, ownerID, "Owner Account", currencyID, accountTypeID, 5000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	testutil.CreateTestUser(t, app, email2, password)
	hackerToken := testutil.GetAuthToken(t, app, email2, password)

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": 999999.00
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", hackerToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	reqBody := `{
		"accountId": 1,
		"newBalance": 1000.00
	}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceNegativeBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustnegative@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Negative Balance Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": -500.00
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify balance is negative
	var balance float64
	err := app.DB.QueryRow("SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	if err != nil {
		t.Fatalf("Failed to verify balance: %v", err)
	}
	if balance != -500.00 {
		t.Errorf("Expected balance -500.00, got %f", balance)
	}
}

// ==================== Filter Tests ====================

func TestGetAccountsFilterHidden(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "filterhidden@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create visible account
	visibleID := testutil.CreateTestAccount(t, app.DB, userID, "Visible Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, visibleID)

	// Create hidden account
	hiddenID := testutil.CreateTestAccount(t, app.DB, userID, "Hidden Account", currencyID, accountTypeID, 2000.00)
	defer testutil.DeleteTestAccount(t, app.DB, hiddenID)
	_, _ = app.DB.Exec("UPDATE accounts SET is_hidden = true WHERE id = $1", hiddenID)

	// Get accounts without hidden
	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Hidden account should not be in list
	for _, acc := range response {
		if int(acc["id"].(float64)) == hiddenID {
			t.Error("Hidden account should not be in default list")
		}
	}

	// Get accounts with hidden
	req2 := httptest.NewRequest(http.MethodGet, "/accounts/?includeHidden=true", nil)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec2, req2)

	var response2 []map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	foundHidden := false
	for _, acc := range response2 {
		if int(acc["id"].(float64)) == hiddenID {
			foundHidden = true
			break
		}
	}
	if !foundHidden {
		t.Error("Hidden account should be in list when includeHidden=true")
	}
}

func TestGetAccountsFilterArchived(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "filterarchived@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create active account
	activeID := testutil.CreateTestAccount(t, app.DB, userID, "Active Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, activeID)

	// Create archived account
	archivedID := testutil.CreateTestAccount(t, app.DB, userID, "Archived Account", currencyID, accountTypeID, 2000.00)
	defer testutil.DeleteTestAccount(t, app.DB, archivedID)
	_, _ = app.DB.Exec("UPDATE accounts SET is_archived = true WHERE id = $1", archivedID)

	// Get accounts without archived
	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Archived account should not be in list
	for _, acc := range response {
		if int(acc["id"].(float64)) == archivedID {
			t.Error("Archived account should not be in default list")
		}
	}

	// Get accounts with archived
	req2 := httptest.NewRequest(http.MethodGet, "/accounts/?includeArchived=true", nil)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec2, req2)

	var response2 []map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	foundArchived := false
	for _, acc := range response2 {
		if int(acc["id"].(float64)) == archivedID {
			foundArchived = true
			break
		}
	}
	if !foundArchived {
		t.Error("Archived account should be in list when includeArchived=true")
	}
}

func TestGetAccountsArchivedOnly(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "archivedonly@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create active account
	activeID := testutil.CreateTestAccount(t, app.DB, userID, "Active Account", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, activeID)

	// Create archived account
	archivedID := testutil.CreateTestAccount(t, app.DB, userID, "Archived Account", currencyID, accountTypeID, 2000.00)
	defer testutil.DeleteTestAccount(t, app.DB, archivedID)
	_, _ = app.DB.Exec("UPDATE accounts SET is_archived = true WHERE id = $1", archivedID)

	// Get only archived accounts
	req := httptest.NewRequest(http.MethodGet, "/accounts/?archivedOnly=true", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// All accounts should be archived
	for _, acc := range response {
		if acc["isArchived"] != true {
			t.Error("All accounts should be archived when archivedOnly=true")
		}
	}
}

// ==================== Additional Account Tests ====================

func TestCreateAccountInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "invalidjson@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountWithZeroBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "zerobalance@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Zero Balance Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 0,
		"balance": 0,
		"isHidden": false
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["balance"].(float64) != 0 {
		t.Errorf("Expected balance 0, got '%v'", response["balance"])
	}
}

func TestCreateAccountWithNegativeBalance(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "negativebalance@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Negative Balance Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": -500.00,
		"balance": -500.00,
		"isHidden": false
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["balance"].(float64) != -500.00 {
		t.Errorf("Expected balance -500.00, got '%v'", response["balance"])
	}
}

func TestGetAccountDetailsInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "invalidid@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodGet, "/accounts/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deleteinvalid@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	req := httptest.NewRequest(http.MethodDelete, "/accounts/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAccountInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updateinvalid@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Update Invalid",
		"currency_id": %d,
		"account_type_id": %d,
		"initial_balance": 1000,
		"balance": 1000
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/invalid", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAccountInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updateinvalidjson@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Update", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d", accountID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustinvalidjson@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestArchiveAccountInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "archiveinvalidjson@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
}

func TestArchiveAccountNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "archivenf@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	reqBody := `{"accountId": 999999999, "isArchived": true}`

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Non-existent account should fail
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 401, 404, or 500, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetAccountTypesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/accounts/types/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 401 or 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetAccountTypesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/accounts/types/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountWithLongName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "longname@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Database has 100 char limit, so 200 chars will fail
	longName := strings.Repeat("A", 200)
	reqBody := fmt.Sprintf(`{
		"name": "%s",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000,
		"isHidden": false
	}`, longName, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Very long names exceed database column limit
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountWith100CharName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "name100@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// 100 chars should be accepted
	name100 := strings.Repeat("B", 100)
	reqBody := fmt.Sprintf(`{
		"name": "%s",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000,
		"isHidden": false
	}`, name100, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestCreateAccountWithSpecialCharacters(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "specialchars@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Рахунок на українській / 日本語アカウント",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000,
		"isHidden": false
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "Рахунок на українській / 日本語アカウント" {
		t.Errorf("Expected special character name, got '%v'", response["name"])
	}
}

func TestCreateMultipleAccounts(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "multipleaccounts@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Create first account
	reqBody1 := fmt.Sprintf(`{
		"name": "First Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000
	}`, currencyID, accountTypeID)

	req1 := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("Expected status %d for first account, got %d", http.StatusOK, rec1.Code)
	}

	// Create second account
	reqBody2 := fmt.Sprintf(`{
		"name": "Second Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 2000,
		"balance": 2000
	}`, currencyID, accountTypeID)

	req2 := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("Expected status %d for second account, got %d", http.StatusOK, rec2.Code)
	}

	// Verify both accounts exist
	req3 := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req3.Header.Set("auth-token", token)
	rec3 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec3, req3)

	var accounts []map[string]interface{}
	if err := json.Unmarshal(rec3.Body.Bytes(), &accounts); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(accounts) < 2 {
		t.Errorf("Expected at least 2 accounts, got %d", len(accounts))
	}
}

// ==================== Inactive User / Deleted Account Tests ====================

func TestCreateAccountInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "createaccount-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "Inactive User Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 1000,
		"balance": 1000,
		"isHidden": false
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetAccountsInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "getaccounts-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	// Deactivate user after getting token
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/accounts/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetAccountDetailsInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "accountdetails-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Inactive User Account", currencyID, accountTypeID, 2500.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Deactivate user after getting token and creating account
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestUpdateAccountInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "updateaccount-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Update Inactive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Deactivate user after getting token and creating account
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	reqBody := fmt.Sprintf(`{
		"name": "Updated By Inactive",
		"currency_id": %d,
		"account_type_id": %d,
		"initial_balance": 1000,
		"balance": 1000,
		"is_hidden": false,
		"show_in_reports": true,
		"opening_date": "2024-01-01T00:00:00",
		"comment": "Should fail"
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d", accountID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "deleteaccount-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Delete Inactive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Deactivate user after getting token and creating account
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestSetArchiveStatusInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "archivestatus-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Archive Inactive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Deactivate user after getting token and creating account
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	reqBody := fmt.Sprintf(`{"accountId": %d, "isArchived": true}`, accountID)

	req := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestAdjustBalanceInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "adjustbalance-inactive@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "To Adjust Inactive", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Deactivate user after getting token and creating account
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE email = $1", email)
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}

	reqBody := fmt.Sprintf(`{
		"accountId": %d,
		"newBalance": 2000.00,
		"notes": "Should fail"
	}`, accountID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/adjust-balance", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestGetAccountDetailsDeletedAccount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "accountdetails-deleted@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Deleted Account", currencyID, accountTypeID, 3000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Soft-delete the account
	_, err := app.DB.Exec("UPDATE accounts SET is_deleted = true WHERE id = $1", accountID)
	if err != nil {
		t.Fatalf("Failed to soft-delete account: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", accountID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestAccountWithEURCurrency(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "euraccount@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "EUR")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	reqBody := fmt.Sprintf(`{
		"name": "EUR Account",
		"currencyId": %d,
		"accountTypeId": %d,
		"initialBalance": 500,
		"balance": 500,
		"isHidden": false
	}`, currencyID, accountTypeID)

	req := httptest.NewRequest(http.MethodPost, "/accounts/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if int(response["currencyId"].(float64)) != currencyID {
		t.Errorf("Expected EUR currency ID %d, got '%v'", currencyID, response["currencyId"])
	}
}

// TestUnarchiveAccountClearsArchivedAt verifies that un-archiving an account
// sets archived_at to NULL in the database rather than keeping a stale timestamp.
func TestUnarchiveAccountClearsArchivedAt(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "unarchive-clear-at@example.com"
	password := "Test_password_123"

	testutil.CleanupUserByEmail(t, app.DB, email)
	defer testutil.CleanupUserByEmail(t, app.DB, email)

	userID := testutil.CreateTestUser(t, app, email, password)
	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Archive Test", currencyID, accountTypeID, 1000.00)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	// Step 1: Archive the account via the endpoint
	archiveBody := fmt.Sprintf(`{"accountId": %d, "isArchived": true}`, accountID)
	archiveReq := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(archiveBody))
	archiveReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	archiveReq.Header.Set("auth-token", token)
	archiveRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(archiveRec, archiveReq)

	if archiveRec.Code != http.StatusOK {
		t.Fatalf("Archive request failed: status %d, body: %s", archiveRec.Code, archiveRec.Body.String())
	}

	// Verify archived_at is NOT NULL after archiving
	var archivedAtAfterArchive *string
	err := app.DB.QueryRow("SELECT archived_at::text FROM accounts WHERE id = $1", accountID).Scan(&archivedAtAfterArchive)
	if err != nil {
		t.Fatalf("Failed to query archived_at: %v", err)
	}
	if archivedAtAfterArchive == nil {
		t.Fatal("Expected archived_at to be set after archiving, but got NULL")
	}

	// Step 2: Un-archive the account via the endpoint
	unarchiveBody := fmt.Sprintf(`{"accountId": %d, "isArchived": false}`, accountID)
	unarchiveReq := httptest.NewRequest(http.MethodPut, "/accounts/set-archive-status", strings.NewReader(unarchiveBody))
	unarchiveReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	unarchiveReq.Header.Set("auth-token", token)
	unarchiveRec := httptest.NewRecorder()
	app.Echo.ServeHTTP(unarchiveRec, unarchiveReq)

	if unarchiveRec.Code != http.StatusOK {
		t.Fatalf("Unarchive request failed: status %d, body: %s", unarchiveRec.Code, unarchiveRec.Body.String())
	}

	// Verify archived_at IS NULL after un-archiving
	var archivedAtAfterUnarchive *string
	err = app.DB.QueryRow("SELECT archived_at::text FROM accounts WHERE id = $1", accountID).Scan(&archivedAtAfterUnarchive)
	if err != nil {
		t.Fatalf("Failed to query archived_at: %v", err)
	}
	if archivedAtAfterUnarchive != nil {
		t.Errorf("Expected archived_at to be NULL after un-archiving, but got %q", *archivedAtAfterUnarchive)
	}

	// Also verify is_archived is false
	var isArchived bool
	err = app.DB.QueryRow("SELECT is_archived FROM accounts WHERE id = $1", accountID).Scan(&isArchived)
	if err != nil {
		t.Fatalf("Failed to query is_archived: %v", err)
	}
	if isArchived {
		t.Error("Expected is_archived to be false after un-archiving")
	}
}
