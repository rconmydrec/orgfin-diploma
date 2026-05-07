package currencies_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test user credentials
const (
	testEmail    = "curr-test@example.com"
	testPassword = "TestPass123!"
)

func TestGetCurrenciesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	assert.Greater(t, len(currencies), 0)

	// Check structure
	for _, currency := range currencies {
		assert.NotNil(t, currency["id"])
		assert.NotNil(t, currency["code"])
		assert.NotNil(t, currency["name"])
	}
}

func TestGetCurrenciesIncludesMajorCurrencies(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	// Check for major currencies
	codes := make([]string, 0, len(currencies))
	for _, c := range currencies {
		codes = append(codes, c["code"].(string))
	}

	assert.Contains(t, codes, "USD")
	assert.Contains(t, codes, "EUR")
}

func TestGetCurrenciesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetCurrenciesHasValidStructure(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	// Check that all currencies have required fields
	for _, c := range currencies {
		assert.NotNil(t, c["id"], "Currency should have id")
		assert.NotNil(t, c["code"], "Currency should have code")
		assert.NotNil(t, c["name"], "Currency should have name")
	}
}

func TestGetCurrenciesReturnsNonEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	// Should have multiple currencies
	assert.Greater(t, len(currencies), 5, "Should have at least 5 currencies")
}

func TestGetCurrenciesWithGBP(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	codes := make([]string, 0, len(currencies))
	for _, c := range currencies {
		codes = append(codes, c["code"].(string))
	}

	assert.Contains(t, codes, "GBP")
}

func TestGetCurrenciesWithJPY(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	codes := make([]string, 0, len(currencies))
	for _, c := range currencies {
		codes = append(codes, c["code"].(string))
	}

	assert.Contains(t, codes, "JPY")
}

func TestGetCurrenciesWithUAH(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	codes := make([]string, 0, len(currencies))
	for _, c := range currencies {
		codes = append(codes, c["code"].(string))
	}

	assert.Contains(t, codes, "UAH")
}

func TestGetCurrenciesHasIDs(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	// All currencies should have positive IDs
	for _, c := range currencies {
		id := c["id"].(float64)
		assert.Greater(t, id, float64(0), "Currency ID should be positive")
	}
}

func TestGetCurrenciesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetCurrenciesCodeLength(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var currencies []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &currencies)
	require.NoError(t, err)

	// All currency codes should be 3 characters (ISO 4217)
	for _, c := range currencies {
		code := c["code"].(string)
		assert.Len(t, code, 3, "Currency code should be 3 characters")
	}
}

// ==================== Inactive User Tests (is_active check) ====================

func TestGetCurrenciesInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/currencies/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}
