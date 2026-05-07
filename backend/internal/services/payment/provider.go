// Package payment defines the PaymentProvider interface and supporting types
// for payment gateway integrations. Concrete implementations (e.g. Stripe)
// live in separate files within this package.
package payment

import "time"

// ProviderTypeStripe is the canonical provider type value for Stripe.
// It must match the PostgreSQL enum `paymentprovidertype` which stores the
// value in uppercase.
const ProviderTypeStripe = "STRIPE"

// PaymentProvider abstracts the operations required from a payment gateway.
// Each method works with provider-agnostic DTOs so the business logic layer
// is decoupled from any specific payment vendor.
type PaymentProvider interface {
	// CreateCustomer registers a new customer in the payment provider and
	// returns the external customer ID.
	CreateCustomer(email string, name string, metadata map[string]string) (string, error)

	// GetCustomer retrieves customer details by the provider's customer ID.
	GetCustomer(customerID string) (*Customer, error)

	// UpdateCustomerBalance sets the customer's balance (in cents) in the
	// payment provider. This is used for applying prorated credits.
	UpdateCustomerBalance(customerID string, balanceCents int64) error

	// CreateCheckoutSession creates a new hosted checkout page and returns
	// session details including the redirect URL.
	CreateCheckoutSession(params *CheckoutParams) (*CheckoutSession, error)

	// GetCheckoutSession retrieves an existing checkout session by ID.
	GetCheckoutSession(sessionID string) (*CheckoutSession, error)

	// CreatePortalSession creates a customer billing portal session and
	// returns the portal URL.
	CreatePortalSession(customerID string, returnURL string) (string, error)

	// GetSubscription retrieves a subscription by its provider-side ID.
	GetSubscription(subscriptionID string) (*ProviderSubscription, error)

	// CancelSubscription cancels a subscription. When cancelAtPeriodEnd is
	// true the subscription remains active until the end of the current
	// billing period; otherwise it is terminated immediately.
	CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error

	// ReenableSubscription reverses a pending cancellation so the
	// subscription renews at period end.
	ReenableSubscription(subscriptionID string) error

	// UpdateSubscription changes the subscription (e.g. plan swap) and
	// returns the updated subscription state.
	UpdateSubscription(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error)

	// CreateSubscriptionSchedule creates a schedule that will transition the
	// subscription through the given phases and returns the schedule ID.
	CreateSubscriptionSchedule(subscriptionID string, phases []SchedulePhase) (string, error)

	// ModifySubscriptionSchedule replaces the phases of an existing schedule.
	ModifySubscriptionSchedule(scheduleID string, phases []SchedulePhase) error

	// CancelSubscriptionSchedule cancels (releases) an existing schedule
	// so no further phase transitions occur.
	CancelSubscriptionSchedule(scheduleID string) error

	// VerifyWebhookSignature verifies the cryptographic signature on an
	// incoming webhook payload and returns the verified event JSON.
	VerifyWebhookSignature(payload []byte, signature string) ([]byte, error)
}

// Customer represents a payment provider customer record.
type Customer struct {
	ID      string
	Email   string
	Name    string
	Balance int64 // in cents
}

// CheckoutParams contains the parameters needed to create a checkout session.
type CheckoutParams struct {
	CustomerID string
	PriceID    string
	SuccessURL string
	CancelURL  string
	Metadata   map[string]string
}

// CheckoutSession represents the state of a hosted checkout session.
type CheckoutSession struct {
	ID             string
	URL            string
	Status         string
	PaymentStatus  string
	CustomerID     string
	SubscriptionID string
	AmountTotal    int64
	Currency       string
	Metadata       map[string]string
}

// ProviderSubscription represents a subscription as tracked by the payment
// provider. Field names are intentionally provider-agnostic.
type ProviderSubscription struct {
	ID                 string
	Status             string
	CustomerID         string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	CanceledAt         *time.Time
	Items              []SubscriptionItem
}

// SubscriptionItem represents a single line item within a subscription.
type SubscriptionItem struct {
	PriceID string
}

// UpdateSubscriptionParams holds the parameters for modifying an active
// subscription (e.g. changing the price / plan).
type UpdateSubscriptionParams struct {
	PriceID           string
	ProrationBehavior string // "create_prorations", "none", etc.
}

// SchedulePhase defines a single phase within a subscription schedule.
// Start and end dates are represented as Unix timestamps.
type SchedulePhase struct {
	StartDate int64 // Unix timestamp
	EndDate   int64 // Unix timestamp
	PriceID   string
}

// WebhookResult carries the outcome of processing a single webhook event.
type WebhookResult struct {
	EventType string
	Success   bool
	Message   string
}
