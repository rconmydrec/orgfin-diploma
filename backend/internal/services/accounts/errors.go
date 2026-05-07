package accounts

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the accounts service.
var (
	ErrInvalidUser          = serviceerrors.New(serviceerrors.InvalidInput, "invalid user")
	ErrInvalidCurrency      = serviceerrors.New(serviceerrors.InvalidInput, "invalid currency")
	ErrInvalidAccountType   = serviceerrors.New(serviceerrors.InvalidInput, "invalid account type")
	ErrInvalidAccount       = serviceerrors.New(serviceerrors.InvalidInput, "invalid account")
	ErrAccessDenied         = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
	ErrAccountNotFound      = serviceerrors.New(serviceerrors.NotFound, "account not found")
	ErrBalanceUnchanged     = serviceerrors.New(serviceerrors.NoChange, "balance unchanged")
	ErrAccountLimitExceeded = serviceerrors.New(serviceerrors.LimitExceeded, "account limit exceeded")
	ErrUserNotActivated     = serviceerrors.New(serviceerrors.Unauthorized, "user not activated")
)
