// Package planned_transactions provides planned transactions CRUD with
// ownership checks and occurrence generation from recurrence rules.
//
// Invariants:
//   - All mutations verify ownership via UserID.
//   - Base currency must be set for creation.
//   - Recurrence rules are stored as snake_case JSON in the database.
//   - Occurrence generation respects rule end dates, count limits, and a max iterations guard.
package planned_transactions

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the planned transactions service.
type ServiceInterface interface {
	Create(userID int, baseCurrencyID int, params CreateParams) (*PlannedTransaction, error)
	List(userID int, filters ListFilters) ([]*PlannedTransaction, error)
	GetUpcoming(userID int, days int, includeInactive bool) ([]Occurrence, error)
	GetByID(userID, id int) (*PlannedTransaction, error)
	Update(userID, id int, params UpdateParams) (*PlannedTransaction, error)
	Delete(userID, id int) error
}

// PlannedTransaction represents a planned transaction in the service domain.
type PlannedTransaction struct {
	ID                    int
	UserID                int
	CurrencyID            int
	Amount                decimal.Decimal
	Label                 string
	Notes                 *string
	IsIncome              bool
	PlannedDate           time.Time
	IsRecurring           bool
	RecurrenceRule        *string
	IsExecuted            bool
	ExecutedTransactionID *int
	ExecutionDate         *time.Time
	IsActive              bool
	IsDeleted             bool
	CreatedAt             time.Time
	UpdatedAt             time.Time

	// Relations
	Currency *Currency
}

// Currency represents a currency in the planned transactions service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// PlannedTransactionRepository defines planned transaction data access methods needed by the service.
type PlannedTransactionRepository interface {
	Create(tx *models.PlannedTransaction) (*models.PlannedTransaction, error)
	GetByID(id int) (*models.PlannedTransaction, error)
	GetByUserID(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error)
	GetActiveByUserID(userID int, includeInactive bool) ([]*models.PlannedTransaction, error)
	Update(tx *models.PlannedTransaction) error
	Delete(id int) error
}

// Occurrence represents an upcoming occurrence of a planned transaction
// within a given date range.
type Occurrence struct {
	PlannedTransactionID int             `json:"plannedTransactionId"`
	CurrencyID           int             `json:"currencyId"`
	OccurrenceDate       time.Time       `json:"occurrenceDate"`
	Amount               decimal.Decimal `json:"amount"`
	IsIncome             bool            `json:"isIncome"`
	Label                string          `json:"label"`
	IsActive             bool            `json:"isActive"`
	IsRecurring          bool            `json:"isRecurring"`
}

// CreateParams holds the domain parameters for creating a planned transaction.
type CreateParams struct {
	AccountID      *int
	Amount         decimal.Decimal
	Label          string
	Notes          *string
	IsIncome       bool
	PlannedDate    time.Time
	IsRecurring    bool
	RecurrenceRule *RecurrenceRuleParams
}

// RecurrenceRuleParams holds the recurrence rule parameters
// in domain form (not serialized JSON).
type RecurrenceRuleParams struct {
	Frequency  string
	Interval   int
	EndDate    *time.Time
	Count      *int
	DayOfMonth *int
}

// UpdateParams holds the domain parameters for updating a planned transaction.
type UpdateParams struct {
	Amount         decimal.Decimal
	Label          string
	Notes          *string
	IsIncome       bool
	PlannedDate    time.Time
	IsRecurring    bool
	RecurrenceRule *RecurrenceRuleParams
	IsActive       *bool
}

// ListFilters holds the domain parameters for listing planned transactions.
type ListFilters struct {
	AccountIDs      []int
	FromDate        *string
	ToDate          *string
	IsRecurring     string
	IsExecuted      string
	IsActive        string
	IncludeInactive bool
}
