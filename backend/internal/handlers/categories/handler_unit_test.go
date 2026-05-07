package categories

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	categoriesservice "github.com/go-budget/backend/internal/services/categories"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock category repository
type MockCategoryRepo struct {
	GetByUserIDFunc        func(userID int) ([]*models.UserCategory, error)
	GetByUserIDGroupedFunc func(userID int) ([]*models.UserCategory, error)
	GetByIDFunc            func(id int) (*models.UserCategory, error)
	CreateFunc             func(category *models.UserCategory) (*models.UserCategory, error)
	UpdateFunc             func(category *models.UserCategory) error
	DeleteFunc             func(id int) error
	CopyDefaultFunc        func(userID int) error
}

func (m *MockCategoryRepo) GetByUserID(userID int) ([]*models.UserCategory, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return []*models.UserCategory{}, nil
}

func (m *MockCategoryRepo) GetByUserIDGrouped(userID int) ([]*models.UserCategory, error) {
	if m.GetByUserIDGroupedFunc != nil {
		return m.GetByUserIDGroupedFunc(userID)
	}
	return []*models.UserCategory{}, nil
}

func (m *MockCategoryRepo) GetByID(id int) (*models.UserCategory, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.UserCategory{ID: id, UserID: 1, Name: "Test"}, nil
}

func (m *MockCategoryRepo) Create(category *models.UserCategory) (*models.UserCategory, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(category)
	}
	now := time.Now()
	return &models.UserCategory{ID: 1, Name: category.Name, UserID: category.UserID, CreatedAt: now, UpdatedAt: now}, nil
}

func (m *MockCategoryRepo) Update(category *models.UserCategory) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(category)
	}
	return nil
}

func (m *MockCategoryRepo) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockCategoryRepo) CopyDefaultCategories(userID int) error {
	if m.CopyDefaultFunc != nil {
		return m.CopyDefaultFunc(userID)
	}
	return nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockCategoryRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := &MockCategoryRepo{}
	svc := categoriesservice.New(repo, logger)
	handler := New(svc, logger)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return handler, e, repo
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== GetCategories Tests ====================

func TestGetCategoriesDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetCategoriesSuccess(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	parentID := 1
	repo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 3, UserID: 1, Name: "Salary", IsIncome: true, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Food")
}

func TestGetCategoriesWithChildrenSorting(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	parentID := 1
	repo.GetByUserIDFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Zebra", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 3, UserID: 1, Name: "Apple", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== GetGroupedCategories Tests ====================

func TestGetGroupedCategoriesDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/grouped/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetGroupedCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetGroupedCategoriesSuccess(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	repo.GetByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Salary", IsIncome: true, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/grouped/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetGroupedCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "income")
	assert.Contains(t, rec.Body.String(), "expenses")
}

func TestGetGroupedCategoriesWithChildren(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	parentID := 1
	repo.GetByUserIDGroupedFunc = func(userID int) ([]*models.UserCategory, error) {
		// Return flat list; the service builds the tree
		return []*models.UserCategory{
			{ID: 1, UserID: 1, Name: "Food", IsIncome: false, CreatedAt: now, UpdatedAt: now},
			{ID: 2, UserID: 1, Name: "Groceries", ParentID: &parentID, IsIncome: false, CreatedAt: now, UpdatedAt: now},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/grouped/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetGroupedCategories(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== CreateCategory Tests ====================

func TestCreateCategoryBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCategoryValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCategoryDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.CreateFunc = func(category *models.UserCategory) (*models.UserCategory, error) {
		return nil, errors.New("db error")
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateCategorySuccess(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{"name": "Test", "isIncome": true}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// ==================== UpdateCategory Tests ====================

func TestUpdateCategoryInvalidID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/abc/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateCategoryBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpdateCategoryValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{"isIncome": false}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Validation failed")
}

func TestUpdateCategoryNotFound(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, sql.ErrNoRows
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateCategoryGetByIDDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, errors.New("db connection error")
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to update category")
}

func TestUpdateCategoryAccessDenied(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 999, Name: "Test"}, nil
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateCategoryDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test", CreatedAt: now, UpdatedAt: now}, nil
	}
	repo.UpdateFunc = func(category *models.UserCategory) error {
		return errors.New("db error")
	}

	body := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateCategorySuccess(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test", CreatedAt: now, UpdatedAt: now}, nil
	}

	body := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/1/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.UpdateCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== DeleteCategory Tests ====================

func TestDeleteCategoryInvalidID(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/abc/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteCategoryNotFound(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, sql.ErrNoRows
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteCategoryGetByIDDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return nil, errors.New("db connection error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to delete category")
}

func TestDeleteCategoryAccessDenied(t *testing.T) {
	handler, e, repo := setupTestHandler()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 999, Name: "Test"}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteCategoryDBError(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test", CreatedAt: now, UpdatedAt: now}, nil
	}
	repo.DeleteFunc = func(id int) error {
		return errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDeleteCategorySuccess(t *testing.T) {
	handler, e, repo := setupTestHandler()
	now := time.Now()
	repo.GetByIDFunc = func(id int) (*models.UserCategory, error) {
		return &models.UserCategory{ID: id, UserID: 1, Name: "Test", CreatedAt: now, UpdatedAt: now}, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/1/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.DeleteCategory(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _ := setupTestHandler()

	g := e.Group("/categories")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/categories/"])
	assert.True(t, routePaths["/categories/grouped/"])
	assert.True(t, routePaths["/categories/:id/"])
}
