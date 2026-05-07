// Contract tests for the categories service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package categories_test

import (
	"testing"

	"github.com/go-budget/backend/internal/services/categories"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real categories service connected to the test DB.
func setupTestService(t *testing.T) (categories.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	svc := categories.New(app.CategoryRepo, testutil.TestLogger())
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

// --- GetFlat ---

func TestGetFlat_ReturnsCategories(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-flat@example.com")

	// Registration copies default categories, so user should have some
	result, err := svc.GetFlat(userID)

	require.NoError(t, err)
	require.NotEmpty(t, result, "newly registered user should have default categories")
}

func TestGetFlat_FormatsNamesWithPrefix(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-flatfmt@example.com")

	// Create a known expense category
	cat, err := svc.Create(userID, categories.CreateParams{
		Name:     "TestExpense",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, cat.ID) })

	result, err := svc.GetFlat(userID)
	require.NoError(t, err)

	// Find our category and check format
	found := false
	for _, fc := range result {
		if fc.ID == cat.ID {
			assert.Contains(t, fc.Name, "(-) TestExpense")
			found = true
			break
		}
	}
	assert.True(t, found, "created category should appear in flat list")
}

func TestGetFlat_IncomePrefix(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-flatinc@example.com")

	cat, err := svc.Create(userID, categories.CreateParams{
		Name:     "TestIncome",
		IsIncome: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, cat.ID) })

	result, err := svc.GetFlat(userID)
	require.NoError(t, err)

	found := false
	for _, fc := range result {
		if fc.ID == cat.ID {
			assert.Contains(t, fc.Name, "(+) TestIncome")
			found = true
			break
		}
	}
	assert.True(t, found, "created income category should appear in flat list")
}

func TestGetFlat_HierarchyNotation(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-flathier@example.com")

	// Create parent
	parent, err := svc.Create(userID, categories.CreateParams{
		Name:     "Food",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, parent.ID) })

	// Create child
	child, err := svc.Create(userID, categories.CreateParams{
		Name:     "Groceries",
		ParentID: &parent.ID,
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, child.ID) })

	result, err := svc.GetFlat(userID)
	require.NoError(t, err)

	found := false
	for _, fc := range result {
		if fc.ID == child.ID {
			assert.Contains(t, fc.Name, "Food >> Groceries")
			found = true
			break
		}
	}
	assert.True(t, found, "child category should have hierarchy notation")
}

func TestGetFlat_EmptyForNonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	result, err := svc.GetFlat(999999)

	require.NoError(t, err)
	assert.Empty(t, result, "non-existent user should have no categories")
}

// --- GetGrouped ---

func TestGetGrouped_SplitsIncomeAndExpenses(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-grouped@example.com")

	income, expenses, err := svc.GetGrouped(userID)

	require.NoError(t, err)
	// Default categories include both income and expense types
	assert.NotEmpty(t, income, "should have income categories")
	assert.NotEmpty(t, expenses, "should have expense categories")

	// All items in income should be IsIncome=true
	for _, cat := range income {
		assert.True(t, cat.IsIncome, "income list should only contain income categories")
	}
	// All items in expenses should be IsIncome=false
	for _, cat := range expenses {
		assert.False(t, cat.IsIncome, "expense list should only contain expense categories")
	}
}

func TestGetGrouped_BuildsTreeStructure(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-grptree@example.com")

	// Create parent and child
	parent, err := svc.Create(userID, categories.CreateParams{
		Name:     "ParentCat",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, parent.ID) })

	child, err := svc.Create(userID, categories.CreateParams{
		Name:     "ChildCat",
		ParentID: &parent.ID,
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, child.ID) })

	_, expenses, err := svc.GetGrouped(userID)
	require.NoError(t, err)

	// Find parent in expenses and verify it has child
	found := false
	for _, cat := range expenses {
		if cat.ID == parent.ID {
			require.NotEmpty(t, cat.Children, "parent should have children")
			assert.Equal(t, child.ID, cat.Children[0].ID)
			found = true
			break
		}
	}
	assert.True(t, found, "parent category should appear in expenses list")
}

// --- Create ---

func TestCreate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-create@example.com")

	result, err := svc.Create(userID, categories.CreateParams{
		Name:     "New Category",
		IsIncome: false,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Category", result.Name)
	assert.Equal(t, userID, result.UserID)
	assert.False(t, result.IsIncome)
	assert.NotZero(t, result.ID)

	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, result.ID) })
}

func TestCreate_WithParent(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-createparent@example.com")

	parent, err := svc.Create(userID, categories.CreateParams{
		Name:     "Parent",
		IsIncome: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, parent.ID) })

	child, err := svc.Create(userID, categories.CreateParams{
		Name:     "Child",
		ParentID: &parent.ID,
		IsIncome: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, child.ID) })

	assert.NotNil(t, child.ParentID)
	assert.Equal(t, parent.ID, *child.ParentID)
}

// --- Update ---

func TestUpdate_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-update@example.com")

	cat, err := svc.Create(userID, categories.CreateParams{
		Name:     "Original",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, cat.ID) })

	updated, err := svc.Update(userID, cat.ID, categories.UpdateParams{
		Name:     "Updated",
		IsIncome: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.True(t, updated.IsIncome)
}

func TestUpdate_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-updnf@example.com")

	_, err := svc.Update(userID, 999999, categories.UpdateParams{
		Name: "Nope",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, categories.ErrNotFound)
}

func TestUpdate_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "categories-contract-upd1@example.com")
	user2 := createTestUser(t, app, "categories-contract-upd2@example.com")

	cat, err := svc.Create(user1, categories.CreateParams{
		Name:     "User1Cat",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, cat.ID) })

	_, err = svc.Update(user2, cat.ID, categories.UpdateParams{
		Name: "Stolen",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, categories.ErrAccessDenied)
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-delete@example.com")

	cat, err := svc.Create(userID, categories.CreateParams{
		Name:     "ToDelete",
		IsIncome: false,
	})
	require.NoError(t, err)

	deleted, err := svc.Delete(userID, cat.ID)

	require.NoError(t, err)
	assert.Equal(t, cat.ID, deleted.ID)
}

func TestDelete_NotFound(t *testing.T) {
	svc, app := setupTestService(t)
	userID := createTestUser(t, app, "categories-contract-delnf@example.com")

	_, err := svc.Delete(userID, 999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, categories.ErrNotFound)
}

func TestDelete_AccessDenied(t *testing.T) {
	svc, app := setupTestService(t)
	user1 := createTestUser(t, app, "categories-contract-del1@example.com")
	user2 := createTestUser(t, app, "categories-contract-del2@example.com")

	cat, err := svc.Create(user1, categories.CreateParams{
		Name:     "User1CatDel",
		IsIncome: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.DeleteTestCategory(t, app.DB, cat.ID) })

	_, err = svc.Delete(user2, cat.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, categories.ErrAccessDenied)
}
