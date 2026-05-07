package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-budget/backend/internal/services/budgets"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBudgetService implements BudgetService for testing.
type mockBudgetService struct {
	RecalculateFunc     func(userID int) error
	DailyProcessingFunc func() (*budgets.DailyProcessingResult, error)
	Calls               []int
}

func (m *mockBudgetService) RecalculateCollectedAmounts(userID int) error {
	m.Calls = append(m.Calls, userID)
	if m.RecalculateFunc != nil {
		return m.RecalculateFunc(userID)
	}
	return nil
}

func (m *mockBudgetService) DailyProcessing() (*budgets.DailyProcessingResult, error) {
	if m.DailyProcessingFunc != nil {
		return m.DailyProcessingFunc()
	}
	return &budgets.DailyProcessingResult{}, nil
}

// ==================== NewBudgetUserUpdateTask Tests ====================

func TestNewBudgetUserUpdateTask(t *testing.T) {
	payload := BudgetUserUpdatePayload{
		UserID:    42,
		AccountID: 7,
	}

	task, err := NewBudgetUserUpdateTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeBudgetUserUpdate, task.Type())

	// Verify payload can be round-tripped
	var decoded BudgetUserUpdatePayload
	err = json.Unmarshal(task.Payload(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

// ==================== BudgetUserUpdateHandler ProcessTask Tests ====================

func TestBudgetUserUpdateHandler_ProcessTask_Success(t *testing.T) {
	svc := &mockBudgetService{}
	handler := NewBudgetUserUpdateHandler(svc, testLogger())

	payload := BudgetUserUpdatePayload{UserID: 42, AccountID: 7}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeBudgetUserUpdate, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, []int{42}, svc.Calls)
}

func TestBudgetUserUpdateHandler_ProcessTask_InvalidPayload(t *testing.T) {
	svc := &mockBudgetService{}
	handler := NewBudgetUserUpdateHandler(svc, testLogger())

	task := asynq.NewTask(TypeBudgetUserUpdate, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal budget user update payload")
}

func TestBudgetUserUpdateHandler_ProcessTask_ServiceError(t *testing.T) {
	svc := &mockBudgetService{
		RecalculateFunc: func(userID int) error {
			return errors.New("budget recalculation errors: recalculate budget 1: db error")
		},
	}
	handler := NewBudgetUserUpdateHandler(svc, testLogger())

	payload := BudgetUserUpdatePayload{UserID: 42, AccountID: 7}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	task := asynq.NewTask(TypeBudgetUserUpdate, data)
	err = handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget recalculation errors")
	assert.Equal(t, []int{42}, svc.Calls)
}
