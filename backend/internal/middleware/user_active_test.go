package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-budget/backend/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// MockUserRepo implements repositories.UserRepository for unit tests.
type MockUserRepo struct {
	GetByIDFunc func(id int) (*models.User, error)
}

func (m *MockUserRepo) Create(user *models.User) (*models.User, error) { return nil, nil }
func (m *MockUserRepo) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.User{ID: 1, IsActive: true, BaseCurrencyID: 1}, nil
}
func (m *MockUserRepo) GetByEmail(email string) (*models.User, error)        { return nil, nil }
func (m *MockUserRepo) Update(user *models.User) error                       { return nil }
func (m *MockUserRepo) Activate(userID int) error                            { return nil }
func (m *MockUserRepo) UpdatePassword(userID int, passwordHash string) error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRequireActiveUser_ActiveUser(t *testing.T) {
	userRepo := &MockUserRepo{
		GetByIDFunc: func(id int) (*models.User, error) {
			firstName := "Test"
			lastName := "User"
			return &models.User{ID: id, Email: "test@example.com", IsActive: true, IsDeleted: false, BaseCurrencyID: 1, FirstName: &firstName, LastName: &lastName}, nil
		},
	}

	middleware := RequireActiveUser(userRepo, testLogger())

	handlerCalled := false
	handler := middleware(func(c echo.Context) error {
		handlerCalled = true
		// Verify primitives are set in context
		assert.Equal(t, "test@example.com", c.Get("active_user_email").(string))
		assert.Equal(t, 1, c.Get("active_user_base_currency_id").(int))
		assert.Equal(t, "Test User", c.Get("active_user_display_name").(string))
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 42)

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireActiveUser_InactiveUser(t *testing.T) {
	userRepo := &MockUserRepo{
		GetByIDFunc: func(id int) (*models.User, error) {
			return &models.User{ID: id, IsActive: false, IsDeleted: false}, nil
		},
	}

	middleware := RequireActiveUser(userRepo, testLogger())

	handlerCalled := false
	handler := middleware(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestRequireActiveUser_DeletedUser(t *testing.T) {
	userRepo := &MockUserRepo{
		GetByIDFunc: func(id int) (*models.User, error) {
			return &models.User{ID: id, IsActive: true, IsDeleted: true}, nil
		},
	}

	middleware := RequireActiveUser(userRepo, testLogger())

	handlerCalled := false
	handler := middleware(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}

func TestRequireActiveUser_UserNotFound(t *testing.T) {
	userRepo := &MockUserRepo{
		GetByIDFunc: func(id int) (*models.User, error) {
			return nil, errors.New("user not found")
		},
	}

	middleware := RequireActiveUser(userRepo, testLogger())

	handlerCalled := false
	handler := middleware(func(c echo.Context) error {
		handlerCalled = true
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 999)

	err := handler(c)
	assert.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not activated")
}
