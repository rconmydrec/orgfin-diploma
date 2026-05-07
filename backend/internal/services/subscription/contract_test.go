// Contract tests for the subscription service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package subscription_test

import (
	"testing"

	"github.com/go-budget/backend/internal/services/subscription"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real subscription service connected to the test DB.
func setupTestService(t *testing.T) (subscription.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	svc := subscription.New(
		app.SubscriptionRepo,
		app.SubPlanRepo,
		nil, // planPriceRepo: not needed for basic contract tests
		app.AccountRepo,
		app.BudgetRepo,
		logger,
		14, // trialDurationDays
	)
	return svc, app
}

// createTestUser creates a user and schedules cleanup.
func createTestUser(t *testing.T, app *testutil.TestApp, email string) int {
	t.Helper()
	testutil.CleanupUserByEmail(t, app.DB, email)
	userID := testutil.CreateTestUser(t, app, email, "Password123!")
	t.Cleanup(func() { testutil.CleanupUser(t, app.DB, userID) })
	return userID
}

// --- GetStatus ---

func TestGetStatus_DefaultFreeForNewUser(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-status@example.com")

	// Delete any subscription created during registration
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)

	status, err := svc.GetStatus(userID)

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.IsActive)
}

func TestGetStatus_WithTrialSubscription(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-trial@example.com")

	// Delete any existing subscription
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)

	// Create trial
	err := svc.CreateTrialSubscription(userID)
	require.NoError(t, err)

	status, err := svc.GetStatus(userID)

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "trial", status.PlanType)
	assert.True(t, status.IsActive)
	assert.NotNil(t, status.TrialDaysRemaining)
	assert.Greater(t, *status.TrialDaysRemaining, 0)
}

func TestGetStatus_NonExistentUserReturnsFree(t *testing.T) {
	svc, _ := setupTestService(t)

	// A user with no subscription record should get default free status
	status, err := svc.GetStatus(999999)

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "free", status.PlanType)
}

// --- GetPlans ---

func TestGetPlans_ReturnsActivePlans(t *testing.T) {
	svc, _ := setupTestService(t)

	plans, err := svc.GetPlans()

	require.NoError(t, err)
	require.NotEmpty(t, plans, "there should be active subscription plans in the database")
	for _, p := range plans {
		assert.NotZero(t, p.ID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.PlanType)
	}
}

// --- Upgrade ---

func TestUpgrade_SamePlanRejected(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-sameplan@example.com")

	// Delete any existing subscription and create a trial
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)
	err := svc.CreateTrialSubscription(userID)
	require.NoError(t, err)

	// Get current plan ID
	status, err := svc.GetStatus(userID)
	require.NoError(t, err)

	_, err = svc.Upgrade(userID, status.PlanID)

	require.Error(t, err)
	assert.ErrorIs(t, err, subscription.ErrSamePlan)
}

func TestUpgrade_PlanNotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-badplan@example.com")

	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)
	err := svc.CreateTrialSubscription(userID)
	require.NoError(t, err)

	_, err = svc.Upgrade(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, subscription.ErrPlanNotFound)
}

// --- CreateTrialSubscription ---

func TestCreateTrialSubscription_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-createtrial@example.com")

	// Delete any existing subscription
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)

	err := svc.CreateTrialSubscription(userID)

	require.NoError(t, err)

	// Verify the subscription was created
	status, err := svc.GetStatus(userID)
	require.NoError(t, err)
	assert.Equal(t, "trial", status.PlanType)
}

// --- ScheduleDowngrade ---

func TestScheduleDowngrade_AlreadyFree(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "sub-contract-dowfree@example.com")

	// Ensure the user is on free plan (delete any subscriptions)
	_, _ = app.DB.Exec("DELETE FROM subscriptions WHERE user_id = $1", userID)

	_, err := svc.ScheduleDowngrade(userID, nil, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, subscription.ErrAlreadyOnFreePlan)
}

// --- ProcessRenewals ---

func TestProcessRenewals_RunsWithoutError(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.ProcessRenewals()

	require.NoError(t, err)
}

// --- ProcessExpiredDowngrades ---

func TestProcessExpiredDowngrades_RunsWithoutError(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.ProcessExpiredDowngrades()

	require.NoError(t, err)
}

// --- SetArchivers ---

func TestSetArchivers_DoesNotPanic(t *testing.T) {
	svc, _ := setupTestService(t)

	// SetArchivers should not panic with nil values
	assert.NotPanics(t, func() {
		svc.SetArchivers(nil, nil)
	})
}
