package transactions

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the transactions service.
var (
	ErrInvalidAccount      = serviceerrors.New(serviceerrors.InvalidInput, "invalid account")
	ErrInvalidCategory     = serviceerrors.New(serviceerrors.InvalidInput, "invalid category")
	ErrInvalidUser         = serviceerrors.New(serviceerrors.InvalidInput, "invalid user")
	ErrTransactionNotFound = serviceerrors.New(serviceerrors.NotFound, "transaction not found")
	ErrAccessDenied        = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
	ErrTemplateNotFound    = serviceerrors.New(serviceerrors.NotFound, "template not found")
	ErrUserNotActivated    = serviceerrors.New(serviceerrors.Unauthorized, "user not activated")
	ErrSelfTransfer        = serviceerrors.New(serviceerrors.InvalidInput, "source and target accounts must be different")
)
