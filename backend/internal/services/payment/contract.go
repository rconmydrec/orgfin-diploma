// Package payment provides Stripe payment integration including checkout,
// portal, subscription management, credit calculation, and webhook processing.
//
// Invariants:
//   - Checkout sessions store user_id and plan_id in metadata for webhook verification.
//   - Portal sessions only allow return URLs starting with the configured frontend URL.
//   - Upgrades are applied immediately with prorations.
//   - Downgrades are scheduled for end of current billing period via subscription schedules.
//   - Webhook handler always returns success to the provider (errors are logged internally).
package payment

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the payment service.
type ServiceInterface interface {
	GetOrCreateCustomer(userID int, email string, name string) (string, error)
	CreateCheckout(userID int, planID int, email string, name string) (*CheckoutResult, error)
	CreatePortal(userID int, returnURL *string) (*PortalResult, error)
	GetUpgradePrice(userID int, targetPlanID int) (*UpgradePriceInfo, error)
	ChangePlan(userID int, targetPlanID int) (*ChangePlanResult, error)
	CancelScheduledChange(userID int) error
	GetPaymentStatus(userID int) (*PaymentStatus, error)
	GetSessionStatus(userID int, sessionID string) (*SessionStatus, error)
}

// SubscriptionRepository defines subscription data access methods needed by the service.
type SubscriptionRepository interface {
	GetByID(id int) (*models.Subscription, error)
	GetByUserID(userID int) (*models.Subscription, error)
	UpdateSubscriptionFull(sub *models.Subscription) error
	GetProviderSubscription(subscriptionID int) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByExternalID(externalSubscriptionID string) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByScheduleID(externalScheduleID string) (*models.PaymentProviderSubscription, error)
	CreateProviderSubscription(pps *models.PaymentProviderSubscription) error
	UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error
}

// SubscriptionPlanRepository defines plan data access methods needed by the service.
type SubscriptionPlanRepository interface {
	GetByID(id int) (*models.SubscriptionPlan, error)
	GetFreePlan() (*models.SubscriptionPlan, error)
}

// PlanPriceRepository defines plan price data access methods needed by the service.
type PlanPriceRepository interface {
	GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error)
	GetByExternalPriceID(externalPriceID string) (*models.PlanPrice, error)
}

// CreditCalculator defines the methods needed for prorated credit calculations.
type CreditCalculator interface {
	CalculateRemainingCredit(currentPlanPrice decimal.Decimal, billingPeriodDays int, expiresAt time.Time) decimal.Decimal
	FormatCreditForDisplay(credit decimal.Decimal, currencyCode string) string
}

// CheckoutResult holds the outcome of creating a Stripe Checkout session,
// including any prorated credit that was applied to the customer balance.
type CheckoutResult struct {
	CheckoutURL            string
	SessionID              string
	CreditAppliedCents     int64
	CreditAppliedFormatted string
}

// PortalResult holds the URL for a Stripe Billing Portal session.
type PortalResult struct {
	PortalURL string
}

// UpgradePriceInfo holds the price breakdown for upgrading to a different plan,
// including any prorated credit from the current subscription.
type UpgradePriceInfo struct {
	PlanID                 int
	PlanName               string
	PlanType               string
	OriginalPriceCents     int64
	CreditCents            int64
	FinalPriceCents        int64
	OriginalPriceFormatted string
	CreditFormatted        string
	FinalPriceFormatted    string
	CurrencyCode           string
}

// ChangePlanResult holds the outcome of a plan change operation.
type ChangePlanResult struct {
	Success       bool
	IsUpgrade     bool
	NewPlanName   string
	EffectiveDate string
	Message       string
	ScheduleID    *string
}

// PaymentStatus holds the current payment and subscription state for a user.
type PaymentStatus struct {
	HasStripeCustomer        bool
	StripeSubscriptionActive bool
	StripeSubscriptionID     *string
	CurrentPlanType          string
	LastPaymentFailed        bool
}

// SessionStatus holds the state of a Stripe Checkout session.
type SessionStatus struct {
	SessionID      string
	Status         string
	PaymentStatus  string
	CustomerID     string
	SubscriptionID string
	AmountTotal    int64
	Currency       string
}
