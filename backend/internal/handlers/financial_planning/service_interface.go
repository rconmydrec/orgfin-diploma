package financial_planning

import (
	financialplanning "github.com/go-budget/backend/internal/services/financial_planning"
)

// FinancialPlanningService defines the interface for financial planning service operations
// needed by this handler.
type FinancialPlanningService interface {
	CalculateFutureBalance(userID int, params financialplanning.FutureBalanceParams) (*financialplanning.FutureBalanceResult, error)
	GetBalanceProjection(userID int, params financialplanning.BalanceProjectionParams) (*financialplanning.BalanceProjectionResult, error)
}
