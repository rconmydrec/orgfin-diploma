package payment

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/config"
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/repositories"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock: PaymentProvider (for service tests)
// ---------------------------------------------------------------------------

type svcMockProvider struct {
	CreateCustomerFunc             func(email, name string, metadata map[string]string) (string, error)
	GetCustomerFunc                func(customerID string) (*Customer, error)
	UpdateCustomerBalanceFunc      func(customerID string, balanceCents int64) error
	CreateCheckoutSessionFunc      func(params *CheckoutParams) (*CheckoutSession, error)
	GetCheckoutSessionFunc         func(sessionID string) (*CheckoutSession, error)
	CreatePortalSessionFunc        func(customerID, returnURL string) (string, error)
	GetSubscriptionFunc            func(subscriptionID string) (*ProviderSubscription, error)
	CancelSubscriptionFunc         func(subscriptionID string, cancelAtPeriodEnd bool) error
	ReenableSubscriptionFunc       func(subscriptionID string) error
	UpdateSubscriptionFunc         func(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error)
	CreateSubscriptionScheduleFunc func(subscriptionID string, phases []SchedulePhase) (string, error)
	ModifySubscriptionScheduleFunc func(scheduleID string, phases []SchedulePhase) error
	CancelSubscriptionScheduleFunc func(scheduleID string) error
	VerifyWebhookSignatureFunc     func(payload []byte, signature string) ([]byte, error)
}

func (m *svcMockProvider) CreateCustomer(email, name string, metadata map[string]string) (string, error) {
	if m.CreateCustomerFunc != nil {
		return m.CreateCustomerFunc(email, name, metadata)
	}
	return "cus_default", nil
}

func (m *svcMockProvider) GetCustomer(customerID string) (*Customer, error) {
	if m.GetCustomerFunc != nil {
		return m.GetCustomerFunc(customerID)
	}
	return &Customer{ID: customerID}, nil
}

func (m *svcMockProvider) UpdateCustomerBalance(customerID string, balanceCents int64) error {
	if m.UpdateCustomerBalanceFunc != nil {
		return m.UpdateCustomerBalanceFunc(customerID, balanceCents)
	}
	return nil
}

func (m *svcMockProvider) CreateCheckoutSession(params *CheckoutParams) (*CheckoutSession, error) {
	if m.CreateCheckoutSessionFunc != nil {
		return m.CreateCheckoutSessionFunc(params)
	}
	return &CheckoutSession{ID: "cs_default", URL: "https://checkout.stripe.com/default"}, nil
}

func (m *svcMockProvider) GetCheckoutSession(sessionID string) (*CheckoutSession, error) {
	if m.GetCheckoutSessionFunc != nil {
		return m.GetCheckoutSessionFunc(sessionID)
	}
	return &CheckoutSession{ID: sessionID, Status: "complete", PaymentStatus: "paid"}, nil
}

func (m *svcMockProvider) CreatePortalSession(customerID, returnURL string) (string, error) {
	if m.CreatePortalSessionFunc != nil {
		return m.CreatePortalSessionFunc(customerID, returnURL)
	}
	return "https://billing.stripe.com/default", nil
}

func (m *svcMockProvider) GetSubscription(subscriptionID string) (*ProviderSubscription, error) {
	if m.GetSubscriptionFunc != nil {
		return m.GetSubscriptionFunc(subscriptionID)
	}
	return &ProviderSubscription{ID: subscriptionID, Status: "active"}, nil
}

func (m *svcMockProvider) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error {
	if m.CancelSubscriptionFunc != nil {
		return m.CancelSubscriptionFunc(subscriptionID, cancelAtPeriodEnd)
	}
	return nil
}

func (m *svcMockProvider) ReenableSubscription(subscriptionID string) error {
	if m.ReenableSubscriptionFunc != nil {
		return m.ReenableSubscriptionFunc(subscriptionID)
	}
	return nil
}

func (m *svcMockProvider) UpdateSubscription(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error) {
	if m.UpdateSubscriptionFunc != nil {
		return m.UpdateSubscriptionFunc(subscriptionID, params)
	}
	return &ProviderSubscription{ID: subscriptionID, Status: "active"}, nil
}

func (m *svcMockProvider) CreateSubscriptionSchedule(subscriptionID string, phases []SchedulePhase) (string, error) {
	if m.CreateSubscriptionScheduleFunc != nil {
		return m.CreateSubscriptionScheduleFunc(subscriptionID, phases)
	}
	return "sub_sched_default", nil
}

func (m *svcMockProvider) ModifySubscriptionSchedule(scheduleID string, phases []SchedulePhase) error {
	if m.ModifySubscriptionScheduleFunc != nil {
		return m.ModifySubscriptionScheduleFunc(scheduleID, phases)
	}
	return nil
}

func (m *svcMockProvider) CancelSubscriptionSchedule(scheduleID string) error {
	if m.CancelSubscriptionScheduleFunc != nil {
		return m.CancelSubscriptionScheduleFunc(scheduleID)
	}
	return nil
}

func (m *svcMockProvider) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	if m.VerifyWebhookSignatureFunc != nil {
		return m.VerifyWebhookSignatureFunc(payload, signature)
	}
	return payload, nil
}

// ---------------------------------------------------------------------------
// Mock: SubscriptionRepository (for service tests)
// ---------------------------------------------------------------------------

type svcMockSubRepo struct {
	GetByIDFunc                             func(id int) (*models.Subscription, error)
	GetByUserIDFunc                         func(userID int) (*models.Subscription, error)
	CreateFunc                              func(sub *models.Subscription) error
	UpdateFunc                              func(sub *models.Subscription) error
	UpdateSubscriptionFullFunc              func(sub *models.Subscription) error
	GetExpiredSubscriptionsFunc             func() ([]*models.Subscription, error)
	GetPendingDowngradesFunc                func() ([]*models.Subscription, error)
	GetByPlanTypeFunc                       func(planType string) ([]*models.Subscription, error)
	GetProviderSubscriptionFunc             func(subscriptionID int) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByExternalIDFunc func(externalSubscriptionID string) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByScheduleIDFunc func(externalScheduleID string) (*models.PaymentProviderSubscription, error)
	CreateProviderSubscriptionFunc          func(pps *models.PaymentProviderSubscription) error
	UpdateProviderSubscriptionFunc          func(pps *models.PaymentProviderSubscription) error
}

func (m *svcMockSubRepo) GetByID(id int) (*models.Subscription, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockSubRepo) GetByUserID(userID int) (*models.Subscription, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockSubRepo) Create(sub *models.Subscription) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(sub)
	}
	return nil
}

func (m *svcMockSubRepo) Update(sub *models.Subscription) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(sub)
	}
	return nil
}

func (m *svcMockSubRepo) UpdateSubscriptionFull(sub *models.Subscription) error {
	if m.UpdateSubscriptionFullFunc != nil {
		return m.UpdateSubscriptionFullFunc(sub)
	}
	return nil
}

func (m *svcMockSubRepo) GetExpiredSubscriptions() ([]*models.Subscription, error) {
	if m.GetExpiredSubscriptionsFunc != nil {
		return m.GetExpiredSubscriptionsFunc()
	}
	return nil, nil
}

func (m *svcMockSubRepo) GetPendingDowngrades() ([]*models.Subscription, error) {
	if m.GetPendingDowngradesFunc != nil {
		return m.GetPendingDowngradesFunc()
	}
	return nil, nil
}

func (m *svcMockSubRepo) GetByPlanType(planType string) ([]*models.Subscription, error) {
	if m.GetByPlanTypeFunc != nil {
		return m.GetByPlanTypeFunc(planType)
	}
	return nil, nil
}

func (m *svcMockSubRepo) GetProviderSubscription(subscriptionID int) (*models.PaymentProviderSubscription, error) {
	if m.GetProviderSubscriptionFunc != nil {
		return m.GetProviderSubscriptionFunc(subscriptionID)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockSubRepo) GetProviderSubscriptionByExternalID(externalSubscriptionID string) (*models.PaymentProviderSubscription, error) {
	if m.GetProviderSubscriptionByExternalIDFunc != nil {
		return m.GetProviderSubscriptionByExternalIDFunc(externalSubscriptionID)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockSubRepo) GetProviderSubscriptionByScheduleID(externalScheduleID string) (*models.PaymentProviderSubscription, error) {
	if m.GetProviderSubscriptionByScheduleIDFunc != nil {
		return m.GetProviderSubscriptionByScheduleIDFunc(externalScheduleID)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockSubRepo) CreateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	if m.CreateProviderSubscriptionFunc != nil {
		return m.CreateProviderSubscriptionFunc(pps)
	}
	return nil
}

func (m *svcMockSubRepo) UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	if m.UpdateProviderSubscriptionFunc != nil {
		return m.UpdateProviderSubscriptionFunc(pps)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SubscriptionPlanRepository (for service tests)
// ---------------------------------------------------------------------------

type svcMockPlanRepo struct {
	GetActivePlansFunc func() ([]*models.SubscriptionPlan, error)
	GetByIDFunc        func(id int) (*models.SubscriptionPlan, error)
	GetFreePlanFunc    func() (*models.SubscriptionPlan, error)
}

func (m *svcMockPlanRepo) GetActivePlans() ([]*models.SubscriptionPlan, error) {
	if m.GetActivePlansFunc != nil {
		return m.GetActivePlansFunc()
	}
	return nil, nil
}

func (m *svcMockPlanRepo) GetByID(id int) (*models.SubscriptionPlan, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockPlanRepo) GetFreePlan() (*models.SubscriptionPlan, error) {
	if m.GetFreePlanFunc != nil {
		return m.GetFreePlanFunc()
	}
	return nil, sql.ErrNoRows
}

// ---------------------------------------------------------------------------
// Mock: PlanPriceRepository (for service tests)
// ---------------------------------------------------------------------------

type svcMockPlanPriceRepo struct {
	GetByPlanAndProviderFunc      func(planID int, providerType string) (*models.PlanPrice, error)
	GetByExternalPriceIDFunc      func(externalPriceID string) (*models.PlanPrice, error)
	GetActivePricesByProviderFunc func(providerType string) ([]*models.PlanPrice, error)
}

func (m *svcMockPlanPriceRepo) GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error) {
	if m.GetByPlanAndProviderFunc != nil {
		return m.GetByPlanAndProviderFunc(planID, providerType)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockPlanPriceRepo) GetByExternalPriceID(externalPriceID string) (*models.PlanPrice, error) {
	if m.GetByExternalPriceIDFunc != nil {
		return m.GetByExternalPriceIDFunc(externalPriceID)
	}
	return nil, sql.ErrNoRows
}

func (m *svcMockPlanPriceRepo) GetActivePricesByProvider(providerType string) ([]*models.PlanPrice, error) {
	if m.GetActivePricesByProviderFunc != nil {
		return m.GetActivePricesByProviderFunc(providerType)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func svcTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func svcTestConfig() *config.Config {
	return &config.Config{
		FrontendURL:      "https://app.example.com",
		StripeSuccessURL: "/payment/success",
		StripeCancelURL:  "/payment/cancel",
	}
}

// mockCreditCalculator implements the CreditCalculator interface for testing
// without importing the credit service package directly.
type mockCreditCalculator struct{}

func (m *mockCreditCalculator) CalculateRemainingCredit(currentPlanPrice decimal.Decimal, billingPeriodDays int, expiresAt time.Time) decimal.Decimal {
	if expiresAt.IsZero() || billingPeriodDays <= 0 || !expiresAt.After(time.Now()) {
		return decimal.Zero
	}
	remaining := expiresAt.Sub(time.Now())
	days := int(remaining.Hours()/24) + 1
	dailyRate := currentPlanPrice.Div(decimal.NewFromInt(int64(billingPeriodDays)))
	return dailyRate.Mul(decimal.NewFromInt(int64(days)))
}

func (m *mockCreditCalculator) FormatCreditForDisplay(credit decimal.Decimal, currencyCode string) string {
	return "$" + credit.StringFixed(2)
}

func newSvcTestService(
	provider PaymentProvider,
	subRepo repositories.SubscriptionRepository,
	planRepo repositories.SubscriptionPlanRepository,
	planPriceRepo repositories.PlanPriceRepository,
) *Service {
	return NewService(
		provider,
		subRepo,
		planRepo,
		planPriceRepo,
		&mockCreditCalculator{},
		svcTestConfig(),
		svcTestLogger(),
	)
}

func svcTimeInFuture(days int) *time.Time {
	t := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	return &t
}

// ---------------------------------------------------------------------------
// T13: GetOrCreateCustomer
// ---------------------------------------------------------------------------

func TestGetOrCreateCustomer_ReturnsExistingCustomerID(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				SubscriptionID:     subscriptionID,
				ExternalCustomerID: strPtr("cus_existing_123"),
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	customerID, err := svc.GetOrCreateCustomer(42, "user@example.com", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "cus_existing_123", customerID)
}

func TestGetOrCreateCustomer_CreatesNewAndUpdatesProviderSub(t *testing.T) {
	var updatedPPS *models.PaymentProviderSubscription

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				SubscriptionID:     subscriptionID,
				ExternalCustomerID: nil,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			updatedPPS = pps
			return nil
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_new_456", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	customerID, err := svc.GetOrCreateCustomer(42, "user@example.com", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "cus_new_456", customerID)
	require.NotNil(t, updatedPPS)
	assert.Equal(t, "cus_new_456", *updatedPPS.ExternalCustomerID)
}

func TestGetOrCreateCustomer_CreatesNewAndCreatesProviderSub(t *testing.T) {
	var createdPPS *models.PaymentProviderSubscription

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
		CreateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			createdPPS = pps
			return nil
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_brand_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	customerID, err := svc.GetOrCreateCustomer(42, "user@example.com", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "cus_brand_new", customerID)
	require.NotNil(t, createdPPS)
	assert.Equal(t, "cus_brand_new", *createdPPS.ExternalCustomerID)
	assert.Equal(t, 10, createdPPS.SubscriptionID)
	assert.Equal(t, ProviderTypeStripe, createdPPS.ProviderType)
}

func TestGetOrCreateCustomer_NoSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_no_sub", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	customerID, err := svc.GetOrCreateCustomer(99, "no-sub@example.com", "No Sub")
	require.NoError(t, err)
	assert.Equal(t, "cus_no_sub", customerID)
}

func TestGetOrCreateCustomer_GetByUserIDError(t *testing.T) {
	dbErr := errors.New("database connection lost")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetOrCreateCustomer(1, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

func TestGetOrCreateCustomer_CreateCustomerError(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	providerErr := errors.New("stripe is down")
	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "", providerErr
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetOrCreateCustomer(1, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, providerErr)
}

// ---------------------------------------------------------------------------
// T13: CreateCheckout
// ---------------------------------------------------------------------------

func TestCreateCheckout_SuccessNoCredit(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: strPtr("cus_existing"),
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium Monthly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{
				ID:              1,
				PlanID:          planID,
				ExternalPriceID: "price_monthly",
			}, nil
		},
	}

	var capturedParams *CheckoutParams
	provider := &svcMockProvider{
		CreateCheckoutSessionFunc: func(params *CheckoutParams) (*CheckoutSession, error) {
			capturedParams = params
			return &CheckoutSession{
				ID:  "cs_test_session",
				URL: "https://checkout.stripe.com/pay/cs_test_session",
			}, nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	result, err := svc.CreateCheckout(42, 2, "user@example.com", "Test User")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "cs_test_session", result.SessionID)
	assert.Equal(t, "https://checkout.stripe.com/pay/cs_test_session", result.CheckoutURL)
	assert.Equal(t, int64(0), result.CreditAppliedCents)
	assert.Empty(t, result.CreditAppliedFormatted)

	require.NotNil(t, capturedParams)
	assert.Equal(t, "cus_existing", capturedParams.CustomerID)
	assert.Equal(t, "price_monthly", capturedParams.PriceID)
	assert.Contains(t, capturedParams.SuccessURL, "/payment/success")
	assert.Contains(t, capturedParams.CancelURL, "/payment/cancel")
}

func TestCreateCheckout_SuccessWithProratedCredit(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: strPtr("cus_existing"),
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 2 {
				return &models.SubscriptionPlan{
					ID:           2,
					Name:         "Premium Yearly",
					PlanType:     "premium",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:           1,
				Name:         "Premium Monthly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					Code:         "monthly",
					Name:         "Monthly",
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{
				ID:              1,
				PlanID:          planID,
				ExternalPriceID: "price_yearly",
			}, nil
		},
	}

	var balanceApplied int64
	provider := &svcMockProvider{
		UpdateCustomerBalanceFunc: func(customerID string, balanceCents int64) error {
			balanceApplied = balanceCents
			return nil
		},
		CreateCheckoutSessionFunc: func(params *CheckoutParams) (*CheckoutSession, error) {
			return &CheckoutSession{
				ID:  "cs_credit",
				URL: "https://checkout.stripe.com/pay/cs_credit",
			}, nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	result, err := svc.CreateCheckout(42, 2, "user@example.com", "Test User")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "cs_credit", result.SessionID)
	assert.Greater(t, result.CreditAppliedCents, int64(0))
	assert.NotEmpty(t, result.CreditAppliedFormatted)
	assert.Greater(t, balanceApplied, int64(0))
}

func TestCreateCheckout_PlanNotFound(t *testing.T) {
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.CreateCheckout(1, 999, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPlanNotFound)
}

func TestCreateCheckout_PlanPriceNotFound(t *testing.T) {
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{ID: id, Name: "Plan", Price: decimal.NewFromInt(10)}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(1, 2, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPrice)
}

func TestCreateCheckout_CheckoutSessionCreationError(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: strPtr("cus_test"),
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{ID: id, Name: "Plan", Price: decimal.NewFromInt(10), CurrencyCode: "USD"}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_test"}, nil
		},
	}

	sessionErr := errors.New("stripe session creation failed")
	provider := &svcMockProvider{
		CreateCheckoutSessionFunc: func(params *CheckoutParams) (*CheckoutSession, error) {
			return nil, sessionErr
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(42, 2, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, sessionErr)
}

// ---------------------------------------------------------------------------
// T13: CreatePortal
// ---------------------------------------------------------------------------

func TestCreatePortal_SuccessDefaultReturnURL(t *testing.T) {
	customerID := "cus_portal"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: &customerID,
			}, nil
		},
	}

	var capturedReturnURL string
	provider := &svcMockProvider{
		CreatePortalSessionFunc: func(cid, returnURL string) (string, error) {
			capturedReturnURL = returnURL
			return "https://billing.stripe.com/portal", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	result, err := svc.CreatePortal(42, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://billing.stripe.com/portal", result.PortalURL)
	assert.Equal(t, "https://app.example.com", capturedReturnURL)
}

func TestCreatePortal_SuccessCustomReturnURL(t *testing.T) {
	customerID := "cus_portal"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: &customerID,
			}, nil
		},
	}

	var capturedReturnURL string
	provider := &svcMockProvider{
		CreatePortalSessionFunc: func(cid, returnURL string) (string, error) {
			capturedReturnURL = returnURL
			return "https://billing.stripe.com/portal", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	// URL must start with FrontendURL to be accepted (prevents open redirect).
	customURL := "https://app.example.com/settings"
	result, err := svc.CreatePortal(42, &customURL)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://app.example.com/settings", capturedReturnURL)
}

func TestCreatePortal_RejectsExternalReturnURL(t *testing.T) {
	customerID := "cus_portal"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: &customerID,
			}, nil
		},
	}

	var capturedReturnURL string
	provider := &svcMockProvider{
		CreatePortalSessionFunc: func(cid, returnURL string) (string, error) {
			capturedReturnURL = returnURL
			return "https://billing.stripe.com/portal", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	// External URL should be rejected; default FrontendURL used instead.
	externalURL := "https://evil.example.com/phishing"
	result, err := svc.CreatePortal(42, &externalURL)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://app.example.com", capturedReturnURL)
}

func TestCreatePortal_NoSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(1, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeCustomer)
}

func TestCreatePortal_NoProviderSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(1, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeCustomer)
}

func TestCreatePortal_NoCustomerID(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: nil,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(1, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeCustomer)
}

// ---------------------------------------------------------------------------
// T14: GetUpgradePrice
// ---------------------------------------------------------------------------

func TestGetUpgradePrice_NoCreditNoSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium Yearly",
				PlanType:     "premium",
				Price:        decimal.NewFromFloat(99.99),
				CurrencyCode: "USD",
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(1, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, 5, info.PlanID)
	assert.Equal(t, "Premium Yearly", info.PlanName)
	assert.Equal(t, "premium", info.PlanType)
	assert.Equal(t, int64(0), info.CreditCents)
	assert.Equal(t, info.OriginalPriceCents, info.FinalPriceCents)
	assert.Equal(t, "USD", info.CurrencyCode)
}

func TestGetUpgradePrice_WithProratedCredit(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 5 {
				return &models.SubscriptionPlan{
					ID:           5,
					Name:         "Premium Yearly",
					PlanType:     "premium",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:           1,
				Name:         "Premium Monthly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, 5, info.PlanID)
	assert.Greater(t, info.CreditCents, int64(0))
	assert.Less(t, info.FinalPriceCents, info.OriginalPriceCents)
	assert.NotEmpty(t, info.CreditFormatted)
}

func TestGetUpgradePrice_PlanNotFound(t *testing.T) {
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.GetUpgradePrice(1, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPlanNotFound)
}

// ---------------------------------------------------------------------------
// T14: ChangePlan
// ---------------------------------------------------------------------------

func TestChangePlan_Upgrade(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			assert.Equal(t, 5, sub.PlanID)
			return nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 5 {
				return &models.SubscriptionPlan{
					ID:       5,
					Name:     "Premium Yearly",
					PlanType: "premium",
					Price:    decimal.NewFromInt(100),
					BillingPeriod: &models.BillingPeriod{
						ID:           2,
						DurationDays: 365,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromInt(10),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	var capturedUpdateParams *UpdateSubscriptionParams
	provider := &svcMockProvider{
		UpdateSubscriptionFunc: func(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error) {
			capturedUpdateParams = params
			return &ProviderSubscription{ID: subscriptionID, Status: "active"}, nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	result, err := svc.ChangePlan(42, 5)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.True(t, result.IsUpgrade)
	assert.Equal(t, "Premium Yearly", result.NewPlanName)
	assert.Equal(t, "Plan upgraded successfully", result.Message)
	assert.Nil(t, result.ScheduleID)

	require.NotNil(t, capturedUpdateParams)
	assert.Equal(t, "price_yearly", capturedUpdateParams.PriceID)
	assert.Equal(t, "create_prorations", capturedUpdateParams.ProrationBehavior)
}

func TestChangePlan_Downgrade(t *testing.T) {
	extSubID := "sub_stripe_123"
	periodEnd := time.Now().Add(20 * 24 * time.Hour)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			assert.NotNil(t, pps.ExternalScheduleID)
			return nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			assert.NotNil(t, sub.PendingPlanID)
			assert.Equal(t, 1, *sub.PendingPlanID)
			return nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Premium Monthly",
					PlanType: "premium",
					Price:    decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:       2,
				Name:     "Premium Yearly",
				PlanType: "premium",
				Price:    decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			if planID == 1 {
				return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
			}
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return &ProviderSubscription{
				ID:               subscriptionID,
				Status:           "active",
				CurrentPeriodEnd: periodEnd,
			}, nil
		},
		CreateSubscriptionScheduleFunc: func(subscriptionID string, phases []SchedulePhase) (string, error) {
			require.Len(t, phases, 2)
			assert.Equal(t, "price_yearly", phases[0].PriceID)
			assert.Equal(t, "price_monthly", phases[1].PriceID)
			return "sub_sched_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	result, err := svc.ChangePlan(42, 1)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.False(t, result.IsUpgrade)
	assert.Equal(t, "Premium Monthly", result.NewPlanName)
	assert.Equal(t, "Plan downgrade scheduled", result.Message)
	require.NotNil(t, result.ScheduleID)
	assert.Equal(t, "sub_sched_new", *result.ScheduleID)
}

func TestChangePlan_NoSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(1, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get subscription")
}

func TestChangePlan_NoStripeSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(1, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeSubscription)
}

func TestChangePlan_NoExternalSubscriptionID(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: nil,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(1, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeSubscription)
}

func TestChangePlan_TargetPlanNotFound(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(1, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPlanNotFound)
}

// ---------------------------------------------------------------------------
// T14: CancelScheduledChange
// ---------------------------------------------------------------------------

func TestCancelScheduledChange_Success(t *testing.T) {
	scheduleID := "sub_sched_123"
	pendingPlanID := 1

	var updatedSub *models.Subscription
	var updatedPPS *models.PaymentProviderSubscription

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:            10,
				UserID:        userID,
				PlanID:        2,
				PendingPlanID: &pendingPlanID,
			}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: &scheduleID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			updatedPPS = pps
			return nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			updatedSub = sub
			return nil
		},
	}

	var cancelledScheduleID string
	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			cancelledScheduleID = sid
			return nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.NoError(t, err)

	assert.Equal(t, "sub_sched_123", cancelledScheduleID)
	require.NotNil(t, updatedPPS)
	assert.Nil(t, updatedPPS.ExternalScheduleID)
	require.NotNil(t, updatedSub)
	assert.Nil(t, updatedSub.PendingPlanID)
	assert.Nil(t, updatedSub.PendingDowngradeAccountIDs)
	assert.Nil(t, updatedSub.PendingDowngradeBudgetID)
}

func TestCancelScheduledChange_ReenablesSubscription(t *testing.T) {
	scheduleID := "sub_sched_123"
	extSubID := "sub_ext_123"
	canceledAt := time.Now()

	var reenabled bool

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:         10,
				UserID:     userID,
				PlanID:     2,
				CanceledAt: &canceledAt,
				AutoRenew:  false,
			}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalScheduleID:     &scheduleID,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			assert.Nil(t, sub.CanceledAt)
			assert.True(t, sub.AutoRenew)
			return nil
		},
	}

	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			return nil
		},
		ReenableSubscriptionFunc: func(subscriptionID string) error {
			reenabled = true
			assert.Equal(t, "sub_ext_123", subscriptionID)
			return nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.NoError(t, err)
	assert.True(t, reenabled)
}

func TestCancelScheduledChange_NoSchedule(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: nil,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrScheduleNotFound)
}

func TestCancelScheduledChange_NoProviderSubscription(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrScheduleNotFound)
}

// ---------------------------------------------------------------------------
// T14: GetPaymentStatus
// ---------------------------------------------------------------------------

func TestGetPaymentStatus_FreeUser(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	status, err := svc.GetPaymentStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.CurrentPlanType)
	assert.False(t, status.HasStripeCustomer)
	assert.False(t, status.StripeSubscriptionActive)
	assert.Nil(t, status.StripeSubscriptionID)
	assert.False(t, status.LastPaymentFailed)
}

func TestGetPaymentStatus_SubscriptionNoProviderSub(t *testing.T) {
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2, PlanType: "premium"}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	status, err := svc.GetPaymentStatus(42)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "premium", status.CurrentPlanType)
	assert.False(t, status.HasStripeCustomer)
	assert.False(t, status.StripeSubscriptionActive)
}

func TestGetPaymentStatus_FullProviderSub(t *testing.T) {
	customerID := "cus_123"
	subID := "sub_456"

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2, PlanType: "premium"}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalCustomerID:     &customerID,
				ExternalSubscriptionID: &subID,
				LastPaymentFailed:      true,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	status, err := svc.GetPaymentStatus(42)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "premium", status.CurrentPlanType)
	assert.True(t, status.HasStripeCustomer)
	assert.True(t, status.StripeSubscriptionActive)
	require.NotNil(t, status.StripeSubscriptionID)
	assert.Equal(t, "sub_456", *status.StripeSubscriptionID)
	assert.True(t, status.LastPaymentFailed)
}

// ---------------------------------------------------------------------------
// T14: GetSessionStatus
// ---------------------------------------------------------------------------

func TestGetSessionStatus_Success(t *testing.T) {
	provider := &svcMockProvider{
		GetCheckoutSessionFunc: func(sessionID string) (*CheckoutSession, error) {
			return &CheckoutSession{
				ID:             sessionID,
				Status:         "complete",
				PaymentStatus:  "paid",
				CustomerID:     "cus_abc",
				SubscriptionID: "sub_def",
				AmountTotal:    2000,
				Currency:       "usd",
				Metadata:       map[string]string{"user_id": "42"},
			}, nil
		},
	}

	svc := newSvcTestService(provider, &svcMockSubRepo{}, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	status, err := svc.GetSessionStatus(42, "cs_test_id")
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "cs_test_id", status.SessionID)
	assert.Equal(t, "complete", status.Status)
	assert.Equal(t, "paid", status.PaymentStatus)
	assert.Equal(t, "cus_abc", status.CustomerID)
	assert.Equal(t, "sub_def", status.SubscriptionID)
	assert.Equal(t, int64(2000), status.AmountTotal)
	assert.Equal(t, "usd", status.Currency)
}

func TestGetSessionStatus_ProviderError(t *testing.T) {
	providerErr := errors.New("session not found")
	provider := &svcMockProvider{
		GetCheckoutSessionFunc: func(sessionID string) (*CheckoutSession, error) {
			return nil, providerErr
		},
	}

	svc := newSvcTestService(provider, &svcMockSubRepo{}, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetSessionStatus(42, "cs_invalid")
	require.Error(t, err)
	assert.ErrorIs(t, err, providerErr)
}

func TestGetSessionStatus_OwnershipMismatch(t *testing.T) {
	provider := &svcMockProvider{
		GetCheckoutSessionFunc: func(sessionID string) (*CheckoutSession, error) {
			return &CheckoutSession{
				ID:       sessionID,
				Status:   "complete",
				Metadata: map[string]string{"user_id": "99"},
			}, nil
		},
	}

	svc := newSvcTestService(provider, &svcMockSubRepo{}, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetSessionStatus(42, "cs_other_user")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestGetSessionStatus_NoMetadata(t *testing.T) {
	provider := &svcMockProvider{
		GetCheckoutSessionFunc: func(sessionID string) (*CheckoutSession, error) {
			return &CheckoutSession{
				ID:       sessionID,
				Status:   "complete",
				Metadata: nil,
			}, nil
		},
	}

	svc := newSvcTestService(provider, &svcMockSubRepo{}, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetSessionStatus(42, "cs_no_meta")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// ---------------------------------------------------------------------------
// CreateCheckout — additional error paths
// ---------------------------------------------------------------------------

func TestCreateCheckout_GetOrCreateCustomerError(t *testing.T) {
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_test"}, nil
		},
	}

	// GetOrCreateCustomer calls GetByUserID; return a non-ErrNoRows error.
	dbErr := errors.New("customer db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(1, 2, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateCheckout_UpdateCustomerBalanceError(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: strPtr("cus_existing"),
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 2 {
				return &models.SubscriptionPlan{
					ID:           2,
					Name:         "Premium Yearly",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:           1,
				Name:         "Premium Monthly",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	balanceErr := errors.New("stripe balance update failed")
	provider := &svcMockProvider{
		UpdateCustomerBalanceFunc: func(customerID string, balanceCents int64) error {
			return balanceErr
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(42, 2, "user@example.com", "Test User")
	require.Error(t, err)
	assert.ErrorIs(t, err, balanceErr)
	assert.Contains(t, err.Error(), "apply credit")
}

func TestCreateCheckout_PlanGetByIDDBError(t *testing.T) {
	dbErr := errors.New("plan db error")
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.CreateCheckout(1, 999, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateCheckout_PlanPriceDBError(t *testing.T) {
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{ID: id, Name: "Plan", Price: decimal.NewFromInt(10)}, nil
		},
	}

	dbErr := errors.New("plan price db error")
	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(1, 2, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

// ---------------------------------------------------------------------------
// handleUpgrade — error paths
// ---------------------------------------------------------------------------

func TestChangePlan_UpgradeStripeUpdateError(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 5 {
				return &models.SubscriptionPlan{
					ID:       5,
					Name:     "Premium Yearly",
					PlanType: "premium",
					Price:    decimal.NewFromInt(100),
					BillingPeriod: &models.BillingPeriod{
						ID:           2,
						DurationDays: 365,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromInt(10),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	stripeErr := errors.New("stripe update subscription failed")
	provider := &svcMockProvider{
		UpdateSubscriptionFunc: func(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error) {
			return nil, stripeErr
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeErr)
	assert.Contains(t, err.Error(), "update stripe subscription")
}

func TestChangePlan_UpgradeUpdateSubscriptionFullDBError(t *testing.T) {
	extSubID := "sub_stripe_123"
	dbErr := errors.New("db update full error")

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			return dbErr
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 5 {
				return &models.SubscriptionPlan{
					ID:       5,
					Name:     "Premium Yearly",
					PlanType: "premium",
					Price:    decimal.NewFromInt(100),
					BillingPeriod: &models.BillingPeriod{
						ID:           2,
						DurationDays: 365,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromInt(10),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	provider := &svcMockProvider{
		UpdateSubscriptionFunc: func(subscriptionID string, params *UpdateSubscriptionParams) (*ProviderSubscription, error) {
			return &ProviderSubscription{ID: subscriptionID, Status: "active"}, nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update subscription")
}

// ---------------------------------------------------------------------------
// handleDowngrade — error paths
// ---------------------------------------------------------------------------

func TestChangePlan_DowngradeGetSubscriptionError(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:    1,
					Name:  "Premium Monthly",
					Price: decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:    2,
				Name:  "Premium Yearly",
				Price: decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
		},
	}

	stripeErr := errors.New("stripe get subscription failed")
	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return nil, stripeErr
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeErr)
	assert.Contains(t, err.Error(), "get stripe subscription")
}

func TestChangePlan_DowngradeGetCurrentPlanPriceError(t *testing.T) {
	extSubID := "sub_stripe_123"
	periodEnd := time.Now().Add(20 * 24 * time.Hour)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:    1,
					Name:  "Premium Monthly",
					Price: decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:    2,
				Name:  "Premium Yearly",
				Price: decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	dbErr := errors.New("plan price db error")
	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			// Target plan price (planID=1) succeeds; current plan price (planID=2) fails.
			if planID == 1 {
				return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
			}
			return nil, dbErr
		},
	}

	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return &ProviderSubscription{
				ID:               subscriptionID,
				Status:           "active",
				CurrentPeriodEnd: periodEnd,
			}, nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get current plan price")
}

func TestChangePlan_DowngradeCreateSubscriptionScheduleError(t *testing.T) {
	extSubID := "sub_stripe_123"
	periodEnd := time.Now().Add(20 * 24 * time.Hour)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:    1,
					Name:  "Premium Monthly",
					Price: decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:    2,
				Name:  "Premium Yearly",
				Price: decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			if planID == 1 {
				return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
			}
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	scheduleErr := errors.New("stripe schedule creation failed")
	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return &ProviderSubscription{
				ID:               subscriptionID,
				Status:           "active",
				CurrentPeriodEnd: periodEnd,
			}, nil
		},
		CreateSubscriptionScheduleFunc: func(subscriptionID string, phases []SchedulePhase) (string, error) {
			return "", scheduleErr
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, scheduleErr)
	assert.Contains(t, err.Error(), "create subscription schedule")
}

func TestChangePlan_DowngradeUpdateProviderSubscriptionError(t *testing.T) {
	extSubID := "sub_stripe_123"
	periodEnd := time.Now().Add(20 * 24 * time.Hour)

	dbErr := errors.New("update provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return dbErr
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:    1,
					Name:  "Premium Monthly",
					Price: decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:    2,
				Name:  "Premium Yearly",
				Price: decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			if planID == 1 {
				return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
			}
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return &ProviderSubscription{
				ID:               subscriptionID,
				Status:           "active",
				CurrentPeriodEnd: periodEnd,
			}, nil
		},
		CreateSubscriptionScheduleFunc: func(subscriptionID string, phases []SchedulePhase) (string, error) {
			return "sub_sched_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update provider subscription")
}

func TestChangePlan_DowngradeUpdateSubscriptionFullError(t *testing.T) {
	extSubID := "sub_stripe_123"
	periodEnd := time.Now().Add(20 * 24 * time.Hour)

	dbErr := errors.New("update sub full db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 2}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			return dbErr
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:    1,
					Name:  "Premium Monthly",
					Price: decimal.NewFromInt(10),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						DurationDays: 30,
					},
				}, nil
			}
			return &models.SubscriptionPlan{
				ID:    2,
				Name:  "Premium Yearly",
				Price: decimal.NewFromInt(100),
				BillingPeriod: &models.BillingPeriod{
					ID:           2,
					DurationDays: 365,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			if planID == 1 {
				return &models.PlanPrice{ExternalPriceID: "price_monthly"}, nil
			}
			return &models.PlanPrice{ExternalPriceID: "price_yearly"}, nil
		},
	}

	provider := &svcMockProvider{
		GetSubscriptionFunc: func(subscriptionID string) (*ProviderSubscription, error) {
			return &ProviderSubscription{
				ID:               subscriptionID,
				Status:           "active",
				CurrentPeriodEnd: periodEnd,
			}, nil
		},
		CreateSubscriptionScheduleFunc: func(subscriptionID string, phases []SchedulePhase) (string, error) {
			return "sub_sched_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update subscription")
}

// ---------------------------------------------------------------------------
// CancelScheduledChange — additional error paths
// ---------------------------------------------------------------------------

func TestCancelScheduledChange_GetByUserIDError(t *testing.T) {
	dbErr := errors.New("db connection lost")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get subscription")
}

func TestCancelScheduledChange_GetProviderSubscriptionDBError(t *testing.T) {
	dbErr := errors.New("provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get provider subscription")
}

func TestCancelScheduledChange_CancelSubscriptionScheduleStripeError(t *testing.T) {
	scheduleID := "sub_sched_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: &scheduleID,
			}, nil
		},
	}

	stripeErr := errors.New("stripe cancel schedule failed")
	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			return stripeErr
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeErr)
	assert.Contains(t, err.Error(), "cancel subscription schedule")
}

func TestCancelScheduledChange_ReenableSubscriptionError(t *testing.T) {
	scheduleID := "sub_sched_123"
	extSubID := "sub_ext_123"
	canceledAt := time.Now()

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:         10,
				UserID:     userID,
				CanceledAt: &canceledAt,
				AutoRenew:  false,
			}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalScheduleID:     &scheduleID,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	reenableErr := errors.New("stripe reenable failed")
	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			return nil
		},
		ReenableSubscriptionFunc: func(subscriptionID string) error {
			return reenableErr
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, reenableErr)
	assert.Contains(t, err.Error(), "reenable subscription")
}

func TestCancelScheduledChange_UpdateProviderSubscriptionError(t *testing.T) {
	scheduleID := "sub_sched_123"

	dbErr := errors.New("update provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: &scheduleID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return dbErr
		},
	}

	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			return nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update provider subscription")
}

func TestCancelScheduledChange_UpdateSubscriptionFullError(t *testing.T) {
	scheduleID := "sub_sched_123"

	dbErr := errors.New("update sub full db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: &scheduleID,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return nil
		},
		UpdateSubscriptionFullFunc: func(sub *models.Subscription) error {
			return dbErr
		},
	}

	provider := &svcMockProvider{
		CancelSubscriptionScheduleFunc: func(sid string) error {
			return nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update subscription")
}

// ---------------------------------------------------------------------------
// calculateCurrentCredit — error paths
// ---------------------------------------------------------------------------

func TestCalculateCurrentCredit_GetByUserIDDBError(t *testing.T) {
	dbErr := errors.New("subscription db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Target",
				Price:        decimal.NewFromInt(100),
				CurrencyCode: "USD",
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	// GetUpgradePrice calls calculateCurrentCredit internally.
	_, err := svc.GetUpgradePrice(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get subscription for credit")
}

func TestCalculateCurrentCredit_GetByIDForCurrentPlanDBError(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	dbErr := errors.New("plan db error")
	callCount := 0

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			callCount++
			// First call is for target plan (from GetUpgradePrice); succeeds.
			// Second call is for current plan (from calculateCurrentCredit); fails.
			if callCount == 1 {
				return &models.SubscriptionPlan{
					ID:           id,
					Name:         "Target",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.GetUpgradePrice(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get current plan")
}

func TestCalculateCurrentCredit_SubscriptionExpiredInPast(t *testing.T) {
	billingPeriodID := 1
	pastTime := time.Now().Add(-24 * time.Hour) // expired yesterday

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              &pastTime,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium Yearly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(100),
				CurrencyCode: "USD",
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No credit because subscription has expired.
	assert.Equal(t, int64(0), info.CreditCents)
	assert.Equal(t, info.OriginalPriceCents, info.FinalPriceCents)
}

// ---------------------------------------------------------------------------
// ChangePlan — additional error paths
// ---------------------------------------------------------------------------

func TestChangePlan_GetByPlanAndProviderDBErrorForTargetPrice(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:    id,
				Name:  "Plan",
				Price: decimal.NewFromInt(10),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	dbErr := errors.New("plan price db error")
	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get target plan price")
}

func TestChangePlan_GetByIDErrorForCurrentPlan(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	dbErr := errors.New("current plan db error")
	callCount := 0
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			callCount++
			// First call is for target plan (id=5); succeeds.
			// Second call is for current plan (id=1); fails.
			if callCount == 1 {
				return &models.SubscriptionPlan{
					ID:    id,
					Name:  "Target Plan",
					Price: decimal.NewFromInt(100),
					BillingPeriod: &models.BillingPeriod{
						ID:           2,
						DurationDays: 365,
					},
				}, nil
			}
			return nil, dbErr
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_target"}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get current plan")
}

func TestChangePlan_GetProviderSubscriptionDBError(t *testing.T) {
	dbErr := errors.New("provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get provider subscription")
}

func TestChangePlan_GetByUserIDDBError(t *testing.T) {
	dbErr := errors.New("subscription db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get subscription")
}

func TestChangePlan_TargetPlanDBError(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	dbErr := errors.New("target plan db error")
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.ChangePlan(42, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get target plan")
}

func TestChangePlan_TargetPlanPriceNotFound(t *testing.T) {
	extSubID := "sub_stripe_123"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                     1,
				ExternalSubscriptionID: &extSubID,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:    id,
				Name:  "Plan",
				Price: decimal.NewFromInt(10),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					DurationDays: 30,
				},
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, planPriceRepo)

	_, err := svc.ChangePlan(42, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPrice)
}

// ---------------------------------------------------------------------------
// GetPaymentStatus — additional error paths
// ---------------------------------------------------------------------------

func TestGetPaymentStatus_GetByUserIDDBError(t *testing.T) {
	dbErr := errors.New("subscription db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetPaymentStatus(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

func TestGetPaymentStatus_GetProviderSubscriptionDBError(t *testing.T) {
	dbErr := errors.New("provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID, PlanType: "premium"}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetPaymentStatus(42)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
}

// ---------------------------------------------------------------------------
// CreatePortal — additional error paths
// ---------------------------------------------------------------------------

func TestCreatePortal_GetByUserIDDBError(t *testing.T) {
	dbErr := errors.New("subscription db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(42, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get subscription")
}

func TestCreatePortal_GetProviderSubscriptionDBError(t *testing.T) {
	dbErr := errors.New("provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(42, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get provider subscription")
}

func TestCreatePortal_CreatePortalSessionError(t *testing.T) {
	customerID := "cus_portal"
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: &customerID,
			}, nil
		},
	}

	portalErr := errors.New("stripe portal creation failed")
	provider := &svcMockProvider{
		CreatePortalSessionFunc: func(cid, returnURL string) (string, error) {
			return "", portalErr
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(42, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, portalErr)
	assert.Contains(t, err.Error(), "create portal session")
}

// ---------------------------------------------------------------------------
// GetUpgradePrice — additional error paths
// ---------------------------------------------------------------------------

func TestGetUpgradePrice_PlanDBError(t *testing.T) {
	dbErr := errors.New("plan db error")
	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, &svcMockSubRepo{}, planRepo, &svcMockPlanPriceRepo{})

	_, err := svc.GetUpgradePrice(1, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get target plan")
}

// ---------------------------------------------------------------------------
// GetOrCreateCustomer — additional error paths
// ---------------------------------------------------------------------------

func TestGetOrCreateCustomer_GetProviderSubscriptionDBError(t *testing.T) {
	dbErr := errors.New("provider sub db error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, dbErr
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetOrCreateCustomer(42, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get provider subscription")
}

func TestGetOrCreateCustomer_UpdateProviderSubscriptionError(t *testing.T) {
	dbErr := errors.New("update provider sub error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				SubscriptionID:     subscriptionID,
				ExternalCustomerID: nil,
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return dbErr
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetOrCreateCustomer(42, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "update provider subscription")
}

func TestGetOrCreateCustomer_CreateProviderSubscriptionError(t *testing.T) {
	dbErr := errors.New("create provider sub error")
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return nil, sql.ErrNoRows
		},
		CreateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return dbErr
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_new", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.GetOrCreateCustomer(42, "a@b.com", "A")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "create provider subscription")
}

// ---------------------------------------------------------------------------
// CancelScheduledChange — empty schedule ID string
// ---------------------------------------------------------------------------

func TestCancelScheduledChange_EmptyScheduleIDString(t *testing.T) {
	emptyStr := ""
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalScheduleID: &emptyStr,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	err := svc.CancelScheduledChange(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrScheduleNotFound)
}

// ---------------------------------------------------------------------------
// CreatePortal — empty customer ID string
// ---------------------------------------------------------------------------

func TestCreatePortal_EmptyCustomerIDString(t *testing.T) {
	emptyStr := ""
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: &emptyStr,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	_, err := svc.CreatePortal(1, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoStripeCustomer)
}

// ---------------------------------------------------------------------------
// calculateCurrentCredit — subscription with no billing period on plan
// ---------------------------------------------------------------------------

func TestCalculateCurrentCredit_NoBillingPeriodOnPlan(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			if id == 5 {
				return &models.SubscriptionPlan{
					ID:           5,
					Name:         "Target",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			// Current plan has no billing period.
			return &models.SubscriptionPlan{
				ID:            1,
				Name:          "Free",
				Price:         decimal.NewFromInt(0),
				CurrencyCode:  "USD",
				BillingPeriod: nil,
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No credit because current plan has no billing period.
	assert.Equal(t, int64(0), info.CreditCents)
}

// ---------------------------------------------------------------------------
// calculateCurrentCredit — current plan not found (ErrNoRows)
// ---------------------------------------------------------------------------

func TestCalculateCurrentCredit_CurrentPlanNotFoundReturnsZero(t *testing.T) {
	billingPeriodID := 1
	expiresAt := svcTimeInFuture(15)

	callCount := 0
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              expiresAt,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			callCount++
			// First call is for target plan; succeeds.
			// Second call is for current plan; returns ErrNoRows.
			if callCount == 1 {
				return &models.SubscriptionPlan{
					ID:           id,
					Name:         "Target",
					Price:        decimal.NewFromInt(100),
					CurrencyCode: "USD",
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No credit because current plan was not found.
	assert.Equal(t, int64(0), info.CreditCents)
}

// ---------------------------------------------------------------------------
// calculateCurrentCredit — subscription with no ExpiresAt
// ---------------------------------------------------------------------------

func TestCalculateCurrentCredit_NoExpiresAt(t *testing.T) {
	billingPeriodID := 1

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: &billingPeriodID,
				ExpiresAt:              nil, // no expiry date
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium Yearly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(100),
				CurrencyCode: "USD",
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No credit because ExpiresAt is nil.
	assert.Equal(t, int64(0), info.CreditCents)
}

// ---------------------------------------------------------------------------
// calculateCurrentCredit — subscription with no CurrentBillingPeriodID
// ---------------------------------------------------------------------------

func TestCalculateCurrentCredit_NoBillingPeriodID(t *testing.T) {
	expiresAt := svcTimeInFuture(15)

	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                     10,
				UserID:                 userID,
				PlanID:                 1,
				CurrentBillingPeriodID: nil, // no billing period
				ExpiresAt:              expiresAt,
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium Yearly",
				PlanType:     "premium",
				Price:        decimal.NewFromInt(100),
				CurrencyCode: "USD",
			}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, &svcMockPlanPriceRepo{})

	info, err := svc.GetUpgradePrice(42, 5)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No credit because CurrentBillingPeriodID is nil.
	assert.Equal(t, int64(0), info.CreditCents)
}

// ---------------------------------------------------------------------------
// GetOrCreateCustomer — empty customer ID in existing provider sub
// ---------------------------------------------------------------------------

func TestGetOrCreateCustomer_EmptyCustomerIDInProviderSub(t *testing.T) {
	emptyStr := ""
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{ID: 10, UserID: userID}, nil
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				SubscriptionID:     subscriptionID,
				ExternalCustomerID: &emptyStr, // empty string, not nil
			}, nil
		},
		UpdateProviderSubscriptionFunc: func(pps *models.PaymentProviderSubscription) error {
			return nil
		},
	}

	provider := &svcMockProvider{
		CreateCustomerFunc: func(email, name string, metadata map[string]string) (string, error) {
			return "cus_new_from_empty", nil
		},
	}

	svc := newSvcTestService(provider, subRepo, &svcMockPlanRepo{}, &svcMockPlanPriceRepo{})

	customerID, err := svc.GetOrCreateCustomer(42, "a@b.com", "A")
	require.NoError(t, err)
	assert.Equal(t, "cus_new_from_empty", customerID)
}

// ---------------------------------------------------------------------------
// CreateCheckout — calculateCurrentCredit error propagation
// ---------------------------------------------------------------------------

func TestCreateCheckout_CalculateCurrentCreditError(t *testing.T) {
	// GetOrCreateCustomer must succeed, but calculateCurrentCredit must fail.
	// Both call GetByUserID. Use a counter to succeed on the first call (from
	// GetOrCreateCustomer) and fail on the second call (from calculateCurrentCredit).
	dbErr := errors.New("subscription db intermittent error")
	callCount := 0
	subRepo := &svcMockSubRepo{
		GetByUserIDFunc: func(userID int) (*models.Subscription, error) {
			callCount++
			if callCount == 1 {
				// First call: GetOrCreateCustomer - return a subscription with customer.
				return &models.Subscription{ID: 10, UserID: userID, PlanID: 1}, nil
			}
			// Second call: calculateCurrentCredit - return a DB error.
			return nil, dbErr
		},
		GetProviderSubscriptionFunc: func(subscriptionID int) (*models.PaymentProviderSubscription, error) {
			return &models.PaymentProviderSubscription{
				ID:                 1,
				ExternalCustomerID: strPtr("cus_existing"),
			}, nil
		},
	}

	planRepo := &svcMockPlanRepo{
		GetByIDFunc: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:           id,
				Name:         "Premium",
				Price:        decimal.NewFromInt(10),
				CurrencyCode: "USD",
			}, nil
		},
	}

	planPriceRepo := &svcMockPlanPriceRepo{
		GetByPlanAndProviderFunc: func(planID int, providerType string) (*models.PlanPrice, error) {
			return &models.PlanPrice{ExternalPriceID: "price_test"}, nil
		},
	}

	svc := newSvcTestService(&svcMockProvider{}, subRepo, planRepo, planPriceRepo)

	_, err := svc.CreateCheckout(42, 2, "user@example.com", "Test User")
	require.Error(t, err)
	assert.ErrorIs(t, err, dbErr)
	assert.Contains(t, err.Error(), "get subscription for credit")
}
