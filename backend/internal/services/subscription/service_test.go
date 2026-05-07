package subscription

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations ---

type mockSubscriptionRepo struct {
	getByUserIDFn            func(userID int) (*models.Subscription, error)
	getByIDFn                func(id int) (*models.Subscription, error)
	createFn                 func(sub *models.Subscription) error
	updateFn                 func(sub *models.Subscription) error
	updateSubscriptionFullFn func(sub *models.Subscription) error
	getExpiredFn             func() ([]*models.Subscription, error)
	getPendingDowngradesFn   func() ([]*models.Subscription, error)
	getByPlanTypeFn          func(planType string) ([]*models.Subscription, error)

	createCalls                 []*models.Subscription
	updateCalls                 []*models.Subscription
	updateSubscriptionFullCalls []*models.Subscription
}

func (m *mockSubscriptionRepo) GetByUserID(userID int) (*models.Subscription, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(userID)
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionRepo) GetByID(id int) (*models.Subscription, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionRepo) Create(sub *models.Subscription) error {
	m.createCalls = append(m.createCalls, sub)
	if m.createFn != nil {
		return m.createFn(sub)
	}
	return nil
}

func (m *mockSubscriptionRepo) Update(sub *models.Subscription) error {
	m.updateCalls = append(m.updateCalls, sub)
	if m.updateFn != nil {
		return m.updateFn(sub)
	}
	return nil
}

func (m *mockSubscriptionRepo) UpdateSubscriptionFull(sub *models.Subscription) error {
	m.updateSubscriptionFullCalls = append(m.updateSubscriptionFullCalls, sub)
	if m.updateSubscriptionFullFn != nil {
		return m.updateSubscriptionFullFn(sub)
	}
	return nil
}

func (m *mockSubscriptionRepo) GetExpiredSubscriptions() ([]*models.Subscription, error) {
	if m.getExpiredFn != nil {
		return m.getExpiredFn()
	}
	return nil, nil
}

func (m *mockSubscriptionRepo) GetPendingDowngrades() ([]*models.Subscription, error) {
	if m.getPendingDowngradesFn != nil {
		return m.getPendingDowngradesFn()
	}
	return nil, nil
}

func (m *mockSubscriptionRepo) GetByPlanType(planType string) ([]*models.Subscription, error) {
	if m.getByPlanTypeFn != nil {
		return m.getByPlanTypeFn(planType)
	}
	return nil, nil
}

func (m *mockSubscriptionRepo) GetProviderSubscription(subscriptionID int) (*models.PaymentProviderSubscription, error) {
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionRepo) GetProviderSubscriptionByExternalID(externalSubscriptionID string) (*models.PaymentProviderSubscription, error) {
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionRepo) GetProviderSubscriptionByScheduleID(externalScheduleID string) (*models.PaymentProviderSubscription, error) {
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionRepo) CreateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	return nil
}

func (m *mockSubscriptionRepo) UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	return nil
}

type mockSubscriptionPlanRepo struct {
	getActivePlansFn func() ([]*models.SubscriptionPlan, error)
	getByIDFn        func(id int) (*models.SubscriptionPlan, error)
	getFreePlanFn    func() (*models.SubscriptionPlan, error)
}

func (m *mockSubscriptionPlanRepo) GetActivePlans() ([]*models.SubscriptionPlan, error) {
	if m.getActivePlansFn != nil {
		return m.getActivePlansFn()
	}
	return nil, nil
}

func (m *mockSubscriptionPlanRepo) GetByID(id int) (*models.SubscriptionPlan, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockSubscriptionPlanRepo) GetFreePlan() (*models.SubscriptionPlan, error) {
	if m.getFreePlanFn != nil {
		return m.getFreePlanFn()
	}
	return &models.SubscriptionPlan{
		ID:       1,
		Name:     "Free",
		PlanType: "free",
		Price:    decimal.Zero,
	}, nil
}

type mockPlanPriceRepo struct {
	getByPlanAndProviderFn    func(planID int, providerType string) (*models.PlanPrice, error)
	getByExternalPriceIDFn    func(externalPriceID string) (*models.PlanPrice, error)
	getActivePricesByProvider func(providerType string) ([]*models.PlanPrice, error)
}

func (m *mockPlanPriceRepo) GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error) {
	if m.getByPlanAndProviderFn != nil {
		return m.getByPlanAndProviderFn(planID, providerType)
	}
	return nil, sql.ErrNoRows
}

func (m *mockPlanPriceRepo) GetByExternalPriceID(externalPriceID string) (*models.PlanPrice, error) {
	if m.getByExternalPriceIDFn != nil {
		return m.getByExternalPriceIDFn(externalPriceID)
	}
	return nil, sql.ErrNoRows
}

func (m *mockPlanPriceRepo) GetActivePricesByProvider(providerType string) ([]*models.PlanPrice, error) {
	if m.getActivePricesByProvider != nil {
		return m.getActivePricesByProvider(providerType)
	}
	return nil, nil
}

type mockAccountCounter struct {
	countFn func(userID int) (int, error)
}

func (m *mockAccountCounter) CountActiveByUserID(userID int) (int, error) {
	if m.countFn != nil {
		return m.countFn(userID)
	}
	return 0, nil
}

type mockBudgetCounter struct {
	countFn func(userID int) (int, error)
}

func (m *mockBudgetCounter) CountActiveByUserID(userID int) (int, error) {
	if m.countFn != nil {
		return m.countFn(userID)
	}
	return 0, nil
}

type mockAccountArchiver struct {
	archiveFn func(userID int, keepIDs []int) error
	called    bool
	userID    int
	keepIDs   []int
}

func (m *mockAccountArchiver) ArchiveByUserExcept(userID int, keepIDs []int) error {
	m.called = true
	m.userID = userID
	m.keepIDs = keepIDs
	if m.archiveFn != nil {
		return m.archiveFn(userID, keepIDs)
	}
	return nil
}

type mockBudgetArchiver struct {
	archiveFn func(userID int, keepID *int) error
	called    bool
	userID    int
	keepID    *int
}

func (m *mockBudgetArchiver) ArchiveByUserExcept(userID int, keepID *int) error {
	m.called = true
	m.userID = userID
	m.keepID = keepID
	if m.archiveFn != nil {
		return m.archiveFn(userID, keepID)
	}
	return nil
}

// --- Helpers ---

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestService(
	subRepo *mockSubscriptionRepo,
	planRepo *mockSubscriptionPlanRepo,
	priceRepo *mockPlanPriceRepo,
	acctCounter *mockAccountCounter,
	budgetCounter *mockBudgetCounter,
) *Service {
	if subRepo == nil {
		subRepo = &mockSubscriptionRepo{}
	}
	if planRepo == nil {
		planRepo = &mockSubscriptionPlanRepo{}
	}
	if priceRepo == nil {
		priceRepo = &mockPlanPriceRepo{}
	}
	if acctCounter == nil {
		acctCounter = &mockAccountCounter{}
	}
	if budgetCounter == nil {
		budgetCounter = &mockBudgetCounter{}
	}

	return New(subRepo, planRepo, priceRepo, acctCounter, budgetCounter, discardLogger(), 14)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(v int) *int {
	return &v
}

// --- Tests ---

func TestValidatePlanTransition(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	tests := []struct {
		name        string
		current     string
		target      string
		expectedErr error
	}{
		{
			name:        "same plan returns ErrSamePlan",
			current:     "free",
			target:      "free",
			expectedErr: ErrSamePlan,
		},
		{
			name:        "same plan premium returns ErrSamePlan",
			current:     "premium",
			target:      "premium",
			expectedErr: ErrSamePlan,
		},
		{
			name:        "any to trial returns ErrInvalidPlanTransition",
			current:     "free",
			target:      "trial",
			expectedErr: ErrInvalidPlanTransition,
		},
		{
			name:        "premium to trial returns ErrInvalidPlanTransition",
			current:     "premium",
			target:      "trial",
			expectedErr: ErrInvalidPlanTransition,
		},
		{
			name:        "trial to free returns ErrInvalidPlanTransition",
			current:     "trial",
			target:      "free",
			expectedErr: ErrInvalidPlanTransition,
		},
		{
			name:        "premium to free returns ErrDowngradeNotAllowed",
			current:     "premium",
			target:      "free",
			expectedErr: ErrDowngradeNotAllowed,
		},
		{
			name:        "free to premium is allowed",
			current:     "free",
			target:      "premium",
			expectedErr: nil,
		},
		{
			name:        "trial to premium is allowed",
			current:     "trial",
			target:      "premium",
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.validatePlanTransition(tc.current, tc.target)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectedErr),
					"expected %v, got %v", tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetStatus_FreeUserNoSubscription(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.IsActive)
	assert.Equal(t, PlanLimits["free"], status.Limits)
	assert.Nil(t, status.TrialDaysRemaining)
	assert.False(t, status.RequiresDowngrade)
	assert.Equal(t, 0, status.PlanID)
}

func TestGetStatus_TrialUserActive(t *testing.T) {
	trialEnd := time.Now().Add(7 * 24 * time.Hour) // 7 days from now
	subscribedAt := time.Now().Add(-7 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           10,
				UserID:       1,
				PlanID:       2,
				PlanType:     "trial",
				IsActive:     true,
				TrialEndsAt:  &trialEnd,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "trial", status.PlanType)
	assert.True(t, status.IsActive)
	assert.Equal(t, 2, status.PlanID)
	require.NotNil(t, status.TrialDaysRemaining)
	assert.True(t, *status.TrialDaysRemaining > 0, "trial days remaining should be positive")
	assert.True(t, *status.TrialDaysRemaining <= 7, "trial days remaining should be at most 7")
	assert.Nil(t, status.Limits) // trial has no limits (nil in PlanLimits)
}

func TestGetStatus_PremiumUser(t *testing.T) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	subscribedAt := time.Now().Add(-1 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    20,
				UserID:                1,
				PlanID:                3,
				PlanType:              "premium",
				IsActive:              true,
				SubscribedAt:          &subscribedAt,
				ExpiresAt:             &expiresAt,
				HasStripeSubscription: true,
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "premium", status.PlanType)
	assert.True(t, status.IsActive)
	assert.Equal(t, 3, status.PlanID)
	assert.Nil(t, status.TrialDaysRemaining)
	assert.Nil(t, status.Limits) // premium has no limits
	assert.True(t, status.HasStripeSubscription)
	assert.False(t, status.RequiresDowngrade)
}

func TestGetStatus_TrialExpiredAutoDowngrade(t *testing.T) {
	trialEnd := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
	subscribedAt := time.Now().Add(-15 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           10,
				UserID:       1,
				PlanID:       2,
				PlanType:     "trial",
				IsActive:     true,
				TrialEndsAt:  &trialEnd,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) { return 1, nil },
	}
	budgetCounter := &mockBudgetCounter{
		countFn: func(userID int) (int, error) { return 0, nil },
	}

	svc := newTestService(subRepo, planRepo, nil, acctCounter, budgetCounter)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.IsActive)
	assert.Equal(t, PlanLimits["free"], status.Limits)
	assert.False(t, status.RequiresDowngrade)

	// Verify downgrade was persisted
	require.Len(t, subRepo.updateCalls, 1)
	updated := subRepo.updateCalls[0]
	assert.Equal(t, 1, updated.PlanID)
	assert.Equal(t, "free", updated.PlanType)
	assert.True(t, updated.IsActive)
}

func TestGetStatus_FreeUserExceedsLimitsRequiresDowngrade(t *testing.T) {
	subscribedAt := time.Now().Add(-30 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           5,
				UserID:       1,
				PlanID:       1,
				PlanType:     "free",
				IsActive:     true,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	// User has 5 accounts (free limit is 2)
	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) { return 5, nil },
	}
	budgetCounter := &mockBudgetCounter{
		countFn: func(userID int) (int, error) { return 0, nil },
	}

	svc := newTestService(subRepo, nil, nil, acctCounter, budgetCounter)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.RequiresDowngrade)
}

func TestGetStatus_FreeUserExceedsBudgetLimitsRequiresDowngrade(t *testing.T) {
	subscribedAt := time.Now().Add(-30 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           5,
				UserID:       1,
				PlanID:       1,
				PlanType:     "free",
				IsActive:     true,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	// User is within account limit but exceeds budget limit (free limit is 1)
	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) { return 1, nil },
	}
	budgetCounter := &mockBudgetCounter{
		countFn: func(userID int) (int, error) { return 3, nil },
	}

	svc := newTestService(subRepo, nil, nil, acctCounter, budgetCounter)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.RequiresDowngrade)
}

func TestGetStatus_FreeUserWithinLimits(t *testing.T) {
	subscribedAt := time.Now().Add(-30 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           5,
				UserID:       1,
				PlanID:       1,
				PlanType:     "free",
				IsActive:     true,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	// User within limits
	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) { return 1, nil },
	}
	budgetCounter := &mockBudgetCounter{
		countFn: func(userID int) (int, error) { return 1, nil },
	}

	svc := newTestService(subRepo, nil, nil, acctCounter, budgetCounter)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.False(t, status.RequiresDowngrade)
}

func TestGetStatus_RepoError(t *testing.T) {
	dbErr := errors.New("database connection lost")
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.Error(t, err)
	assert.Nil(t, status)
	assert.True(t, errors.Is(err, dbErr))
}

func TestGetStatus_TrialExpiredDowngradeError(t *testing.T) {
	trialEnd := time.Now().Add(-1 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:          10,
				UserID:      1,
				PlanID:      2,
				PlanType:    "trial",
				IsActive:    true,
				TrialEndsAt: &trialEnd,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return nil, errors.New("plan lookup failed")
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.Error(t, err)
	assert.Nil(t, status)
}

func TestGetStatus_PendingPlanName(t *testing.T) {
	subscribedAt := time.Now().Add(-1 * 24 * time.Hour)
	pendingID := 99

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:            20,
				UserID:        1,
				PlanID:        3,
				PlanType:      "premium",
				IsActive:      true,
				SubscribedAt:  &subscribedAt,
				ExpiresAt:     timePtr(time.Now().Add(30 * 24 * time.Hour)),
				PendingPlanID: &pendingID,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			assert.Equal(t, 99, id)
			return &models.SubscriptionPlan{
				ID:       99,
				Name:     "Premium Yearly",
				PlanType: "premium",
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "premium", status.PlanType)
	require.NotNil(t, status.PendingPlanID)
	assert.Equal(t, 99, *status.PendingPlanID)
	require.NotNil(t, status.PendingPlanName)
	assert.Equal(t, "Premium Yearly", *status.PendingPlanName)
}

func TestGetStatus_PendingPlanLookupErrorNonFatal(t *testing.T) {
	subscribedAt := time.Now().Add(-1 * 24 * time.Hour)
	pendingID := 99

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:            20,
				UserID:        1,
				PlanID:        3,
				PlanType:      "premium",
				IsActive:      true,
				SubscribedAt:  &subscribedAt,
				ExpiresAt:     timePtr(time.Now().Add(30 * 24 * time.Hour)),
				PendingPlanID: &pendingID,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return nil, errors.New("plan lookup failed")
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err) // should be non-fatal
	require.NotNil(t, status)

	assert.Equal(t, "premium", status.PlanType)
	assert.Nil(t, status.PendingPlanName) // name is nil due to lookup failure
}

func TestGetStatus_AccountCounterErrorNonFatal(t *testing.T) {
	subscribedAt := time.Now().Add(-30 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           5,
				UserID:       1,
				PlanID:       1,
				PlanType:     "free",
				IsActive:     true,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) {
			return 0, errors.New("account count failed")
		},
	}

	svc := newTestService(subRepo, nil, nil, acctCounter, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err) // non-fatal
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.False(t, status.RequiresDowngrade) // defaults to false on error
}

func TestGetPlans(t *testing.T) {
	expectedPlans := []*models.SubscriptionPlan{
		{ID: 1, Name: "Free", PlanType: "free", Price: decimal.Zero},
		{ID: 2, Name: "Trial", PlanType: "trial", Price: decimal.Zero},
		{ID: 3, Name: "Premium Monthly", PlanType: "premium", Price: decimal.NewFromFloat(9.99)},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return expectedPlans, nil
		},
	}

	svc := newTestService(nil, planRepo, nil, nil, nil)

	plans, err := svc.GetPlans()
	require.NoError(t, err)
	assert.Len(t, plans, 3)
	assert.Len(t, plans, len(expectedPlans))
	for i, plan := range plans {
		assert.Equal(t, expectedPlans[i].ID, plan.ID)
		assert.Equal(t, expectedPlans[i].Name, plan.Name)
		assert.Equal(t, expectedPlans[i].PlanType, plan.PlanType)
	}
}

func TestGetPlans_RepoError(t *testing.T) {
	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return nil, errors.New("db error")
		},
	}

	svc := newTestService(nil, planRepo, nil, nil, nil)

	plans, err := svc.GetPlans()
	require.Error(t, err)
	assert.Nil(t, plans)
}

func TestGetPlans_EmptyList(t *testing.T) {
	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return []*models.SubscriptionPlan{}, nil
		},
	}

	svc := newTestService(nil, planRepo, nil, nil, nil)

	plans, err := svc.GetPlans()
	require.NoError(t, err)
	assert.Empty(t, plans)
}

func TestGetEffectivePlanType(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	tests := []struct {
		name     string
		sub      *models.Subscription
		expected string
	}{
		{
			name: "trial not expired returns trial",
			sub: &models.Subscription{
				PlanType:    "trial",
				TrialEndsAt: timePtr(time.Now().Add(7 * 24 * time.Hour)),
			},
			expected: "trial",
		},
		{
			name: "trial expired returns free",
			sub: &models.Subscription{
				PlanType:    "trial",
				TrialEndsAt: timePtr(time.Now().Add(-1 * time.Hour)),
			},
			expected: "free",
		},
		{
			name: "trial with nil TrialEndsAt returns trial",
			sub: &models.Subscription{
				PlanType:    "trial",
				TrialEndsAt: nil,
			},
			expected: "trial",
		},
		{
			name: "premium not expired returns premium",
			sub: &models.Subscription{
				PlanType:  "premium",
				ExpiresAt: timePtr(time.Now().Add(30 * 24 * time.Hour)),
			},
			expected: "premium",
		},
		{
			name: "premium expired returns free",
			sub: &models.Subscription{
				PlanType:  "premium",
				ExpiresAt: timePtr(time.Now().Add(-1 * time.Hour)),
			},
			expected: "free",
		},
		{
			name: "premium with nil ExpiresAt returns premium",
			sub: &models.Subscription{
				PlanType:  "premium",
				ExpiresAt: nil,
			},
			expected: "premium",
		},
		{
			name: "free returns free",
			sub: &models.Subscription{
				PlanType: "free",
			},
			expected: "free",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.GetEffectivePlanType(tc.sub)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// --- T11: Mutation tests ---

func TestUpgrade_NoExistingSubscription_NewCreated(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       10,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromFloat(9.99),
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "upgrade", result.ChangeType)
	assert.Equal(t, "Subscribed to Premium Monthly plan", result.Message)
	assert.Equal(t, 10, result.Subscription.PlanID)
	assert.Equal(t, "premium", result.Subscription.PlanType)
	assert.True(t, result.Subscription.IsActive)
	assert.Equal(t, 1, result.Subscription.UserID)

	require.Len(t, subRepo.createCalls, 1)
	assert.Equal(t, 10, subRepo.createCalls[0].PlanID)
}

func TestUpgrade_NoExistingSubscription_PlanNotFound(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 999)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrPlanNotFound))
}

func TestUpgrade_FreeToPremium(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:       1,
				UserID:   1,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			switch id {
			case 1:
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Free",
					PlanType: "free",
					Price:    decimal.Zero,
				}, nil
			case 10:
				return &models.SubscriptionPlan{
					ID:       10,
					Name:     "Premium Monthly",
					PlanType: "premium",
					Price:    decimal.NewFromFloat(9.99),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						Code:         "monthly",
						Name:         "Monthly",
						DurationDays: 30,
					},
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "upgrade", result.ChangeType)
	assert.Contains(t, result.Message, "Premium Monthly")
	assert.Equal(t, 10, result.Subscription.PlanID)
	assert.Equal(t, "premium", result.Subscription.PlanType)
	assert.True(t, result.Subscription.IsActive)
	assert.NotNil(t, result.Subscription.ExpiresAt)
	assert.NotNil(t, result.Subscription.SubscribedAt)

	require.Len(t, subRepo.updateCalls, 1)
}

func TestUpgrade_SamePlanReturnsErrSamePlan(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:        1,
				UserID:    1,
				PlanID:    10,
				PlanType:  "premium",
				IsActive:  true,
				ExpiresAt: timePtr(time.Now().Add(30 * 24 * time.Hour)),
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       10,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromFloat(9.99),
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrSamePlan))
}

func TestUpgrade_ToTrialReturnsErrInvalidPlanTransition(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:       1,
				UserID:   1,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			switch id {
			case 1:
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Free",
					PlanType: "free",
					Price:    decimal.Zero,
				}, nil
			case 5:
				return &models.SubscriptionPlan{
					ID:       5,
					Name:     "Trial",
					PlanType: "trial",
					Price:    decimal.Zero,
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 5)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrInvalidPlanTransition))
}

func TestUpgrade_PremiumToFreeReturnsErrDowngradeNotAllowed(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:        1,
				UserID:    1,
				PlanID:    10,
				PlanType:  "premium",
				IsActive:  true,
				ExpiresAt: timePtr(time.Now().Add(30 * 24 * time.Hour)),
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			switch id {
			case 10:
				return &models.SubscriptionPlan{
					ID:       10,
					Name:     "Premium Monthly",
					PlanType: "premium",
					Price:    decimal.NewFromFloat(9.99),
				}, nil
			case 1:
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Free",
					PlanType: "free",
					Price:    decimal.Zero,
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 1)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrDowngradeNotAllowed))
}

func TestUpgrade_RepoError(t *testing.T) {
	dbErr := errors.New("db connection failed")
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get subscription")
}

func TestUpgrade_DeterminesPlanChange_ShorterBillingPeriod(t *testing.T) {
	// Use an expired premium subscription so the effective plan type is "free",
	// which allows the "free" -> "premium" transition through validation.
	// The current plan has a yearly billing period (365 days) and the target has
	// monthly (30 days), so determineChangeType returns "plan_change".
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:        1,
				UserID:    1,
				PlanID:    10,
				PlanType:  "premium",
				IsActive:  true,
				ExpiresAt: timePtr(time.Now().Add(-1 * time.Hour)), // expired
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			switch id {
			case 10:
				return &models.SubscriptionPlan{
					ID:       10,
					Name:     "Premium Yearly",
					PlanType: "premium",
					Price:    decimal.NewFromFloat(99.99),
					BillingPeriod: &models.BillingPeriod{
						ID:           2,
						Code:         "yearly",
						Name:         "Yearly",
						DurationDays: 365,
					},
				}, nil
			case 11:
				return &models.SubscriptionPlan{
					ID:       11,
					Name:     "Premium Monthly",
					PlanType: "premium",
					Price:    decimal.NewFromFloat(9.99),
					BillingPeriod: &models.BillingPeriod{
						ID:           1,
						Code:         "monthly",
						Name:         "Monthly",
						DurationDays: 30,
					},
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 11)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "plan_change", result.ChangeType)
	assert.Equal(t, 11, result.Subscription.PlanID)
	assert.NotNil(t, result.Subscription.ExpiresAt)
}

func TestUpgrade_WithBillingPeriodSetsExpiry(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       10,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromFloat(9.99),
				BillingPeriod: &models.BillingPeriod{
					ID:           1,
					Code:         "monthly",
					Name:         "Monthly",
					DurationDays: 30,
				},
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	before := time.Now()
	result, err := svc.Upgrade(1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)

	sub := result.Subscription
	require.NotNil(t, sub.SubscribedAt)
	require.NotNil(t, sub.ExpiresAt)
	require.NotNil(t, sub.CurrentBillingPeriodID)
	assert.Equal(t, 1, *sub.CurrentBillingPeriodID)

	// ExpiresAt should be ~30 days from now
	expectedExpiry := before.AddDate(0, 0, 30)
	assert.WithinDuration(t, expectedExpiry, *sub.ExpiresAt, 5*time.Second)
}

// --- ScheduleDowngrade tests ---

func TestScheduleDowngrade_PremiumWithStripe_Scheduled(t *testing.T) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    1,
				UserID:                1,
				PlanID:                10,
				PlanType:              "premium",
				IsActive:              true,
				HasStripeSubscription: true,
				ExpiresAt:             &expiresAt,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1, 2}, intPtr(5))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Scheduled)
	assert.Equal(t, "Downgrade scheduled for end of billing period", result.Message)
	assert.NotNil(t, result.ExpiresAt)

	require.Len(t, subRepo.updateSubscriptionFullCalls, 1)
	saved := subRepo.updateSubscriptionFullCalls[0]
	require.NotNil(t, saved.PendingPlanID)
	assert.Equal(t, 1, *saved.PendingPlanID)
	assert.Equal(t, []int{1, 2}, saved.PendingDowngradeAccountIDs)
	require.NotNil(t, saved.PendingDowngradeBudgetID)
	assert.Equal(t, 5, *saved.PendingDowngradeBudgetID)
}

func TestScheduleDowngrade_PremiumWithoutStripe_Immediate(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    1,
				UserID:                1,
				PlanID:                10,
				PlanType:              "premium",
				IsActive:              true,
				HasStripeSubscription: false,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, intPtr(3))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Scheduled)
	assert.Equal(t, "Downgraded to free plan immediately", result.Message)

	require.Len(t, subRepo.updateSubscriptionFullCalls, 1)
	saved := subRepo.updateSubscriptionFullCalls[0]
	assert.Equal(t, 1, saved.PlanID)
	assert.Equal(t, "free", saved.PlanType)
	assert.Nil(t, saved.PendingPlanID)
	assert.Nil(t, saved.PendingDowngradeAccountIDs)
	assert.Nil(t, saved.PendingDowngradeBudgetID)
	assert.True(t, saved.IsActive)
}

func TestScheduleDowngrade_FreeUserReturnsErrAlreadyOnFreePlan(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:       1,
				UserID:   1,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrAlreadyOnFreePlan))
}

func TestScheduleDowngrade_NoSubscriptionReturnsErrAlreadyOnFreePlan(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrAlreadyOnFreePlan))
}

func TestScheduleDowngrade_TooManyAccountsReturnsErrInvalidEntitySelection(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:        1,
				UserID:    1,
				PlanID:    10,
				PlanType:  "premium",
				IsActive:  true,
				ExpiresAt: timePtr(time.Now().Add(30 * 24 * time.Hour)),
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	// Free plan limit is 2 accounts; passing 3 should fail
	result, err := svc.ScheduleDowngrade(1, []int{1, 2, 3}, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrInvalidEntitySelection))
}

func TestScheduleDowngrade_RepoError(t *testing.T) {
	dbErr := errors.New("database error")
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get subscription")
}

// --- ApplyDowngradeEntitySelection tests ---

func TestApplyDowngradeEntitySelection_Success(t *testing.T) {
	acctArchiver := &mockAccountArchiver{}
	budgetArchiver := &mockBudgetArchiver{}

	svc := newTestService(nil, nil, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	budgetID := 5
	err := svc.ApplyDowngradeEntitySelection(1, []int{1, 2}, &budgetID)
	require.NoError(t, err)

	assert.True(t, acctArchiver.called)
	assert.Equal(t, 1, acctArchiver.userID)
	assert.Equal(t, []int{1, 2}, acctArchiver.keepIDs)

	assert.True(t, budgetArchiver.called)
	assert.Equal(t, 1, budgetArchiver.userID)
	require.NotNil(t, budgetArchiver.keepID)
	assert.Equal(t, 5, *budgetArchiver.keepID)
}

func TestApplyDowngradeEntitySelection_NilArchiversReturnsNil(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)
	// Archivers not set (nil by default)

	err := svc.ApplyDowngradeEntitySelection(1, []int{1}, intPtr(2))
	assert.NoError(t, err)
}

func TestApplyDowngradeEntitySelection_AccountArchiverError(t *testing.T) {
	archiveErr := errors.New("archive failed")
	acctArchiver := &mockAccountArchiver{
		archiveFn: func(userID int, keepIDs []int) error {
			return archiveErr
		},
	}
	budgetArchiver := &mockBudgetArchiver{}

	svc := newTestService(nil, nil, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	err := svc.ApplyDowngradeEntitySelection(1, []int{1}, intPtr(2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archive accounts")
	assert.True(t, errors.Is(err, archiveErr))

	// Budget archiver should NOT be called since account archiver failed first
	assert.False(t, budgetArchiver.called)
}

func TestApplyDowngradeEntitySelection_BudgetArchiverError(t *testing.T) {
	archiveErr := errors.New("budget archive failed")
	acctArchiver := &mockAccountArchiver{}
	budgetArchiver := &mockBudgetArchiver{
		archiveFn: func(userID int, keepID *int) error {
			return archiveErr
		},
	}

	svc := newTestService(nil, nil, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	err := svc.ApplyDowngradeEntitySelection(1, []int{1}, intPtr(2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archive budgets")
	assert.True(t, errors.Is(err, archiveErr))

	// Account archiver should have been called successfully
	assert.True(t, acctArchiver.called)
}

// --- CreateTrialSubscription tests ---

func TestCreateTrialSubscription_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepo{}

	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return []*models.SubscriptionPlan{
				{ID: 1, Name: "Free", PlanType: "free", Price: decimal.Zero},
				{ID: 2, Name: "Trial", PlanType: "trial", Price: decimal.Zero},
				{ID: 3, Name: "Premium", PlanType: "premium", Price: decimal.NewFromFloat(9.99)},
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	before := time.Now()
	err := svc.CreateTrialSubscription(42)
	require.NoError(t, err)

	require.Len(t, subRepo.createCalls, 1)
	created := subRepo.createCalls[0]
	assert.Equal(t, 42, created.UserID)
	assert.Equal(t, 2, created.PlanID)
	assert.Equal(t, "trial", created.PlanType)
	assert.True(t, created.IsActive)
	assert.False(t, created.AutoRenew)
	require.NotNil(t, created.TrialStartedAt)
	require.NotNil(t, created.TrialEndsAt)

	// Trial should end approximately 14 days from now (default trialDurationDays)
	expectedEnd := before.AddDate(0, 0, 14)
	assert.WithinDuration(t, expectedEnd, *created.TrialEndsAt, 5*time.Second)
}

func TestCreateTrialSubscription_NoTrialPlanFound(t *testing.T) {
	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return []*models.SubscriptionPlan{
				{ID: 1, Name: "Free", PlanType: "free", Price: decimal.Zero},
				{ID: 3, Name: "Premium", PlanType: "premium", Price: decimal.NewFromFloat(9.99)},
			}, nil
		},
	}

	svc := newTestService(nil, planRepo, nil, nil, nil)

	err := svc.CreateTrialSubscription(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active trial plan found")
}

func TestCreateTrialSubscription_RepoCreateError(t *testing.T) {
	createErr := errors.New("insert failed")
	subRepo := &mockSubscriptionRepo{
		createFn: func(sub *models.Subscription) error {
			return createErr
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return []*models.SubscriptionPlan{
				{ID: 2, Name: "Trial", PlanType: "trial", Price: decimal.Zero},
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	err := svc.CreateTrialSubscription(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create trial subscription")
	assert.True(t, errors.Is(err, createErr))
}

func TestCreateTrialSubscription_GetActivePlansError(t *testing.T) {
	plansErr := errors.New("cannot fetch plans")
	planRepo := &mockSubscriptionPlanRepo{
		getActivePlansFn: func() ([]*models.SubscriptionPlan, error) {
			return nil, plansErr
		},
	}

	svc := newTestService(nil, planRepo, nil, nil, nil)

	err := svc.CreateTrialSubscription(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get active plans")
	assert.True(t, errors.Is(err, plansErr))
}

// --- T12: Worker method tests ---

// --- ProcessRenewals tests ---

func TestProcessRenewals_NoExpiredSubscriptions(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)
	assert.Empty(t, subRepo.updateCalls)
}

func TestProcessRenewals_TrialExpired_DowngradedToFree(t *testing.T) {
	trialEnd := time.Now().Add(-1 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:          1,
					UserID:      10,
					PlanID:      2,
					PlanType:    "trial",
					IsActive:    true,
					TrialEndsAt: &trialEnd,
				},
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	require.Len(t, subRepo.updateCalls, 1)
	updated := subRepo.updateCalls[0]
	assert.Equal(t, 1, updated.PlanID)
	assert.Equal(t, "free", updated.PlanType)
	assert.Nil(t, updated.TrialStartedAt)
	assert.Nil(t, updated.TrialEndsAt)
	assert.True(t, updated.IsActive)
}

func TestProcessRenewals_PremiumExpiredWithoutStripe_Deactivated(t *testing.T) {
	expiresAt := time.Now().Add(-1 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:                    2,
					UserID:                20,
					PlanID:                10,
					PlanType:              "premium",
					IsActive:              true,
					ExpiresAt:             &expiresAt,
					HasStripeSubscription: false,
				},
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	require.Len(t, subRepo.updateCalls, 1)
	updated := subRepo.updateCalls[0]
	assert.False(t, updated.IsActive)
	assert.Equal(t, 10, updated.PlanID) // plan ID unchanged
}

func TestProcessRenewals_PremiumExpiredWithStripe_Skipped(t *testing.T) {
	expiresAt := time.Now().Add(-1 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:                    3,
					UserID:                30,
					PlanID:                10,
					PlanType:              "premium",
					IsActive:              true,
					ExpiresAt:             &expiresAt,
					HasStripeSubscription: true,
				},
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	// No updates should be made (Stripe handles renewal)
	assert.Empty(t, subRepo.updateCalls)
}

func TestProcessRenewals_GetFreePlanErrorForTrial_LoggedContinues(t *testing.T) {
	trialEnd := time.Now().Add(-1 * time.Hour)
	expiresAt := time.Now().Add(-1 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:          1,
					UserID:      10,
					PlanID:      2,
					PlanType:    "trial",
					IsActive:    true,
					TrialEndsAt: &trialEnd,
				},
				{
					ID:                    2,
					UserID:                20,
					PlanID:                10,
					PlanType:              "premium",
					IsActive:              true,
					ExpiresAt:             &expiresAt,
					HasStripeSubscription: false,
				},
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return nil, errors.New("free plan lookup failed")
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	// The trial one should have been skipped due to error,
	// but the premium one should still be deactivated.
	require.Len(t, subRepo.updateCalls, 1)
	assert.Equal(t, 2, subRepo.updateCalls[0].ID)
	assert.False(t, subRepo.updateCalls[0].IsActive)
}

func TestProcessRenewals_UpdateError_LoggedContinues(t *testing.T) {
	expiresAt := time.Now().Add(-1 * time.Hour)
	updateCallCount := 0
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:                    1,
					UserID:                10,
					PlanID:                10,
					PlanType:              "premium",
					IsActive:              true,
					ExpiresAt:             &expiresAt,
					HasStripeSubscription: false,
				},
				{
					ID:                    2,
					UserID:                20,
					PlanID:                11,
					PlanType:              "premium",
					IsActive:              true,
					ExpiresAt:             &expiresAt,
					HasStripeSubscription: false,
				},
			}, nil
		},
		updateFn: func(sub *models.Subscription) error {
			updateCallCount++
			if updateCallCount == 1 {
				return errors.New("update failed")
			}
			return nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	// Both should have been attempted
	assert.Equal(t, 2, updateCallCount)
}

func TestProcessRenewals_GetExpiredSubscriptionsError(t *testing.T) {
	dbErr := errors.New("query failed")
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get expired subscriptions")
}

func TestProcessRenewals_UnknownPlanType_WarnedContinues(t *testing.T) {
	expiresAt := time.Now().Add(-1 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getExpiredFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:        1,
					UserID:    10,
					PlanID:    99,
					PlanType:  "enterprise",
					IsActive:  true,
					ExpiresAt: &expiresAt,
				},
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessRenewals()
	require.NoError(t, err)

	// No updates should be performed for unknown plan types
	assert.Empty(t, subRepo.updateCalls)
}

// --- ProcessExpiredDowngrades tests ---

func TestProcessExpiredDowngrades_NoPendingDowngrades(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)
	assert.Empty(t, subRepo.updateSubscriptionFullCalls)
}

func TestProcessExpiredDowngrades_PendingDowngradeApplied_FreePlanWithArchival(t *testing.T) {
	freePlanID := 1
	budgetID := 5
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:                         1,
					UserID:                     10,
					PlanID:                     10,
					PlanType:                   "premium",
					IsActive:                   true,
					PendingPlanID:              &freePlanID,
					PendingDowngradeAccountIDs: []int{1, 2},
					PendingDowngradeBudgetID:   &budgetID,
				},
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			if id == 1 {
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Free",
					PlanType: "free",
					Price:    decimal.Zero,
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	acctArchiver := &mockAccountArchiver{}
	budgetArchiver := &mockBudgetArchiver{}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)

	require.Len(t, subRepo.updateSubscriptionFullCalls, 1)
	updated := subRepo.updateSubscriptionFullCalls[0]
	assert.Equal(t, 1, updated.PlanID)
	assert.Equal(t, "free", updated.PlanType)
	assert.Nil(t, updated.PendingPlanID)
	assert.Nil(t, updated.PendingDowngradeAccountIDs)
	assert.Nil(t, updated.PendingDowngradeBudgetID)
	assert.True(t, updated.IsActive)

	// Verify archival was called
	assert.True(t, acctArchiver.called)
	assert.Equal(t, 10, acctArchiver.userID)
	assert.Equal(t, []int{1, 2}, acctArchiver.keepIDs)

	assert.True(t, budgetArchiver.called)
	assert.Equal(t, 10, budgetArchiver.userID)
	require.NotNil(t, budgetArchiver.keepID)
	assert.Equal(t, 5, *budgetArchiver.keepID)
}

func TestProcessExpiredDowngrades_PendingPlanLookupError_LoggedContinues(t *testing.T) {
	pendingID := 99
	pendingID2 := 1
	updateCount := 0
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:            1,
					UserID:        10,
					PlanID:        10,
					PlanType:      "premium",
					IsActive:      true,
					PendingPlanID: &pendingID,
				},
				{
					ID:            2,
					UserID:        20,
					PlanID:        11,
					PlanType:      "premium",
					IsActive:      true,
					PendingPlanID: &pendingID2,
				},
			}, nil
		},
		updateSubscriptionFullFn: func(sub *models.Subscription) error {
			updateCount++
			return nil
		},
	}

	callCount := 0
	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			callCount++
			if id == 99 {
				return nil, errors.New("plan not found")
			}
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)

	// First one should be skipped due to plan lookup error, second should succeed
	assert.Equal(t, 2, callCount)
	assert.Equal(t, 1, updateCount)
}

func TestProcessExpiredDowngrades_NilPendingPlanID_Skipped(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:            1,
					UserID:        10,
					PlanID:        10,
					PlanType:      "premium",
					IsActive:      true,
					PendingPlanID: nil, // should not happen but guard defensively
				},
			}, nil
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)

	assert.Empty(t, subRepo.updateSubscriptionFullCalls)
}

func TestProcessExpiredDowngrades_UpdateSubscriptionFullError_LoggedContinues(t *testing.T) {
	pendingID1 := 1
	pendingID2 := 1
	updateAttempts := 0
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:            1,
					UserID:        10,
					PlanID:        10,
					PlanType:      "premium",
					IsActive:      true,
					PendingPlanID: &pendingID1,
				},
				{
					ID:            2,
					UserID:        20,
					PlanID:        11,
					PlanType:      "premium",
					IsActive:      true,
					PendingPlanID: &pendingID2,
				},
			}, nil
		},
		updateSubscriptionFullFn: func(sub *models.Subscription) error {
			updateAttempts++
			if updateAttempts == 1 {
				return errors.New("update failed")
			}
			return nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)

	// Both should be attempted
	assert.Equal(t, 2, updateAttempts)
}

func TestProcessExpiredDowngrades_GetPendingDowngradesError(t *testing.T) {
	dbErr := errors.New("query failed")
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return nil, dbErr
		},
	}

	svc := newTestService(subRepo, nil, nil, nil, nil)

	err := svc.ProcessExpiredDowngrades()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get pending downgrades")
}

func TestProcessExpiredDowngrades_NonFreePendingPlan_NoArchival(t *testing.T) {
	pendingPlanID := 5
	subRepo := &mockSubscriptionRepo{
		getPendingDowngradesFn: func() ([]*models.Subscription, error) {
			return []*models.Subscription{
				{
					ID:                         1,
					UserID:                     10,
					PlanID:                     10,
					PlanType:                   "premium",
					IsActive:                   true,
					PendingPlanID:              &pendingPlanID,
					PendingDowngradeAccountIDs: []int{1},
					PendingDowngradeBudgetID:   intPtr(3),
				},
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       5,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromFloat(9.99),
			}, nil
		},
	}

	acctArchiver := &mockAccountArchiver{}
	budgetArchiver := &mockBudgetArchiver{}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	err := svc.ProcessExpiredDowngrades()
	require.NoError(t, err)

	require.Len(t, subRepo.updateSubscriptionFullCalls, 1)
	updated := subRepo.updateSubscriptionFullCalls[0]
	assert.Equal(t, 5, updated.PlanID)
	assert.Equal(t, "premium", updated.PlanType)

	// Archival should NOT be called for non-free plans
	assert.False(t, acctArchiver.called)
	assert.False(t, budgetArchiver.called)
}

// --- Additional error path tests ---

// buildFreeStatus: checkRequiresDowngrade returns an error after trial-to-free downgrade.
// The error should be swallowed gracefully (non-fatal), and RequiresDowngrade defaults to false.
func TestGetStatus_TrialExpiredAutoDowngrade_CheckRequiresDowngradeError(t *testing.T) {
	trialEnd := time.Now().Add(-1 * time.Hour) // expired
	subscribedAt := time.Now().Add(-15 * 24 * time.Hour)

	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:           10,
				UserID:       1,
				PlanID:       2,
				PlanType:     "trial",
				IsActive:     true,
				TrialEndsAt:  &trialEnd,
				SubscribedAt: &subscribedAt,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	// Account counter returns an error, which makes checkRequiresDowngrade fail
	acctCounter := &mockAccountCounter{
		countFn: func(userID int) (int, error) {
			return 0, errors.New("account count unavailable")
		},
	}

	svc := newTestService(subRepo, planRepo, nil, acctCounter, nil)

	status, err := svc.GetStatus(1)
	require.NoError(t, err) // the error is non-fatal
	require.NotNil(t, status)

	assert.Equal(t, "free", status.PlanType)
	assert.True(t, status.IsActive)
	assert.False(t, status.RequiresDowngrade) // defaults to false on error
	assert.Equal(t, PlanLimits["free"], status.Limits)

	// Verify the downgrade itself was persisted
	require.Len(t, subRepo.updateCalls, 1)
	assert.Equal(t, "free", subRepo.updateCalls[0].PlanType)
}

// Upgrade: GetByID(sub.PlanID) fails when loading the current plan (line ~372-375).
func TestUpgrade_ExistingSubscription_GetCurrentPlanFails(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:       1,
				UserID:   1,
				PlanID:   50, // current plan ID
				PlanType: "free",
				IsActive: true,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			if id == 10 {
				// Target plan lookup succeeds
				return &models.SubscriptionPlan{
					ID:       10,
					Name:     "Premium Monthly",
					PlanType: "premium",
					Price:    decimal.NewFromFloat(9.99),
				}, nil
			}
			if id == 50 {
				// Current plan lookup fails
				return nil, errors.New("current plan database error")
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get current plan")
}

// Upgrade: targetPlan.BillingPeriod is nil during update of existing subscription (line ~398-403).
// When BillingPeriod is nil, ExpiresAt and SubscribedAt should NOT be set on the subscription.
func TestUpgrade_ExistingSubscription_TargetPlanNilBillingPeriod(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:       1,
				UserID:   1,
				PlanID:   1,
				PlanType: "free",
				IsActive: true,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getByIDFn: func(id int) (*models.SubscriptionPlan, error) {
			switch id {
			case 1:
				return &models.SubscriptionPlan{
					ID:       1,
					Name:     "Free",
					PlanType: "free",
					Price:    decimal.Zero,
				}, nil
			case 10:
				return &models.SubscriptionPlan{
					ID:            10,
					Name:          "Premium Lifetime",
					PlanType:      "premium",
					Price:         decimal.NewFromFloat(199.99),
					BillingPeriod: nil, // no billing period
				}, nil
			}
			return nil, sql.ErrNoRows
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.Upgrade(1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "upgrade", result.ChangeType)
	assert.Equal(t, 10, result.Subscription.PlanID)
	assert.Equal(t, "premium", result.Subscription.PlanType)
	assert.True(t, result.Subscription.IsActive)
	// With nil BillingPeriod, these fields should remain unset from the upgrade path
	assert.Nil(t, result.Subscription.CurrentBillingPeriodID)
	assert.Nil(t, result.Subscription.SubscribedAt)
	assert.Nil(t, result.Subscription.ExpiresAt)

	require.Len(t, subRepo.updateCalls, 1)
}

// determineChangeType: only current.BillingPeriod is nil (target has billing period).
// Should return "upgrade".
func TestDetermineChangeType_CurrentNilBillingPeriod(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	current := &models.SubscriptionPlan{
		ID:            1,
		Name:          "Free",
		PlanType:      "free",
		BillingPeriod: nil, // current has no billing period
	}
	target := &models.SubscriptionPlan{
		ID:       10,
		Name:     "Premium Monthly",
		PlanType: "premium",
		BillingPeriod: &models.BillingPeriod{
			ID:           1,
			Code:         "monthly",
			Name:         "Monthly",
			DurationDays: 30,
		},
	}

	result := svc.determineChangeType(current, target)
	assert.Equal(t, "upgrade", result)
}

// determineChangeType: only target.BillingPeriod is nil (current has billing period).
// Should also return "upgrade".
func TestDetermineChangeType_TargetNilBillingPeriod(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil)

	current := &models.SubscriptionPlan{
		ID:       10,
		Name:     "Premium Monthly",
		PlanType: "premium",
		BillingPeriod: &models.BillingPeriod{
			ID:           1,
			Code:         "monthly",
			Name:         "Monthly",
			DurationDays: 30,
		},
	}
	target := &models.SubscriptionPlan{
		ID:            20,
		Name:          "Premium Lifetime",
		PlanType:      "premium",
		BillingPeriod: nil, // target has no billing period
	}

	result := svc.determineChangeType(current, target)
	assert.Equal(t, "upgrade", result)
}

// ScheduleDowngrade: UpdateSubscriptionFull fails during immediate downgrade (no Stripe).
func TestScheduleDowngrade_ImmediateDowngrade_UpdateSubscriptionFullFails(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    1,
				UserID:                1,
				PlanID:                10,
				PlanType:              "premium",
				IsActive:              true,
				HasStripeSubscription: false,
			}, nil
		},
		updateSubscriptionFullFn: func(sub *models.Subscription) error {
			return errors.New("update full failed")
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, intPtr(3))
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "save immediate downgrade")
}

// ScheduleDowngrade: ApplyDowngradeEntitySelection error during immediate downgrade is non-fatal.
// The downgrade itself succeeds, but the archival step fails gracefully.
func TestScheduleDowngrade_ImmediateDowngrade_ArchivalErrorNonFatal(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    1,
				UserID:                1,
				PlanID:                10,
				PlanType:              "premium",
				IsActive:              true,
				HasStripeSubscription: false,
			}, nil
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	// Set up archivers where account archiver fails
	acctArchiver := &mockAccountArchiver{
		archiveFn: func(userID int, keepIDs []int) error {
			return errors.New("archival failed")
		},
	}
	budgetArchiver := &mockBudgetArchiver{}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)
	svc.SetArchivers(acctArchiver, budgetArchiver)

	result, err := svc.ScheduleDowngrade(1, []int{1}, intPtr(3))
	require.NoError(t, err) // archival error is non-fatal
	require.NotNil(t, result)

	assert.False(t, result.Scheduled)
	assert.Equal(t, "Downgraded to free plan immediately", result.Message)

	// Verify the subscription was still saved
	require.Len(t, subRepo.updateSubscriptionFullCalls, 1)
	saved := subRepo.updateSubscriptionFullCalls[0]
	assert.Equal(t, 1, saved.PlanID)
	assert.Equal(t, "free", saved.PlanType)
	assert.True(t, saved.IsActive)

	// Verify archiver was attempted
	assert.True(t, acctArchiver.called)
}

// ScheduleDowngrade: UpdateSubscriptionFull fails during scheduled downgrade (with Stripe).
func TestScheduleDowngrade_ScheduledDowngrade_UpdateSubscriptionFullFails(t *testing.T) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	subRepo := &mockSubscriptionRepo{
		getByUserIDFn: func(userID int) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                    1,
				UserID:                1,
				PlanID:                10,
				PlanType:              "premium",
				IsActive:              true,
				HasStripeSubscription: true,
				ExpiresAt:             &expiresAt,
			}, nil
		},
		updateSubscriptionFullFn: func(sub *models.Subscription) error {
			return errors.New("save scheduled downgrade failed")
		},
	}

	planRepo := &mockSubscriptionPlanRepo{
		getFreePlanFn: func() (*models.SubscriptionPlan, error) {
			return &models.SubscriptionPlan{
				ID:       1,
				Name:     "Free",
				PlanType: "free",
				Price:    decimal.Zero,
			}, nil
		},
	}

	svc := newTestService(subRepo, planRepo, nil, nil, nil)

	result, err := svc.ScheduleDowngrade(1, []int{1}, intPtr(3))
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "save scheduled downgrade")
}
