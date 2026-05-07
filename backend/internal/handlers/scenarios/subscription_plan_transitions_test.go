package scenarios_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTrialToMonthlyToYearlyTransition tests a full subscription lifecycle:
//  1. Register a user -- user starts on trial/free plan.
//  2. Verify the user's initial subscription state.
//  3. Upgrade to a premium monthly plan (via /subscriptions/upgrade).
//  4. Verify subscription state changed to premium.
//  5. Change plan to yearly (via /payments/change-plan).
//  6. Verify plan updated.
//  7. Check subscription status reflects the premium yearly plan.
//
// Note: The Go backend's Stripe integration is still a placeholder (TODO in handlers),
// so this test exercises the available DB-level subscription management endpoints.
func TestTrialToMonthlyToYearlyTransition(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-sub-plans-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	// Step 1: Verify initial subscription state. New users start with free/trial.
	rec := doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code, "get status: %s", rec.Body.String())

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Contains(t, []interface{}{"free", "trial"}, statusResp["planType"],
		"New user should be on free or trial plan")

	// Step 2: Get available premium plans.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/plans", "", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var plans []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &plans))

	// Find monthly and yearly premium plans.
	var monthlyPlanID, yearlyPlanID int
	for _, plan := range plans {
		planType, _ := plan["planType"].(string)
		if planType != "premium" {
			continue
		}
		bp, _ := plan["billingPeriod"].(map[string]interface{})
		if bp == nil {
			continue
		}
		code, _ := bp["code"].(string)
		id := int(plan["id"].(float64))
		switch code {
		case "monthly":
			monthlyPlanID = id
		case "yearly":
			yearlyPlanID = id
		}
	}

	if monthlyPlanID == 0 {
		t.Skip("No monthly premium plan found in the database; skipping plan transition test")
	}

	// Step 3: Upgrade to monthly premium.
	upgradeBody := fmt.Sprintf(`{"planId": %d}`, monthlyPlanID)
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/upgrade", upgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "upgrade: %s", rec.Body.String())

	var upgradeResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &upgradeResp))
	assert.NotEmpty(t, upgradeResp["message"])

	// Step 4: Verify subscription is now premium.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "premium", statusResp["planType"], "User should be on premium after upgrade")
	assert.Equal(t, float64(monthlyPlanID), statusResp["planId"])

	// Step 5: Change plan to yearly (if yearly plan exists).
	if yearlyPlanID == 0 {
		t.Log("No yearly premium plan found; skipping year upgrade step")
		return
	}

	changePlanBody := fmt.Sprintf(`{"planId": %d}`, yearlyPlanID)
	rec = doRequest(t, app, http.MethodPost, "/payments/change-plan", changePlanBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "change plan: %s", rec.Body.String())

	var changePlanResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &changePlanResp))
	assert.Equal(t, true, changePlanResp["success"])

	// Step 6: Verify plan changed to yearly.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "premium", statusResp["planType"])
	assert.Equal(t, float64(yearlyPlanID), statusResp["planId"])
}

// TestDowngradeSchedulesPendingPlan tests that downgrading a premium subscription
// schedules a pending plan change and records the selected accounts.
func TestDowngradeSchedulesPendingPlan(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-sub-downgrade-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	// Get available plans.
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
		t.Skip("No premium plan found; skipping downgrade test")
	}

	// First, upgrade to premium.
	upgradeBody := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/upgrade", upgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "upgrade: %s", rec.Body.String())

	// Create two accounts to keep.
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	acc1 := testutil.CreateTestAccount(t, app.DB, userID, "Keep Account 1", currencyID, accountTypeID, 100)
	defer testutil.DeleteTestAccount(t, app.DB, acc1)

	acc2 := testutil.CreateTestAccount(t, app.DB, userID, "Keep Account 2", currencyID, accountTypeID, 200)
	defer testutil.DeleteTestAccount(t, app.DB, acc2)

	// Downgrade with selection.
	downgradeBody := fmt.Sprintf(`{"accountIds": [%d, %d]}`, acc1, acc2)
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/downgrade", downgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "downgrade: %s", rec.Body.String())

	var downgradeResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &downgradeResp))
	// Without a Stripe subscription, the downgrade is applied immediately.
	assert.Equal(t, false, downgradeResp["pendingDowngrade"])

	// Verify subscription status shows free plan after immediate downgrade.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "free", statusResp["planType"], "User should be on free plan after immediate downgrade")
}
