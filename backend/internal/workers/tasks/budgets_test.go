package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/go-budget/backend/internal/services/budgets"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

// ==================== NewBudgetProcessingTask Tests ====================

func TestNewBudgetProcessingTask(t *testing.T) {
	task := NewBudgetProcessingTask()
	assert.Equal(t, TypeBudgetProcessing, task.Type())
	assert.Nil(t, task.Payload())
}

// ==================== BudgetProcessingHandler ProcessTask Tests ====================

func TestBudgetProcessingHandler_ProcessTask_Success(t *testing.T) {
	svc := &mockBudgetService{
		DailyProcessingFunc: func() (*budgets.DailyProcessingResult, error) {
			return &budgets.DailyProcessingResult{Processed: 5}, nil
		},
	}
	handler := NewBudgetProcessingHandler(svc, testLogger())

	task := asynq.NewTask(TypeBudgetProcessing, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)
}

func TestBudgetProcessingHandler_ProcessTask_Error(t *testing.T) {
	svc := &mockBudgetService{
		DailyProcessingFunc: func() (*budgets.DailyProcessingResult, error) {
			return nil, errors.New("database connection lost")
		},
	}
	handler := NewBudgetProcessingHandler(svc, testLogger())

	task := asynq.NewTask(TypeBudgetProcessing, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget daily processing")
}
