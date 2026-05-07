package types

import (
	"time"

	"github.com/shopspring/decimal"
)

// BudgetTransactionRow represents a single transaction row returned by
// GetExpenseTransactionsForBudget. It contains only the fields needed
// for budget recalculation with currency conversion.
type BudgetTransactionRow struct {
	Amount       decimal.Decimal
	CurrencyCode string
	DateTime     time.Time
}
