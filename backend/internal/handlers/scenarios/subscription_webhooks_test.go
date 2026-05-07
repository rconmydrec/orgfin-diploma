package scenarios_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookReceivesAndAcknowledges verifies that the webhook endpoint accepts
// Stripe-like payloads and returns a successful acknowledgement.
//
// The Go backend's Stripe integration is not yet fully implemented (the handler
// has a TODO for Stripe signature verification and event handling). This test
// validates the current behavior: the endpoint reads the body, checks for a
// signature header (non-strictly in non-production), and returns a success JSON.
func TestWebhookReceivesAndAcknowledges(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Simulate a checkout.session.completed webhook payload.
	webhookBody := `{
		"id": "evt_test_001",
		"type": "checkout.session.completed",
		"data": {
			"object": {
				"id": "cs_test_001",
				"customer": "cus_test_001",
				"subscription": "sub_test_001",
				"metadata": {"user_id": "999", "plan_id": "1"}
			}
		}
	}`

	rec := doRequest(t, app, http.MethodPost, "/payments/webhook", webhookBody, "")
	// Webhook should be reachable without auth (it uses Stripe signature instead).
	require.Equal(t, http.StatusOK, rec.Code, "webhook: %s", rec.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["received"])
	assert.Equal(t, true, resp["success"])
}

// TestWebhookWithStripeSignatureHeader verifies that providing a Stripe-Signature
// header does not break the webhook processing.
func TestWebhookWithStripeSignatureHeader(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	webhookBody := `{"test": "payload"}`

	req := makeRequestWithHeaders(t, http.MethodPost, "/payments/webhook", webhookBody, map[string]string{
		"Stripe-Signature": "test_signature_value",
		"Content-Type":     "application/json",
	})
	rec := executeRequest(t, app, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["received"])
}

// TestWebhookMultipleEventTypes sends multiple different webhook event types
// and verifies they are all acknowledged.
func TestWebhookMultipleEventTypes(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	eventTypes := []string{
		"checkout.session.completed",
		"customer.subscription.updated",
		"invoice.payment_failed",
		"invoice.paid",
		"subscription_schedule.completed",
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			body := fmt.Sprintf(`{
				"id": "evt_test_%s",
				"type": "%s",
				"data": {"object": {"id": "obj_test_001"}}
			}`, eventType, eventType)

			req := makeRequestWithHeaders(t, http.MethodPost, "/payments/webhook", body, map[string]string{
				"Stripe-Signature": "test_sig",
				"Content-Type":     "application/json",
			})
			rec := executeRequest(t, app, req)
			require.Equal(t, http.StatusOK, rec.Code, "event %s: %s", eventType, rec.Body.String())

			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, true, resp["received"])
		})
	}
}

// TestSubscriptionStateUpdateViaUpgradeAndWebhook simulates a full
// subscription state change: upgrade via API, then send a webhook.
//
// Scenario:
//  1. Register user, starts on free plan.
//  2. Upgrade to premium via /subscriptions/upgrade.
//  3. Verify subscription is now premium.
//  4. Send a webhook event (simulating Stripe callback).
//  5. Verify webhook acknowledged.
//  6. Verify subscription state is still premium (not broken by webhook).
func TestSubscriptionStateUpdateViaUpgradeAndWebhook(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-webhooks-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	// Get premium plans.
	rec := doRequest(t, app, http.MethodGet, "/subscriptions/plans", "", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var plans []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &plans))

	var premiumPlanID int
	for _, plan := range plans {
		if plan["planType"] == "premium" {
			premiumPlanID = int(plan["id"].(float64))
			break
		}
	}

	if premiumPlanID == 0 {
		t.Skip("No premium plan found; skipping webhook state test")
	}

	// Step 2: Upgrade.
	upgradeBody := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/upgrade", upgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "upgrade: %s", rec.Body.String())

	// Step 3: Verify premium.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "premium", statusResp["planType"])

	// Step 4: Simulate webhook event.
	webhookBody := fmt.Sprintf(`{
		"id": "evt_scenario_001",
		"type": "checkout.session.completed",
		"data": {
			"object": {
				"id": "cs_scenario_001",
				"customer": "cus_scenario_001",
				"subscription": "sub_scenario_001",
				"metadata": {"user_id": "%d", "plan_id": "%d"}
			}
		}
	}`, userID, premiumPlanID)

	req := makeRequestWithHeaders(t, http.MethodPost, "/payments/webhook", webhookBody, map[string]string{
		"Stripe-Signature": "test_signature",
		"Content-Type":     "application/json",
	})
	webhookRec := executeRequest(t, app, req)
	require.Equal(t, http.StatusOK, webhookRec.Code)

	var webhookResp map[string]interface{}
	require.NoError(t, json.Unmarshal(webhookRec.Body.Bytes(), &webhookResp))
	assert.Equal(t, true, webhookResp["success"])

	// Step 5: Verify subscription state is still premium (webhook did not break it).
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "premium", statusResp["planType"],
		"Subscription should remain premium after webhook")
}

// TestPaymentStatusAfterUpgrade verifies that /payments/status reflects the
// correct state after a subscription upgrade.
func TestPaymentStatusAfterUpgrade(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-pay-status-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	// Initial payment status.
	rec := doRequest(t, app, http.MethodGet, "/payments/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var initialPayStatus map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &initialPayStatus))
	assert.Equal(t, "free", initialPayStatus["currentPlanType"])

	// Upgrade.
	plansRec := doRequest(t, app, http.MethodGet, "/subscriptions/plans", "", "")
	require.Equal(t, http.StatusOK, plansRec.Code)

	var plans []map[string]interface{}
	require.NoError(t, json.Unmarshal(plansRec.Body.Bytes(), &plans))

	var premiumPlanID int
	for _, plan := range plans {
		if plan["planType"] == "premium" {
			premiumPlanID = int(plan["id"].(float64))
			break
		}
	}

	if premiumPlanID == 0 {
		t.Skip("No premium plan found; skipping payment status test")
	}

	upgradeBody := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/upgrade", upgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify payment status changed.
	rec = doRequest(t, app, http.MethodGet, "/payments/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var updatedPayStatus map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &updatedPayStatus))
	// The payments handler returns the plan type as stored in DB (may be uppercase),
	// whereas the subscriptions handler normalizes to lowercase. Compare case-insensitively.
	assert.Equal(t, "premium", strings.ToLower(updatedPayStatus["currentPlanType"].(string)),
		"Payment status should reflect premium after upgrade")
}
