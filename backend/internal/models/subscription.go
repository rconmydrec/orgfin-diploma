package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Subscription represents the user subscription data.
type Subscription struct {
	ID                         int
	UserID                     int
	PlanID                     int
	PlanType                   string
	CurrentBillingPeriodID     *int
	TrialStartedAt             *time.Time
	TrialEndsAt                *time.Time
	SubscribedAt               *time.Time
	ExpiresAt                  *time.Time
	AutoRenew                  bool
	CanceledAt                 *time.Time
	PendingPlanID              *int
	PendingDowngradeAccountIDs []int
	PendingDowngradeBudgetID   *int
	IsActive                   bool
	HasStripeSubscription      bool
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// SubscriptionPlan represents a subscription plan.
type SubscriptionPlan struct {
	ID             int
	Name           string
	TranslationKey *string
	PlanType       string
	Price          decimal.Decimal
	CurrencyCode   string
	IsFeatured     bool
	Description    *string
	SortOrder      int
	BillingPeriod  *BillingPeriod
}

// BillingPeriod represents a billing period.
type BillingPeriod struct {
	ID           int
	Code         string
	Name         string
	DurationDays int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PaymentProviderSubscription represents a payment provider subscription record.
type PaymentProviderSubscription struct {
	ID                     int
	SubscriptionID         int
	ProviderType           string // stripe
	ExternalCustomerID     *string
	ExternalSubscriptionID *string
	ExternalScheduleID     *string
	PaymentMethodID        *string
	LastPaymentFailed      bool
	ProviderMetadata       map[string]any
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
