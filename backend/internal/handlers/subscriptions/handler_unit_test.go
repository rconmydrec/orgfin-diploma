package subscriptions

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/subscription"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// MockSubscriptionService implements SubscriptionServiceInterface for testing.
type MockSubscriptionService struct {
	GetStatusFunc         func(userID int) (*subscription.SubscriptionStatus, error)
	GetPlansFunc          func() ([]*subscription.SubscriptionPlan, error)
	UpgradeFunc           func(userID int, planID int) (*subscription.UpgradeResult, error)
	ScheduleDowngradeFunc func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error)
}

func (m *MockSubscriptionService) GetStatus(userID int) (*subscription.SubscriptionStatus, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(userID)
	}
	return &subscription.SubscriptionStatus{
		PlanType: "free",
		IsActive: true,
		Limits:   &subscription.Limits{Accounts: 2, Budgets: 1, PlanningDays: 14},
	}, nil
}

func (m *MockSubscriptionService) GetPlans() ([]*subscription.SubscriptionPlan, error) {
	if m.GetPlansFunc != nil {
		return m.GetPlansFunc()
	}
	return []*subscription.SubscriptionPlan{}, nil
}

func (m *MockSubscriptionService) Upgrade(userID int, planID int) (*subscription.UpgradeResult, error) {
	if m.UpgradeFunc != nil {
		return m.UpgradeFunc(userID, planID)
	}
	return &subscription.UpgradeResult{
		ChangeType:   "upgrade",
		Subscription: &subscription.Subscription{ID: 1, UserID: userID, PlanID: planID, IsActive: true},
		Message:      "Subscribed to Premium plan",
	}, nil
}

func (m *MockSubscriptionService) ScheduleDowngrade(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
	if m.ScheduleDowngradeFunc != nil {
		return m.ScheduleDowngradeFunc(userID, accountIDs, budgetID)
	}
	return &subscription.DowngradeResult{
		Scheduled: false,
		Message:   "Downgraded to free plan immediately",
	}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockSubscriptionService) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := &MockSubscriptionService{}
	handler := New(svc, logger)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return handler, e, svc
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== GetPlans Tests ====================

func TestGetPlansDBError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetPlansFunc = func() ([]*subscription.SubscriptionPlan, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetPlans(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetPlansWithBillingPeriod(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetPlansFunc = func() ([]*subscription.SubscriptionPlan, error) {
		return []*subscription.SubscriptionPlan{
			{
				ID:       1,
				Name:     "Premium Monthly",
				PlanType: "premium",
				Price:    decimal.NewFromInt(10),
				BillingPeriod: &subscription.BillingPeriod{
					ID:           1,
					Code:         "monthly",
					Name:         "Monthly",
					DurationDays: 30,
				},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetPlans(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Premium Monthly")
}

func TestGetPlansAllTypes(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetPlansFunc = func() ([]*subscription.SubscriptionPlan, error) {
		return []*subscription.SubscriptionPlan{
			{ID: 1, Name: "Free", PlanType: "free", Price: decimal.Zero},
			{ID: 2, Name: "Trial", PlanType: "trial", Price: decimal.Zero},
			{ID: 3, Name: "Premium", PlanType: "premium", Price: decimal.NewFromInt(10)},
			{ID: 4, Name: "Unknown", PlanType: "unknown", Price: decimal.Zero},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetPlans(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== GetStatus Tests ====================

func TestGetStatusNoSubscription(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType: "free",
			IsActive: true,
			Limits:   &subscription.Limits{Accounts: 2, Budgets: 1, PlanningDays: 14},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "free")
}

func TestGetStatusWithTrial(t *testing.T) {
	handler, e, svc := setupTestHandler()
	trialDays := 7
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType:           "trial",
			IsActive:           true,
			PlanID:             1,
			TrialDaysRemaining: &trialDays,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "trial")
}

func TestGetStatusTrialExpired(t *testing.T) {
	handler, e, svc := setupTestHandler()
	zeroDays := 0
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType:           "trial",
			IsActive:           true,
			PlanID:             1,
			TrialDaysRemaining: &zeroDays,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetStatusWithPendingPlan(t *testing.T) {
	handler, e, svc := setupTestHandler()
	pendingID := 2
	pendingName := "Free Plan"
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType:        "premium",
			IsActive:        true,
			PlanID:          1,
			PendingPlanID:   &pendingID,
			PendingPlanName: &pendingName,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Free Plan")
}

func TestGetStatusPendingPlanError(t *testing.T) {
	// The service handles the pending plan lookup internally.
	// If the pending plan lookup fails, the service returns PendingPlanName as nil.
	handler, e, svc := setupTestHandler()
	pendingID := 2
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType:      "premium",
			IsActive:      true,
			PlanID:        1,
			PendingPlanID: &pendingID,
			// PendingPlanName is nil because the service failed to look it up
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetStatusAllPlanTypes(t *testing.T) {
	handler, e, svc := setupTestHandler()

	// Test unknown type — the service returns whatever plan type it has;
	// the handler maps it via mapPlanType which defaults to "free".
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType: "unknown",
			IsActive: true,
			PlanID:   1,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetStatusServiceError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return nil, errors.New("database connection error")
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal server error")
}

func TestGetStatusNilLimits(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType: "premium",
			IsActive: true,
			PlanID:   1,
			Limits:   nil,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// limits should be null in JSON since no limits for premium
	assert.Contains(t, rec.Body.String(), `"limits":null`)
}

func TestGetStatusZeroPlanID(t *testing.T) {
	// When PlanID is 0 (default free status with no subscription record),
	// the handler should return null for planId.
	handler, e, svc := setupTestHandler()
	svc.GetStatusFunc = func(userID int) (*subscription.SubscriptionStatus, error) {
		return &subscription.SubscriptionStatus{
			PlanType: "free",
			IsActive: true,
			PlanID:   0,
			Limits:   &subscription.Limits{Accounts: 2, Budgets: 1, PlanningDays: 14},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"planId":null`)
}

// ==================== Upgrade Tests ====================

func TestUpgradeBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpgradeValidateError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUpgradePlanNotFound(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, subscription.ErrPlanNotFound
	}

	body := `{"planId": 999}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpgradeNewSubscription(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return &subscription.UpgradeResult{
			ChangeType:   "upgrade",
			Subscription: &subscription.Subscription{ID: 1, UserID: userID, PlanID: planID, IsActive: true},
			Message:      "Subscribed to Premium plan",
		}, nil
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpgradeNewSubscriptionCreateError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, errors.New("create subscription: db error")
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpgradeExistingSubscription(t *testing.T) {
	handler, e, svc := setupTestHandler()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return &subscription.UpgradeResult{
			ChangeType: "plan_change",
			Subscription: &subscription.Subscription{
				ID:        1,
				UserID:    userID,
				PlanID:    planID,
				IsActive:  true,
				ExpiresAt: &expiresAt,
			},
			Message: "Plan changed to Premium",
		}, nil
	}

	body := `{"planId": 2}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpgradeExistingSubscriptionUpdateError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, errors.New("update subscription: db error")
	}

	body := `{"planId": 2}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpgradeSamePlan(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, subscription.ErrSamePlan
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Already on this plan")
}

func TestUpgradeInvalidTransition(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, subscription.ErrInvalidPlanTransition
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid plan transition")
}

func TestUpgradeDowngradeNotAllowed(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.UpgradeFunc = func(userID int, planID int) (*subscription.UpgradeResult, error) {
		return nil, subscription.ErrDowngradeNotAllowed
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/upgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Upgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Downgrade not allowed via upgrade endpoint")
}

// ==================== Downgrade Tests ====================

func TestDowngradeBindError(t *testing.T) {
	handler, e, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDowngradeNoSubscription(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return nil, subscription.ErrAlreadyOnFreePlan
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Already on free plan")
}

func TestDowngradeFreePlanError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return nil, errors.New("get free plan: not found")
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDowngradeUpdateError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return nil, errors.New("save immediate downgrade: db error")
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDowngradeSuccess(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return &subscription.DowngradeResult{
			Scheduled: false,
			Message:   "Downgraded to free plan immediately",
		}, nil
	}

	body := `{"accountIds": [1, 2], "budgetId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDowngradeScheduled(t *testing.T) {
	handler, e, svc := setupTestHandler()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return &subscription.DowngradeResult{
			Scheduled: true,
			Message:   "Downgrade scheduled for end of billing period",
			ExpiresAt: &expiresAt,
		}, nil
	}

	body := `{"accountIds": [1], "budgetId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Downgrade scheduled")
	assert.Contains(t, rec.Body.String(), `"pendingDowngrade":true`)
}

func TestDowngradeInvalidEntitySelection(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.ScheduleDowngradeFunc = func(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error) {
		return nil, subscription.ErrInvalidEntitySelection
	}

	body := `{"accountIds": [1, 2, 3, 4, 5]}`
	req := httptest.NewRequest(http.MethodPost, "/downgrade", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.Downgrade(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Entity selection violates plan limits")
}

// ==================== Error Mapping Tests ====================

func TestMapServiceErrorAllCases(t *testing.T) {
	handler, e, _ := setupTestHandler()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"PlanNotFound", subscription.ErrPlanNotFound, http.StatusNotFound, "Plan not found"},
		{"SamePlan", subscription.ErrSamePlan, http.StatusBadRequest, "Already on this plan"},
		{"InvalidPlanTransition", subscription.ErrInvalidPlanTransition, http.StatusBadRequest, "Invalid plan transition"},
		{"DowngradeNotAllowed", subscription.ErrDowngradeNotAllowed, http.StatusBadRequest, "Downgrade not allowed via upgrade endpoint"},
		{"AlreadyOnFreePlan", subscription.ErrAlreadyOnFreePlan, http.StatusBadRequest, "Already on free plan"},
		{"InvalidEntitySelection", subscription.ErrInvalidEntitySelection, http.StatusBadRequest, "Entity selection violates plan limits"},
		{"UnknownError", errors.New("unknown"), http.StatusInternalServerError, "Internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.mapServiceError(c, tt.err)
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantBody)
		})
	}
}

// ==================== RegisterRoutes Tests ====================

func TestRegisterRoutes(t *testing.T) {
	handler, e, _ := setupTestHandler()

	g := e.Group("/subscriptions")
	handler.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/subscriptions/plans"])
	assert.True(t, routePaths["/subscriptions/status"])
	assert.True(t, routePaths["/subscriptions/upgrade"])
	assert.True(t, routePaths["/subscriptions/downgrade"])
}
