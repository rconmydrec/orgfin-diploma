package payment

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"
)

// TestStripeClientImplementsPaymentProvider verifies at compile time that
// StripeClient satisfies the PaymentProvider interface.
func TestStripeClientImplementsPaymentProvider(t *testing.T) {
	var _ PaymentProvider = (*StripeClient)(nil)
}

func TestNewStripeClient(t *testing.T) {
	const testKey = "sk_test_abc123"
	const testWebhookSecret = "whsec_test_secret"

	client := NewStripeClient(testKey, testWebhookSecret)

	assert.Equal(t, testKey, client.secretKey)
	assert.Equal(t, testWebhookSecret, client.webhookSecret)
	assert.Equal(t, testKey, stripe.Key, "stripe.Key should be set to the provided secret key")
}

func TestMapStripeError_CardError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type: stripe.ErrorTypeCard,
		Msg:  "Your card was declined",
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrPaymentFailed))
	assert.Contains(t, result.Error(), "Your card was declined")
}

func TestMapStripeError_InvalidRequestError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type: stripe.ErrorTypeInvalidRequest,
		Msg:  "No such customer",
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrInvalidRequest))
	assert.Contains(t, result.Error(), "No such customer")
}

func TestMapStripeError_APIError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type: stripe.ErrorTypeAPI,
		Msg:  "Internal server error",
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
	assert.Contains(t, result.Error(), "Internal server error")
}

func TestMapStripeError_AuthenticationError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type:           stripe.ErrorTypeAPI,
		Msg:            "Invalid API Key provided",
		HTTPStatusCode: http.StatusUnauthorized,
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
	assert.Contains(t, result.Error(), "Invalid API Key provided")
}

func TestMapStripeError_RateLimitError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type:           stripe.ErrorTypeAPI,
		Msg:            "Too many requests",
		HTTPStatusCode: http.StatusTooManyRequests,
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
	assert.Contains(t, result.Error(), "Too many requests")
}

func TestMapStripeError_401StatusCode(t *testing.T) {
	// Test that a non-standard error type with 401 status maps to
	// ErrProviderUnavailable.
	stripeErr := &stripe.Error{
		Type:           stripe.ErrorTypeTemporarySessionExpired,
		Msg:            "Authentication required",
		HTTPStatusCode: http.StatusUnauthorized,
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
}

func TestMapStripeError_429StatusCode(t *testing.T) {
	// Test that a non-standard error type with 429 status maps to
	// ErrProviderUnavailable.
	stripeErr := &stripe.Error{
		Type:           stripe.ErrorTypeTemporarySessionExpired,
		Msg:            "Rate limited",
		HTTPStatusCode: http.StatusTooManyRequests,
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
}

func TestMapStripeError_NonStripeError(t *testing.T) {
	plainErr := errors.New("network timeout")

	result := mapStripeError(plainErr)

	assert.Equal(t, plainErr, result, "non-Stripe errors should be returned as-is")
}

func TestMapStripeError_UnknownType(t *testing.T) {
	stripeErr := &stripe.Error{
		Type:           "some_unknown_type",
		Msg:            "Something went wrong",
		HTTPStatusCode: http.StatusInternalServerError,
	}

	result := mapStripeError(stripeErr)

	assert.True(t, errors.Is(result, ErrProviderUnavailable))
	assert.Contains(t, result.Error(), "Something went wrong")
}

func TestVerifyWebhookSignature_InvalidSignature(t *testing.T) {
	client := &StripeClient{
		secretKey:     "sk_test_key",
		webhookSecret: "whsec_test_secret",
	}

	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed"}`)
	invalidSig := "t=1234567890,v1=invalid_signature"

	result, err := client.VerifyWebhookSignature(payload, invalidSig)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrWebhookSignatureInvalid))
}

func TestVerifyWebhookSignature_EmptySignature(t *testing.T) {
	client := &StripeClient{
		secretKey:     "sk_test_key",
		webhookSecret: "whsec_test_secret",
	}

	payload := []byte(`{"id":"evt_test"}`)

	result, err := client.VerifyWebhookSignature(payload, "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrWebhookSignatureInvalid))
}

func TestMapCheckoutSession(t *testing.T) {
	stripeSession := &stripe.CheckoutSession{
		ID:            "cs_test_123",
		URL:           "https://checkout.stripe.com/pay/cs_test_123",
		Status:        stripe.CheckoutSessionStatusOpen,
		PaymentStatus: stripe.CheckoutSessionPaymentStatusUnpaid,
		AmountTotal:   999,
		Currency:      "usd",
		Customer: &stripe.Customer{
			ID: "cus_test_456",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test_789",
		},
	}

	result := mapCheckoutSession(stripeSession)

	assert.Equal(t, "cs_test_123", result.ID)
	assert.Equal(t, "https://checkout.stripe.com/pay/cs_test_123", result.URL)
	assert.Equal(t, "open", result.Status)
	assert.Equal(t, "unpaid", result.PaymentStatus)
	assert.Equal(t, int64(999), result.AmountTotal)
	assert.Equal(t, "usd", result.Currency)
	assert.Equal(t, "cus_test_456", result.CustomerID)
	assert.Equal(t, "sub_test_789", result.SubscriptionID)
}

func TestMapCheckoutSession_NilCustomerAndSubscription(t *testing.T) {
	stripeSession := &stripe.CheckoutSession{
		ID:            "cs_test_123",
		Status:        stripe.CheckoutSessionStatusComplete,
		PaymentStatus: stripe.CheckoutSessionPaymentStatusPaid,
	}

	result := mapCheckoutSession(stripeSession)

	assert.Equal(t, "cs_test_123", result.ID)
	assert.Empty(t, result.CustomerID)
	assert.Empty(t, result.SubscriptionID)
}

func TestMapSubscription(t *testing.T) {
	sub := &stripe.Subscription{
		ID:                "sub_test_123",
		Status:            stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd: false,
		CanceledAt:        1700000000,
		Customer: &stripe.Customer{
			ID: "cus_test_456",
		},
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					CurrentPeriodStart: 1700000000,
					CurrentPeriodEnd:   1702592000,
					Price: &stripe.Price{
						ID: "price_test_abc",
					},
				},
			},
		},
	}

	result := mapSubscription(sub)

	assert.Equal(t, "sub_test_123", result.ID)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, "cus_test_456", result.CustomerID)
	assert.False(t, result.CancelAtPeriodEnd)
	require.NotNil(t, result.CanceledAt)
	assert.Equal(t, int64(1700000000), result.CanceledAt.Unix())
	assert.Equal(t, int64(1700000000), result.CurrentPeriodStart.Unix())
	assert.Equal(t, int64(1702592000), result.CurrentPeriodEnd.Unix())
	require.Len(t, result.Items, 1)
	assert.Equal(t, "price_test_abc", result.Items[0].PriceID)
}

func TestMapSubscription_NilCustomer(t *testing.T) {
	sub := &stripe.Subscription{
		ID:     "sub_test_123",
		Status: stripe.SubscriptionStatusActive,
	}

	result := mapSubscription(sub)

	assert.Empty(t, result.CustomerID)
}

func TestMapSubscription_NoCanceledAt(t *testing.T) {
	sub := &stripe.Subscription{
		ID:     "sub_test_123",
		Status: stripe.SubscriptionStatusActive,
	}

	result := mapSubscription(sub)

	assert.Nil(t, result.CanceledAt)
}

func TestMapSubscription_MultipleItems(t *testing.T) {
	sub := &stripe.Subscription{
		ID:     "sub_test_123",
		Status: stripe.SubscriptionStatusActive,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					CurrentPeriodStart: 1700000000,
					CurrentPeriodEnd:   1702592000,
					Price: &stripe.Price{
						ID: "price_one",
					},
				},
				{
					CurrentPeriodStart: 1700100000,
					CurrentPeriodEnd:   1702692000,
					Price: &stripe.Price{
						ID: "price_two",
					},
				},
			},
		},
	}

	result := mapSubscription(sub)

	require.Len(t, result.Items, 2)
	assert.Equal(t, "price_one", result.Items[0].PriceID)
	assert.Equal(t, "price_two", result.Items[1].PriceID)
	// Period should come from the first item.
	assert.Equal(t, int64(1700000000), result.CurrentPeriodStart.Unix())
	assert.Equal(t, int64(1702592000), result.CurrentPeriodEnd.Unix())
}

func TestMapSubscription_NilPrice(t *testing.T) {
	sub := &stripe.Subscription{
		ID:     "sub_test_123",
		Status: stripe.SubscriptionStatusActive,
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					CurrentPeriodStart: 1700000000,
					CurrentPeriodEnd:   1702592000,
					Price:              nil,
				},
			},
		},
	}

	result := mapSubscription(sub)

	require.Len(t, result.Items, 1)
	assert.Empty(t, result.Items[0].PriceID)
}

func TestMapSchedulePhases(t *testing.T) {
	phases := []SchedulePhase{
		{
			StartDate: 1700000000,
			EndDate:   1702592000,
			PriceID:   "price_abc",
		},
		{
			StartDate: 1702592000,
			EndDate:   1705184000,
			PriceID:   "price_def",
		},
	}

	result := mapSchedulePhases(phases)

	require.Len(t, result, 2)

	assert.Equal(t, int64(1700000000), *result[0].StartDate)
	assert.Equal(t, int64(1702592000), *result[0].EndDate)
	require.Len(t, result[0].Items, 1)
	assert.Equal(t, "price_abc", *result[0].Items[0].Price)

	assert.Equal(t, int64(1702592000), *result[1].StartDate)
	assert.Equal(t, int64(1705184000), *result[1].EndDate)
	require.Len(t, result[1].Items, 1)
	assert.Equal(t, "price_def", *result[1].Items[0].Price)
}

func TestMapSchedulePhases_Empty(t *testing.T) {
	result := mapSchedulePhases(nil)
	assert.Empty(t, result)
}

func TestMapStripeError_IdempotencyError(t *testing.T) {
	stripeErr := &stripe.Error{
		Type: stripe.ErrorTypeIdempotency,
		Msg:  "Idempotency key already used",
	}

	result := mapStripeError(stripeErr)

	// Idempotency errors fall into the default case and map to
	// ErrProviderUnavailable.
	assert.True(t, errors.Is(result, ErrProviderUnavailable))
}

// TestMapCheckoutSession_JSONRoundTrip verifies that the domain CheckoutSession
// can be serialized and deserialized correctly.
func TestMapCheckoutSession_JSONRoundTrip(t *testing.T) {
	cs := &CheckoutSession{
		ID:             "cs_test",
		URL:            "https://example.com",
		Status:         "open",
		PaymentStatus:  "unpaid",
		CustomerID:     "cus_test",
		SubscriptionID: "sub_test",
		AmountTotal:    1000,
		Currency:       "usd",
	}

	data, err := json.Marshal(cs)
	require.NoError(t, err)

	var decoded CheckoutSession
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cs.ID, decoded.ID)
	assert.Equal(t, cs.URL, decoded.URL)
	assert.Equal(t, cs.AmountTotal, decoded.AmountTotal)
}
