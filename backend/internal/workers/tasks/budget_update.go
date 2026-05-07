package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/services/budgets"
	"github.com/hibiken/asynq"
)

// BudgetUserUpdatePayload is the payload for budget:user_update tasks.
type BudgetUserUpdatePayload struct {
	UserID    int `json:"user_id"`
	AccountID int `json:"account_id"`
}

// NewBudgetUserUpdateTask creates a new budget:user_update task that triggers
// recalculation of budget collected amounts for the given user.
func NewBudgetUserUpdateTask(payload BudgetUserUpdatePayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal budget user update payload: %w", err)
	}

	return asynq.NewTask(
		TypeBudgetUserUpdate,
		data,
		asynq.MaxRetry(10),
		asynq.Queue("critical"),
		asynq.Timeout(5*time.Minute),
	), nil
}

// BudgetService handles budget business logic including recalculation
// and daily processing of outdated budgets (copy creation + archival).
type BudgetService interface {
	RecalculateCollectedAmounts(userID int) error
	DailyProcessing() (*budgets.DailyProcessingResult, error)
}

// BudgetUserUpdateHandler processes budget:user_update tasks by delegating
// to the budget service for recalculation with proper currency conversion.
type BudgetUserUpdateHandler struct {
	budgetSvc BudgetService
	logger    *slog.Logger
}

// NewBudgetUserUpdateHandler creates a new handler for budget:user_update tasks.
func NewBudgetUserUpdateHandler(budgetSvc BudgetService, logger *slog.Logger) *BudgetUserUpdateHandler {
	return &BudgetUserUpdateHandler{
		budgetSvc: budgetSvc,
		logger:    logger,
	}
}

// ProcessTask recalculates collected amounts for all active budgets belonging
// to the user specified in the payload, delegating to the budget service.
func (h *BudgetUserUpdateHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload BudgetUserUpdatePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.logger.Error("failed to unmarshal budget user update payload", "error", err)
		return fmt.Errorf("unmarshal budget user update payload: %w", err)
	}

	h.logger.Info("processing budget user update",
		"userID", payload.UserID,
		"accountID", payload.AccountID,
	)

	if err := h.budgetSvc.RecalculateCollectedAmounts(payload.UserID); err != nil {
		h.logger.Error("budget recalculation failed",
			"userID", payload.UserID,
			"error", err,
		)
		return err
	}

	h.logger.Info("budget user update completed",
		"userID", payload.UserID,
		"accountID", payload.AccountID,
	)

	return nil
}
