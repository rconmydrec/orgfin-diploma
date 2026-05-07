package payments

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/services/payment"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// Mock services

type MockPaymentService struct {
	CreateCheckoutFunc        func(userID int, planID int, email string, name string) (*payment.CheckoutResult, error)
	CreatePortalFunc          func(userID int, returnURL *string) (*payment.PortalResult, error)
	GetPaymentStatusFunc      func(userID int) (*payment.PaymentStatus, error)
	GetSessionStatusFunc      func(userID int, sessionID string) (*payment.SessionStatus, error)
	GetUpgradePriceFunc       func(userID int, targetPlanID int) (*payment.UpgradePriceInfo, error)
	ChangePlanFunc            func(userID int, targetPlanID int) (*payment.ChangePlanResult, error)
	CancelScheduledChangeFunc func(userID int) error
}

func (m *MockPaymentService) CreateCheckout(userID int, planID int, email string, name string) (*payment.CheckoutResult, error) {
	if m.CreateCheckoutFunc != nil {
		return m.CreateCheckoutFunc(userID, planID, email, name)
	}
	return &payment.CheckoutResult{
		CheckoutURL:            "https://checkout.stripe.com/test",
		SessionID:              "cs_test_123",
		CreditAppliedCents:     0,
		CreditAppliedFormatted: "",
	}, nil
}

func (m *MockPaymentService) CreatePortal(userID int, returnURL *string) (*payment.PortalResult, error) {
	if m.CreatePortalFunc != nil {
		return m.CreatePortalFunc(userID, returnURL)
	}
	return &payment.PortalResult{
		PortalURL: "https://billing.stripe.com/test",
	}, nil
}

func (m *MockPaymentService) GetPaymentStatus(userID int) (*payment.PaymentStatus, error) {
	if m.GetPaymentStatusFunc != nil {
		return m.GetPaymentStatusFunc(userID)
	}
	return &payment.PaymentStatus{
		CurrentPlanType: "free",
	}, nil
}

func (m *MockPaymentService) GetSessionStatus(userID int, sessionID string) (*payment.SessionStatus, error) {
	if m.GetSessionStatusFunc != nil {
		return m.GetSessionStatusFunc(userID, sessionID)
	}
	return &payment.SessionStatus{
		SessionID:     sessionID,
		Status:        "complete",
		PaymentStatus: "paid",
	}, nil
}

func (m *MockPaymentService) GetUpgradePrice(userID int, targetPlanID int) (*payment.UpgradePriceInfo, error) {
	if m.GetUpgradePriceFunc != nil {
		return m.GetUpgradePriceFunc(userID, targetPlanID)
	}
	return &payment.UpgradePriceInfo{
		PlanID:                 targetPlanID,
		PlanName:               "Premium",
		PlanType:               "premium",
		OriginalPriceCents:     1000,
		CreditCents:            0,
		FinalPriceCents:        1000,
		OriginalPriceFormatted: "10.00 USD",
		FinalPriceFormatted:    "10.00 USD",
		CurrencyCode:           "USD",
	}, nil
}

func (m *MockPaymentService) ChangePlan(userID int, targetPlanID int) (*payment.ChangePlanResult, error) {
	if m.ChangePlanFunc != nil {
		return m.ChangePlanFunc(userID, targetPlanID)
	}
	return &payment.ChangePlanResult{
		Success:       true,
		IsUpgrade:     true,
		NewPlanName:   "Premium",
		EffectiveDate: "immediately",
		Message:       "Plan changed successfully",
	}, nil
}

func (m *MockPaymentService) CancelScheduledChange(userID int) error {
	if m.CancelScheduledChangeFunc != nil {
		return m.CancelScheduledChangeFunc(userID)
	}
	return nil
}

type MockWebhookHandler struct {
	HandleWebhookFunc func(payload []byte, signature string) *payment.WebhookResult
}

func (m *MockWebhookHandler) HandleWebhook(payload []byte, signature string) *payment.WebhookResult {
	if m.HandleWebhookFunc != nil {
		return m.HandleWebhookFunc(payload, signature)
	}
	return &payment.WebhookResult{
		EventType: "test.event",
		Success:   true,
		Message:   "event processed",
	}
}

func setupTestHandler() (*Handler, *echo.Echo, *MockPaymentService, *MockWebhookHandler) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	paymentSvc := &MockPaymentService{}
	webhookHandler := &MockWebhookHandler{}
	handler := New(paymentSvc, webhookHandler, logger)
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	return handler, e, paymentSvc, webhookHandler
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// ==================== CreateCheckout Tests ====================

func TestCreateCheckoutBindError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCheckout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCheckoutValidateError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCheckout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateCheckoutPlanNotFound(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.CreateCheckoutFunc = func(userID int, planID int, email string, name string) (*payment.CheckoutResult, error) {
		return nil, payment.ErrPlanNotFound
	}

	body := `{"planId": 999}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCheckout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateCheckoutSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCheckout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "checkoutUrl")
}

func TestCreateCheckoutServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.CreateCheckoutFunc = func(userID int, planID int, email string, name string) (*payment.CheckoutResult, error) {
		return nil, payment.ErrInvalidPrice
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreateCheckout(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ==================== CreatePortal Tests ====================

func TestCreatePortalBindError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/portal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreatePortal(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreatePortalSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"returnUrl": "http://localhost/return"}`
	req := httptest.NewRequest(http.MethodPost, "/portal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreatePortal(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "portalUrl")
}

func TestCreatePortalServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.CreatePortalFunc = func(userID int, returnURL *string) (*payment.PortalResult, error) {
		return nil, payment.ErrNoStripeCustomer
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/portal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CreatePortal(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ==================== GetStatus Tests ====================

func TestGetStatusNoSubscription(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.GetPaymentStatusFunc = func(userID int) (*payment.PaymentStatus, error) {
		return &payment.PaymentStatus{
			CurrentPlanType: "free",
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

func TestGetStatusWithSubscription(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	subID := "sub_123"
	paymentSvc.GetPaymentStatusFunc = func(userID int) (*payment.PaymentStatus, error) {
		return &payment.PaymentStatus{
			HasStripeCustomer:        true,
			StripeSubscriptionActive: true,
			StripeSubscriptionID:     &subID,
			CurrentPlanType:          "premium",
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "premium")
}

func TestGetStatusServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.GetPaymentStatusFunc = func(userID int) (*payment.PaymentStatus, error) {
		return nil, errors.New("unexpected error")
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== GetSessionStatus Tests ====================

func TestGetSessionStatus(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/session/cs_test_123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("session_id")
	c.SetParamValues("cs_test_123")
	setUserContext(c, 1)

	err := handler.GetSessionStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "cs_test_123")
}

func TestGetSessionStatusServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.GetSessionStatusFunc = func(userID int, sessionID string) (*payment.SessionStatus, error) {
		return nil, payment.ErrProviderUnavailable
	}

	req := httptest.NewRequest(http.MethodGet, "/session/cs_test_123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("session_id")
	c.SetParamValues("cs_test_123")
	setUserContext(c, 1)

	err := handler.GetSessionStatus(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

// ==================== GetUpgradePrice Tests ====================

func TestGetUpgradePriceInvalidID(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/upgrade-price/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("plan_id")
	c.SetParamValues("abc")
	setUserContext(c, 1)

	err := handler.GetUpgradePrice(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetUpgradePricePlanNotFound(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.GetUpgradePriceFunc = func(userID int, targetPlanID int) (*payment.UpgradePriceInfo, error) {
		return nil, payment.ErrPlanNotFound
	}

	req := httptest.NewRequest(http.MethodGet, "/upgrade-price/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("plan_id")
	c.SetParamValues("999")
	setUserContext(c, 1)

	err := handler.GetUpgradePrice(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetUpgradePriceSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/upgrade-price/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("plan_id")
	c.SetParamValues("1")
	setUserContext(c, 1)

	err := handler.GetUpgradePrice(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "planId")
}

// ==================== ChangePlan Tests ====================

func TestChangePlanBindError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestChangePlanValidateError(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestChangePlanNotFound(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.ChangePlanFunc = func(userID int, targetPlanID int) (*payment.ChangePlanResult, error) {
		return nil, payment.ErrPlanNotFound
	}

	body := `{"planId": 999}`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestChangePlanNoSubscription(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.ChangePlanFunc = func(userID int, targetPlanID int) (*payment.ChangePlanResult, error) {
		return nil, payment.ErrNoStripeSubscription
	}

	body := `{"planId": 1}`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestChangePlanServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.ChangePlanFunc = func(userID int, targetPlanID int) (*payment.ChangePlanResult, error) {
		return nil, errors.New("unexpected error")
	}

	body := `{"planId": 2}`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestChangePlanSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"planId": 2}`
	req := httptest.NewRequest(http.MethodPost, "/change-plan", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.ChangePlan(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "success")
}

// ==================== CancelScheduledChange Tests ====================

func TestCancelScheduledChangeNoSubscription(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.CancelScheduledChangeFunc = func(userID int) error {
		return payment.ErrScheduleNotFound
	}

	req := httptest.NewRequest(http.MethodPost, "/cancel-scheduled-change", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CancelScheduledChange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancelScheduledChangeServiceError(t *testing.T) {
	handler, e, paymentSvc, _ := setupTestHandler()
	paymentSvc.CancelScheduledChangeFunc = func(userID int) error {
		return errors.New("unexpected error")
	}

	req := httptest.NewRequest(http.MethodPost, "/cancel-scheduled-change", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CancelScheduledChange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCancelScheduledChangeSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/cancel-scheduled-change", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.CancelScheduledChange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== Webhook Tests ====================

func TestWebhookSuccess(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	body := `{"type": "checkout.session.completed"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Stripe-Signature", "test_signature")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Webhook(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"received":true`)
}

func TestWebhookHandlerError(t *testing.T) {
	handler, e, _, webhookH := setupTestHandler()
	webhookH.HandleWebhookFunc = func(payload []byte, signature string) *payment.WebhookResult {
		return &payment.WebhookResult{
			EventType: "unknown",
			Success:   false,
			Message:   "signature verification failed",
		}
	}

	body := `{"type": "checkout.session.completed"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Webhook(c)
	assert.NoError(t, err)
	// Webhook always returns 200
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"success":false`)
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _, _ := setupTestHandler()

	g := e.Group("/payments")
	handler.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/payments/webhook"])
	assert.True(t, routePaths["/payments/checkout"])
	assert.True(t, routePaths["/payments/portal"])
	assert.True(t, routePaths["/payments/status"])
}

// ==================== Error Mapping Tests ====================

func TestMapServiceErrorAllTypes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedMsg    string
	}{
		{"PlanNotFound", payment.ErrPlanNotFound, http.StatusNotFound, "Plan not found"},
		{"SamePlan", payment.ErrSamePlan, http.StatusBadRequest, "Already on the target plan"},
		{"InvalidPlanTransition", payment.ErrInvalidPlanTransition, http.StatusBadRequest, "Invalid plan transition"},
		{"DowngradeNotAllowed", payment.ErrDowngradeNotAllowed, http.StatusBadRequest, "Downgrade not allowed"},
		{"InvalidEntitySelection", payment.ErrInvalidEntitySelection, http.StatusBadRequest, "Invalid entity selection"},
		{"InvalidPrice", payment.ErrInvalidPrice, http.StatusBadRequest, "Invalid price configuration"},
		{"CheckoutSessionFailed", payment.ErrCheckoutSessionFailed, http.StatusBadRequest, "Checkout session creation failed"},
		{"NoStripeCustomer", payment.ErrNoStripeCustomer, http.StatusBadRequest, "No Stripe customer found"},
		{"NoStripeSubscription", payment.ErrNoStripeSubscription, http.StatusBadRequest, "No active Stripe subscription"},
		{"ScheduleNotFound", payment.ErrScheduleNotFound, http.StatusBadRequest, "No active subscription schedule"},
		{"SessionNotFound", payment.ErrSessionNotFound, http.StatusForbidden, "Session not found"},
		{"ProviderUnavailable", payment.ErrProviderUnavailable, http.StatusServiceUnavailable, "Payment provider unavailable"},
		{"UnknownError", errors.New("unknown"), http.StatusInternalServerError, "Internal server error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := mapServiceError(tc.err)
			assert.Equal(t, tc.expectedStatus, status)
			assert.Equal(t, tc.expectedMsg, msg)
		})
	}
}
