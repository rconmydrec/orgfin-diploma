package planned_transactions

import (
	plannedtransactions "github.com/go-budget/backend/internal/services/planned_transactions"
)

// PlannedTransactionsService defines the interface for planned transaction service operations
// needed by this handler.
type PlannedTransactionsService interface {
	Create(userID int, baseCurrencyID int, params plannedtransactions.CreateParams) (*plannedtransactions.PlannedTransaction, error)
	List(userID int, filters plannedtransactions.ListFilters) ([]*plannedtransactions.PlannedTransaction, error)
	GetUpcoming(userID int, days int, includeInactive bool) ([]plannedtransactions.Occurrence, error)
	GetByID(userID, id int) (*plannedtransactions.PlannedTransaction, error)
	Update(userID, id int, params plannedtransactions.UpdateParams) (*plannedtransactions.PlannedTransaction, error)
	Delete(userID, id int) error
}
