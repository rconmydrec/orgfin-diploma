// Package financial_planning provides financial planning and projection business
// logic including future balance calculation and balance projection over time.
//
// Invariants:
//   - All amounts are converted to the user's base currency.
//   - If the exchange rates table is empty, the operation fails.
//   - Account projections show balances in the account's own currency (not converted).
//   - Only accounts with ShowInReports=true are included.
package financial_planning

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/services/planned_transactions"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the financial planning service.
type ServiceInterface interface {
	CalculateFutureBalance(userID int, params FutureBalanceParams) (*FutureBalanceResult, error)
	GetBalanceProjection(userID int, params BalanceProjectionParams) (*BalanceProjectionResult, error)
}

// PlannedTransactionService defines the methods this package needs
// from the planned_transactions service (consumer-side interface).
type PlannedTransactionService interface {
	GetUpcoming(userID int, days int, includeInactive bool) ([]planned_transactions.Occurrence, error)
}

// CurrencyService defines the currency conversion methods needed by financial planning.
type CurrencyService interface {
	NewRateCache() *currency.RateCache
	ConvertToBaseCurrency(amount decimal.Decimal, sourceCurrencyCode, baseCurrencyCode string, dateStr string, rc *currency.RateCache) (decimal.Decimal, error)
}

// AccountRepository defines account data access methods needed by the service.
type AccountRepository interface {
	GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error)
}

// CurrencyRepository defines currency data access methods needed by the service.
type CurrencyRepository interface {
	GetByID(id int) (*models.Currency, error)
}

// FutureBalanceParams holds the domain parameters for calculating future balance.
type FutureBalanceParams struct {
	TargetDate      time.Time
	AccountIDs      []int
	IncludeInactive bool
	BaseCurrencyID  int
	IncludeHidden   bool
}

// BalanceProjectionParams holds the domain parameters for generating a balance projection.
type BalanceProjectionParams struct {
	StartDate       time.Time
	EndDate         time.Time
	Period          string // "daily", "weekly", "monthly"
	AccountIDs      []int
	IncludeInactive bool
	BaseCurrencyID  int
	IncludeHidden   bool
}

// AccountProjection represents one account's projected balance in its own currency.
type AccountProjection struct {
	AccountID            int
	AccountName          string
	CurrencyCode         string
	CurrentBalance       float64
	ProjectedBalance     float64
	TotalPlannedIncome   float64
	TotalPlannedExpenses float64
}

// FutureBalanceResult contains the complete future balance calculation result.
type FutureBalanceResult struct {
	TargetDate            time.Time
	BaseCurrencyCode      string
	TotalCurrentBalance   float64
	TotalProjectedBalance float64
	TotalPlannedIncome    float64
	TotalPlannedExpenses  float64
	IncomeCount           int
	ExpensesCount         int
	Accounts              []AccountProjection
}

// ProjectionPoint represents one data point in a balance projection timeline.
type ProjectionPoint struct {
	Date     time.Time
	Balance  float64
	Income   float64
	Expenses float64
}

// BalanceProjectionResult contains the complete balance projection result.
type BalanceProjectionResult struct {
	StartDate        time.Time
	EndDate          time.Time
	Period           string
	BaseCurrencyCode string
	ProjectionPoints []ProjectionPoint
}
