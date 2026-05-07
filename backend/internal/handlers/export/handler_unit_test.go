package export

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock export service
type MockExportService struct {
	GenerateExcelFunc func(userID int, startDate, endDate string) ([]byte, error)
}

func (m *MockExportService) GenerateExcel(userID int, startDate, endDate string) ([]byte, error) {
	if m.GenerateExcelFunc != nil {
		return m.GenerateExcelFunc(userID, startDate, endDate)
	}
	return []byte("fake excel content"), nil
}

// Mock email service
type MockEmailService struct {
	SendWithAttachmentFunc func(to, subject, body string, attachment []byte, filename string) error
}

func (m *MockEmailService) SendWithAttachment(to, subject, body string, attachment []byte, filename string) error {
	if m.SendWithAttachmentFunc != nil {
		return m.SendWithAttachmentFunc(to, subject, body, attachment, filename)
	}
	return nil
}

// MockEnqueuer implements workers.TaskEnqueuer for unit tests.
type MockEnqueuer struct {
	EnqueueFunc func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

func (m *MockEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task, opts...)
	}
	return &asynq.TaskInfo{}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockExportService, *MockEmailService, *MockEnqueuer) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	exportSvc := &MockExportService{}
	emailSvc := &MockEmailService{}
	enqueuer := &MockEnqueuer{}
	handler := New(exportSvc, emailSvc, enqueuer, logger, 1096)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return handler, e, exportSvc, emailSvc, enqueuer
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== DownloadExport Tests ====================

func TestDownloadExportBindError(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDownloadExportValidationError(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDownloadExportInvalidStartDateFormat(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "not-a-date", "end_date": "2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "start_date")
}

func TestDownloadExportInvalidEndDateFormat(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "2026-01-01", "end_date": "not-a-date"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "end_date")
}

func TestDownloadExportStartAfterEnd(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "2026-02-01", "end_date": "2026-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "start_date must be before")
}

func TestDownloadExportDateRangeExceedsMaxDays(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "2020-01-01", "end_date": "2026-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "1096 days")
}

func TestDownloadExportServiceError(t *testing.T) {
	handler, e, exportSvc, _, _ := setupTestHandler()
	exportSvc.GenerateExcelFunc = func(userID int, startDate, endDate string) ([]byte, error) {
		return nil, errors.New("service error")
	}

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDownloadExportSuccess(t *testing.T) {
	handler, e, exportSvc, _, _ := setupTestHandler()
	exportSvc.GenerateExcelFunc = func(userID int, startDate, endDate string) ([]byte, error) {
		return []byte("fake excel bytes"), nil
	}

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/download", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.DownloadExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "transactions_export.xlsx")
}

// ==================== EmailExport Tests ====================

func TestEmailExportBindError(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/email", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.EmailExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestEmailExportSuccess(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/email", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.EmailExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Contains(t, rec.Body.String(), "Export will be sent to your email shortly")
}

func TestEmailExportEnqueueError(t *testing.T) {
	handler, e, _, _, enqueuer := setupTestHandler()
	enqueuer.EnqueueFunc = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		return nil, errors.New("redis connection refused")
	}

	body := `{"start_date": "2026-01-01", "end_date": "2026-01-31"}`
	req := httptest.NewRequest(http.MethodPost, "/email", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.EmailExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to schedule export")
}

func TestEmailExportStartAfterEnd(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	body := `{"start_date": "2026-02-01", "end_date": "2026-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/email", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.EmailExport(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// ==================== RegisterRoutes Tests ====================

func TestRegisterRoutes(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	g := e.Group("/export")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/export/download"])
	assert.True(t, routePaths["/export/email"])
}
