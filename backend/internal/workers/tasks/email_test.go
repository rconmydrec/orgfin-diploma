package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NewEmailSendTask Tests ====================

func TestNewEmailSendTask(t *testing.T) {
	payload := EmailSendPayload{
		To:      "user@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	task, err := NewEmailSendTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeEmailSend, task.Type())

	// Verify payload can be round-tripped
	var decoded EmailSendPayload
	err = json.Unmarshal(task.Payload(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestNewEmailSendTask_WithAttachment(t *testing.T) {
	payload := EmailSendPayload{
		To:             "user@example.com",
		Subject:        "Test Subject",
		Body:           "Test Body",
		AttachmentData: []byte("file content"),
		AttachmentName: "report.pdf",
	}

	task, err := NewEmailSendTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeEmailSend, task.Type())

	var decoded EmailSendPayload
	err = json.Unmarshal(task.Payload(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, payload.AttachmentData, decoded.AttachmentData)
	assert.Equal(t, payload.AttachmentName, decoded.AttachmentName)
}

// ==================== EmailSendHandler ProcessTask Tests ====================

func TestEmailSendHandler_ProcessTask_Success_PlainEmail(t *testing.T) {
	emailSvc := &mockEmailService{}
	handler := NewEmailSendHandler(emailSvc, testLogger())

	payload := EmailSendPayload{
		To:      "user@example.com",
		Subject: "Hello",
		Body:    "World",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailSend, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Should have called Send, not SendWithAttachment
	require.Len(t, emailSvc.SendCalls, 1)
	assert.Empty(t, emailSvc.Calls) // No attachment calls
	call := emailSvc.SendCalls[0]
	assert.Equal(t, "user@example.com", call.To)
	assert.Equal(t, "Hello", call.Subject)
	assert.Equal(t, "World", call.Body)
}

func TestEmailSendHandler_ProcessTask_Success_WithAttachment(t *testing.T) {
	emailSvc := &mockEmailService{}
	handler := NewEmailSendHandler(emailSvc, testLogger())

	payload := EmailSendPayload{
		To:             "user@example.com",
		Subject:        "Report",
		Body:           "See attached",
		AttachmentData: []byte("pdf content"),
		AttachmentName: "report.pdf",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailSend, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Should have called SendWithAttachment
	require.Len(t, emailSvc.Calls, 1)
	assert.Empty(t, emailSvc.SendCalls) // No plain send calls
	call := emailSvc.Calls[0]
	assert.Equal(t, "user@example.com", call.To)
	assert.Equal(t, "Report", call.Subject)
	assert.Equal(t, "See attached", call.Body)
	assert.Equal(t, []byte("pdf content"), call.Attachment)
	assert.Equal(t, "report.pdf", call.Filename)
}

func TestEmailSendHandler_ProcessTask_InvalidPayload(t *testing.T) {
	emailSvc := &mockEmailService{}
	handler := NewEmailSendHandler(emailSvc, testLogger())

	task := asynq.NewTask(TypeEmailSend, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal email send payload")
}

func TestEmailSendHandler_ProcessTask_SendError(t *testing.T) {
	emailSvc := &mockEmailService{
		SendFunc: func(to, subject, body string) error {
			return errors.New("SMTP connection refused")
		},
	}
	handler := NewEmailSendHandler(emailSvc, testLogger())

	payload := EmailSendPayload{
		To:      "user@example.com",
		Subject: "Hello",
		Body:    "World",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailSend, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send email")
}

func TestEmailSendHandler_ProcessTask_SendWithAttachmentError(t *testing.T) {
	emailSvc := &mockEmailService{
		SendWithAttachmentFunc: func(to, subject, body string, attachment []byte, filename string) error {
			return errors.New("attachment too large")
		},
	}
	handler := NewEmailSendHandler(emailSvc, testLogger())

	payload := EmailSendPayload{
		To:             "user@example.com",
		Subject:        "Report",
		Body:           "See attached",
		AttachmentData: []byte("large file"),
		AttachmentName: "report.pdf",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailSend, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send email with attachment")
}
