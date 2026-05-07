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

// mockTaskEnqueuer implements TaskEnqueuer for testing.
type mockTaskEnqueuer struct {
	EnqueueFunc func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
	Tasks       []enqueuedTask
}

type enqueuedTask struct {
	Type    string
	Payload []byte
}

func (m *mockTaskEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.Tasks = append(m.Tasks, enqueuedTask{
		Type:    task.Type(),
		Payload: task.Payload(),
	})
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task, opts...)
	}
	return &asynq.TaskInfo{}, nil
}

// ==================== NewActivationEmailTask Tests ====================

func TestNewActivationEmailTask(t *testing.T) {
	payload := ActivationEmailPayload{
		UserID:      42,
		Email:       "user@example.com",
		Token:       "abc123def456",
		AppURL:      "http://localhost:8080",
		FrontendURL: "http://localhost:5173",
	}

	task, err := NewActivationEmailTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeEmailActivation, task.Type())

	// Verify payload can be round-tripped
	var decoded ActivationEmailPayload
	err = json.Unmarshal(task.Payload(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

// ==================== ActivationEmailHandler ProcessTask Tests ====================

func TestActivationEmailHandler_ProcessTask_Success(t *testing.T) {
	enqueuer := &mockTaskEnqueuer{}
	handler := NewActivationEmailHandler(enqueuer, testLogger())

	payload := ActivationEmailPayload{
		UserID:      42,
		Email:       "user@example.com",
		Token:       "abc123def456",
		AppURL:      "http://localhost:8080",
		FrontendURL: "http://localhost:5173",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailActivation, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify an email:send task was enqueued
	require.Len(t, enqueuer.Tasks, 1)
	assert.Equal(t, TypeEmailSend, enqueuer.Tasks[0].Type)

	// Verify the chained email:send payload
	var emailPayload EmailSendPayload
	err = json.Unmarshal(enqueuer.Tasks[0].Payload, &emailPayload)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", emailPayload.To)
	assert.Equal(t, "Activate Your Budget Tracker Account", emailPayload.Subject)
	assert.Contains(t, emailPayload.Body, "http://localhost:5173/activate/abc123def456")
	assert.Empty(t, emailPayload.AttachmentData)
	assert.Empty(t, emailPayload.AttachmentName)
}

func TestActivationEmailHandler_ProcessTask_InvalidPayload(t *testing.T) {
	enqueuer := &mockTaskEnqueuer{}
	handler := NewActivationEmailHandler(enqueuer, testLogger())

	task := asynq.NewTask(TypeEmailActivation, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal activation email payload")

	// No tasks should have been enqueued
	assert.Empty(t, enqueuer.Tasks)
}

func TestActivationEmailHandler_ProcessTask_EnqueueError(t *testing.T) {
	enqueuer := &mockTaskEnqueuer{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			return nil, errors.New("redis connection refused")
		},
	}
	handler := NewActivationEmailHandler(enqueuer, testLogger())

	payload := ActivationEmailPayload{
		UserID:      42,
		Email:       "user@example.com",
		Token:       "abc123def456",
		AppURL:      "http://localhost:8080",
		FrontendURL: "http://localhost:5173",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailActivation, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enqueue email send task")
}

func TestActivationEmailHandler_ProcessTask_ActivationLinkFormat(t *testing.T) {
	enqueuer := &mockTaskEnqueuer{}
	handler := NewActivationEmailHandler(enqueuer, testLogger())

	payload := ActivationEmailPayload{
		UserID:      1,
		Email:       "test@example.com",
		Token:       "token123",
		AppURL:      "https://api.example.com",
		FrontendURL: "https://app.example.com",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeEmailActivation, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	require.Len(t, enqueuer.Tasks, 1)

	var emailPayload EmailSendPayload
	err = json.Unmarshal(enqueuer.Tasks[0].Payload, &emailPayload)
	require.NoError(t, err)

	// Verify activation link uses FrontendURL, not AppURL
	assert.Contains(t, emailPayload.Body, "https://app.example.com/activate/token123")
	assert.NotContains(t, emailPayload.Body, "https://api.example.com")
}
