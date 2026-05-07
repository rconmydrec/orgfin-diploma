package budgets_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail    = "budget-test@example.com"
	testPassword = "TestPass123!"
)

// Helper function to create a test budget
func createTestBudget(t *testing.T, app *testutil.TestApp, userID int, name string, currencyID int) int {
	t.Helper()

	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO budgets (user_id, name, currency_id, target_amount, collected_amount, period, repeat, start_date, end_date, included_categories, is_archived)
		VALUES ($1, $2, $3, 500.00, 0, 'MONTHLY', false, $4, $5, '', false)
		RETURNING id
	`, userID, name, currencyID, startDate, endDate).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test budget: %v", err)
	}
	return id
}

// Helper function to delete a test budget
func deleteTestBudget(t *testing.T, app *testutil.TestApp, budgetID int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM budgets WHERE id = $1", budgetID)
}

// ==================== Create Budget Tests ====================

func TestCreateBudgetSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Monthly Food Budget",
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d]
	}`, currencyID, startDate, endDate, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Monthly Food Budget", response["name"])
	assert.Equal(t, float64(500), response["targetAmount"])
}

func TestCreateBudgetDateOnlyFormat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Groceries", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{
		"name": "Date Only Budget",
		"currencyId": %d,
		"targetAmount": 300.00,
		"period": "MONTHLY",
		"repeat": true,
		"startDate": "2026-02-27",
		"endDate": "2026-03-27",
		"categories": [%d]
	}`, currencyID, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Date Only Budget", response["name"])
	assert.Equal(t, float64(300), response["targetAmount"])
	assert.Equal(t, true, response["repeat"])
}

func TestCreateBudgetLowercasePeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	body := fmt.Sprintf(`{
		"name": "Lowercase Period Budget",
		"currencyId": %d,
		"targetAmount": 100.00,
		"period": "daily",
		"repeat": true,
		"startDate": "2026-02-27",
		"endDate": "2026-02-27",
		"categories": []
	}`, currencyID)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Lowercase Period Budget", response["name"])
	assert.Equal(t, "daily", response["period"])
}

func TestUpdateBudgetLowercasePeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Original Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Updated Lowercase Period",
		"currencyId": %d,
		"targetAmount": 200.00,
		"period": "weekly",
		"repeat": false,
		"startDate": "2026-03-01",
		"endDate": "2026-03-07",
		"categories": []
	}`, budgetID, currencyID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Lowercase Period", response["name"])
	assert.Equal(t, "weekly", response["period"])
}

func TestCreateBudgetInvalidCurrency(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Invalid Budget",
		"currencyId": 999999,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateBudgetUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Unauthorized Budget",
		"currencyId": 1,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Budgets Tests ====================

func TestGetBudgetsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Test Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	req := httptest.NewRequest(http.MethodGet, "/budgets/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var budgets []map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &budgets)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(budgets), 1)
}

func TestGetBudgetsFilterActive(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Active Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	req := httptest.NewRequest(http.MethodGet, "/budgets/?include=active", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var budgets []map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &budgets)
	require.NoError(t, err)

	// All returned budgets should not be archived
	for _, b := range budgets {
		assert.Equal(t, false, b["isArchived"])
	}
}

func TestGetBudgetsEmptyList(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "no-budgets@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/budgets/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var budgets []map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &budgets)
	require.NoError(t, err)

	assert.Empty(t, budgets)
}

func TestGetBudgetsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/budgets/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Budget Tests ====================

func TestUpdateBudgetDateOnlyFormat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Original Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Updated With Date Only",
		"currencyId": %d,
		"targetAmount": 900.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2026-03-01",
		"endDate": "2026-03-31",
		"categories": []
	}`, budgetID, currencyID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated With Date Only", response["name"])
	assert.Equal(t, float64(900), response["targetAmount"])
}

func TestUpdateBudgetSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Original Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Updated Budget",
		"currencyId": %d,
		"targetAmount": 750.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, budgetID, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Budget", response["name"])
	assert.Equal(t, float64(750), response["targetAmount"])
}

func TestUpdateBudgetNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"id": 999999,
		"name": "Nonexistent",
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPut, "/budgets/999999/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateBudgetOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with budget
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID1, "User1 Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	// Create second user
	otherEmail := "other-budget@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Hacked Budget",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, budgetID, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ==================== Delete Budget Tests ====================

func TestDeleteBudgetSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "To Delete", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/budgets/%d/", budgetID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteBudgetNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/999999/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteBudgetOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with budget
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID1, "User1 Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	// Create second user
	otherEmail := "other-del-budget@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/budgets/%d/", budgetID), nil)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ==================== Archive Budget Tests ====================

func TestArchiveBudgetSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "To Archive", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/archive/", budgetID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestArchiveBudgetNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodPut, "/budgets/999999/archive/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestArchiveBudgetOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with budget
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID1, "User1 Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	// Create second user
	otherEmail := "other-arc-budget@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/archive/", budgetID), nil)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ==================== Additional Budget Tests ====================

func TestCreateBudgetMissingName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateBudgetWithRepeat(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Repeating Budget",
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": true,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["repeat"])
}

func TestCreateBudgetWeeklyPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Weekly Budget",
		"currencyId": %d,
		"targetAmount": 100.00,
		"period": "WEEKLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "weekly", response["period"])
}

func TestCreateBudgetYearlyPeriod(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(1, 0, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Yearly Budget",
		"currencyId": %d,
		"targetAmount": 5000.00,
		"period": "YEARLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "yearly", response["period"])
}

func TestCreateBudgetWithComment(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Budget with Comment",
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [],
		"comment": "This is a test budget"
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "This is a test budget", response["comment"])
}

func TestGetBudgetsFilterArchived(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	budgetID := createTestBudget(t, app, userID, "Active Budget", currencyID)
	defer deleteTestBudget(t, app, budgetID)

	// Archive the budget
	_, err := app.DB.Exec("UPDATE budgets SET is_archived = true WHERE id = $1", budgetID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/budgets/?include=archived", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var budgets []map[string]any
	err = json.Unmarshal(rec.Body.Bytes(), &budgets)
	require.NoError(t, err)

	// Should include the archived budget
	found := false
	for _, b := range budgets {
		if int(b["id"].(float64)) == budgetID {
			found = true
			assert.Equal(t, true, b["isArchived"])
		}
	}
	assert.True(t, found, "Archived budget should be in the list")
}

func TestUpdateBudgetInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"name": "Updated",
		"currencyId": %d,
		"targetAmount": 500.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": []
	}`, currencyID, startDate, endDate)

	req := httptest.NewRequest(http.MethodPut, "/budgets/not-a-number/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteBudgetInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/budgets/invalid/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestArchiveBudgetInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodPut, "/budgets/invalid/archive/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateBudgetUnauthorized2(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/budgets/1/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestArchiveBudgetUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodPut, "/budgets/1/archive/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateBudgetUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"name": "Test",
		"currencyId": 1,
		"targetAmount": 100,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-01-31T00:00:00Z",
		"categories": []
	}`

	req := httptest.NewRequest(http.MethodPut, "/budgets/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Recalculation on Create/Update Tests ====================
//
// These tests verify that POST /budgets/add/ and PUT /budgets/:id/ recalculate
// the budget's collected_amount from existing matching transactions
// synchronously after persistence. The recalculated value must be reflected in
// the HTTP response.

// seedBudgetExchangeRates inserts a USD-base exchange rate for the given date
// so that ConvertAmount can resolve USD->EUR (and back) during recalculation.
// The cleanup is registered with t.Cleanup.
func seedBudgetExchangeRates(t *testing.T, app *testutil.TestApp, date time.Time) {
	t.Helper()
	rate := &models.ExchangeRateHistory{
		Rates: map[string]float64{
			"USD": 1.0,
			"EUR": 0.5, // simple ratio for predictable conversion: 100 USD -> 50 EUR
		},
		ActualDate:       date,
		BaseCurrencyCode: "USD",
		ServiceName:      "test-budget-recalc",
	}
	require.NoError(t, app.ExchangeRateRepo.SaveRate(rate))
	t.Cleanup(func() {
		_, _ = app.DB.Exec(
			"DELETE FROM exchange_rates WHERE service_name = 'test-budget-recalc' AND actual_date = $1",
			date.Format("2006-01-02"),
		)
	})
}

// setBudgetTransactionDate updates a transaction's date_time so that it falls
// within a known budget window during integration tests.
func setBudgetTransactionDate(t *testing.T, app *testutil.TestApp, txID int, when time.Time) {
	t.Helper()
	_, err := app.DB.Exec(
		"UPDATE transactions SET date_time = $1 WHERE id = $2",
		when, txID,
	)
	require.NoError(t, err)
}

func TestCreateBudgetRecalculatesFromExistingTransactions(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "budget-recalc-create@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Recalc Acc", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Pre-existing matching transactions in USD.
	tx1 := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &categoryID, 75.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx1)
	tx2 := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &categoryID, 25.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx2)

	// Anchor transactions to today so they fall within the budget window.
	now := time.Now()
	setBudgetTransactionDate(t, app, tx1, now)
	setBudgetTransactionDate(t, app, tx2, now)

	startDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := now.AddDate(0, 0, 30).Format("2006-01-02")

	body := fmt.Sprintf(`{
		"name": "Budget With Pre-Existing Transactions",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d]
	}`, currencyID, startDate, endDate, categoryID)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	// 75 + 25 = 100 USD; budget is in USD so no conversion.
	assert.Equal(t, float64(100), response["collectedAmount"],
		"collectedAmount should reflect summed pre-existing transactions; full body: %s", rec.Body.String())
}

func TestUpdateBudgetAddCategoryIncreasesCollectedAmount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "budget-recalc-update-add@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Acc", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	catA := testutil.CreateTestCategory(t, app.DB, userID, "Cat A", false)
	defer testutil.DeleteTestCategory(t, app.DB, catA)
	catB := testutil.CreateTestCategory(t, app.DB, userID, "Cat B", false)
	defer testutil.DeleteTestCategory(t, app.DB, catB)

	now := time.Now()
	txA := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &catA, 40.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txA)
	setBudgetTransactionDate(t, app, txA, now)
	txB := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &catB, 60.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txB)
	setBudgetTransactionDate(t, app, txB, now)

	startDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := now.AddDate(0, 0, 30).Format("2006-01-02")

	// Step 1: create budget with only catA.
	createBody := fmt.Sprintf(`{
		"name": "Update-Add Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d]
	}`, currencyID, startDate, endDate, catA)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(createBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, float64(40), created["collectedAmount"])
	budgetID := int(created["id"].(float64))
	defer deleteTestBudget(t, app, budgetID)

	// Step 2: update to add catB.
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"name": "Update-Add Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d, %d]
	}`, budgetID, currencyID, startDate, endDate, catA, catB)

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(updateBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec = httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	// 40 + 60 = 100
	assert.Equal(t, float64(100), updated["collectedAmount"],
		"collectedAmount should grow when a category with matching transactions is added; full body: %s", rec.Body.String())
}

func TestUpdateBudgetRemoveCategoryDecreasesCollectedAmount(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "budget-recalc-update-remove@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Acc", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	catA := testutil.CreateTestCategory(t, app.DB, userID, "Cat A", false)
	defer testutil.DeleteTestCategory(t, app.DB, catA)
	catB := testutil.CreateTestCategory(t, app.DB, userID, "Cat B", false)
	defer testutil.DeleteTestCategory(t, app.DB, catB)

	now := time.Now()
	txA := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &catA, 30.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txA)
	setBudgetTransactionDate(t, app, txA, now)
	txB := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &catB, 70.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txB)
	setBudgetTransactionDate(t, app, txB, now)

	startDate := now.AddDate(0, 0, -1).Format("2006-01-02")
	endDate := now.AddDate(0, 0, 30).Format("2006-01-02")

	// Step 1: create budget with both categories.
	createBody := fmt.Sprintf(`{
		"name": "Update-Remove Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d, %d]
	}`, currencyID, startDate, endDate, catA, catB)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(createBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, float64(100), created["collectedAmount"])
	budgetID := int(created["id"].(float64))
	defer deleteTestBudget(t, app, budgetID)

	// Step 2: update to keep only catA (drop catB).
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"name": "Update-Remove Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "%s",
		"endDate": "%s",
		"categories": [%d]
	}`, budgetID, currencyID, startDate, endDate, catA)

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(updateBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec = httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	// Only catA remains -> 30
	assert.Equal(t, float64(30), updated["collectedAmount"],
		"collectedAmount should shrink when a category is removed; full body: %s", rec.Body.String())
}

func TestUpdateBudgetChangeWindowRecalculates(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "budget-recalc-update-window@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Acc", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	cat := testutil.CreateTestCategory(t, app.DB, userID, "Win", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat)

	// One transaction inside the initial window, one outside it but inside the
	// updated window.
	insideDate := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	outsideDate := time.Date(2025, 7, 15, 12, 0, 0, 0, time.UTC)

	txInside := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat, 10.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txInside)
	setBudgetTransactionDate(t, app, txInside, insideDate)

	txOutside := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat, 99.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, txOutside)
	setBudgetTransactionDate(t, app, txOutside, outsideDate)

	// Step 1: create budget covering June only.
	createBody := fmt.Sprintf(`{
		"name": "Window Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2025-06-01",
		"endDate": "2025-06-30",
		"categories": [%d]
	}`, currencyID, cat)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(createBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, float64(10), created["collectedAmount"], "June-only window matches only the inside transaction")
	budgetID := int(created["id"].(float64))
	defer deleteTestBudget(t, app, budgetID)

	// Step 2: extend the window to cover July as well.
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"name": "Window Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2025-06-01",
		"endDate": "2025-07-31",
		"categories": [%d]
	}`, budgetID, currencyID, cat)

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(updateBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec = httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	// Both transactions now match: 10 + 99 = 109
	assert.Equal(t, float64(109), updated["collectedAmount"],
		"extended window should pick up additional transactions; full body: %s", rec.Body.String())
}

func TestUpdateBudgetChangeCurrencyRecalculatesWithConversion(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := "budget-recalc-update-currency@example.com"
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, testPassword)
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")
	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	// USD-denominated account so transactions are reported in USD.
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "USD Acc", usdID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	cat := testutil.CreateTestCategory(t, app.DB, userID, "FX", false)
	defer testutil.DeleteTestCategory(t, app.DB, cat)

	// Anchor the transaction to a specific date so the seeded exchange rate
	// applies to it.
	txDate := time.Date(2025, 5, 15, 12, 0, 0, 0, time.UTC)
	tx := testutil.CreateTestTransaction(t, app.DB, userID, accountID, &cat, 100.00, false)
	defer testutil.DeleteTestTransaction(t, app.DB, tx)
	setBudgetTransactionDate(t, app, tx, txDate)

	// Seed USD->EUR exchange rate effective on the transaction date.
	seedBudgetExchangeRates(t, app, txDate)

	// Step 1: create the budget in USD.
	createBody := fmt.Sprintf(`{
		"name": "FX Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2025-05-01",
		"endDate": "2025-05-31",
		"categories": [%d]
	}`, usdID, cat)

	req := httptest.NewRequest(http.MethodPost, "/budgets/add/", strings.NewReader(createBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, float64(100), created["collectedAmount"], "USD budget: no conversion needed")
	budgetID := int(created["id"].(float64))
	defer deleteTestBudget(t, app, budgetID)

	// Step 2: switch the budget to EUR.
	updateBody := fmt.Sprintf(`{
		"id": %d,
		"name": "FX Test",
		"currencyId": %d,
		"targetAmount": 1000.00,
		"period": "MONTHLY",
		"repeat": false,
		"startDate": "2025-05-01",
		"endDate": "2025-05-31",
		"categories": [%d]
	}`, budgetID, eurID, cat)

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/budgets/%d/", budgetID), strings.NewReader(updateBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec = httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var updated map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updated))
	// 100 USD * 0.5 (USD->EUR) = 50 EUR
	assert.Equal(t, float64(50), updated["collectedAmount"],
		"EUR budget: USD transaction should be converted; full body: %s", rec.Body.String())
}
