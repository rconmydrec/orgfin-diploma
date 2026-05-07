package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// NewBudgetProcessingTask creates a new budget:processing task with no payload.
func NewBudgetProcessingTask() *asynq.Task {
	return asynq.NewTask(
		TypeBudgetProcessing,
		nil,
		asynq.MaxRetry(10),
		asynq.Queue("default"),
		asynq.Timeout(10*time.Minute),
	)
}

// BudgetProcessingHandler processes budget:processing tasks by delegating
// to BudgetService.DailyProcessing() which creates copies of repeating
// budgets with shifted dates and archives all outdated budgets.
type BudgetProcessingHandler struct {
	budgetSvc BudgetService
	logger    *slog.Logger
}

// NewBudgetProcessingHandler creates a new handler for budget:processing tasks.
func NewBudgetProcessingHandler(budgetSvc BudgetService, logger *slog.Logger) *BudgetProcessingHandler {
	return &BudgetProcessingHandler{
		budgetSvc: budgetSvc,
		logger:    logger,
	}
}

// ProcessTask delegates to BudgetService.DailyProcessing() to process all
// outdated budgets. For repeating budgets, new copies with shifted dates are
// created before archival. Partial failures are handled by the service layer:
// individual budget errors are logged and processing continues.
func (h *BudgetProcessingHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("processing budget daily processing task")

	result, err := h.budgetSvc.DailyProcessing()
	if err != nil {
		h.logger.Error("budget daily processing failed", "error", err)
		return fmt.Errorf("budget daily processing: %w", err)
	}

	h.logger.Info("budget daily processing completed",
		"processed", result.Processed,
	)

	return nil
}
