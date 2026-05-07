// Package budgets provides budget business logic including CRUD operations,
// recalculation of collected amounts with currency conversion, and daily
// processing of outdated budgets.
//
// Invariants:
//   - End dates are always stored as start+1 day (to make the range inclusive).
//   - Period is normalized to uppercase before storage.
//   - Categories are stored as a comma-separated string of IDs.
//   - DailyProcessing handles partial failures: individual budget errors are logged, not propagated.
package budgets

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the budgets service.
type ServiceInterface interface {
	Create(userID int, params CreateParams) (*Budget, error)
	GetByUserID(userID int, include string) ([]*Budget, error)
	Update(userID, budgetID int, params UpdateParams) (*Budget, error)
	Delete(userID, budgetID int) error
	Archive(userID, budgetID int) error
	DailyProcessing() (*DailyProcessingResult, error)
	RecalculateCollectedAmounts(userID int) error
}

// Budget represents a budget in the budgets service domain.
type Budget struct {
	ID                 int
	UserID             int
	Name               string
	CurrencyID         int
	TargetAmount       decimal.Decimal
	CollectedAmount    decimal.Decimal
	Period             string
	Repeat             bool
	StartDate          time.Time
	EndDate            time.Time
	IncludedCategories string
	IsArchived         bool
	Comment            *string
	IsDeleted          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time

	// Relations
	Currency *Currency
}

// Currency represents a currency in the budgets service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// CurrencyConverter converts amounts between currencies using exchange rates.
type CurrencyConverter interface {
	ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal
}

// BudgetRepository defines budget CRUD methods needed by the service.
type BudgetRepository interface {
	GetByUserID(userID int, include string) ([]*models.Budget, error)
	UpdateCollectedAmount(budgetID int, amount decimal.Decimal) error
	Create(budget *models.Budget) (*models.Budget, error)
	GetByID(id int) (*models.Budget, error)
	GetOutdatedBudgets() ([]*models.Budget, error)
	Update(budget *models.Budget) error
	Delete(id int) error
	Archive(id int) error
}

// TransactionRepository queries transactions for budget recalculation.
type TransactionRepository interface {
	GetExpenseTransactionsForBudget(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error)
}

// CurrencyRepository provides currency lookup for validation.
type CurrencyRepository interface {
	GetByID(id int) (*models.Currency, error)
}

// CreateParams holds the domain parameters for creating a budget.
type CreateParams struct {
	Name         string
	CurrencyID   int
	TargetAmount decimal.Decimal
	Period       string
	Repeat       bool
	StartDate    string // raw date string, parsed by service
	EndDate      string // raw date string, parsed by service
	Categories   []int
	Comment      *string
}

// UpdateParams holds the domain parameters for updating a budget.
type UpdateParams struct {
	Name         string
	CurrencyID   int
	TargetAmount decimal.Decimal
	Period       string
	Repeat       bool
	StartDate    string
	EndDate      string
	Categories   []int
	Comment      *string
}

// DailyProcessingResult holds the result of the daily processing operation.
type DailyProcessingResult struct {
	Processed int
}
