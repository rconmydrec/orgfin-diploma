package planned_transactions

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the planned transactions service.
var (
	ErrBaseCurrencyNotSet = serviceerrors.New(serviceerrors.InvalidInput, "base currency not set")
	ErrNotFound           = serviceerrors.New(serviceerrors.NotFound, "planned transaction not found")
	ErrAccessDenied       = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
)
