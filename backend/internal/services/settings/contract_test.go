// Contract tests for the settings service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package settings_test

import (
	"testing"

	"github.com/go-budget/backend/internal/services/settings"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real settings service connected to the test DB.
func setupTestService(t *testing.T) (settings.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	svc := settings.New(app.SettingsRepo, app.UserRepo, app.CurrencyRepo, app.LanguageRepo, testutil.TestLogger())
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

// --- GetSettings ---

func TestGetSettings_AutoCreatesDefaults(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-get@example.com")

	// Delete any settings that were auto-created during registration
	_, _ = app.DB.Exec("DELETE FROM user_settings WHERE user_id = $1", userID)

	result, err := svc.GetSettings(userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.NotEmpty(t, result.Settings.Language)
}

func TestGetSettings_ReturnsExisting(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-existing@example.com")

	// First call ensures settings exist
	first, err := svc.GetSettings(userID)
	require.NoError(t, err)

	// Second call returns the same settings
	second, err := svc.GetSettings(userID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
}

// --- UpdateSettings ---

func TestUpdateSettings_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-update@example.com")

	projEnd := "2026-12-31"
	projPeriod := "MONTHLY"
	result, err := svc.UpdateSettings(userID, settings.UpdateParams{
		Language:          "uk",
		ProjectionEndDate: &projEnd,
		ProjectionPeriod:  &projPeriod,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "uk", result.Settings.Language)
	assert.Equal(t, "2026-12-31", *result.Settings.ProjectionEndDate)
	assert.Equal(t, "MONTHLY", *result.Settings.ProjectionPeriod)
}

func TestUpdateSettings_AutoCreatesIfNotExist(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-update-auto@example.com")

	// Delete settings to force auto-creation
	_, _ = app.DB.Exec("DELETE FROM user_settings WHERE user_id = $1", userID)

	result, err := svc.UpdateSettings(userID, settings.UpdateParams{
		Language: "en",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "en", result.Settings.Language)
}

// --- GetBaseCurrency ---

func TestGetBaseCurrency_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-basecurr@example.com")

	// Ensure user has a base currency set (registration sets USD by default)
	result, err := svc.GetBaseCurrency(userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Code)
	assert.NotZero(t, result.ID)
}

func TestGetBaseCurrency_UserNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetBaseCurrency(999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrUserNotFound)
}

// Note: ErrBaseCurrencyNotSet is a defensive sentinel. With the current schema,
// base_currency_id is NOT NULL, so this error path cannot be triggered via the
// public API without an inconsistent DB state. The error sentinel exists as
// defense-in-depth and is tested indirectly through unit tests with mocks.

// --- UpdateBaseCurrency ---

func TestUpdateBaseCurrency_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-updcurr@example.com")

	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")

	result, err := svc.UpdateBaseCurrency(userID, eurID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "EUR", result.Code)
	assert.Equal(t, eurID, result.ID)
}

func TestUpdateBaseCurrency_InvalidCurrency(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "settings-contract-invalidcurr@example.com")

	_, err := svc.UpdateBaseCurrency(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrInvalidCurrency)
}

func TestUpdateBaseCurrency_UserNotFound(t *testing.T) {
	svc, app := setupTestService(t)
	usdID := testutil.GetCurrencyID(t, app.DB, "USD")

	_, err := svc.UpdateBaseCurrency(999999, usdID)

	require.Error(t, err)
	assert.ErrorIs(t, err, settings.ErrUserNotFound)
}

// --- GetLanguages ---

func TestGetLanguages_ReturnsNonEmptyList(t *testing.T) {
	svc, _ := setupTestService(t)

	langs, err := svc.GetLanguages()

	require.NoError(t, err)
	require.NotEmpty(t, langs)
	// Each language should have an ID and code
	for _, lang := range langs {
		assert.NotZero(t, lang.ID)
		assert.NotEmpty(t, lang.Code)
	}
}
