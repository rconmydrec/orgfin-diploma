package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

// mockActivationTokenRepo implements ActivationTokenRepository for testing.
type mockActivationTokenRepo struct {
	DeleteExpiredFunc  func() error
	DeleteExpiredCalls int
}

func (m *mockActivationTokenRepo) DeleteExpired() error {
	m.DeleteExpiredCalls++
	if m.DeleteExpiredFunc != nil {
		return m.DeleteExpiredFunc()
	}
	return nil
}

// ==================== NewTokenCleanupTask Tests ====================

func TestNewTokenCleanupTask(t *testing.T) {
	task := NewTokenCleanupTask()
	assert.Equal(t, TypeTokenCleanup, task.Type())
	assert.Nil(t, task.Payload())
}

// ==================== TokenCleanupHandler ProcessTask Tests ====================

func TestTokenCleanupHandler_ProcessTask_Success(t *testing.T) {
	repo := &mockActivationTokenRepo{}
	handler := NewTokenCleanupHandler(repo, testLogger())

	task := asynq.NewTask(TypeTokenCleanup, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteExpiredCalls)
}

func TestTokenCleanupHandler_ProcessTask_Error(t *testing.T) {
	repo := &mockActivationTokenRepo{
		DeleteExpiredFunc: func() error {
			return errors.New("database connection lost")
		},
	}
	handler := NewTokenCleanupHandler(repo, testLogger())

	task := asynq.NewTask(TypeTokenCleanup, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete expired tokens")
	assert.Equal(t, 1, repo.DeleteExpiredCalls)
}
