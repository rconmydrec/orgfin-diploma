package payments_test

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
	testEmail    = "payments-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Create Checkout Tests ====================

func TestCreateCheckoutInvalidPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"planId": 999999}`

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateCheckoutMissingPlanID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCheckoutUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"planId": 1}`

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Create Portal Tests ====================

func TestCreatePortalNoCustomer(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPost, "/payments/portal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// User without a Stripe customer should get 400
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePortalUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodPost, "/payments/portal", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Status Tests ====================

func TestGetPaymentStatusSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["hasStripeCustomer"])
	assert.NotNil(t, response["currentPlanType"])
}

func TestGetPaymentStatusUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/payments/status", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Session Status Tests ====================

func TestGetSessionStatusSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Configure the stub to return the correct user_id in session metadata.
	app.StubPaymentProv.SessionUserID = fmt.Sprintf("%d", userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/session/test-session-123", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-session-123", response["sessionId"])
}

func TestGetSessionStatusUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/payments/session/test-session-123", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Upgrade Price Tests ====================

func TestGetUpgradePriceInvalidPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/upgrade-price/999999", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetUpgradePriceUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/payments/upgrade-price/1", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Change Plan Tests ====================

func TestChangePlanInvalidPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"planId": 999999}`

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Service checks subscription first; user without subscription gets 500
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestChangePlanUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"planId": 1}`

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Cancel Scheduled Change Tests ====================

func TestCancelScheduledChangeNoSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodPost, "/payments/cancel-scheduled-change", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// User without subscription gets an internal error from the service
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCancelScheduledChangeUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodPost, "/payments/cancel-scheduled-change", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Webhook Tests ====================

func TestWebhookSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Webhook doesn't require auth
	body := `{"type": "test.event"}`

	req := httptest.NewRequest(http.MethodPost, "/payments/webhook", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["received"])
}

// ==================== Additional Payment Tests ====================

func TestCreateCheckoutInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestChangePlanMissingPlanID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestChangePlanInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetUpgradePriceInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/upgrade-price/not-a-number", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetSessionStatusDifferentSession(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Configure the stub to return the correct user_id in session metadata.
	app.StubPaymentProv.SessionUserID = fmt.Sprintf("%d", userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/session/another-session-456", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "another-session-456", response["sessionId"])
}

func TestWebhookEmptyBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodPost, "/payments/webhook", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetPaymentStatusReturnsCurrentPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/payments/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// By default user should be on free plan
	assert.Equal(t, "free", response["currentPlanType"])
}

// ==================== Additional Success Path Tests ====================

// Helper function to create a test subscription plan
func createTestPlan(t *testing.T, app *testutil.TestApp, planType string, name string) int {
	t.Helper()
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
	_, _ = app.DB.Exec("DELETE FROM payment_provider_subscriptions WHERE subscription_id = $1", id)
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE id = $1", id)
}

// createTestPlanPrice creates a plan_prices record for a given plan and provider.
func createTestPlanPrice(t *testing.T, app *testutil.TestApp, planID int, providerType string, externalPriceID string) int {
	t.Helper()
	var id int
	err := app.DB.QueryRow(`
		INSERT INTO plan_prices (plan_id, provider_type, external_price_id, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id
	`, planID, providerType, externalPriceID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test plan price: %v", err)
	}
	return id
}

func deleteTestPlanPrice(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM plan_prices WHERE id = $1", id)
}

// createTestProviderSubscription creates a payment_provider_subscriptions record.
func createTestProviderSubscription(t *testing.T, app *testutil.TestApp, subID int, customerID string, subscriptionID *string, scheduleID *string) int {
	t.Helper()
	var id int
	err := app.DB.QueryRow(`
		INSERT INTO payment_provider_subscriptions (subscription_id, provider_type, external_customer_id, external_subscription_id, external_schedule_id, last_payment_failed)
		VALUES ($1, 'STRIPE', $2, $3, $4, false)
		RETURNING id
	`, subID, customerID, subscriptionID, scheduleID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test provider subscription: %v", err)
	}
	return id
}

func deleteTestProviderSubscription(t *testing.T, app *testutil.TestApp, id int) {
	t.Helper()
	_, _ = app.DB.Exec("DELETE FROM payment_provider_subscriptions WHERE id = $1", id)
}

func TestCreateCheckoutWithValidPlanNoPlanPrice(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "checkout-valid@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create a premium plan (without Stripe price - service will return ErrInvalidPrice)
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Checkout")
	defer deleteTestPlan(t, app, planID)

	body := fmt.Sprintf(`{"planId": %d}`, planID)

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Plan exists but has no Stripe price configured, so service returns ErrInvalidPrice -> 400
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreatePortalWithReturnURL(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "portal-return@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Portal requires a subscription with a Stripe customer ID
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Portal")
	defer deleteTestPlan(t, app, planID)

	subID := createTestSubscription(t, app, userID, planID)
	defer deleteTestSubscription(t, app, subID)

	provSubID := createTestProviderSubscription(t, app, subID, "cus_stub_portal", nil, nil)
	defer deleteTestProviderSubscription(t, app, provSubID)

	body := `{"returnUrl": "http://localhost:5173/settings"}`

	req := httptest.NewRequest(http.MethodPost, "/payments/portal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["portalUrl"])
}

func TestGetUpgradePriceWithValidPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "upgrade-price@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create a premium plan
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Price")
	defer deleteTestPlan(t, app, planID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/payments/upgrade-price/%d", planID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(planID), response["planId"])
	assert.Equal(t, "Test Premium Price", response["planName"])
	assert.NotNil(t, response["originalPriceCents"])
	assert.NotNil(t, response["finalPriceCents"])
}

func TestChangePlanWithValidPlanNoSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "change-plan-nosub@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create a premium plan
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Change")
	defer deleteTestPlan(t, app, planID)

	body := fmt.Sprintf(`{"planId": %d}`, planID)

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Without subscription, the service returns an error (user needs to checkout first)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestChangePlanWithExistingSubscriptionNoStripe(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "change-plan-withsub@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plans (without Stripe prices)
	freePlanID := createTestPlan(t, app, "FREE", "Test Free Change")
	defer deleteTestPlan(t, app, freePlanID)

	premiumPlanID := createTestPlan(t, app, "PREMIUM", "Test Premium Change Sub")
	defer deleteTestPlan(t, app, premiumPlanID)

	// Create existing subscription without provider subscription
	subID := createTestSubscription(t, app, userID, freePlanID)
	defer deleteTestSubscription(t, app, subID)

	body := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)

	req := httptest.NewRequest(http.MethodPost, "/payments/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Subscription exists but has no Stripe subscription -> 400
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancelScheduledChangeNoPendingChange(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "cancel-nopending@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create subscription without pending change
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium No Pending")
	defer deleteTestPlan(t, app, planID)

	subID := createTestSubscription(t, app, userID, planID)
	defer deleteTestSubscription(t, app, subID)

	req := httptest.NewRequest(http.MethodPost, "/payments/cancel-scheduled-change", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should return bad request because no pending change exists
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancelScheduledChangeSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "cancel-pending@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plans
	premiumPlanID := createTestPlan(t, app, "PREMIUM", "Test Premium Cancel")
	defer deleteTestPlan(t, app, premiumPlanID)

	freePlanID := createTestPlan(t, app, "FREE", "Test Free Cancel")
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

	// Create provider subscription with schedule ID (required for cancel)
	scheduleID := "sub_sched_stub_cancel"
	provSubID := createTestProviderSubscription(t, app, subID, "cus_stub_cancel", nil, &scheduleID)
	defer deleteTestProviderSubscription(t, app, provSubID)

	req := httptest.NewRequest(http.MethodPost, "/payments/cancel-scheduled-change", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Scheduled change canceled", response["message"])
}

func TestGetPaymentStatusWithSubscription(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "payment-status-sub@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	// Create plan and subscription
	planID := createTestPlan(t, app, "PREMIUM", "Test Premium Status")
	defer deleteTestPlan(t, app, planID)

	subID := createTestSubscription(t, app, userID, planID)
	defer deleteTestSubscription(t, app, subID)

	req := httptest.NewRequest(http.MethodGet, "/payments/status", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should reflect the subscription plan type (lowercased by the repository)
	assert.Equal(t, "premium", response["currentPlanType"])
}
