package payment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Mock: PaymentProvider
// ---------------------------------------------------------------------------

type mockPaymentProvider struct {
	// VerifyWebhookSignature
	verifyResult []byte
	verifyErr    error

	// GetSubscription
	getSubResult *ProviderSubscription
	getSubErr    error

	// Track calls
	getSubCalledWith string
}

func (m *mockPaymentProvider) CreateCustomer(string, string, map[string]string) (string, error) {
	return "", nil
}
func (m *mockPaymentProvider) GetCustomer(string) (*Customer, error) { return nil, nil }
func (m *mockPaymentProvider) UpdateCustomerBalance(string, int64) error {
	return nil
}
func (m *mockPaymentProvider) CreateCheckoutSession(*CheckoutParams) (*CheckoutSession, error) {
	return nil, nil
}
func (m *mockPaymentProvider) GetCheckoutSession(string) (*CheckoutSession, error) {
	return nil, nil
}
func (m *mockPaymentProvider) CreatePortalSession(string, string) (string, error) {
	return "", nil
}
func (m *mockPaymentProvider) GetSubscription(id string) (*ProviderSubscription, error) {
	m.getSubCalledWith = id
	return m.getSubResult, m.getSubErr
}
func (m *mockPaymentProvider) CancelSubscription(string, bool) error { return nil }
func (m *mockPaymentProvider) ReenableSubscription(string) error     { return nil }
func (m *mockPaymentProvider) UpdateSubscription(string, *UpdateSubscriptionParams) (*ProviderSubscription, error) {
	return nil, nil
}
func (m *mockPaymentProvider) CreateSubscriptionSchedule(string, []SchedulePhase) (string, error) {
	return "", nil
}
func (m *mockPaymentProvider) ModifySubscriptionSchedule(string, []SchedulePhase) error {
	return nil
}
func (m *mockPaymentProvider) CancelSubscriptionSchedule(string) error { return nil }
func (m *mockPaymentProvider) VerifyWebhookSignature(payload []byte, sig string) ([]byte, error) {
	return m.verifyResult, m.verifyErr
}

// ---------------------------------------------------------------------------
// Mock: SubscriptionRepository
// ---------------------------------------------------------------------------

type mockSubRepo struct {
	// GetByID
	getByIDResult map[int]*models.Subscription
	getByIDErr    error

	// GetByUserID
	getByUserIDResult map[int]*models.Subscription
	getByUserIDErr    error

	// GetProviderSubscription (by internal subscription ID)
	getProvSubResult map[int]*models.PaymentProviderSubscription
	getProvSubErr    error

	// GetProviderSubscriptionByExternalID (by Stripe subscription ID)
	getProvSubByExtResult map[string]*models.PaymentProviderSubscription
	getProvSubByExtErr    map[string]error

	// GetProviderSubscriptionByScheduleID
	getProvSubBySchedResult map[string]*models.PaymentProviderSubscription
	getProvSubBySchedErr    map[string]error

	// CreateProviderSubscription
	createProvSubErr    error
	createdProvSub      *models.PaymentProviderSubscription
	createProvSubCalled bool

	// UpdateProviderSubscription
	updateProvSubErr    error
	updatedProvSub      *models.PaymentProviderSubscription
	updateProvSubCalled bool

	// UpdateSubscriptionFull
	updateSubFullErr    error
	updatedSub          *models.Subscription
	updateSubFullCalled bool

	// Unused stubs
	createErr error
	updateErr error
}

func (m *mockSubRepo) GetByID(id int) (*models.Subscription, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if sub, ok := m.getByIDResult[id]; ok {
		return sub, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubRepo) GetByUserID(userID int) (*models.Subscription, error) {
	if m.getByUserIDErr != nil {
		return nil, m.getByUserIDErr
	}
	if sub, ok := m.getByUserIDResult[userID]; ok {
		return sub, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubRepo) Create(*models.Subscription) error { return m.createErr }
func (m *mockSubRepo) Update(*models.Subscription) error { return m.updateErr }

func (m *mockSubRepo) UpdateSubscriptionFull(sub *models.Subscription) error {
	m.updateSubFullCalled = true
	m.updatedSub = sub
	return m.updateSubFullErr
}

func (m *mockSubRepo) GetExpiredSubscriptions() ([]*models.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepo) GetPendingDowngrades() ([]*models.Subscription, error) {
	return nil, nil
}
func (m *mockSubRepo) GetByPlanType(string) ([]*models.Subscription, error) {
	return nil, nil
}

func (m *mockSubRepo) GetProviderSubscription(subID int) (*models.PaymentProviderSubscription, error) {
	if m.getProvSubErr != nil {
		return nil, m.getProvSubErr
	}
	if pps, ok := m.getProvSubResult[subID]; ok {
		return pps, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubRepo) GetProviderSubscriptionByExternalID(extID string) (*models.PaymentProviderSubscription, error) {
	if errMap := m.getProvSubByExtErr; errMap != nil {
		if e, ok := errMap[extID]; ok {
			return nil, e
		}
	}
	if pps, ok := m.getProvSubByExtResult[extID]; ok {
		return pps, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubRepo) GetProviderSubscriptionByScheduleID(schedID string) (*models.PaymentProviderSubscription, error) {
	if errMap := m.getProvSubBySchedErr; errMap != nil {
		if e, ok := errMap[schedID]; ok {
			return nil, e
		}
	}
	if pps, ok := m.getProvSubBySchedResult[schedID]; ok {
		return pps, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubRepo) CreateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	m.createProvSubCalled = true
	m.createdProvSub = pps
	return m.createProvSubErr
}

func (m *mockSubRepo) UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	m.updateProvSubCalled = true
	m.updatedProvSub = pps
	return m.updateProvSubErr
}

// ---------------------------------------------------------------------------
// Mock: SubscriptionPlanRepository
// ---------------------------------------------------------------------------

type mockPlanRepo struct {
	getByIDResult map[int]*models.SubscriptionPlan
	getByIDErr    error

	freePlan    *models.SubscriptionPlan
	freePlanErr error
}

func (m *mockPlanRepo) GetActivePlans() ([]*models.SubscriptionPlan, error) {
	return nil, nil
}

func (m *mockPlanRepo) GetByID(id int) (*models.SubscriptionPlan, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if plan, ok := m.getByIDResult[id]; ok {
		return plan, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockPlanRepo) GetFreePlan() (*models.SubscriptionPlan, error) {
	if m.freePlanErr != nil {
		return nil, m.freePlanErr
	}
	return m.freePlan, nil
}

// ---------------------------------------------------------------------------
// Mock: PlanPriceRepository
// ---------------------------------------------------------------------------

type mockPlanPriceRepo struct {
	getByExtResult map[string]*models.PlanPrice
	getByExtErr    map[string]error
}

func (m *mockPlanPriceRepo) GetByPlanAndProvider(int, string) (*models.PlanPrice, error) {
	return nil, nil
}

func (m *mockPlanPriceRepo) GetByExternalPriceID(extPriceID string) (*models.PlanPrice, error) {
	if errMap := m.getByExtErr; errMap != nil {
		if e, ok := errMap[extPriceID]; ok {
			return nil, e
		}
	}
	if pp, ok := m.getByExtResult[extPriceID]; ok {
		return pp, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockPlanPriceRepo) GetActivePricesByProvider(string) ([]*models.PlanPrice, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// buildWebhookPayload constructs a Stripe-style webhook event JSON payload.
func buildWebhookPayload(t *testing.T, eventType string, object any) []byte {
	t.Helper()
	objBytes, err := json.Marshal(object)
	require.NoError(t, err)

	event := map[string]any{
		"id":   "evt_test_123",
		"type": eventType,
		"data": map[string]any{
			"object": json.RawMessage(objBytes),
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return payload
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// newTestHandler creates a WebhookHandler with the given mock dependencies.
func newTestHandler(
	provider *mockPaymentProvider,
	subRepo *mockSubRepo,
	planRepo *mockPlanRepo,
	planPriceRepo *mockPlanPriceRepo,
) *WebhookHandler {
	return NewWebhookHandler(provider, subRepo, planRepo, planPriceRepo, testLogger())
}

// ===========================================================================
// T15-test: Part 1 tests
// ===========================================================================

// ---------------------------------------------------------------------------
// 1. HandleWebhook top-level tests
// ---------------------------------------------------------------------------

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	provider := &mockPaymentProvider{
		verifyErr: ErrWebhookSignatureInvalid,
	}
	h := newTestHandler(provider, &mockSubRepo{}, &mockPlanRepo{}, &mockPlanPriceRepo{})

	result := h.HandleWebhook([]byte(`{"id":"evt_1"}`), "bad_sig")

	assert.False(t, result.Success)
	assert.Equal(t, "signature verification failed", result.Message)
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
	}
	h := newTestHandler(provider, &mockSubRepo{}, &mockPlanRepo{}, &mockPlanPriceRepo{})

	result := h.HandleWebhook([]byte(`not json at all{{{`), "valid_sig")

	assert.False(t, result.Success)
	assert.Equal(t, "failed to parse event", result.Message)
}

func TestHandleWebhook_UnknownEventType(t *testing.T) {
	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
	}
	h := newTestHandler(provider, &mockSubRepo{}, &mockPlanRepo{}, &mockPlanPriceRepo{})

	payload := buildWebhookPayload(t, "unknown.event.type", map[string]string{"foo": "bar"})
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "event processed", result.Message)
	assert.Equal(t, "unknown.event.type", result.EventType)
}

// ---------------------------------------------------------------------------
// 2. handleCheckoutSessionCompleted tests
// ---------------------------------------------------------------------------

func TestCheckoutSessionCompleted_FullSuccess_CreatesProviderSub(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	bpID := 10

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_123",
			Status:           "active",
			CurrentPeriodEnd: periodEnd,
		},
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {
				ID:       100,
				UserID:   42,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			},
		},
		// No existing provider subscription -- sql.ErrNoRows
		getProvSubResult: map[int]*models.PaymentProviderSubscription{},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {
				ID:       5,
				PlanType: "premium",
				BillingPeriod: &models.BillingPeriod{
					ID:   bpID,
					Code: "monthly",
				},
			},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "checkout.session.completed", result.EventType)

	// Provider subscription was created (not updated).
	assert.True(t, subRepo.createProvSubCalled)
	assert.False(t, subRepo.updateProvSubCalled)
	require.NotNil(t, subRepo.createdProvSub)
	assert.Equal(t, 100, subRepo.createdProvSub.SubscriptionID)
	assert.Equal(t, ProviderTypeStripe, subRepo.createdProvSub.ProviderType)
	assert.Equal(t, "cus_abc", *subRepo.createdProvSub.ExternalCustomerID)
	assert.Equal(t, "sub_stripe_123", *subRepo.createdProvSub.ExternalSubscriptionID)

	// Subscription was updated to the premium plan.
	assert.True(t, subRepo.updateSubFullCalled)
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, 5, subRepo.updatedSub.PlanID)
	assert.Equal(t, "premium", subRepo.updatedSub.PlanType)
	assert.True(t, subRepo.updatedSub.IsActive)
	assert.True(t, subRepo.updatedSub.AutoRenew)
	assert.True(t, subRepo.updatedSub.HasStripeSubscription)
	assert.NotNil(t, subRepo.updatedSub.SubscribedAt)
	assert.NotNil(t, subRepo.updatedSub.ExpiresAt)
	assert.Equal(t, &bpID, subRepo.updatedSub.CurrentBillingPeriodID)

	// Trial and pending fields are cleared.
	assert.Nil(t, subRepo.updatedSub.TrialStartedAt)
	assert.Nil(t, subRepo.updatedSub.TrialEndsAt)
	assert.Nil(t, subRepo.updatedSub.PendingPlanID)
	assert.Nil(t, subRepo.updatedSub.PendingDowngradeAccountIDs)
	assert.Nil(t, subRepo.updatedSub.PendingDowngradeBudgetID)
}

func TestCheckoutSessionCompleted_UpdatesExistingProviderSub(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_456",
			Status:           "active",
			CurrentPeriodEnd: periodEnd,
		},
	}

	existingProvSub := &models.PaymentProviderSubscription{
		ID:             99,
		SubscriptionID: 100,
		ProviderType:   ProviderTypeStripe,
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {
				ID:       100,
				UserID:   42,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			},
		},
		getProvSubResult: map[int]*models.PaymentProviderSubscription{
			100: existingProvSub,
		},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {
				ID:       5,
				PlanType: "premium",
			},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_456",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)

	// Provider subscription was updated (not created).
	assert.True(t, subRepo.updateProvSubCalled)
	assert.False(t, subRepo.createProvSubCalled)
	assert.Equal(t, "cus_abc", *subRepo.updatedProvSub.ExternalCustomerID)
	assert.Equal(t, "sub_stripe_456", *subRepo.updatedProvSub.ExternalSubscriptionID)
}

func TestCheckoutSessionCompleted_InvalidUserID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
	subRepo := &mockSubRepo{}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "not_a_number",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Logged error, but still returns success.
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_InvalidPlanID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
	subRepo := &mockSubRepo{}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "abc",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_SubscriptionNotFound(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
	// GetByUserID will return sql.ErrNoRows for user 42 since no entry.
	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

// ---------------------------------------------------------------------------
// 3. handleSubscriptionUpdated tests
// ---------------------------------------------------------------------------

func TestSubscriptionUpdated_UpdatesPlanWhenPriceChanges(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_stripe_789"
	bpID := 20

	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:       100,
				PlanID:   5,
				PlanType: "premium",
				IsActive: true,
			},
		},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			10: {
				ID:       10,
				PlanType: "premium_annual",
				BillingPeriod: &models.BillingPeriod{
					ID:   bpID,
					Code: "annual",
				},
			},
		},
	}

	planPriceRepo := &mockPlanPriceRepo{
		getByExtResult: map[string]*models.PlanPrice{
			"price_new_abc": {
				ID:              1,
				PlanID:          10,
				ExternalPriceID: "price_new_abc",
			},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, planPriceRepo)

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(365 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items": map[string]any{
			"data": []map[string]any{
				{"price": map[string]string{"id": "price_new_abc"}},
			},
		},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.True(t, subRepo.updateSubFullCalled)
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, 10, subRepo.updatedSub.PlanID)
	assert.Equal(t, "premium_annual", subRepo.updatedSub.PlanType)
	assert.Equal(t, &bpID, subRepo.updatedSub.CurrentBillingPeriodID)
}

func TestSubscriptionUpdated_CancelAtPeriodEndTrue(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_stripe_cancel"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:       100,
				PlanID:   5,
				PlanType: "premium",
				IsActive: true,
			},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": true,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.NotNil(t, subRepo.updatedSub.CanceledAt, "CanceledAt should be set when cancel_at_period_end is true")
	assert.False(t, subRepo.updatedSub.AutoRenew, "AutoRenew should be false when cancel_at_period_end is true")
}

func TestSubscriptionUpdated_CancelAtPeriodEndFalse(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_stripe_reactivate"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	now := time.Now()
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:         100,
				PlanID:     5,
				PlanType:   "premium",
				IsActive:   true,
				CanceledAt: &now,
				AutoRenew:  false,
			},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.Nil(t, subRepo.updatedSub.CanceledAt, "CanceledAt should be cleared when cancel_at_period_end is false")
	assert.True(t, subRepo.updatedSub.AutoRenew, "AutoRenew should be true when cancel_at_period_end is false")
}

func TestSubscriptionUpdated_UnknownStripeSubscriptionID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   "sub_unknown",
		"status":               "active",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled, "should not update subscription for unknown Stripe ID")
}

func TestSubscriptionUpdated_ActiveStatus(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_active"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:       100,
				IsActive: false,
			},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.True(t, subRepo.updatedSub.IsActive, "active status should map to IsActive=true")
}

func TestSubscriptionUpdated_CanceledStatus(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_canceled"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:       100,
				IsActive: true,
			},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "canceled",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.False(t, subRepo.updatedSub.IsActive, "canceled status should map to IsActive=false")
}

// ===========================================================================
// T16-test: Part 2 tests
// ===========================================================================

// ---------------------------------------------------------------------------
// 4. handleSubscriptionDeleted tests
// ---------------------------------------------------------------------------

func TestSubscriptionDeleted_DowngradesToFreePlan(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_to_delete"
	scheduleID := "sched_old"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		ExternalScheduleID:     &scheduleID,
	}

	pendingPlanID := 99
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:                         100,
				PlanID:                     5,
				PlanType:                   "premium",
				IsActive:                   true,
				AutoRenew:                  true,
				PendingPlanID:              &pendingPlanID,
				PendingDowngradeAccountIDs: []int{1, 2, 3},
				PendingDowngradeBudgetID:   intPtr(10),
			},
		},
	}

	freePlan := &models.SubscriptionPlan{
		ID:       1,
		PlanType: "free",
	}
	planRepo := &mockPlanRepo{freePlan: freePlan}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "customer.subscription.deleted", result.EventType)

	// Subscription should be downgraded to free plan.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, 1, subRepo.updatedSub.PlanID)
	assert.True(t, subRepo.updatedSub.IsActive)
	assert.False(t, subRepo.updatedSub.AutoRenew)
	assert.NotNil(t, subRepo.updatedSub.CanceledAt)
	assert.Nil(t, subRepo.updatedSub.ExpiresAt)
	assert.Nil(t, subRepo.updatedSub.CurrentBillingPeriodID)
	assert.Nil(t, subRepo.updatedSub.PendingPlanID)
	assert.Nil(t, subRepo.updatedSub.PendingDowngradeAccountIDs)
	assert.Nil(t, subRepo.updatedSub.PendingDowngradeBudgetID)

	// Provider subscription should have Stripe data cleared.
	require.NotNil(t, subRepo.updatedProvSub)
	assert.Nil(t, subRepo.updatedProvSub.ExternalSubscriptionID)
	assert.Nil(t, subRepo.updatedProvSub.ExternalScheduleID)
}

func TestSubscriptionDeleted_UnknownSubscriptionID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": "sub_nonexistent"}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

// ---------------------------------------------------------------------------
// 5. handleInvoicePaid tests
// ---------------------------------------------------------------------------

func TestInvoicePaid_ClearsLastPaymentFailed(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_paid"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		LastPaymentFailed:      true,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.paid", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "invoice.paid", result.EventType)
	assert.True(t, subRepo.updateProvSubCalled)
	require.NotNil(t, subRepo.updatedProvSub)
	assert.False(t, subRepo.updatedProvSub.LastPaymentFailed)
}

func TestInvoicePaid_NoSubscription_Skipped(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
	subRepo := &mockSubRepo{}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "",
	}
	payload := buildWebhookPayload(t, "invoice.paid", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

func TestInvoicePaid_UnknownSubscription(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_unknown",
	}
	payload := buildWebhookPayload(t, "invoice.paid", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

// ---------------------------------------------------------------------------
// 6. handleInvoicePaymentFailed tests
// ---------------------------------------------------------------------------

func TestInvoicePaymentFailed_SetsLastPaymentFailed(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_failed"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		LastPaymentFailed:      false,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.payment_failed", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "invoice.payment_failed", result.EventType)
	assert.True(t, subRepo.updateProvSubCalled)
	require.NotNil(t, subRepo.updatedProvSub)
	assert.True(t, subRepo.updatedProvSub.LastPaymentFailed)
}

func TestInvoicePaymentFailed_NoSubscription_Skipped(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
	subRepo := &mockSubRepo{}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "",
	}
	payload := buildWebhookPayload(t, "invoice.payment_failed", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

func TestInvoicePaymentFailed_UnknownSubscription(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_unknown",
	}
	payload := buildWebhookPayload(t, "invoice.payment_failed", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

// ===========================================================================
// T17-test: Part 3 tests
// ===========================================================================

// ---------------------------------------------------------------------------
// 7. handleScheduleCompleted tests
// ---------------------------------------------------------------------------

func TestScheduleCompleted_AppliesPendingPlan(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_complete"
	extSubID := "sub_sched"
	pendingPlanID := 10
	bpID := 30

	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		ExternalScheduleID:     &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:            100,
				PlanID:        5,
				PlanType:      "premium",
				PendingPlanID: &pendingPlanID,
			},
		},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			10: {
				ID:       10,
				PlanType: "premium_annual",
				BillingPeriod: &models.BillingPeriod{
					ID:   bpID,
					Code: "annual",
				},
			},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "subscription_schedule.completed", result.EventType)

	// Subscription should be updated to the pending plan.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, 10, subRepo.updatedSub.PlanID)
	assert.Equal(t, &bpID, subRepo.updatedSub.CurrentBillingPeriodID)
	assert.Nil(t, subRepo.updatedSub.PendingPlanID, "PendingPlanID should be cleared")

	// Schedule ID should be cleared from provider subscription.
	require.NotNil(t, subRepo.updatedProvSub)
	assert.Nil(t, subRepo.updatedProvSub.ExternalScheduleID)
}

func TestScheduleCompleted_NoPendingPlan_ClearsScheduleOnly(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_no_pending"
	extSubID := "sub_sched2"

	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		ExternalScheduleID:     &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:            100,
				PlanID:        5,
				PlanType:      "premium",
				PendingPlanID: nil, // No pending plan.
			},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)

	// Subscription should remain on the same plan.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, 5, subRepo.updatedSub.PlanID)

	// Schedule ID should still be cleared.
	require.NotNil(t, subRepo.updatedProvSub)
	assert.Nil(t, subRepo.updatedProvSub.ExternalScheduleID)
}

func TestScheduleCompleted_UnknownScheduleID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": "sched_nonexistent"}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
	assert.False(t, subRepo.updateProvSubCalled)
}

// ---------------------------------------------------------------------------
// 8. handleScheduleReleased tests
// ---------------------------------------------------------------------------

func TestScheduleReleased_ClearsScheduleID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_released"
	extSubID := "sub_sched3"

	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		ExternalScheduleID:     &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.released", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.Equal(t, "subscription_schedule.released", result.EventType)

	assert.True(t, subRepo.updateProvSubCalled)
	require.NotNil(t, subRepo.updatedProvSub)
	assert.Nil(t, subRepo.updatedProvSub.ExternalScheduleID)
}

func TestScheduleReleased_UnknownScheduleID(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": "sched_unknown"}
	payload := buildWebhookPayload(t, "subscription_schedule.released", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

// ===========================================================================
// Additional edge case tests
// ===========================================================================

func TestCheckoutSessionCompleted_PlanNotFound(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42, PlanID: 1},
		},
	}

	// Plan 5 does not exist in the mock.
	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Still returns success but subscription is not updated.
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_GetStripeSubscriptionFails(t *testing.T) {
	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubErr:    errors.New("stripe API error"),
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42, PlanID: 1},
		},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {ID: 5, PlanType: "premium"},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_PlanWithNoBillingPeriod(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_123",
			Status:           "active",
			CurrentPeriodEnd: periodEnd,
		},
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42, PlanID: 1},
		},
		getProvSubResult: map[int]*models.PaymentProviderSubscription{},
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {
				ID:            5,
				PlanType:      "premium",
				BillingPeriod: nil, // No billing period.
			},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.Nil(t, subRepo.updatedSub.CurrentBillingPeriodID,
		"CurrentBillingPeriodID should be nil when target plan has no billing period")
}

func TestSubscriptionUpdated_PastDueStatus(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_past_due"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, IsActive: false},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "past_due",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.True(t, subRepo.updatedSub.IsActive, "past_due status should map to IsActive=true")
}

func TestSubscriptionUpdated_UnpaidStatus(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_unpaid"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, IsActive: true},
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "unpaid",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	require.NotNil(t, subRepo.updatedSub)
	assert.False(t, subRepo.updatedSub.IsActive, "unpaid status should map to IsActive=false")
}

func TestSubscriptionUpdated_NonStripeDBError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_dberr"
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubByExtErr: map[string]error{
			extSubID: fmt.Errorf("connection refused"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Still returns success (errors are logged, not propagated).
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestSubscriptionDeleted_FreePlanNotFound(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_delete_no_free"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PlanID: 5},
		},
	}

	planRepo := &mockPlanRepo{
		freePlanErr: errors.New("no free plan configured"),
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Error is logged but success is still returned (handler errors don't affect the result).
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestScheduleCompleted_GetPendingPlanFails(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_plan_err"
	pendingPlanID := 99

	provSub := &models.PaymentProviderSubscription{
		ID:                 1,
		SubscriptionID:     100,
		ExternalScheduleID: &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {
				ID:            100,
				PlanID:        5,
				PendingPlanID: &pendingPlanID,
			},
		},
	}

	// Plan 99 does not exist.
	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// The handler returns an error, which is logged. Result is still success.
	assert.True(t, result.Success)
}

func TestCheckoutSessionCompleted_CreateProviderSubError(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_123",
			CurrentPeriodEnd: periodEnd,
		},
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42},
		},
		getProvSubResult: map[int]*models.PaymentProviderSubscription{},
		createProvSubErr: errors.New("DB insert failed"),
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {ID: 5, PlanType: "premium"},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	// Subscription should NOT have been updated because provider sub creation failed.
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_UpdateProviderSubError(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_456",
			CurrentPeriodEnd: periodEnd,
		},
	}

	existingProvSub := &models.PaymentProviderSubscription{
		ID:             99,
		SubscriptionID: 100,
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42},
		},
		getProvSubResult: map[int]*models.PaymentProviderSubscription{
			100: existingProvSub,
		},
		updateProvSubErr: errors.New("DB update failed"),
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {ID: 5, PlanType: "premium"},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_456",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	// Subscription should NOT have been updated because provider sub update failed.
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_UpdateSubscriptionFullError(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_123",
			CurrentPeriodEnd: periodEnd,
		},
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42},
		},
		getProvSubResult: map[int]*models.PaymentProviderSubscription{},
		updateSubFullErr: errors.New("DB update sub failed"),
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {ID: 5, PlanType: "premium"},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Result still shows success even though internal update failed.
	assert.True(t, result.Success)
	assert.True(t, subRepo.updateSubFullCalled)
}

func TestCheckoutSessionCompleted_GetProviderSubDBError(t *testing.T) {
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	provider := &mockPaymentProvider{
		verifyResult: []byte(`ok`),
		getSubResult: &ProviderSubscription{
			ID:               "sub_stripe_123",
			CurrentPeriodEnd: periodEnd,
		},
	}

	subRepo := &mockSubRepo{
		getByUserIDResult: map[int]*models.Subscription{
			42: {ID: 100, UserID: 42},
		},
		getProvSubErr: errors.New("connection timeout"), // Non-ErrNoRows error.
	}

	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{
			5: {ID: 5, PlanType: "premium"},
		},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	sessionObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": "sub_stripe_123",
		"metadata": map[string]string{
			"user_id": "42",
			"plan_id": "5",
		},
	}
	payload := buildWebhookPayload(t, "checkout.session.completed", sessionObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestSubscriptionUpdated_UpdateSubFullError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_update_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100},
		},
		updateSubFullErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Result still success -- errors are logged internally.
	assert.True(t, result.Success)
}

func TestSubscriptionUpdated_GetSubByIDError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_getbyid_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Unix(),
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{}},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestSubscriptionUpdated_PriceNotFoundInSystem(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_unknown_price"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	originalPlanID := 5
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PlanID: originalPlanID, PlanType: "premium"},
		},
	}

	// Price not found in the plan price repo.
	planPriceRepo := &mockPlanPriceRepo{
		getByExtResult: map[string]*models.PlanPrice{},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, planPriceRepo)

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items": map[string]any{
			"data": []map[string]any{
				{"price": map[string]string{"id": "price_unknown"}},
			},
		},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	// Subscription was still updated (other fields like expiration), just plan not changed.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, originalPlanID, subRepo.updatedSub.PlanID, "plan should remain unchanged when price not found")
}

func TestInvoicePaid_UpdateProviderSubError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_inv_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
		LastPaymentFailed:      true,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		updateProvSubErr: errors.New("DB update error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.paid", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Error logged but still success.
	assert.True(t, result.Success)
}

func TestInvoicePaymentFailed_UpdateProviderSubError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_fail_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		updateProvSubErr: errors.New("DB update error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.payment_failed", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestInvoicePaid_LookupDBError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_lookup_err"
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubByExtErr: map[string]error{
			extSubID: fmt.Errorf("connection refused"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.paid", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Error is returned by handler, logged by HandleWebhook. Still success.
	assert.True(t, result.Success)
}

func TestInvoicePaymentFailed_LookupDBError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_fail_lookup_err"
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubByExtErr: map[string]error{
			extSubID: fmt.Errorf("connection refused"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	invObj := map[string]any{
		"customer":     "cus_abc",
		"subscription": extSubID,
	}
	payload := buildWebhookPayload(t, "invoice.payment_failed", invObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestSubscriptionDeleted_GetSubByIDError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_del_getbyid_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Handler returns error which is logged.
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestSubscriptionDeleted_UpdateSubFullError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_del_upd_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100},
		},
		updateSubFullErr: errors.New("DB error"),
	}

	planRepo := &mockPlanRepo{
		freePlan: &models.SubscriptionPlan{ID: 1, PlanType: "free"},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestSubscriptionDeleted_UpdateProvSubError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_del_prov_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100},
		},
		updateProvSubErr: errors.New("DB prov error"),
	}

	planRepo := &mockPlanRepo{
		freePlan: &models.SubscriptionPlan{ID: 1, PlanType: "free"},
	}

	h := newTestHandler(provider, subRepo, planRepo, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestScheduleCompleted_GetSubByIDError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_getbyid_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                 1,
		SubscriptionID:     100,
		ExternalScheduleID: &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestScheduleCompleted_UpdateSubFullError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_upd_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                 1,
		SubscriptionID:     100,
		ExternalScheduleID: &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PendingPlanID: nil},
		},
		updateSubFullErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestScheduleCompleted_UpdateProvSubError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_upd_prov_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                 1,
		SubscriptionID:     100,
		ExternalScheduleID: &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PendingPlanID: nil},
		},
		updateProvSubErr: errors.New("DB prov error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestScheduleReleased_UpdateProvSubError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_rel_upd_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                 1,
		SubscriptionID:     100,
		ExternalScheduleID: &scheduleID,
	}

	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{
			scheduleID: provSub,
		},
		updateProvSubErr: errors.New("DB error"),
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.released", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
}

func TestSubscriptionUpdated_PlanPriceLookupDBError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_price_dberr"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	originalPlanID := 5
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PlanID: originalPlanID, PlanType: "premium"},
		},
	}

	planPriceRepo := &mockPlanPriceRepo{
		getByExtErr: map[string]error{
			"price_db_err": fmt.Errorf("connection timeout"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, planPriceRepo)

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items": map[string]any{
			"data": []map[string]any{
				{"price": map[string]string{"id": "price_db_err"}},
			},
		},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	// Subscription was still updated (other fields), but plan not changed.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, originalPlanID, subRepo.updatedSub.PlanID)
}

func TestSubscriptionUpdated_PlanLookupError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_plan_err"
	provSub := &models.PaymentProviderSubscription{
		ID:                     1,
		SubscriptionID:         100,
		ExternalSubscriptionID: &extSubID,
	}

	originalPlanID := 5
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{
			extSubID: provSub,
		},
		getByIDResult: map[int]*models.Subscription{
			100: {ID: 100, PlanID: originalPlanID, PlanType: "premium"},
		},
	}

	planPriceRepo := &mockPlanPriceRepo{
		getByExtResult: map[string]*models.PlanPrice{
			"price_plan_err": {PlanID: 999},
		},
	}

	// Plan 999 does not exist.
	planRepo := &mockPlanRepo{
		getByIDResult: map[int]*models.SubscriptionPlan{},
	}

	h := newTestHandler(provider, subRepo, planRepo, planPriceRepo)

	subObj := map[string]any{
		"id":                   extSubID,
		"status":               "active",
		"current_period_end":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"cancel_at_period_end": false,
		"items": map[string]any{
			"data": []map[string]any{
				{"price": map[string]string{"id": "price_plan_err"}},
			},
		},
	}
	payload := buildWebhookPayload(t, "customer.subscription.updated", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	// Subscription was still updated (other fields), but plan not changed.
	require.NotNil(t, subRepo.updatedSub)
	assert.Equal(t, originalPlanID, subRepo.updatedSub.PlanID)
}

func TestSubscriptionDeleted_LookupDBError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	extSubID := "sub_del_lookup_err"
	subRepo := &mockSubRepo{
		getProvSubByExtResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubByExtErr: map[string]error{
			extSubID: fmt.Errorf("connection refused"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	subObj := map[string]any{"id": extSubID}
	payload := buildWebhookPayload(t, "customer.subscription.deleted", subObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// Handler returns an error, which is logged. Still success.
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestScheduleCompleted_LookupScheduleError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_lookup_err"
	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubBySchedErr: map[string]error{
			scheduleID: fmt.Errorf("connection timeout"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.completed", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	// The handler logs a warning and returns nil. Result is still success.
	assert.True(t, result.Success)
	assert.False(t, subRepo.updateSubFullCalled)
}

func TestScheduleReleased_LookupScheduleError(t *testing.T) {
	provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}

	scheduleID := "sched_rel_lookup_err"
	subRepo := &mockSubRepo{
		getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{},
		getProvSubBySchedErr: map[string]error{
			scheduleID: fmt.Errorf("connection timeout"),
		},
	}

	h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

	schedObj := map[string]any{"id": scheduleID}
	payload := buildWebhookPayload(t, "subscription_schedule.released", schedObj)
	result := h.HandleWebhook(payload, "valid_sig")

	assert.True(t, result.Success)
	assert.False(t, subRepo.updateProvSubCalled)
}

// Verify that all event routing works and each event type reaches its handler.
func TestHandleWebhook_AllEventTypesRoute(t *testing.T) {
	eventTypes := []string{
		"checkout.session.completed",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"invoice.paid",
		"invoice.payment_failed",
		"subscription_schedule.completed",
		"subscription_schedule.released",
	}

	for _, et := range eventTypes {
		t.Run(et, func(t *testing.T) {
			provider := &mockPaymentProvider{verifyResult: []byte(`ok`)}
			subRepo := &mockSubRepo{
				getProvSubByExtResult:   map[string]*models.PaymentProviderSubscription{},
				getProvSubBySchedResult: map[string]*models.PaymentProviderSubscription{},
				getByUserIDResult:       map[int]*models.Subscription{},
			}

			h := newTestHandler(provider, subRepo, &mockPlanRepo{}, &mockPlanPriceRepo{})

			// Build a minimal valid object for each event type.
			var obj any
			switch et {
			case "checkout.session.completed":
				obj = map[string]any{
					"customer":     "cus_1",
					"subscription": "sub_1",
					"metadata":     map[string]string{"user_id": "bad", "plan_id": "bad"},
				}
			case "customer.subscription.updated":
				obj = map[string]any{
					"id": "sub_unknown", "status": "active",
					"current_period_end": time.Now().Unix(),
					"items":              map[string]any{"data": []any{}},
				}
			case "customer.subscription.deleted":
				obj = map[string]any{"id": "sub_unknown"}
			case "invoice.paid", "invoice.payment_failed":
				obj = map[string]any{"customer": "cus_1", "subscription": ""}
			case "subscription_schedule.completed", "subscription_schedule.released":
				obj = map[string]any{"id": "sched_unknown"}
			}

			payload := buildWebhookPayload(t, et, obj)
			result := h.HandleWebhook(payload, "valid_sig")

			assert.True(t, result.Success, "event type %s should return success", et)
			assert.Equal(t, et, result.EventType)
		})
	}
}
