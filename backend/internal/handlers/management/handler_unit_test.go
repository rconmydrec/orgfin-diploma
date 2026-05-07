package management

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockBackupJobCreator implements BackupJobCreator for unit tests.
type mockBackupJobCreator struct {
	CreateFunc func(ctx context.Context) (string, error)
}

func (m *mockBackupJobCreator) CreateBackupJob(ctx context.Context) (string, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx)
	}
	return "orgfin-api-go-backup-manual-20260315-110000", nil
}

func setupTestHandler() (*Handler, *echo.Echo, *mockBackupJobCreator) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mock := &mockBackupJobCreator{}
	handler := New(mock, logger)
	e := echo.New()
	return handler, e, mock
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "admin@test.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Admin User")
}

// ==================== BackupDB Tests ====================

func TestBackupDBSuccess(t *testing.T) {
	handler, e, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/backup/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BackupDB(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Backup job created")
	assert.Contains(t, rec.Body.String(), "orgfin-api-go-backup-manual-")
}

func TestBackupDBServiceError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mock := &mockBackupJobCreator{
		CreateFunc: func(ctx context.Context) (string, error) {
			return "", errors.New("k8s api error")
		},
	}
	handler := New(mock, logger)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/backup/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BackupDB(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to create backup job")
}

func TestBackupDBNilService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := New(nil, logger)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/backup/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.BackupDB(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "Backup is only available in Kubernetes environment")
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _ := setupTestHandler()

	g := e.Group("/management")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/management/backup"])
}
