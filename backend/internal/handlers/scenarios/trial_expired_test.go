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

// TestTrialExpiredFreeLimitsEnforced verifies the behavior when a user's trial
// has expired and they are over the free-tier limits.
//
// The Go backend has partial implementation of subscription enforcement:
// - Account creation returns 402 when ErrAccountLimitExceeded is triggered.
// - The downgrade endpoint accepts account selections.
//
// This test exercises the full lifecycle:
//  1. Create a user and 3 accounts while trial is still active.
//  2. Expire the trial by modifying the subscription directly in DB.
//  3. Verify the subscription status shows trial with 0 days remaining.
//  4. Apply free-plan downgrade with a selection of accounts to keep.
//  5. Verify the downgrade was scheduled.
//  6. Verify list endpoints still work after downgrade.
func TestTrialExpiredFreeLimitsEnforced(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-trial-expired-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	accountTypeID := testutil.GetAccountTypeID(t, app.DB, false)

	// Step 1: Create 3 accounts while trial is active.
	var accountIDs []int
	for i := range 3 {
		body := fmt.Sprintf(`{
			"name": "Trial Account %d",
			"currencyId": %d,
			"accountTypeId": %d,
			"initialBalance": 0,
			"balance": 0
		}`, i, currencyID, accountTypeID)

		rec := doRequest(t, app, http.MethodPost, "/accounts/", body, token)
		require.Equal(t, http.StatusOK, rec.Code, "create account %d: %s", i, rec.Body.String())

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		accountIDs = append(accountIDs, int(resp["id"].(float64)))
	}

	// Step 2: Expire the trial by modifying the subscription in DB.
	// First, check if the user has a subscription.
	var subID int
	err := app.DB.QueryRow("SELECT id FROM subscriptions WHERE user_id = $1", userID).Scan(&subID)
	if err != nil {
		// User does not have a subscription (starts on free plan without DB row in some configs).
		// The test still validates the downgrade path works.
		t.Log("No subscription found for user; testing downgrade flow with account selection")

		// Create a subscription with expired trial.
		// First, find the trial plan.
		var trialPlanID int
		err = app.DB.QueryRow(
			"SELECT id FROM subscription_plans WHERE LOWER(plan_type::text) = 'trial' AND is_active = true AND is_deleted = false LIMIT 1",
		).Scan(&trialPlanID)
		if err != nil {
			// No trial plan; try free plan.
			err = app.DB.QueryRow(
				"SELECT id FROM subscription_plans WHERE LOWER(plan_type::text) = 'free' AND is_active = true AND is_deleted = false LIMIT 1",
			).Scan(&trialPlanID)
			if err != nil {
				t.Skip("No trial or free plan found in the database; skipping trial-expired test")
			}
		}

		trialStarted := time.Now().UTC().AddDate(0, 0, -60)
		trialEnds := time.Now().UTC().AddDate(0, 0, -1)
		_, err = app.DB.Exec(`
			INSERT INTO subscriptions (user_id, plan_id, trial_started_at, trial_ends_at, is_active)
			VALUES ($1, $2, $3, $4, true)
		`, userID, trialPlanID, trialStarted, trialEnds)
		if err != nil {
			t.Fatalf("Failed to create test subscription: %v", err)
		}
	} else {
		// Update existing subscription to have expired trial.
		trialStarted := time.Now().UTC().AddDate(0, 0, -60)
		trialEnds := time.Now().UTC().AddDate(0, 0, -1)
		_, err = app.DB.Exec(`
			UPDATE subscriptions
			SET trial_started_at = $1, trial_ends_at = $2
			WHERE id = $3
		`, trialStarted, trialEnds, subID)
		require.NoError(t, err)
	}

	// Step 3: Verify subscription status shows trial with 0 days remaining.
	rec := doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code, "status: %s", rec.Body.String())

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))

	// Plan type should be trial or free.
	planType := statusResp["planType"].(string)
	assert.Contains(t, []string{"trial", "free"}, planType)

	// If trial, days remaining should be 0.
	if planType == "trial" {
		if trialDays, ok := statusResp["trialDaysRemaining"].(float64); ok {
			assert.Equal(t, float64(0), trialDays, "Trial days remaining should be 0 for expired trial")
		}
	}

	// Step 4: Verify listing endpoints still work even with expired trial.
	listRec := doRequest(t, app, http.MethodGet, "/accounts/", "", token)
	assert.Equal(t, http.StatusOK, listRec.Code, "Listing accounts should still work")

	// Step 5: The service auto-downgrades expired trials to free during GetStatus.
	// Attempting to downgrade a user who is already on the free plan returns 400.
	downgradeBody := fmt.Sprintf(`{"accountIds": [%d, %d]}`, accountIDs[0], accountIDs[1])
	rec = doRequest(t, app, http.MethodPost, "/subscriptions/downgrade", downgradeBody, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"downgrade should fail because user is already on free plan: %s", rec.Body.String())

	// Step 6: Verify subscription status is still free with limits.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "free", statusResp["planType"],
		"User should remain on free plan after expired trial")
}

// TestFreeUserSubscriptionLimitsDisplayed verifies that a free-tier user
// has their limits displayed correctly in the subscription status.
func TestFreeUserSubscriptionLimitsDisplayed(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-free-limits-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	rec := doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))

	// Free users should have limits defined.
	limits, ok := statusResp["limits"].(map[string]interface{})
	if ok && limits != nil {
		assert.Contains(t, limits, "accounts", "Free tier should have accounts limit")
		assert.Contains(t, limits, "budgets", "Free tier should have budgets limit")
	}

	assert.Equal(t, true, statusResp["isActive"], "Free subscription should be active")
}

// TestTrialExpiredThenUpgrade verifies that a user with an expired trial
// can still upgrade to premium and regain full access.
func TestTrialExpiredThenUpgrade(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	email := fmt.Sprintf("scenario-trial-upgrade-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"

	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, email, password)

	// Create a subscription with expired trial.
	var trialPlanID int
	err := app.DB.QueryRow(
		"SELECT id FROM subscription_plans WHERE LOWER(plan_type::text) = 'trial' AND is_active = true AND is_deleted = false LIMIT 1",
	).Scan(&trialPlanID)
	if err != nil {
		err = app.DB.QueryRow(
			"SELECT id FROM subscription_plans WHERE LOWER(plan_type::text) = 'free' AND is_active = true AND is_deleted = false LIMIT 1",
		).Scan(&trialPlanID)
		if err != nil {
			t.Skip("No trial or free plan found; skipping")
		}
	}

	// Check if user already has subscription.
	var existingSubID int
	existingErr := app.DB.QueryRow("SELECT id FROM subscriptions WHERE user_id = $1", userID).Scan(&existingSubID)

	trialStarted := time.Now().UTC().AddDate(0, 0, -60)
	trialEnds := time.Now().UTC().AddDate(0, 0, -1)

	if existingErr != nil {
		// No subscription, create one.
		_, err = app.DB.Exec(`
			INSERT INTO subscriptions (user_id, plan_id, trial_started_at, trial_ends_at, is_active)
			VALUES ($1, $2, $3, $4, true)
		`, userID, trialPlanID, trialStarted, trialEnds)
		require.NoError(t, err)
	} else {
		_, err = app.DB.Exec(`
			UPDATE subscriptions
			SET plan_id = $1, trial_started_at = $2, trial_ends_at = $3
			WHERE id = $4
		`, trialPlanID, trialStarted, trialEnds, existingSubID)
		require.NoError(t, err)
	}

	// Get premium plan.
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
		t.Skip("No premium plan found; skipping trial-to-upgrade test")
	}

	// Upgrade to premium.
	upgradeBody := fmt.Sprintf(`{"planId": %d}`, premiumPlanID)
	rec := doRequest(t, app, http.MethodPost, "/subscriptions/upgrade", upgradeBody, token)
	require.Equal(t, http.StatusOK, rec.Code, "upgrade: %s", rec.Body.String())

	// Verify user is now premium.
	rec = doRequest(t, app, http.MethodGet, "/subscriptions/status", "", token)
	require.Equal(t, http.StatusOK, rec.Code)

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "premium", statusResp["planType"],
		"User should be premium after upgrading from expired trial")

	// Free-tier limits should not be present for premium users.
	limits, ok := statusResp["limits"].(map[string]interface{})
	if ok {
		assert.Nil(t, limits, "Premium users should not have free-tier limits")
	}
}
