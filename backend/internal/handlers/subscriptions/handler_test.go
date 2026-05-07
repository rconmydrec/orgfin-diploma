package subscriptions_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail    = "subscriptions-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Get Plans Tests ====================

func TestGetPlansSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Plans endpoint is public
	req := httptest.NewRequest(http.MethodGet, "/subscriptions/plans", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
}

// ==================== Get Status Tests ====================

func TestGetStatusSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["planType"])
	assert.NotNil(t, response["isActive"])
}

func TestGetStatusFreeUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// New user should be on free plan
	assert.Equal(t, "free", response["planType"])
	assert.Equal(t, true, response["isActive"])
	assert.NotNil(t, response["limits"])
}

func TestGetStatusUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Upgrade Tests ====================

func TestUpgradeInvalidPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"planId": 999999}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpgradeMissingPlanID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpgradeUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"planId": 1}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Downgrade Tests ====================

func TestDowngradeNoSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"accountIds": []}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// User without subscription is effectively on the free plan already
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDowngradeUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"accountIds": []}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Subscription Tests ====================

func TestGetPlansReturnsArray(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/plans", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify it returns an array (even if empty)
	var response []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
}

func TestGetStatusHasLimits(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Free user should have limits
	limits := response["limits"].(map[string]interface{})
	assert.NotNil(t, limits["accounts"])
	assert.NotNil(t, limits["budgets"])
}

func TestUpgradeInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDowngradeInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDowngradeWithAccountIds(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"accountIds": [1, 2], "budgetId": 1}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// No subscription means already on free plan
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetStatusIsActive(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Free plan should be active
	assert.Equal(t, true, response["isActive"])
}

// Helper function to create a test subscription plan
func createTestPlan(t *testing.T, app *testutil.TestApp, planType string, name string) int {
	t.Helper()
	// Get USD currency ID
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	var id int
	err := app.DB.QueryRow(`
		INSERT INTO subscription_plans (name, plan_type, price, currency_id, is_featured, sort_order, is_active, is_deleted)
		VALUES ($1, $2, 9.99, $3, false, 99, true, false)
		RETURNING id
	`, name, planType, currencyID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test plan: %v", err)
	}
	return id
}

func deleteTestPlan(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM subscription_plans WHERE id = $1", id)
}

func createTestSubscription(t *testing.T, app *testutil.TestApp, userID, planID int) int {
	t.Helper()
	var id int
	err := app.DB.QueryRow(`
		INSERT INTO subscriptions (user_id, plan_id, is_active, subscribed_at)
		VALUES ($1, $2, true, NOW())
		RETURNING id
	`, userID, planID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}
	return id
}

func deleteTestSubscription(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE id = $1", id)
}

func TestGetStatusWithPremiumSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create a premium plan (DB enum requires uppercase)
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium")
	defer deleteTestPlan(t, app, planID)

	// Create subscription for user
	subID := createTestSubscription(t, app, userID, planID)
	defer deleteTestSubscription(t, app, subID)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Response uses normalized lowercase plan type
	assert.Equal(t, "premium", response["planType"])
	assert.Equal(t, true, response["isActive"])
	assert.NotNil(t, response["planId"])
}

func TestGetStatusWithTrialSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "trial-sub-test@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create a trial plan (DB enum requires uppercase)
	planID := createTestPlan(t, app, "TRIAL", "Test Trial")
	defer deleteTestPlan(t, app, planID)

	// Create trial subscription with trial_ends_at in the future
	var subID int
	err := app.DB.QueryRow(`
		INSERT INTO subscriptions (user_id, plan_id, is_active, trial_started_at, trial_ends_at)
		VALUES ($1, $2, true, NOW(), NOW() + INTERVAL '7 days')
		RETURNING id
	`, userID, planID).Scan(&subID)
	require.NoError(t, err)
	defer deleteTestSubscription(t, app, subID)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "trial", response["planType"])
	assert.NotNil(t, response["trialDaysRemaining"])
}

func TestUpgradeToExistingPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "upgrade-test@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create a premium plan (DB enum requires uppercase)
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Plan")
	defer deleteTestPlan(t, app, planID)

	body := fmt.Sprintf(`{"planId": %d}`, planID)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["message"], "Test Premium Plan")
	assert.NotNil(t, response["subscription"])

	// Cleanup created subscription
	app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)
}

func TestUpgradeExistingSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "upgrade-existing@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plans (DB enum requires uppercase)
	freePlanID := createTestPlan(t, app, "FREE", "Test Free")
	defer deleteTestPlan(t, app, freePlanID)

	premiumPlanID := createTestPlan(t, app, "PREMIUM", "Test Premium Upgrade")
	defer deleteTestPlan(t, app, premiumPlanID)

	// Create initial subscription on free plan
	subID := createTestSubscription(t, app, userID, freePlanID)
	defer deleteTestSubscription(t, app, subID)

	// Upgrade to premium
	body := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["message"], "Test Premium Upgrade")
}

func TestDowngradeWithSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "downgrade-test@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plans (DB enum requires uppercase)
	premiumPlanID := createTestPlan(t, app, "PREMIUM", "Test Premium Downgrade")
	defer deleteTestPlan(t, app, premiumPlanID)

	freePlanID := createTestPlan(t, app, "FREE", "Test Free Downgrade")
	defer deleteTestPlan(t, app, freePlanID)

	// Create subscription on premium plan
	subID := createTestSubscription(t, app, userID, premiumPlanID)
	defer deleteTestSubscription(t, app, subID)

	body := `{"accountIds": [1, 2]}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Without a Stripe subscription, the downgrade is applied immediately
	assert.Contains(t, response["message"], "Downgraded to free plan immediately")
	assert.Equal(t, false, response["pendingDowngrade"])
}

func TestGetStatusWithPendingDowngrade(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "pending-downgrade@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plans (DB enum requires uppercase)
	premiumPlanID := createTestPlan(t, app, "PREMIUM", "Premium Pending")
	defer deleteTestPlan(t, app, premiumPlanID)

	freePlanID := createTestPlan(t, app, "FREE", "Free Pending")
	defer deleteTestPlan(t, app, freePlanID)

	// Create subscription with pending downgrade
	var subID int
	err := app.DB.QueryRow(`
		INSERT INTO subscriptions (user_id, plan_id, is_active, pending_plan_id)
		VALUES ($1, $2, true, $3)
		RETURNING id
	`, userID, premiumPlanID, freePlanID).Scan(&subID)
	require.NoError(t, err)
	defer deleteTestSubscription(t, app, subID)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "premium", response["planType"])
	assert.NotNil(t, response["pendingPlanId"])
	assert.NotNil(t, response["pendingPlanName"])
}
