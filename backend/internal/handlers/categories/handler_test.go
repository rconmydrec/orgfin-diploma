package categories_test

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
	testEmail    = "cat-test@example.com"
	testPassword = "TestPass123!"
)

// ==================== Get Categories Tests ====================

func TestGetCategoriesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create a test category
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Test Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	req := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var categories []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &categories)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(categories), 1)

	// Check structure
	for _, cat := range categories {
		assert.NotNil(t, cat["id"])
		assert.NotNil(t, cat["name"])
		assert.NotNil(t, cat["isIncome"])
	}
}

func TestGetCategoriesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Get Grouped Categories Tests ====================

func TestGetGroupedCategoriesSuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create income and expense categories
	incomeID := testutil.CreateTestCategory(t, app.DB, userID, "Income Category", true)
	defer testutil.DeleteTestCategory(t, app.DB, incomeID)

	expenseID := testutil.CreateTestCategory(t, app.DB, userID, "Expense Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, expenseID)

	req := httptest.NewRequest(http.MethodGet, "/categories/grouped/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check grouping structure
	assert.NotNil(t, response["income"])
	assert.NotNil(t, response["expenses"])

	income := response["income"].([]interface{})
	expenses := response["expenses"].([]interface{})

	assert.GreaterOrEqual(t, len(income), 1)
	assert.GreaterOrEqual(t, len(expenses), 1)
}

func TestGetGroupedCategoriesUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories/grouped/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Create Category Tests ====================

func TestCreateExpenseCategorySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"name": "New Expense Category",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "New Expense Category", response["name"])
	assert.Equal(t, false, response["isIncome"])
}

func TestCreateIncomeCategorySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"name": "New Income Category",
		"isIncome": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "New Income Category", response["name"])
	assert.Equal(t, true, response["isIncome"])
}

func TestCreateCategoryMissingName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCategoryUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"name": "Unauthorized Category",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Update Category Tests ====================

func TestUpdateCategorySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Original Name", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Updated Category Name",
		"isIncome": false
	}`, categoryID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Category Name", response["name"])
}

func TestUpdateCategoryNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"id": 999999,
		"name": "Nonexistent",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/categories/999999/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateCategoryOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with category
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID1, "User1 Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Create second user
	otherEmail := "other-cat@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Stolen Category",
		"isIncome": false
	}`, categoryID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateCategoryUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	body := `{
		"id": 1,
		"name": "Unauthorized Update",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/categories/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Delete Category Tests ====================

func TestDeleteCategorySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "To Be Deleted", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d/", categoryID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteCategoryNotFound(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/categories/999999/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteCategoryOtherUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	// Create first user with category
	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID1 := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID1)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID1, "User1 Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Create second user
	otherEmail := "other-cat-del@example.com"
	testutil.CleanupUserByEmail(t, app.DB, otherEmail)
	userID2 := testutil.CreateTestUser(t, app, otherEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID2)

	token2 := testutil.GetAuthToken(t, app, otherEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d/", categoryID), nil)
	req.Header.Set("auth-token", token2)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteCategoryUnauthorized(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodDelete, "/categories/1/", nil)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== Subcategory Tests ====================

func TestCreateSubcategorySuccess(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create parent category
	parentID := testutil.CreateTestCategory(t, app.DB, userID, "Food", false)
	defer testutil.DeleteTestCategory(t, app.DB, parentID)

	// Create subcategory
	body := fmt.Sprintf(`{
		"name": "Groceries",
		"isIncome": false,
		"parentId": %d
	}`, parentID)

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Groceries", response["name"])
	assert.Equal(t, float64(parentID), response["parentId"])
}

func TestGetCategoriesWithSubcategories(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create parent category
	parentID := testutil.CreateTestCategory(t, app.DB, userID, "Entertainment", false)
	defer testutil.DeleteTestCategory(t, app.DB, parentID)

	// Create subcategory directly in DB
	var childID int
	err := app.DB.QueryRow(`
		INSERT INTO user_categories (user_id, name, parent_id, is_income)
		VALUES ($1, 'Movies', $2, false)
		RETURNING id
	`, userID, parentID).Scan(&childID)
	require.NoError(t, err)
	defer testutil.DeleteTestCategory(t, app.DB, childID)

	req := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var categories []map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &categories)
	require.NoError(t, err)

	// Verify subcategory appears in list
	found := false
	for _, cat := range categories {
		name := cat["name"].(string)
		if strings.Contains(name, "Movies") {
			found = true
			break
		}
	}
	assert.True(t, found, "Subcategory should appear in the list")
}

func TestUpdateCategoryToSubcategory(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create two categories
	parentID := testutil.CreateTestCategory(t, app.DB, userID, "Parent Cat", false)
	defer testutil.DeleteTestCategory(t, app.DB, parentID)

	childID := testutil.CreateTestCategory(t, app.DB, userID, "To Become Child", false)
	defer testutil.DeleteTestCategory(t, app.DB, childID)

	// Update childID to have parentID as parent
	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Now A Subcategory",
		"isIncome": false,
		"parentId": %d
	}`, childID, parentID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", childID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(parentID), response["parentId"])
}

func TestChangeCategoryIncomeType(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create expense category
	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Mixed Category", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Change to income category
	body := fmt.Sprintf(`{
		"id": %d,
		"name": "Now Income",
		"isIncome": true
	}`, categoryID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["isIncome"])
}

func TestDeleteCategoryInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	req := httptest.NewRequest(http.MethodDelete, "/categories/not-a-number/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCategoryInvalidID(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"name": "Whatever",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPut, "/categories/invalid/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetGroupedCategoriesEmpty(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	newEmail := "no-categories@example.com"
	testutil.CleanupUserByEmail(t, app.DB, newEmail)
	userID := testutil.CreateTestUser(t, app, newEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, newEmail, testPassword)

	req := httptest.NewRequest(http.MethodGet, "/categories/grouped/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Both income and expenses should exist (even if empty)
	assert.NotNil(t, response["income"])
	assert.NotNil(t, response["expenses"])
}

// ==================== Additional Category Tests ====================

func TestCreateCategoryInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateCategoryInvalidJSON(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Update Target", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCategoryWithLongName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	longName := strings.Repeat("A", 100)
	body := fmt.Sprintf(`{
		"name": "%s",
		"isIncome": false
	}`, longName)

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	// Should accept the category
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreateCategoryWithSpecialCharacters(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{
		"name": "Їжа / 食べ物 🍕",
		"isIncome": false
	}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Їжа / 食べ物 🍕", response["name"])
}

func TestGetCategoriesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetGroupedCategoriesInvalidToken(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	req := httptest.NewRequest(http.MethodGet, "/categories/grouped/", nil)
	req.Header.Set("auth-token", "invalid-token")
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateMultipleCategories(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Create first category
	body1 := `{"name": "Category 1", "isIncome": false}`
	req1 := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req1.Header.Set("auth-token", token)
	rec1 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	// Create second category
	body2 := `{"name": "Category 2", "isIncome": true}`
	req2 := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("auth-token", token)
	rec2 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusCreated, rec2.Code)

	// Verify both exist
	req3 := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	req3.Header.Set("auth-token", token)
	rec3 := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec3, req3)

	var categories []map[string]interface{}
	err := json.Unmarshal(rec3.Body.Bytes(), &categories)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(categories), 2)
}

func TestDeleteCategoryAndVerifyRemoved(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "To Be Deleted", false)

	// Delete the category
	reqDel := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d/", categoryID), nil)
	reqDel.Header.Set("auth-token", token)
	recDel := httptest.NewRecorder()
	app.Echo.ServeHTTP(recDel, reqDel)
	assert.Equal(t, http.StatusOK, recDel.Code)

	// Verify it's gone
	reqGet := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	reqGet.Header.Set("auth-token", token)
	recGet := httptest.NewRecorder()
	app.Echo.ServeHTTP(recGet, reqGet)

	var categories []map[string]interface{}
	err := json.Unmarshal(recGet.Body.Bytes(), &categories)
	require.NoError(t, err)

	for _, cat := range categories {
		if int(cat["id"].(float64)) == categoryID {
			t.Error("Deleted category should not appear in list")
		}
	}
}

func TestCreateCategoryEmptyName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	body := `{"name": "", "isIncome": false}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== Inactive User Tests (is_active check) ====================

func TestGetCategoriesInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/categories/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}

func TestGetGroupedCategoriesInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/categories/grouped/", nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}

func TestCreateCategoryInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	body := `{"name": "New Category", "isIncome": false}`

	req := httptest.NewRequest(http.MethodPost, "/categories/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}

func TestUpdateCategoryInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "To Update", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	body := fmt.Sprintf(`{"id": %d, "name": "Updated", "isIncome": false}`, categoryID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}

func TestDeleteCategoryInactiveUser(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "To Delete", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	// Deactivate user
	_, err := app.DB.Exec("UPDATE users SET is_active = false WHERE id = $1", userID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d/", categoryID), nil)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "User not activated", response["detail"])
}

// ==================== UpdateCategory Validation Test ====================

func TestUpdateCategoryEmptyName(t *testing.T) {
	app := testutil.NewTestApp(t)
	app.SetupAllRoutes()

	testutil.CleanupUserByEmail(t, app.DB, testEmail)
	userID := testutil.CreateTestUser(t, app, testEmail, testPassword)
	defer testutil.CleanupUser(t, app.DB, userID)

	token := testutil.GetAuthToken(t, app, testEmail, testPassword)

	categoryID := testutil.CreateTestCategory(t, app.DB, userID, "Original Name", false)
	defer testutil.DeleteTestCategory(t, app.DB, categoryID)

	body := fmt.Sprintf(`{"id": %d, "name": "", "isIncome": false}`, categoryID)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/categories/%d/", categoryID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("auth-token", token)
	rec := httptest.NewRecorder()

	app.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
