package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

// mockSubscriptionService implements SubscriptionService for testing.
type mockSubscriptionService struct {
	processRenewalsErr             error
	processExpiredDowngradesErr    error
	processRenewalsCalled          bool
	processExpiredDowngradesCalled bool
}

func (m *mockSubscriptionService) ProcessRenewals() error {
	m.processRenewalsCalled = true
	return m.processRenewalsErr
}

func (m *mockSubscriptionService) ProcessExpiredDowngrades() error {
	m.processExpiredDowngradesCalled = true
	return m.processExpiredDowngradesErr
}

// ==================== NewSubscriptionRenewalTask Tests ====================

func TestNewSubscriptionRenewalTask(t *testing.T) {
	task := NewSubscriptionRenewalTask()
	assert.Equal(t, TypeSubscriptionRenewal, task.Type())
	assert.Nil(t, task.Payload())
}

// ==================== NewSubscriptionDowngradeTask Tests ====================

func TestNewSubscriptionDowngradeTask(t *testing.T) {
	task := NewSubscriptionDowngradeTask()
	assert.Equal(t, TypeSubscriptionDowngrade, task.Type())
	assert.Nil(t, task.Payload())
}

// ==================== SubscriptionRenewalHandler ProcessTask Tests ====================

func TestSubscriptionRenewalHandler_ProcessTask_Success(t *testing.T) {
	svc := &mockSubscriptionService{}
	handler := NewSubscriptionRenewalHandler(svc, testLogger())

	task := asynq.NewTask(TypeSubscriptionRenewal, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)
	assert.True(t, svc.processRenewalsCalled)
}

func TestSubscriptionRenewalHandler_ProcessTask_ServiceError(t *testing.T) {
	svc := &mockSubscriptionService{
		processRenewalsErr: errors.New("renewal processing failed"),
	}
	handler := NewSubscriptionRenewalHandler(svc, testLogger())

	task := asynq.NewTask(TypeSubscriptionRenewal, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "process renewals")
	assert.True(t, svc.processRenewalsCalled)
}

// ==================== SubscriptionDowngradeHandler ProcessTask Tests ====================

func TestSubscriptionDowngradeHandler_ProcessTask_Success(t *testing.T) {
	svc := &mockSubscriptionService{}
	handler := NewSubscriptionDowngradeHandler(svc, testLogger())

	task := asynq.NewTask(TypeSubscriptionDowngrade, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)
	assert.True(t, svc.processExpiredDowngradesCalled)
}

func TestSubscriptionDowngradeHandler_ProcessTask_ServiceError(t *testing.T) {
	svc := &mockSubscriptionService{
		processExpiredDowngradesErr: errors.New("downgrade processing failed"),
	}
	handler := NewSubscriptionDowngradeHandler(svc, testLogger())

	task := asynq.NewTask(TypeSubscriptionDowngrade, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "process expired downgrades")
	assert.True(t, svc.processExpiredDowngradesCalled)
}
