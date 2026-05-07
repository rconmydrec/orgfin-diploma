package budgets

import (
	budgetsservice "github.com/go-budget/backend/internal/services/budgets"
)

// BudgetsService defines the interface for budget service operations
// needed by this handler.
type BudgetsService interface {
	Create(userID int, params budgetsservice.CreateParams) (*budgetsservice.Budget, error)
	GetByUserID(userID int, include string) ([]*budgetsservice.Budget, error)
	Update(userID, budgetID int, params budgetsservice.UpdateParams) (*budgetsservice.Budget, error)
	Delete(userID, budgetID int) error
	Archive(userID, budgetID int) error
	DailyProcessing() (*budgetsservice.DailyProcessingResult, error)
}
