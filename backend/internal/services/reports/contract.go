// Package reports provides report generation business logic including cash flow,
// balance snapshots, expenses by category, diagram data, and raw expense data.
//
// Invariants:
//   - All amounts are converted to the user's base currency using exchange rates.
//   - If the exchange rates table is empty (ErrNoExchangeRates), the report fails.
//   - If a specific currency is missing from rates, the transaction is skipped with a warning.
//   - The "Other" threshold (2%) combines small categories into a single entry.
package reports

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the reports service.
type ServiceInterface interface {
	CashFlowReport(userID int, params CashFlowParams) (*CashFlowResult, error)
	BalanceReport(userID int, params BalanceParams) ([]BalanceItem, error)
	ExpensesByCategories(userID int, params ExpensesParams) ([]ExpensesCategoryItem, error)
	DiagramData(userID int, params DiagramParams) (*DiagramResult, error)
	ExpensesData(userID int, params ExpensesParams) ([]ExpenseDataEntry, error)
}

// CurrencyService defines the currency conversion methods needed by reports.
type CurrencyService interface {
	NewRateCache() *currency.RateCache
	ConvertToBaseCurrency(amount decimal.Decimal, sourceCurrencyCode, baseCurrencyCode string, dateStr string, rc *currency.RateCache) (decimal.Decimal, error)
}

// TransactionRepository defines transaction data access methods needed by the service.
type TransactionRepository interface {
	GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error)
}

// AccountRepository defines account data access methods needed by the service.
type AccountRepository interface {
	GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error)
}

// CategoryRepository defines category data access methods needed by the service.
type CategoryRepository interface {
	GetByUserID(userID int) ([]*models.UserCategory, error)
}

// CurrencyRepository defines currency data access methods needed by the service.
type CurrencyRepository interface {
	GetByID(id int) (*models.Currency, error)
}

// CashFlowParams holds the domain parameters for generating a cash flow report.
type CashFlowParams struct {
	StartDate      *time.Time // nil means "use default"
	EndDate        *time.Time // nil means "use default"
	Period         string     // "daily", "monthly"
	BaseCurrencyID int
}

// BalanceParams holds the domain parameters for generating a balance report.
type BalanceParams struct {
	AccountIDs     []int
	BalanceDate    string // "2006-01-02" format
	ExcludeHidden  bool
	BaseCurrencyID int
}

// ExpensesParams holds the domain parameters for expenses-by-categories
// and expenses-data reports.
type ExpensesParams struct {
	StartDate           string // "2006-01-02" format
	EndDate             string // "2006-01-02" format
	Categories          []int  // empty = all categories
	HideEmptyCategories bool
	BaseCurrencyID      int
}

// DiagramParams holds the domain parameters for diagram data generation.
type DiagramParams struct {
	StartDate      string // "2006-01-02" format
	EndDate        string // "2006-01-02" format
	DiagramType    string // e.g., "pie" (currently unused in logic but passed through)
	BaseCurrencyID int
}

// CashFlowResult contains the aggregated cash flow data.
type CashFlowResult struct {
	CurrencyCode  string
	TotalIncome   map[string]float64
	TotalExpenses map[string]float64
	NetFlow       map[string]float64
}

// BalanceItem represents one account's balance at a point in time.
type BalanceItem struct {
	AccountID           int
	AccountName         string
	AccountTypeID       int
	CurrencyCode        string
	Balance             float64
	BaseCurrencyBalance float64
	BaseCurrencyCode    string
	ReportDate          string
}

// ExpensesCategoryItem represents one category's total expenses.
type ExpensesCategoryItem struct {
	ID            int
	Name          string // display name, e.g., "Parent >> Child"
	ParentID      *int
	ParentName    *string
	TotalExpenses float64
	CurrencyCode  string
	IsParent      bool
}

// ExpenseDataEntry represents one parent-level expense entry for diagrams.
type ExpenseDataEntry struct {
	Label        string
	Amount       float64
	CategoryID   int
	CategoryName string
}

// DiagramResult contains labels and data arrays for chart rendering.
type DiagramResult struct {
	Labels       []string
	Data         []float64
	CurrencyCode string
}
