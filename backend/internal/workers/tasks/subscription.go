package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// SubscriptionService defines the minimal interface needed by subscription
// task handlers. It delegates all business logic to the service layer.
type SubscriptionService interface {
	ProcessRenewals() error
	ProcessExpiredDowngrades() error
}

// NewSubscriptionRenewalTask creates a new subscription:renewal task with no payload.
func NewSubscriptionRenewalTask() *asynq.Task {
	return asynq.NewTask(
		TypeSubscriptionRenewal,
		nil,
		asynq.MaxRetry(10),
		asynq.Queue("default"),
		asynq.Timeout(10*time.Minute),
	)
}

// NewSubscriptionDowngradeTask creates a new subscription:downgrade task with no payload.
func NewSubscriptionDowngradeTask() *asynq.Task {
	return asynq.NewTask(
		TypeSubscriptionDowngrade,
		nil,
		asynq.MaxRetry(10),
		asynq.Queue("default"),
		asynq.Timeout(10*time.Minute),
	)
}

// SubscriptionRenewalHandler processes subscription:renewal tasks by
// delegating to the SubscriptionService for renewal logic.
type SubscriptionRenewalHandler struct {
	svc    SubscriptionService
	logger *slog.Logger
}

// NewSubscriptionRenewalHandler creates a new handler for subscription:renewal tasks.
func NewSubscriptionRenewalHandler(svc SubscriptionService, logger *slog.Logger) *SubscriptionRenewalHandler {
	return &SubscriptionRenewalHandler{
		svc:    svc,
		logger: logger,
	}
}

// ProcessTask delegates expired subscription processing to the service layer.
func (h *SubscriptionRenewalHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("processing subscription renewal task")

	if err := h.svc.ProcessRenewals(); err != nil {
		h.logger.Error("subscription renewal failed", "error", err)
		return fmt.Errorf("process renewals: %w", err)
	}

	h.logger.Info("subscription renewal task completed")
	return nil
}

// SubscriptionDowngradeHandler processes subscription:downgrade tasks by
// delegating to the SubscriptionService for downgrade logic.
type SubscriptionDowngradeHandler struct {
	svc    SubscriptionService
	logger *slog.Logger
}

// NewSubscriptionDowngradeHandler creates a new handler for subscription:downgrade tasks.
func NewSubscriptionDowngradeHandler(svc SubscriptionService, logger *slog.Logger) *SubscriptionDowngradeHandler {
	return &SubscriptionDowngradeHandler{
		svc:    svc,
		logger: logger,
	}
}

// ProcessTask delegates pending downgrade processing to the service layer.
func (h *SubscriptionDowngradeHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("processing subscription downgrade task")

	if err := h.svc.ProcessExpiredDowngrades(); err != nil {
		h.logger.Error("subscription downgrade failed", "error", err)
		return fmt.Errorf("process expired downgrades: %w", err)
	}

	h.logger.Info("subscription downgrade task completed")
	return nil
}
