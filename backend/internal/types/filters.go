// Package types provides shared data transfer types used across repository
// and service layers. These are pure value structs with no behavior, no methods,
// and no imports of models, services, or repositories.
package types

// AccountFilters for filtering accounts.
type AccountFilters struct {
	IncludeHidden     bool
	IncludeArchived   bool
	ArchivedOnly      bool
	IncludeDeleted    bool
	OnlyShowInReports bool
}

// TransactionFilters represents filters for transaction queries.
type TransactionFilters struct {
	AccountIDs  []int
	CategoryIDs []int
	DateFrom    *string
	DateTo      *string
	Type        *string
	Limit       int
	Offset      int
	NoLimit     bool // When true, skip the LIMIT clause entirely (used by reports)
}

// PlannedTxFilters for filtering planned transactions.
type PlannedTxFilters struct {
	AccountIDs      []int
	FromDate        *string
	ToDate          *string
	IsRecurring     string
	IsExecuted      string
	IsActive        string
	IncludeInactive bool
}
