package categories

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the categories service.
var (
	ErrNotFound     = serviceerrors.New(serviceerrors.NotFound, "category not found")
	ErrAccessDenied = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
)
