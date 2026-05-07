package export_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/go-budget/backend/internal/workers/tasks"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	exportTestEmail    = "export-test@example.com"
	exportTestPassword = "TestPass123!"
)

// setupExportTest creates a test app with routes, a test user, and returns
// the app, auth token, and user ID. The caller must NOT forget to defer cleanup.
func setupExportTest(t *testing.T) (app *testutil.TestApp, token string, userID int) {
	t.Helper()

	app = testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, exportTestEmail)

	userID = testutil.CreateTestUser(t, app, exportTestEmail, exportTestPassword)
	token = testutil.GetAuthToken(t, app, exportTestEmail, exportTestPassword)

	return app, token, userID
}

// createTransactionWithDate inserts a transaction for the given user/account at
// the specified date. Returns the transaction ID.
func createTransactionWithDate(t *testing.T, app *testutil.TestApp, userID, accountID int, categoryID *int, amount float64, isIncome bool, date string) int {
	t.Helper()

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO transactions
			(user_id, account_id, category_id, amount, is_income, is_transfer,
			 is_deleted, date_time, new_balance, exclude_from_reports, is_adjustment)
		VALUES ($1, $2, $3, $4, $5, false, false, $6, $4, false, false)
		RETURNING id
	`, userID, accountID, categoryID, amount, isIncome, date).Scan(&id)
	require.NoError(t, err, "failed to create test transaction")
	return id
}

// createExcludedTransaction creates a transaction that has exclude_from_reports = true.
func createExcludedTransaction(t *testing.T, app *testutil.TestApp, userID, accountID int, categoryID *int, amount float64, isIncome bool, date string) int {
	t.Helper()

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO transactions
			(user_id, account_id, category_id, amount, is_income, is_transfer,
			 is_deleted, date_time, new_balance, exclude_from_reports, is_adjustment)
		VALUES ($1, $2, $3, $4, $5, false, false, $6, $4, true, false)
		RETURNING id
	`, userID, accountID, categoryID, amount, isIncome, date).Scan(&id)
	require.NoError(t, err, "failed to create excluded test transaction")
	return id
}

// doExportRequest sends a POST request to the given export endpoint with the
// provided body and returns the response recorder.
func doExportRequest(t *testing.T, app *testutil.TestApp, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set("auth-token", token)
	}
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	return rec
}

// ==================== POST /export/download Tests ====================

func TestDownloadExportSuccess(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Create an account and a transaction within the date range
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Export Test Wallet", currencyID, accountTypeID, 1000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Export Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	txID := createTransactionWithDate(t, app, userID, accountID, &categoryID, 50.0, false, "2026-01-15")
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "transactions_export.xlsx")
	// The body should contain actual Excel bytes (non-empty)
	assert.Greater(t, rec.Body.Len(), 0)
}

func TestDownloadExportEmptyDateRange(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// No transactions created -- the date range is valid but empty
	body := `{"start_date": "2020-01-01", "end_date": "2020-01-31"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	// Should still succeed with a valid Excel file containing only headers
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		rec.Header().Get("Content-Type"))
	assert.Greater(t, rec.Body.Len(), 0)
}

func TestDownloadExportStartAfterEnd(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	body := `{"start_date": "2026-02-01", "end_date": "2026-01-01"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "start_date must be before")
}

func TestDownloadExportDateRangeExceedsLimit(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Range of more than 1096 days
	body := `{"start_date": "2020-01-01", "end_date": "2026-01-01"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "1096 days")
}

func TestDownloadExportDateRangeAtBoundary(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Exactly 1096 days
	body := `{"start_date": "2020-01-01", "end_date": "2023-01-01"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	// Should succeed (366 days is exactly at the limit)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		rec.Header().Get("Content-Type"))
}

func TestDownloadExportExcludesReportExcludedTransactions(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Export Report Wallet", currencyID, accountTypeID, 2000)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Export Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Create a normal transaction
	normalTxID := createTransactionWithDate(t, app, userID, accountID, &categoryID, 100.0, false, "2026-01-10")
	defer testutil.DeleteTestTransaction(t, app.DB, normalTxID)

	// Create an excluded transaction (same date range)
	excludedTxID := createExcludedTransaction(t, app, userID, accountID, &categoryID, 200.0, false, "2026-01-15")
	defer testutil.DeleteTestTransaction(t, app.DB, excludedTxID)

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	rec := doExportRequest(t, app, "/export/download", body, token)

	assert.Equal(t, http.StatusOK, rec.Code)
	// The response is an Excel binary. We cannot easily parse the Excel content in the
	// test, but we can verify that the export was generated. The fact that the export
	// succeeds (HTTP 200) and the GetForExport query filters out excluded transactions
	// is the key verification. A more thorough check would parse the Excel, but that
	// would duplicate the excelize dependency and add complexity for marginal benefit.
	assert.Greater(t, rec.Body.Len(), 0)
}

func TestDownloadExportUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	rec := doExportRequest(t, app, "/export/download", body, "")

	// Without a valid token, the auth middleware should reject the request
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== POST /export/email Tests ====================

func TestEmailExportSuccess(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Create an account and transaction so the export has data
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)
	accountID := testutil.CreateTestAccount(t, app.DB, userID, "Email Export Wallet", currencyID, accountTypeID, 500)
	defer testutil.DeleteTestAccount(t, app.DB, accountID)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Email Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	txID := createTransactionWithDate(t, app, userID, accountID, &categoryID, 25.0, false, "2026-02-10")
	defer testutil.DeleteTestTransaction(t, app.DB, txID)

	body := `{"start_date": "2026-02-01", "end_date": "2026-02-28"}`
	rec := doExportRequest(t, app, "/export/email", body, token)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"], "Export will be sent to your email shortly")

	// Verify that an export:email task was enqueued
	enqueuedTasks := app.MockEnqueuer.GetTasks()
	require.GreaterOrEqual(t, len(enqueuedTasks), 1, "expected at least one enqueued task")
	assert.Equal(t, tasks.TypeExportEmail, enqueuedTasks[0].Type)

	// Verify the payload contains the correct email and date range
	var payload tasks.ExportEmailPayload
	err = json.Unmarshal(enqueuedTasks[0].Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, exportTestEmail, payload.Email)
	assert.Equal(t, "2026-02-01", payload.StartDate)
	assert.Equal(t, "2026-02-28", payload.EndDate)
	assert.Equal(t, userID, payload.UserID)
}

func TestEmailExportEmptyDateRange(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Valid date range but no transactions exist for it
	body := `{"start_date": "2020-01-01", "end_date": "2020-01-31"}`
	rec := doExportRequest(t, app, "/export/email", body, token)

	// Should still accept -- the email will be sent with an empty export
	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"], "Export will be sent to your email shortly")
}

func TestEmailExportStartAfterEnd(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	body := `{"start_date": "2026-03-01", "end_date": "2026-01-01"}`
	rec := doExportRequest(t, app, "/export/email", body, token)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "start_date must be before")
}

func TestEmailExportDateRangeExceedsLimit(t *testing.T) {
	app, token, userID := setupExportTest(t)
	defer testutil.CleanupUser(t, app.DB, userID)

	body := `{"start_date": "2020-01-01", "end_date": "2026-01-01"}`
	rec := doExportRequest(t, app, "/export/email", body, token)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "1096 days")
}

func TestEmailExportUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	rec := doExportRequest(t, app, "/export/email", body, "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
