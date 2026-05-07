package budgets

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the budgets service.
var (
	ErrNotFound         = serviceerrors.New(serviceerrors.NotFound, "budget not found")
	ErrAccessDenied     = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
	ErrInvalidCurrency  = serviceerrors.New(serviceerrors.InvalidInput, "invalid currency")
	ErrInvalidStartDate = serviceerrors.New(serviceerrors.InvalidInput, "invalid start date format")
	ErrInvalidEndDate   = serviceerrors.New(serviceerrors.InvalidInput, "invalid end date format")
)
