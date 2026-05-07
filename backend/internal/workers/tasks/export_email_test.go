package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExportService implements ExportService for testing.
type mockExportService struct {
	GenerateExcelFunc func(userID int, startDate, endDate string) ([]byte, error)
}

func (m *mockExportService) GenerateExcel(userID int, startDate, endDate string) ([]byte, error) {
	if m.GenerateExcelFunc != nil {
		return m.GenerateExcelFunc(userID, startDate, endDate)
	}
	return []byte("fake excel content"), nil
}

// mockEmailService implements EmailService for testing.
type mockEmailService struct {
	SendFunc               func(to, subject, body string) error
	SendWithAttachmentFunc func(to, subject, body string, attachment []byte, filename string) error
	Calls                  []emailCall
	SendCalls              []sendCall
}

type emailCall struct {
	To, Subject, Body string
	Attachment        []byte
	Filename          string
}

type sendCall struct {
	To, Subject, Body string
}

func (m *mockEmailService) Send(to, subject, body string) error {
	m.SendCalls = append(m.SendCalls, sendCall{
		To: to, Subject: subject, Body: body,
	})
	if m.SendFunc != nil {
		return m.SendFunc(to, subject, body)
	}
	return nil
}

func (m *mockEmailService) SendWithAttachment(to, subject, body string, attachment []byte, filename string) error {
	m.Calls = append(m.Calls, emailCall{
		To: to, Subject: subject, Body: body,
		Attachment: attachment, Filename: filename,
	})
	if m.SendWithAttachmentFunc != nil {
		return m.SendWithAttachmentFunc(to, subject, body, attachment, filename)
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ==================== NewExportEmailTask Tests ====================

func TestNewExportEmailTask(t *testing.T) {
	payload := ExportEmailPayload{
		UserID:    42,
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		Email:     "user@example.com",
	}

	task, err := NewExportEmailTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeExportEmail, task.Type())

	// Verify payload can be round-tripped
	var decoded ExportEmailPayload
	err = json.Unmarshal(task.Payload(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

// ==================== ProcessTask Tests ====================

func TestExportEmailHandler_ProcessTask_Success(t *testing.T) {
	exportSvc := &mockExportService{
		GenerateExcelFunc: func(userID int, startDate, endDate string) ([]byte, error) {
			assert.Equal(t, 42, userID)
			assert.Equal(t, "2026-01-01", startDate)
			assert.Equal(t, "2026-01-31", endDate)
			return []byte("excel data"), nil
		},
	}
	emailSvc := &mockEmailService{}
	handler := NewExportEmailHandler(exportSvc, emailSvc, testLogger())

	payload := ExportEmailPayload{
		UserID:    42,
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		Email:     "user@example.com",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeExportEmail, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify email was sent with correct parameters
	require.Len(t, emailSvc.Calls, 1)
	call := emailSvc.Calls[0]
	assert.Equal(t, "user@example.com", call.To)
	assert.Contains(t, call.Subject, "Transactions Export")
	assert.Contains(t, call.Subject, "2026-01-01")
	assert.Contains(t, call.Subject, "2026-01-31")
	assert.Contains(t, call.Body, "2026-01-01")
	assert.Contains(t, call.Body, "2026-01-31")
	assert.Equal(t, "transactions_export.xlsx", call.Filename)
	assert.Equal(t, []byte("excel data"), call.Attachment)
}

func TestExportEmailHandler_ProcessTask_InvalidPayload(t *testing.T) {
	exportSvc := &mockExportService{}
	emailSvc := &mockEmailService{}
	handler := NewExportEmailHandler(exportSvc, emailSvc, testLogger())

	task := asynq.NewTask(TypeExportEmail, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal export email payload")
}

func TestExportEmailHandler_ProcessTask_GenerateExcelError(t *testing.T) {
	exportSvc := &mockExportService{
		GenerateExcelFunc: func(userID int, startDate, endDate string) ([]byte, error) {
			return nil, errors.New("database timeout")
		},
	}
	emailSvc := &mockEmailService{}
	handler := NewExportEmailHandler(exportSvc, emailSvc, testLogger())

	payload := ExportEmailPayload{
		UserID:    42,
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		Email:     "user@example.com",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeExportEmail, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate excel")

	// Email should NOT have been sent
	assert.Empty(t, emailSvc.Calls)
}

func TestExportEmailHandler_ProcessTask_SendEmailError(t *testing.T) {
	exportSvc := &mockExportService{}
	emailSvc := &mockEmailService{
		SendWithAttachmentFunc: func(to, subject, body string, attachment []byte, filename string) error {
			return errors.New("SMTP connection refused")
		},
	}
	handler := NewExportEmailHandler(exportSvc, emailSvc, testLogger())

	payload := ExportEmailPayload{
		UserID:    42,
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		Email:     "user@example.com",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeExportEmail, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send export email")
}
