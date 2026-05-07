package payment

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/subscriptionschedule"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeClient implements PaymentProvider using the Stripe API.
type StripeClient struct {
	secretKey     string
	webhookSecret string
}

// Compile-time check that StripeClient implements PaymentProvider.
var _ PaymentProvider = (*StripeClient)(nil)

// NewStripeClient creates a new StripeClient and sets the global Stripe API key.
func NewStripeClient(secretKey, webhookSecret string) *StripeClient {
	stripe.Key = secretKey
	return &StripeClient{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// CreateCustomer registers a new customer in Stripe and returns the external
// customer ID.
func (c *StripeClient) CreateCustomer(email string, name string, metadata map[string]string) (string, error) {
	params := &stripe.CustomerParams{
		Email:    stripe.String(email),
		Name:     stripe.String(name),
		Metadata: metadata,
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", mapStripeError(err)
	}

	return cust.ID, nil
}

// GetCustomer retrieves customer details by the Stripe customer ID.
func (c *StripeClient) GetCustomer(customerID string) (*Customer, error) {
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return &Customer{
		ID:      cust.ID,
		Email:   cust.Email,
		Name:    cust.Name,
		Balance: cust.Balance,
	}, nil
}

// UpdateCustomerBalance sets the customer's balance in Stripe. The balance is
// negated because Stripe convention treats negative values as credit.
func (c *StripeClient) UpdateCustomerBalance(customerID string, balanceCents int64) error {
	params := &stripe.CustomerParams{
		Balance: stripe.Int64(-balanceCents),
	}

	_, err := customer.Update(customerID, params)
	if err != nil {
		return mapStripeError(err)
	}

	return nil
}

// CreateCheckoutSession creates a new Stripe Checkout session in subscription
// mode and returns the session details.
func (c *StripeClient) CreateCheckoutSession(params *CheckoutParams) (*CheckoutSession, error) {
	stripeParams := &stripe.CheckoutSessionParams{
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer: stripe.String(params.CustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(params.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(params.SuccessURL),
		CancelURL:  stripe.String(params.CancelURL),
		Metadata:   params.Metadata,
	}

	sess, err := checkoutsession.New(stripeParams)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return mapCheckoutSession(sess), nil
}

// GetCheckoutSession retrieves an existing Stripe Checkout session by ID.
func (c *StripeClient) GetCheckoutSession(sessionID string) (*CheckoutSession, error) {
	sess, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return mapCheckoutSession(sess), nil
}

// CreatePortalSession creates a Stripe Billing Portal session and returns the
// portal URL.
func (c *StripeClient) CreatePortalSession(customerID string, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return "", mapStripeError(err)
	}

	return sess.URL, nil
}

// GetSubscription retrieves a Stripe subscription by ID, expanding the
// items.data.price field.
func (c *StripeClient) GetSubscription(subscriptionID string) (*ProviderSubscription, error) {
	params := &stripe.SubscriptionParams{}
	params.AddExpand("items.data.price")

	sub, err := subscription.Get(subscriptionID, params)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return mapSubscription(sub), nil
}

// CancelSubscription cancels a Stripe subscription. If cancelAtPeriodEnd is
// true, the subscription is set to cancel at the end of the current billing
// period. Otherwise it is cancelled immediately.
func (c *StripeClient) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error {
	if cancelAtPeriodEnd {
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		_, err := subscription.Update(subscriptionID, params)
		if err != nil {
			return mapStripeError(err)
		}
		return nil
	}

	_, err := subscription.Cancel(subscriptionID, nil)
	if err != nil {
		return mapStripeError(err)
	}

	return nil
}

// ReenableSubscription reverses a pending cancellation so the subscription
// renews at the end of the current billing period.
func (c *StripeClient) ReenableSubscription(subscriptionID string) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}

	_, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return mapStripeError(err)
	}

	return nil
}

// UpdateSubscription changes the subscription's price and proration behavior
// and returns the updated subscription state.
func (c *StripeClient) UpdateSubscription(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error) {
	stripeParams := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(params.PriceID),
			},
		},
		ProrationBehavior: stripe.String(params.ProrationBehavior),
	}
	stripeParams.AddExpand("items.data.price")

	sub, err := subscription.Update(subscriptionID, stripeParams)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return mapSubscription(sub), nil
}

// CreateSubscriptionSchedule creates a schedule from an existing subscription
// with the given phases and returns the schedule ID.
func (c *StripeClient) CreateSubscriptionSchedule(subscriptionID string, phases []SchedulePhase) (string, error) {
	params := &stripe.SubscriptionScheduleParams{
		FromSubscription: stripe.String(subscriptionID),
		Phases:           mapSchedulePhases(phases),
	}

	sched, err := subscriptionschedule.New(params)
	if err != nil {
		return "", mapStripeError(err)
	}

	return sched.ID, nil
}

// ModifySubscriptionSchedule replaces the phases of an existing schedule.
func (c *StripeClient) ModifySubscriptionSchedule(scheduleID string, phases []SchedulePhase) error {
	params := &stripe.SubscriptionScheduleParams{
		Phases: mapSchedulePhases(phases),
	}

	_, err := subscriptionschedule.Update(scheduleID, params)
	if err != nil {
		return mapStripeError(err)
	}

	return nil
}

// CancelSubscriptionSchedule releases an existing schedule so no further
// phase transitions occur. The underlying subscription is left in place.
func (c *StripeClient) CancelSubscriptionSchedule(scheduleID string) error {
	_, err := subscriptionschedule.Release(scheduleID, nil)
	if err != nil {
		return mapStripeError(err)
	}

	return nil
}

// VerifyWebhookSignature verifies the cryptographic signature on an incoming
// webhook payload and returns the verified payload bytes on success. The caller
// can safely parse the original payload knowing the signature was validated.
func (c *StripeClient) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	_, err := webhook.ConstructEventWithOptions(payload, signature, c.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookSignatureInvalid, err)
	}

	return payload, nil
}

// mapStripeError converts a Stripe API error into a domain error.
func mapStripeError(err error) error {
	var stripeErr *stripe.Error
	if !errors.As(err, &stripeErr) {
		return err
	}

	switch stripeErr.Type {
	case stripe.ErrorTypeCard:
		return fmt.Errorf("%w: %s", ErrPaymentFailed, stripeErr.Msg)
	case stripe.ErrorTypeInvalidRequest:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, stripeErr.Msg)
	case stripe.ErrorTypeAPI:
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, stripeErr.Msg)
	default:
		// Covers authentication errors, rate limiting, and any other
		// unexpected error types.
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, stripeErr.Msg)
	}
}

// mapCheckoutSession converts a Stripe CheckoutSession to the domain type.
func mapCheckoutSession(sess *stripe.CheckoutSession) *CheckoutSession {
	result := &CheckoutSession{
		ID:            sess.ID,
		URL:           sess.URL,
		Status:        string(sess.Status),
		PaymentStatus: string(sess.PaymentStatus),
		AmountTotal:   sess.AmountTotal,
		Currency:      string(sess.Currency),
		Metadata:      sess.Metadata,
	}

	if sess.Customer != nil {
		result.CustomerID = sess.Customer.ID
	}

	if sess.Subscription != nil {
		result.SubscriptionID = sess.Subscription.ID
	}

	return result
}

// mapSubscription converts a Stripe Subscription to the domain type.
func mapSubscription(sub *stripe.Subscription) *ProviderSubscription {
	result := &ProviderSubscription{
		ID:                sub.ID,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}

	if sub.Customer != nil {
		result.CustomerID = sub.Customer.ID
	}

	if sub.CanceledAt != 0 {
		t := time.Unix(sub.CanceledAt, 0)
		result.CanceledAt = &t
	}

	// In stripe-go v82, CurrentPeriodStart/End live on the subscription
	// items, not on the subscription itself. Use the first item's values.
	if sub.Items != nil {
		for _, item := range sub.Items.Data {
			si := SubscriptionItem{}
			if item.Price != nil {
				si.PriceID = item.Price.ID
			}
			result.Items = append(result.Items, si)

			// Use the first item's billing period for the subscription-level fields.
			if result.CurrentPeriodStart.IsZero() {
				result.CurrentPeriodStart = time.Unix(item.CurrentPeriodStart, 0)
				result.CurrentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0)
			}
		}
	}

	return result
}

// mapSchedulePhases converts domain SchedulePhase values to Stripe params.
// An EndDate of 0 is treated as open-ended (no end date set), which tells
// Stripe to continue the phase indefinitely.
func mapSchedulePhases(phases []SchedulePhase) []*stripe.SubscriptionSchedulePhaseParams {
	result := make([]*stripe.SubscriptionSchedulePhaseParams, len(phases))
	for i, phase := range phases {
		p := &stripe.SubscriptionSchedulePhaseParams{
			StartDate: stripe.Int64(phase.StartDate),
			Items: []*stripe.SubscriptionSchedulePhaseItemParams{
				{
					Price: stripe.String(phase.PriceID),
				},
			},
		}
		if phase.EndDate != 0 {
			p.EndDate = stripe.Int64(phase.EndDate)
		}
		result[i] = p
	}
	return result
}
