package categories

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mock repository ====================

type mockCategoryRepo struct {
	getByUserIDFunc        func(userID int) ([]*models.UserCategory, error)
	getByUserIDGroupedFunc func(userID int) ([]*models.UserCategory, error)
	getByIDFunc            func(id int) (*models.UserCategory, error)
	createFunc             func(category *models.UserCategory) (*models.UserCategory, error)
	updateFunc             func(category *models.UserCategory) error
	deleteFunc             func(id int) error
	copyDefaultFunc        func(userID int) error
}

func (m *mockCategoryRepo) GetByUserID(userID int) ([]*models.UserCategory, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID)
	}
	return []*models.UserCategory{}, nil
}

func (m *mockCategoryRepo) GetByUserIDGrouped(userID int) ([]*models.UserCategory, error) {
	if m.getByUserIDGroupedFunc != nil {
		return m.getByUserIDGroupedFunc(userID)
	}
	return []*models.UserCategory{}, nil
}

func (m *mockCategoryRepo) GetByID(id int) (*models.UserCategory, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.UserCategory{ID: id, UserID: 1, Name: "Test"}, nil
}

func (m *mockCategoryRepo) Create(category *models.UserCategory) (*models.UserCategory, error) {
	if m.createFunc != nil {
		return m.createFunc(category)
	}
	now := time.Now()
	category.ID = 1
	category.CreatedAt = now
	category.UpdatedAt = now
	return category, nil
}

func (m *mockCategoryRepo) Update(category *models.UserCategory) error {
	if m.updateFunc != nil {
		return m.updateFunc(category)
	}
	return nil
}

func (m *mockCategoryRepo) Delete(id int) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *mockCategoryRepo) CopyDefaultCategories(userID int) error {
	if m.copyDefaultFunc != nil {
		return m.copyDefaultFunc(userID)
	}
	return nil
}

func newTestService() (*Service, *mockCategoryRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := &mockCategoryRepo{}
	svc := New(repo, logger)
	return svc, repo
}

// ==================== GetFlat Tests ====================

func TestGetFlat_EmptyList(t *testing.T) {
	svc, repo := newTestService()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetFlat_RootOnlyCategories(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Salary", IsIncome: true, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "(-) Food", result[0].Name)
	assert.Equal(t, "(+) Salary", result[1].Name)
}

func TestGetFlat_IncomePrefixPlus(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Salary", IsIncome: true, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "(+) Salary", result[0].Name)
	assert.True(t, result[0].IsIncome)
}

func TestGetFlat_ExpensePrefixMinus(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "(-) Food", result[0].Name)
	assert.False(t, result[0].IsIncome)
}

func TestGetFlat_ChildFormatting(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	parentID := 1
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "(-) Food", result[0].Name)
	assert.Equal(t, "(-) Food >> Groceries", result[1].Name)
}

func TestGetFlat_ChildrenSortedAlphabetically(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	parentID := 1
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Zebra", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 3, UserID: 1, Name: "Apple", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "(-) Food", result[0].Name)
	assert.Equal(t, "(-) Food >> Apple", result[1].Name)
	assert.Equal(t, "(-) Food >> Zebra", result[2].Name)
}

func TestGetFlat_NestedChildren(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	parentID1 := 1
	parentID2 := 2
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID1, IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 3, UserID: 1, Name: "Organic", ParentID: &parentID2, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(1)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "(-) Food", result[0].Name)
	assert.Equal(t, "(-) Food >> Groceries", result[1].Name)
	assert.Equal(t, "(-) Food >> Groceries >> Organic", result[2].Name)
}

func TestGetFlat_PreservesFields(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 42, UserID: 7, Name: "Test", IsIncome: true, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	result, err := svc.GetFlat(7)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 42, result[0].ID)
	assert.Equal(t, 7, result[0].UserID)
	assert.True(t, result[0].IsIncome)
	assert.Nil(t, result[0].ParentID)
	assert.Equal(t, now, result[0].CreatedAt)
	assert.Equal(t, now, result[0].UpdatedAt)
}

func TestGetFlat_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.GetFlat(1)
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== GetGrouped Tests ====================

func TestGetGrouped_EmptyCategories(t *testing.T) {
	svc, repo := newTestService()
	repo.getByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{}, nil
	}

	income, expenses, err := svc.GetGrouped(1)
	require.NoError(t, err)
	assert.Empty(t, income)
	assert.Empty(t, expenses)
}

func TestGetGrouped_SplitsIntoIncomeAndExpenses(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		// Return flat list; tree building is done by the service
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Salary", IsIncome: true, CreatedAt: now, UpdatedAt: now},
			{ID: 3, UserID: 1, Name: "Transport", IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	income, expenses, err := svc.GetGrouped(1)
	require.NoError(t, err)
	assert.Len(t, income, 1)
	assert.Len(t, expenses, 2)
	assert.Equal(t, "Salary", income[0].Name)
	assert.Equal(t, "Food", expenses[0].Name)
	assert.Equal(t, "Transport", expenses[1].Name)
}

func TestGetGrouped_OnlyRootCategories(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	parentID := 1
	repo.getByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		// Flat list with parent-child relationship; service builds the tree
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	income, expenses, err := svc.GetGrouped(1)
	require.NoError(t, err)
	assert.Empty(t, income)
	assert.Len(t, expenses, 1)
	assert.Equal(t, "Food", expenses[0].Name)
	// Verify child is attached to parent
	assert.Len(t, expenses[0].Children, 1)
	assert.Equal(t, "Groceries", expenses[0].Children[0].Name)
}

func TestGetGrouped_ChildrenAttachedToParents(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	parentID := 1
	repo.getByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		// Flat list; service builds the tree
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	income, expenses, err := svc.GetGrouped(1)
	require.NoError(t, err)
	assert.Empty(t, income)
	require.Len(t, expenses, 1)
	assert.Len(t, expenses[0].Children, 1)
	assert.Equal(t, "Groceries", expenses[0].Children[0].Name)
}

func TestGetGrouped_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	_, _, err := svc.GetGrouped(1)
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== Create Tests ====================

func TestCreate_Success(t *testing.T) {
	svc, repo := newTestService()

	var captured *models.UserCategory
	repo.createFunc = func(category *models.UserCategory) (*models.UserCategory, error) {
		captured = category
		category.ID = 1
		now := time.Now()
		category.CreatedAt = now
		category.UpdatedAt = now
		return category, nil
	}

	parentID := 5
	params := CreateParams{
		Name:     "Test Category",
		ParentID: &parentID,
		IsIncome: true,
	}

	result, err := svc.Create(42, params)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, 42, captured.UserID)
	assert.Equal(t, "Test Category", captured.Name)
	assert.Equal(t, &parentID, captured.ParentID)
	assert.True(t, captured.IsIncome)
}

func TestCreate_UserIDFromParam(t *testing.T) {
	svc, repo := newTestService()

	var capturedUserID int
	repo.createFunc = func(category *models.UserCategory) (*models.UserCategory, error) {
		capturedUserID = category.UserID
		category.ID = 1
		return category, nil
	}

	_, err := svc.Create(99, CreateParams{Name: "Test"})
	require.NoError(t, err)
	assert.Equal(t, 99, capturedUserID)
}

func TestCreate_ParentIDNilWhenNotProvided(t *testing.T) {
	svc, repo := newTestService()

	var capturedParentID *int
	repo.createFunc = func(category *models.UserCategory) (*models.UserCategory, error) {
		capturedParentID = category.ParentID
		category.ID = 1
		return category, nil
	}

	_, err := svc.Create(1, CreateParams{Name: "Test"})
	require.NoError(t, err)
	assert.Nil(t, capturedParentID)
}

func TestCreate_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.createFunc = func(category *models.UserCategory) (*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.Create(1, CreateParams{Name: "Test"})
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== Update Tests ====================

func TestUpdate_NotFound_NoRows(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, sql.ErrNoRows
	}

	_, err := svc.Update(1, 99, UpdateParams{Name: "Test"})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_GetByIDDBError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, errors.New("db connection error")
	}

	_, err := svc.Update(1, 99, UpdateParams{Name: "Test"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrAccessDenied)
	assert.Equal(t, "db connection error", err.Error())
}

func TestUpdate_AccessDenied(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 999, Name: "Test"}, nil
	}

	_, err := svc.Update(1, 1, UpdateParams{Name: "Updated"})
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestUpdate_SuccessFieldsUpdated(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{
			ID:        id,
			UserID:    1,
			Name:      "Old Name",
			IsIncome:  false,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	var captured *models.UserCategory
	repo.updateFunc = func(category *models.UserCategory) error {
		captured = category
		return nil
	}

	parentID := 10
	params := UpdateParams{
		Name:     "New Name",
		ParentID: &parentID,
		IsIncome: true,
	}

	result, err := svc.Update(1, 5, params)
	require.NoError(t, err)
	assert.Equal(t, "New Name", captured.Name)
	assert.Equal(t, &parentID, captured.ParentID)
	assert.True(t, captured.IsIncome)
	assert.Equal(t, captured.ID, result.ID)
	assert.Equal(t, captured.Name, result.Name)
	assert.Equal(t, captured.IsIncome, result.IsIncome)
}

func TestUpdate_RepoUpdateError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test"}, nil
	}
	repo.updateFunc = func(category *models.UserCategory) error {
		return errors.New("db error")
	}

	_, err := svc.Update(1, 1, UpdateParams{Name: "Updated"})
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== Delete Tests ====================

func TestDelete_NotFound_NoRows(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, sql.ErrNoRows
	}

	_, err := svc.Delete(1, 99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_GetByIDDBError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, errors.New("db connection error")
	}

	_, err := svc.Delete(1, 99)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
	assert.NotErrorIs(t, err, ErrAccessDenied)
	assert.Equal(t, "db connection error", err.Error())
}

func TestDelete_AccessDenied(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 999, Name: "Test"}, nil
	}

	_, err := svc.Delete(1, 1)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestDelete_SuccessCorrectID(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test", CreatedAt: now, UpdatedAt: now}, nil
	}

	var deletedID int
	repo.deleteFunc = func(id int) error {
		deletedID = id
		return nil
	}

	_, err := svc.Delete(1, 42)
	require.NoError(t, err)
	assert.Equal(t, 42, deletedID)
}

func TestDelete_ReturnsDeletedCategory(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Deleted Cat", CreatedAt: now, UpdatedAt: now}, nil
	}

	result, err := svc.Delete(1, 42)
	require.NoError(t, err)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "Deleted Cat", result.Name)
}

func TestDelete_RepoDeleteError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test"}, nil
	}
	repo.deleteFunc = func(id int) error {
		return errors.New("db error")
	}

	_, err := svc.Delete(1, 1)
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}
