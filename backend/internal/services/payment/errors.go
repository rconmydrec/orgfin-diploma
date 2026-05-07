package payment

import "github.com/go-budget/backend/internal/serviceerrors"

// Payment provider errors
var (
	ErrInvalidRequest          = serviceerrors.New(serviceerrors.InvalidInput, "invalid request to payment provider")
	ErrPaymentFailed           = serviceerrors.New(serviceerrors.ProviderError, "payment failed")
	ErrProviderUnavailable     = serviceerrors.New(serviceerrors.ProviderError, "payment provider unavailable")
	ErrCustomerNotFound        = serviceerrors.New(serviceerrors.NotFound, "stripe customer not found")
	ErrSubscriptionNotFound    = serviceerrors.New(serviceerrors.NotFound, "stripe subscription not found")
	ErrInvalidPrice            = serviceerrors.New(serviceerrors.InvalidInput, "invalid or non-existent price ID")
	ErrCheckoutSessionFailed   = serviceerrors.New(serviceerrors.ProviderError, "checkout session creation failed")
	ErrWebhookSignatureInvalid = serviceerrors.New(serviceerrors.ProviderError, "webhook signature verification failed")
)

// Business logic errors
var (
	ErrPlanNotFound           = serviceerrors.New(serviceerrors.NotFound, "subscription plan not found")
	ErrSamePlan               = serviceerrors.New(serviceerrors.Conflict, "already on the target plan")
	ErrInvalidPlanTransition  = serviceerrors.New(serviceerrors.InvalidInput, "invalid plan transition")
	ErrDowngradeNotAllowed    = serviceerrors.New(serviceerrors.InvalidInput, "downgrade not allowed via upgrade endpoint")
	ErrAlreadyOnFreePlan      = serviceerrors.New(serviceerrors.Conflict, "already on free plan")
	ErrInvalidEntitySelection = serviceerrors.New(serviceerrors.InvalidInput, "entity selection violates plan limits")
	ErrNoStripeSubscription   = serviceerrors.New(serviceerrors.Conflict, "no active stripe subscription")
	ErrNoStripeCustomer       = serviceerrors.New(serviceerrors.Conflict, "no stripe customer found")
	ErrScheduleNotFound       = serviceerrors.New(serviceerrors.NotFound, "no active subscription schedule")
	ErrSessionNotFound        = serviceerrors.New(serviceerrors.NotFound, "checkout session not found")
)
