package subscription

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the subscription service.
var (
	ErrPlanNotFound           = serviceerrors.New(serviceerrors.NotFound, "subscription plan not found")
	ErrSamePlan               = serviceerrors.New(serviceerrors.Conflict, "already on the target plan")
	ErrInvalidPlanTransition  = serviceerrors.New(serviceerrors.InvalidInput, "invalid plan transition")
	ErrDowngradeNotAllowed    = serviceerrors.New(serviceerrors.InvalidInput, "downgrade not allowed via upgrade endpoint")
	ErrAlreadyOnFreePlan      = serviceerrors.New(serviceerrors.Conflict, "already on free plan")
	ErrInvalidEntitySelection = serviceerrors.New(serviceerrors.LimitExceeded, "entity selection violates plan limits")
)
