package payments

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	"github.com/go-budget/backend/internal/services/payment"
	"github.com/labstack/echo/v4"
)

// PaymentServiceInterface defines the business operations that the handler
// delegates to. This allows the handler to remain a thin HTTP adapter.
type PaymentServiceInterface interface {
	CreateCheckout(userID int, planID int, email string, name string) (*payment.CheckoutResult, error)
	CreatePortal(userID int, returnURL *string) (*payment.PortalResult, error)
	GetPaymentStatus(userID int) (*payment.PaymentStatus, error)
	GetSessionStatus(userID int, sessionID string) (*payment.SessionStatus, error)
	GetUpgradePrice(userID int, targetPlanID int) (*payment.UpgradePriceInfo, error)
	ChangePlan(userID int, targetPlanID int) (*payment.ChangePlanResult, error)
	CancelScheduledChange(userID int) error
}

// WebhookHandlerInterface defines the contract for processing incoming
// webhook events from a payment provider.
type WebhookHandlerInterface interface {
	HandleWebhook(payload []byte, signature string) *payment.WebhookResult
}

type Handler struct {
	paymentSvc     PaymentServiceInterface
	webhookHandler WebhookHandlerInterface
	logger         *slog.Logger
}

func New(
	paymentSvc PaymentServiceInterface,
	webhookHandler WebhookHandlerInterface,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		paymentSvc:     paymentSvc,
		webhookHandler: webhookHandler,
		logger:         logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group, middlewares ...echo.MiddlewareFunc) {
	// Webhook - no auth, uses Stripe signature
	g.POST("/webhook", h.Webhook)

	// Protected routes
	protected := g.Group("")
	for _, mw := range middlewares {
		protected.Use(mw)
	}
	protected.POST("/checkout", h.CreateCheckout)
	protected.POST("/portal", h.CreatePortal)
	protected.GET("/status", h.GetStatus)
	protected.GET("/session/:session_id", h.GetSessionStatus)
	protected.GET("/upgrade-price/:plan_id", h.GetUpgradePrice)
	protected.POST("/change-plan", h.ChangePlan)
	protected.POST("/cancel-scheduled-change", h.CancelScheduledChange)
}

// DTOs
type CheckoutRequest struct {
	PlanID int `json:"planId" validate:"required"`
}

type CheckoutResponse struct {
	CheckoutURL            string `json:"checkoutUrl"`
	SessionID              string `json:"sessionId"`
	CreditAppliedCents     int64  `json:"creditAppliedCents"`
	CreditAppliedFormatted string `json:"creditAppliedFormatted"`
}

type PortalRequest struct {
	ReturnURL *string `json:"returnUrl"`
}

type PortalResponse struct {
	PortalURL string `json:"portalUrl"`
}

type PaymentStatusResponse struct {
	HasStripeCustomer        bool    `json:"hasStripeCustomer"`
	StripeSubscriptionActive bool    `json:"stripeSubscriptionActive"`
	StripeSubscriptionID     *string `json:"stripeSubscriptionId"`
	CurrentPlanType          string  `json:"currentPlanType"`
	LastPaymentFailed        bool    `json:"lastPaymentFailed"`
}

type CheckoutSessionStatus struct {
	SessionID      string `json:"sessionId"`
	Status         string `json:"status"`
	PaymentStatus  string `json:"paymentStatus"`
	CustomerID     string `json:"customerId"`
	SubscriptionID string `json:"subscriptionId"`
	AmountTotal    int64  `json:"amountTotal"`
	Currency       string `json:"currency"`
}

type UpgradePriceInfo struct {
	PlanID                 int    `json:"planId"`
	PlanName               string `json:"planName"`
	PlanType               string `json:"planType"`
	OriginalPriceCents     int64  `json:"originalPriceCents"`
	CreditCents            int64  `json:"creditCents"`
	FinalPriceCents        int64  `json:"finalPriceCents"`
	OriginalPriceFormatted string `json:"originalPriceFormatted"`
	CreditFormatted        string `json:"creditFormatted"`
	FinalPriceFormatted    string `json:"finalPriceFormatted"`
	CurrencyCode           string `json:"currencyCode"`
}

type ChangePlanRequest struct {
	PlanID int `json:"planId" validate:"required"`
}

type ChangePlanResponse struct {
	Success       bool    `json:"success"`
	IsUpgrade     bool    `json:"isUpgrade"`
	NewPlanName   string  `json:"newPlanName"`
	EffectiveDate string  `json:"effectiveDate"`
	Message       string  `json:"message"`
	ScheduleID    *string `json:"scheduleId"`
}

type WebhookResponse struct {
	Received  bool    `json:"received"`
	Success   bool    `json:"success"`
	EventType string  `json:"eventType"`
	Error     *string `json:"error"`
}

// mapServiceError translates a service-layer error into an HTTP status code
// and a user-friendly message. Returns (statusCode, message).
func mapServiceError(err error) (int, string) {
	kind := serviceerrors.GetKind(err)
	msg := serviceerrors.Message(err)

	switch kind {
	case serviceerrors.NotFound:
		switch msg {
		case "subscription plan not found":
			return http.StatusNotFound, "Plan not found"
		case "no active subscription schedule":
			return http.StatusBadRequest, "No active subscription schedule"
		case "checkout session not found":
			return http.StatusForbidden, "Session not found"
		default:
			return http.StatusNotFound, "Not found"
		}
	case serviceerrors.Conflict:
		switch msg {
		case "already on the target plan":
			return http.StatusBadRequest, "Already on the target plan"
		case "no stripe customer found":
			return http.StatusBadRequest, "No Stripe customer found"
		case "no active stripe subscription":
			return http.StatusBadRequest, "No active Stripe subscription"
		default:
			return http.StatusBadRequest, "Conflict"
		}
	case serviceerrors.InvalidInput:
		switch msg {
		case "invalid plan transition":
			return http.StatusBadRequest, "Invalid plan transition"
		case "downgrade not allowed via upgrade endpoint":
			return http.StatusBadRequest, "Downgrade not allowed"
		case "entity selection violates plan limits":
			return http.StatusBadRequest, "Invalid entity selection"
		case "invalid or non-existent price ID":
			return http.StatusBadRequest, "Invalid price configuration"
		default:
			return http.StatusBadRequest, "Invalid input"
		}
	case serviceerrors.ProviderError:
		switch msg {
		case "checkout session creation failed":
			return http.StatusBadRequest, "Checkout session creation failed"
		default:
			return http.StatusServiceUnavailable, "Payment provider unavailable"
		}
	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}

func (h *Handler) CreateCheckout(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req CheckoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Creating checkout session", "user_id", userID, "plan_id", req.PlanID)

	email := common.ActiveUserEmail(c)
	displayName := common.ActiveUserDisplayName(c)

	result, err := h.paymentSvc.CreateCheckout(userID, req.PlanID, email, displayName)
	if err != nil {
		h.logger.Error("failed to create checkout", "error", err, "user_id", userID, "plan_id", req.PlanID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, CheckoutResponse{
		CheckoutURL:            result.CheckoutURL,
		SessionID:              result.SessionID,
		CreditAppliedCents:     result.CreditAppliedCents,
		CreditAppliedFormatted: result.CreditAppliedFormatted,
	})
}

func (h *Handler) CreatePortal(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req PortalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Creating portal session", "user_id", userID)

	result, err := h.paymentSvc.CreatePortal(userID, req.ReturnURL)
	if err != nil {
		h.logger.Error("failed to create portal", "error", err, "user_id", userID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, PortalResponse{
		PortalURL: result.PortalURL,
	})
}

func (h *Handler) GetStatus(c echo.Context) error {
	userID := c.Get("user_id").(int)

	result, err := h.paymentSvc.GetPaymentStatus(userID)
	if err != nil {
		h.logger.Error("failed to get payment status", "error", err, "user_id", userID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, PaymentStatusResponse{
		HasStripeCustomer:        result.HasStripeCustomer,
		StripeSubscriptionActive: result.StripeSubscriptionActive,
		StripeSubscriptionID:     result.StripeSubscriptionID,
		CurrentPlanType:          result.CurrentPlanType,
		LastPaymentFailed:        result.LastPaymentFailed,
	})
}

func (h *Handler) GetSessionStatus(c echo.Context) error {
	userID := c.Get("user_id").(int)

	sessionID := c.Param("session_id")

	h.logger.Info("Getting checkout session status", "session_id", sessionID)

	result, err := h.paymentSvc.GetSessionStatus(userID, sessionID)
	if err != nil {
		h.logger.Error("failed to get session status", "error", err, "session_id", sessionID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, CheckoutSessionStatus{
		SessionID:      result.SessionID,
		Status:         result.Status,
		PaymentStatus:  result.PaymentStatus,
		CustomerID:     result.CustomerID,
		SubscriptionID: result.SubscriptionID,
		AmountTotal:    result.AmountTotal,
		Currency:       result.Currency,
	})
}

func (h *Handler) GetUpgradePrice(c echo.Context) error {
	userID := c.Get("user_id").(int)

	// Parse plan_id from path
	planIDStr := c.Param("plan_id")
	var planID int
	if _, err := fmt.Sscanf(planIDStr, "%d", &planID); err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid plan ID"})
	}

	h.logger.Info("Getting upgrade price info", "user_id", userID, "plan_id", planID)

	result, err := h.paymentSvc.GetUpgradePrice(userID, planID)
	if err != nil {
		h.logger.Error("failed to get upgrade price", "error", err, "user_id", userID, "plan_id", planID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, UpgradePriceInfo{
		PlanID:                 result.PlanID,
		PlanName:               result.PlanName,
		PlanType:               result.PlanType,
		OriginalPriceCents:     result.OriginalPriceCents,
		CreditCents:            result.CreditCents,
		FinalPriceCents:        result.FinalPriceCents,
		OriginalPriceFormatted: result.OriginalPriceFormatted,
		CreditFormatted:        result.CreditFormatted,
		FinalPriceFormatted:    result.FinalPriceFormatted,
		CurrencyCode:           result.CurrencyCode,
	})
}

func (h *Handler) ChangePlan(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req ChangePlanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Changing subscription plan", "user_id", userID, "plan_id", req.PlanID)

	result, err := h.paymentSvc.ChangePlan(userID, req.PlanID)
	if err != nil {
		h.logger.Error("failed to change plan", "error", err, "user_id", userID, "plan_id", req.PlanID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, ChangePlanResponse{
		Success:       result.Success,
		IsUpgrade:     result.IsUpgrade,
		NewPlanName:   result.NewPlanName,
		EffectiveDate: result.EffectiveDate,
		Message:       result.Message,
		ScheduleID:    result.ScheduleID,
	})
}

func (h *Handler) CancelScheduledChange(c echo.Context) error {
	userID := c.Get("user_id").(int)

	h.logger.Info("Canceling scheduled plan change", "user_id", userID)

	if err := h.paymentSvc.CancelScheduledChange(userID); err != nil {
		h.logger.Error("failed to cancel scheduled change", "error", err, "user_id", userID)
		status, msg := mapServiceError(err)
		return c.JSON(status, common.ErrorResponse{Detail: msg})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Scheduled change canceled",
	})
}

func (h *Handler) Webhook(c echo.Context) error {
	// Limit webhook body to 64KB to prevent memory exhaustion.
	const maxWebhookBodySize = 64 * 1024 // 64KB
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxWebhookBodySize))
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		return c.JSON(http.StatusOK, WebhookResponse{
			Received:  true,
			Success:   false,
			EventType: "unknown",
		})
	}

	// Get Stripe signature
	signature := c.Request().Header.Get("Stripe-Signature")

	h.logger.Info("Received Stripe webhook", "body_length", len(body))

	result := h.webhookHandler.HandleWebhook(body, signature)

	return c.JSON(http.StatusOK, WebhookResponse{
		Received:  true,
		Success:   result.Success,
		EventType: result.EventType,
	})
}
