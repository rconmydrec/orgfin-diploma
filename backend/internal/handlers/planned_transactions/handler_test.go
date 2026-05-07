package planned_transactions_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail    = "planned-tx-test@example.com"
	testPassword = "TestPass123!"
)

// Helper function to create a planned transaction with currency_id
func createTestPlannedTx(t *testing.T, app *testutil.TestApp, userID int, currencyID int, label string, isIncome bool, isRecurring bool) int {
	t.Helper()

	plannedDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO planned_transactions (user_id, currency_id, amount, label, is_income, planned_date, is_recurring, is_active)
		VALUES ($1, $2, 100.00, $3, $4, $5, $6, true)
		RETURNING id
	`, userID, currencyID, label, isIncome, plannedDate, isRecurring).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test planned transaction: %v", err)
	}
	return id
}

// Helper function to delete a planned transaction
func deleteTestPlannedTx(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM planned_transactions WHERE id = $1", id)
}

// ==================== Create Planned Transaction Tests ====================

func TestCreatePlannedTxSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 0, 14).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 250.50,
		"label": "Monthly Rent",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": false
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["id"])
	assert.Equal(t, "Monthly Rent", response["label"])
	assert.Equal(t, 250.5, response["amount"])
	assert.Equal(t, false, response["isIncome"])
	assert.Equal(t, false, response["isRecurring"])
	assert.Equal(t, true, response["isActive"])

	// Cleanup
	if id, ok := response["id"].(float64); ok {
		deleteTestPlannedTx(t, app, int(id))
	}
}

func TestCreatePlannedTxWithRecurrenceRule(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 100.00,
		"label": "Weekly Groceries",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": true,
		"recurrenceRule": {
			"frequency": "weekly",
			"interval": 1
		}
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isRecurring"])
	assert.NotNil(t, response["recurrenceRule"])

	rule := response["recurrenceRule"].(map[string]interface{})
	assert.Equal(t, "weekly", rule["frequency"])
	assert.Equal(t, float64(1), rule["interval"])

	// Cleanup
	if id, ok := response["id"].(float64); ok {
		deleteTestPlannedTx(t, app, int(id))
	}
}

func TestCreatePlannedTxIncome(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 5000.00,
		"label": "Expected Salary",
		"isIncome": true,
		"plannedDate": "%s",
		"isRecurring": true,
		"recurrenceRule": {
			"frequency": "monthly",
			"interval": 1,
			"dayOfMonth": 25
		}
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isIncome"])
	assert.Equal(t, float64(5000), response["amount"])

	// Cleanup
	if id, ok := response["id"].(float64); ok {
		deleteTestPlannedTx(t, app, int(id))
	}
}

func TestCreatePlannedTxWithNotes(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 50.00,
		"label": "Subscription",
		"notes": "Netflix annual subscription",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": false
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Netflix annual subscription", response["notes"])

	// Cleanup
	if id, ok := response["id"].(float64); ok {
		deleteTestPlannedTx(t, app, int(id))
	}
}

func TestCreatePlannedTxUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"amount": 150.00,
		"label": "Test",
		"isIncome": false,
		"plannedDate": "2024-12-31T00:00:00Z",
		"isRecurring": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== List Planned Transactions Tests ====================

func TestListPlannedTxSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create a planned transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Test Planned", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(response), 1)
}

func TestListPlannedTxEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, len(response))
}

func TestListPlannedTxUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Upcoming Occurrences Tests ====================

func TestGetUpcomingSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/upcoming/occurrences", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUpcomingWithDays(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/upcoming/occurrences?days=60", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUpcomingUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/upcoming/occurrences", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get By ID Tests ====================

func TestGetByIDSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Test Planned", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/planned-transactions/%d", ptxID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(ptxID), response["id"])
	assert.Equal(t, "Test Planned", response["label"])
}

func TestGetByIDNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetByIDOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with a planned transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "User1 Planned", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	// Create second user
	otherEmail := "other-planned-tx@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	otherUserID := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, otherUserID)

	otherToken := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	// Try to access first user's planned transaction
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/planned-transactions/%d", ptxID), nil)
	req.Header.Set("auth-token", otherToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetByIDUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Tests ====================

func TestUpdatePlannedTxSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Original Label", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	plannedDate := time.Now().AddDate(0, 0, 14).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 200.00,
		"label": "Updated Label",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": false
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/planned-transactions/%d", ptxID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(ptxID), response["id"])
	assert.Equal(t, "Updated Label", response["label"])
	assert.Equal(t, float64(200), response["amount"])
}

func TestUpdatePlannedTxWithRecurrenceRule(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Make Recurring", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 100.00,
		"label": "Now Recurring",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": true,
		"recurrenceRule": {
			"frequency": "monthly",
			"interval": 1
		}
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/planned-transactions/%d", ptxID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isRecurring"])
	assert.NotNil(t, response["recurrenceRule"])
}

func TestUpdatePlannedTxChangeIsActive(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Deactivate Me", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := fmt.Sprintf(`{
		"amount": 100.00,
		"label": "Deactivate Me",
		"isIncome": false,
		"plannedDate": "%s",
		"isRecurring": false,
		"isActive": false
	}`, plannedDate)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/planned-transactions/%d", ptxID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["isActive"])
}

func TestUpdatePlannedTxNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := `{
		"amount": 100.00,
		"label": "Test",
		"isIncome": false,
		"plannedDate": "` + plannedDate + `",
		"isRecurring": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/planned-transactions/999999", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdatePlannedTxUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"amount": 100.00,
		"label": "Test",
		"isIncome": false,
		"plannedDate": "2024-12-31T00:00:00Z",
		"isRecurring": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/planned-transactions/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Delete Tests ====================

func TestDeletePlannedTxSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "To Delete", false, false)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/planned-transactions/%d", ptxID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["deleted"])
}

func TestDeletePlannedTxNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/planned-transactions/999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeletePlannedTxUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/planned-transactions/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Planned Transaction Tests ====================

func TestDeletePlannedTxOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with planned transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "User1 Planned", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	// Create second user
	otherEmail := "other-del-planned@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	otherUserID := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, otherUserID)

	otherToken := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/planned-transactions/%d", ptxID), nil)
	req.Header.Set("auth-token", otherToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetByIDInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeletePlannedTxInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/planned-transactions/invalid", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdatePlannedTxInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"amount": 100.00,
		"label": "Test",
		"isIncome": false,
		"plannedDate": "2024-12-31T00:00:00Z",
		"isRecurring": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/planned-transactions/invalid", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListPlannedTxInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUpcomingInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/upcoming/occurrences", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListPlannedTxWithRecurring(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create recurring planned transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Recurring Payment", false, true)
	defer deleteTestPlannedTx(t, app, ptxID)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Find the recurring transaction
	found := false
	for _, ptx := range response {
		if ptx["label"] == "Recurring Payment" {
			found = true
			assert.Equal(t, true, ptx["isRecurring"])
			break
		}
	}
	assert.True(t, found, "Should find the recurring planned transaction")
}

func TestListPlannedTxIncome(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create income planned transaction
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "Expected Salary", true, true)
	defer deleteTestPlannedTx(t, app, ptxID)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Find the income transaction
	found := false
	for _, ptx := range response {
		if ptx["label"] == "Expected Salary" {
			found = true
			assert.Equal(t, true, ptx["isIncome"])
			break
		}
	}
	assert.True(t, found, "Should find the income planned transaction")
}

func TestCreatePlannedTxInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/planned-transactions/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdatePlannedTxInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "To Update", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/planned-transactions/%d", ptxID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdatePlannedTxOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with planned transaction
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	ptxID := createTestPlannedTx(t, app, userID, currencyID, "User1 Planned", false, false)
	defer deleteTestPlannedTx(t, app, ptxID)

	// Create second user
	otherEmail := "other-upd-planned@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	otherUserID := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, otherUserID)

	otherToken := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	plannedDate := time.Now().AddDate(0, 0, 7).Format(time.RFC3339)

	body := `{
		"amount": 100.00,
		"label": "Hacked",
		"isIncome": false,
		"plannedDate": "` + plannedDate + `",
		"isRecurring": false
	}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/planned-transactions/%d", ptxID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", otherToken)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetUpcomingWithInvalidDays(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Negative days
	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/upcoming/occurrences?days=-10", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should still work (service may normalize or reject)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestListMultiplePlannedTx(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	// Create multiple planned transactions
	ptxID1 := createTestPlannedTx(t, app, userID, currencyID, "Rent", false, true)
	defer deleteTestPlannedTx(t, app, ptxID1)

	ptxID2 := createTestPlannedTx(t, app, userID, currencyID, "Salary", true, true)
	defer deleteTestPlannedTx(t, app, ptxID2)

	ptxID3 := createTestPlannedTx(t, app, userID, currencyID, "Gym", false, true)
	defer deleteTestPlannedTx(t, app, ptxID3)

	req := httptest.NewRequest(http.MethodGet, "/planned-transactions/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(response), 3)
}
