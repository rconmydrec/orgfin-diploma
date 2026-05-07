package internal_api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/middleware"
	backupnotify "github.com/go-budget/backend/internal/services/backup_notify"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockBackupNotifyService implements BackupNotifyService for unit tests.
type mockBackupNotifyService struct {
	NotifyFunc func(params backupnotify.NotifyParams) (int, error)
	calls      int
	lastParams backupnotify.NotifyParams
}

func (m *mockBackupNotifyService) NotifyBackupComplete(params backupnotify.NotifyParams) (int, error) {
	m.calls++
	m.lastParams = params
	if m.NotifyFunc != nil {
		return m.NotifyFunc(params)
	}
	return 1, nil
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = middleware.NewValidator()
	return e
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestBackupNotify_SuccessPayload(t *testing.T) {
	mock := &mockBackupNotifyService{
		NotifyFunc: func(params backupnotify.NotifyParams) (int, error) {
			return 2, nil
		},
	}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `{"status":"success","filename":"backup_20260315.sql.gz","fileSize":"42M","gdriveUploaded":true}`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Notification sent")
	assert.Equal(t, 1, mock.calls)
	assert.Equal(t, "success", mock.lastParams.Status)
	assert.Equal(t, "backup_20260315.sql.gz", mock.lastParams.Filename)
	assert.Equal(t, "42M", mock.lastParams.FileSize)
	assert.True(t, mock.lastParams.GdriveUploaded)
}

func TestBackupNotify_ErrorPayload(t *testing.T) {
	mock := &mockBackupNotifyService{}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `{"status":"error","errorMessage":"pg_dump failed: connection refused","gdriveUploaded":false}`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Notification sent")
	assert.Equal(t, 1, mock.calls)
	assert.Equal(t, "error", mock.lastParams.Status)
	assert.Equal(t, "pg_dump failed: connection refused", mock.lastParams.ErrorMessage)
}

func TestBackupNotify_InvalidJSON(t *testing.T) {
	mock := &mockBackupNotifyService{}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request data")
	assert.Equal(t, 0, mock.calls)
}

func TestBackupNotify_MissingStatus(t *testing.T) {
	mock := &mockBackupNotifyService{}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `{"filename":"backup.sql.gz"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request data")
	assert.Equal(t, 0, mock.calls)
}

func TestBackupNotify_InvalidStatusValue(t *testing.T) {
	mock := &mockBackupNotifyService{}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `{"status":"warning"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid request data")
	assert.Equal(t, 0, mock.calls)
}

func TestBackupNotify_ServiceError(t *testing.T) {
	mock := &mockBackupNotifyService{
		NotifyFunc: func(params backupnotify.NotifyParams) (int, error) {
			return 0, assert.AnError
		},
	}
	h := New(mock, newTestLogger())
	e := newTestEcho()

	body := `{"status":"success","filename":"backup.sql.gz"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/backup/notify", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.BackupNotify(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to send notification")
}

func TestRegisterRoutes_InternalAPI(t *testing.T) {
	h := New(nil, newTestLogger())
	e := echo.New()
	g := e.Group("/internal")
	h.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/internal/backup/notify"])
}
