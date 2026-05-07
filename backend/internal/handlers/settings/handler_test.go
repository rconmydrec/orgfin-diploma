package settings_test

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
	testEmail    = "settings-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Get Languages Tests ====================

func TestGetLanguagesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Languages endpoint is public
	req := httptest.NewRequest(http.MethodGet, "/settings/languages", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var languages []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &languages)
	require.NoError(t, err)

	assert.Greater(t, len(languages), 0)

	// Check structure
	for _, lang := range languages {
		assert.NotNil(t, lang["id"])
		assert.NotNil(t, lang["code"])
		assert.NotNil(t, lang["name"])
	}
}

func TestGetLanguagesIncludesCommonLanguages(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings/languages", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var languages []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &languages)
	require.NoError(t, err)

	codes := make([]string, 0, len(languages))
	for _, lang := range languages {
		codes = append(codes, lang["code"].(string))
	}

	assert.Contains(t, codes, "en")
}

// ==================== Get Settings Tests ====================

func TestGetSettingsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["settings"])
}

func TestGetSettingsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Settings Tests ====================

func TestUpdateSettingsSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"language": "en"
	}`

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateSettingsUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"language": "en"
	}`

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Base Currency Tests ====================

func TestGetBaseCurrencySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// First set a base currency
	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")
	_, err := app.DB.Exec("UPDATE users SET base_currency_id = $1 WHERE id = $2", currencyID, userID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/settings/base-currency/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["id"])
	assert.NotNil(t, response["code"])
	assert.NotNil(t, response["name"])
}

func TestGetBaseCurrencyNewUser(t *testing.T) {
	// New users typically have a default base currency set
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "new-user-currency@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings/base-currency/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// New users should have a default currency (typically USD)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["code"])
}

func TestGetBaseCurrencyUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings/base-currency/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Base Currency Tests ====================

func TestUpdateBaseCurrencySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "EUR")

	body := fmt.Sprintf(`{"currencyId": %d}`, currencyID)

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "EUR", response["code"])
}

func TestUpdateBaseCurrencyInvalid(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"currencyId": 999999}`

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateBaseCurrencyMissingID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{}`

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateBaseCurrencyUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"currencyId": 1}`

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Additional Settings Tests ====================

func TestUpdateSettingsWithProjection(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"language": "en",
		"projectionEndDate": "2025-12-31",
		"projectionPeriod": "daily"
	}`

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	settings := response["settings"].(map[string]interface{})
	assert.Equal(t, "en", settings["language"])
	assert.Equal(t, "2025-12-31", settings["projectionEndDate"])
	assert.Equal(t, "daily", settings["projectionPeriod"])
}

func TestUpdateSettingsInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateBaseCurrencyInvalidBody(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetSettingsReturnsDefaults(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "new-settings@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify settings structure exists
	settings := response["settings"].(map[string]interface{})
	assert.NotNil(t, settings["language"])
}

func TestUpdateBaseCurrencyToUSD(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	currencyID := testutil.GetCurrencyID(t, app.DB, "USD")

	body := fmt.Sprintf(`{"currencyId": %d}`, currencyID)

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "USD", response["code"])
}

func TestUpdateSettingsMultipleTimes(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// First update
	body1 := `{"language": "en"}`
	req1 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second update
	body2 := `{"language": "uk"}`
	req2 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec2.Body.Bytes(), &response)
	require.NoError(t, err)

	settings := response["settings"].(map[string]interface{})
	assert.Equal(t, "uk", settings["language"])
}

// ==================== Additional Coverage Tests ====================

func TestGetSettingsCreatesDefaultWhenNotExists(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "no-settings@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Delete user settings if they exist
	_, _ = app.DB.Exec("DELETE FROM user_settings WHERE user_id = $1", userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["settings"])
	assert.NotNil(t, response["userId"])
}

func TestUpdateSettingsCreatesDefaultWhenNotExists(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "update-no-settings@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	// Delete user settings if they exist
	_, _ = app.DB.Exec("DELETE FROM user_settings WHERE user_id = $1", userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	body := `{"language": "de"}`
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	settings := response["settings"].(map[string]interface{})
	assert.Equal(t, "de", settings["language"])
}

func TestUpdateSettingsClearProjectionFields(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// First set projection fields
	body1 := `{
		"language": "en",
		"projectionEndDate": "2025-12-31",
		"projectionPeriod": "monthly"
	}`
	req1 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Then clear them
	body2 := `{
		"language": "en",
		"projectionEndDate": null,
		"projectionPeriod": null
	}`
	req2 := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec2.Body.Bytes(), &response)
	require.NoError(t, err)

	settings := response["settings"].(map[string]interface{})
	assert.Nil(t, settings["projectionEndDate"])
	assert.Nil(t, settings["projectionPeriod"])
}

func TestGetSettingsResponseStructure(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check all required fields
	assert.NotNil(t, response["id"])
	assert.NotNil(t, response["userId"])
	assert.NotNil(t, response["settings"])
	assert.NotNil(t, response["createdAt"])
	assert.NotNil(t, response["updatedAt"])

	// Check settings structure
	settings := response["settings"].(map[string]interface{})
	assert.NotNil(t, settings["language"])
}

func TestUpdateBaseCurrencyVerifyPersistence(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Update to EUR
	eurID := testutil.GetCurrencyID(t, app.DB, "EUR")
	body := fmt.Sprintf(`{"currencyId": %d}`, eurID)

	req := httptest.NewRequest(http.MethodPut, "/settings/base-currency/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify by getting base currency
	req2 := httptest.NewRequest(http.MethodGet, "/settings/base-currency/", nil)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec2.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "EUR", response["code"])
}

func TestGetLanguagesNonDeleted(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings/languages", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var languages []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &languages)
	require.NoError(t, err)

	// All returned languages should not be deleted
	for _, lang := range languages {
		assert.False(t, lang["isDeleted"].(bool))
	}
}

// ==================== Trailing-Slash Regression Tests ====================
//
// Regression coverage for the trailing-slash fix: the frontend calls
// `GET /settings` and `POST /settings` (no trailing slash). Previously the
// backend registered these routes with a trailing slash, causing 404s. These
// tests explicitly assert the canonical no-slash paths work, and that the
// `/base-currency/` variants (which are intentionally registered with a
// trailing slash and called that way by the frontend) remain unchanged.

func TestGetSettingsNoTrailingSlashReturns200(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPostSettingsNoTrailingSlashReturns200(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"language": "en"}`
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetSettingsNoTrailingSlashUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPostSettingsNoTrailingSlashUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{"language": "en"}`
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
