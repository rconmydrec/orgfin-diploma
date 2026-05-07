package reports

import (
	reportsservice "github.com/go-budget/backend/internal/services/reports"
)

// ReportsService defines the interface for report service operations
// needed by this handler.
type ReportsService interface {
	CashFlowReport(userID int, params reportsservice.CashFlowParams) (*reportsservice.CashFlowResult, error)
	BalanceReport(userID int, params reportsservice.BalanceParams) ([]reportsservice.BalanceItem, error)
	ExpensesByCategories(userID int, params reportsservice.ExpensesParams) ([]reportsservice.ExpensesCategoryItem, error)
	DiagramData(userID int, params reportsservice.DiagramParams) (*reportsservice.DiagramResult, error)
	ExpensesData(userID int, params reportsservice.ExpensesParams) ([]reportsservice.ExpenseDataEntry, error)
}
