package financial_planning

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the financial planning service.
var ErrEndDateBeforeStart = serviceerrors.New(serviceerrors.InvalidInput, "end date must be after start date")
