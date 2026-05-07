package subscriptions

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	"github.com/go-budget/backend/internal/services/subscription"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// SubscriptionServiceInterface defines the subscription service methods
// needed by this handler. It is satisfied by *subscription.Service.
type SubscriptionServiceInterface interface {
	GetStatus(userID int) (*subscription.SubscriptionStatus, error)
	GetPlans() ([]*subscription.SubscriptionPlan, error)
	Upgrade(userID int, planID int) (*subscription.UpgradeResult, error)
	ScheduleDowngrade(userID int, accountIDs []int, budgetID *int) (*subscription.DowngradeResult, error)
}

type Handler struct {
	subscriptionSvc SubscriptionServiceInterface
	logger          *slog.Logger
}

func New(
	subscriptionSvc SubscriptionServiceInterface,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		subscriptionSvc: subscriptionSvc,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group, middlewares ...echo.MiddlewareFunc) {
	// Public routes
	g.GET("/plans", h.GetPlans)

	// Protected routes
	protected := g.Group("")
	for _, mw := range middlewares {
		protected.Use(mw)
	}
	protected.GET("/status", h.GetStatus)
	protected.POST("/upgrade", h.Upgrade)
	protected.POST("/downgrade", h.Downgrade)
}

// DTOs
type PlanType string

const (
	PlanTypeFree    PlanType = "free"
	PlanTypeTrial   PlanType = "trial"
	PlanTypePremium PlanType = "premium"
)

type BillingPeriodResponse struct {
	ID           int    `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	DurationDays int    `json:"durationDays"`
}

type SubscriptionPlanResponse struct {
	ID             int                    `json:"id"`
	Name           string                 `json:"name"`
	TranslationKey *string                `json:"translationKey"`
	PlanType       PlanType               `json:"planType"`
	BillingPeriod  *BillingPeriodResponse `json:"billingPeriod"`
	Price          decimal.Decimal        `json:"price"`
	CurrencyCode   string                 `json:"currencyCode"`
	IsFeatured     bool                   `json:"isFeatured"`
	Description    *string                `json:"description"`
	SortOrder      int                    `json:"sortOrder"`
}

type SubscriptionStatusResponse struct {
	PlanType              PlanType       `json:"planType"`
	IsActive              bool           `json:"isActive"`
	PlanID                *int           `json:"planId"`
	TrialDaysRemaining    *int           `json:"trialDaysRemaining"`
	RequiresDowngrade     bool           `json:"requiresDowngrade"`
	Limits                map[string]int `json:"limits"`
	SubscribedAt          *time.Time     `json:"subscribedAt"`
	ExpiresAt             *time.Time     `json:"expiresAt"`
	PendingPlanID         *int           `json:"pendingPlanId"`
	PendingPlanName       *string        `json:"pendingPlanName"`
	HasStripeSubscription bool           `json:"hasStripeSubscription"`
}

type UpgradeRequest struct {
	PlanID        int              `json:"planId" validate:"required"`
	AdjustedPrice *decimal.Decimal `json:"adjustedPrice"`
}

type UpgradeResponse struct {
	Message      string      `json:"message"`
	Subscription interface{} `json:"subscription"`
	ExpiresAt    *time.Time  `json:"expiresAt"`
}

type DowngradeRequest struct {
	AccountIDs []int `json:"accountIds"`
	BudgetID   *int  `json:"budgetId"`
}

type DowngradeResponse struct {
	Message          string     `json:"message"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	PendingDowngrade bool       `json:"pendingDowngrade"`
	PendingPlanID    *int       `json:"pendingPlanId"`
}

// mapPlanType converts a raw plan type string to the PlanType enum.
func mapPlanType(raw string) PlanType {
	switch strings.ToLower(raw) {
	case "free":
		return PlanTypeFree
	case "trial":
		return PlanTypeTrial
	case "premium":
		return PlanTypePremium
	default:
		return PlanTypeFree
	}
}

// mapServiceError maps known service/business errors to appropriate HTTP responses.
func (h *Handler) mapServiceError(c echo.Context, err error) error {
	switch serviceerrors.GetKind(err) {
	case serviceerrors.NotFound:
		return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Plan not found"})
	case serviceerrors.Conflict:
		// ErrSamePlan or ErrAlreadyOnFreePlan
		msg := serviceerrors.Message(err)
		if msg == "already on free plan" {
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Already on free plan"})
		}
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Already on this plan"})
	case serviceerrors.InvalidInput:
		// ErrInvalidPlanTransition or ErrDowngradeNotAllowed
		msg := serviceerrors.Message(err)
		if msg == "downgrade not allowed via upgrade endpoint" {
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Downgrade not allowed via upgrade endpoint"})
		}
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid plan transition"})
	case serviceerrors.LimitExceeded:
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Entity selection violates plan limits"})
	default:
		h.logger.Error("subscription service error", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Internal server error"})
	}
}

func (h *Handler) GetPlans(c echo.Context) error {
	plans, err := h.subscriptionSvc.GetPlans()
	if err != nil {
		h.logger.Error("failed to get subscription plans", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get subscription plans"})
	}

	var response []SubscriptionPlanResponse
	for _, plan := range plans {
		planResp := SubscriptionPlanResponse{
			ID:             plan.ID,
			Name:           plan.Name,
			TranslationKey: plan.TranslationKey,
			PlanType:       mapPlanType(plan.PlanType),
			Price:          plan.Price,
			CurrencyCode:   plan.CurrencyCode,
			IsFeatured:     plan.IsFeatured,
			Description:    plan.Description,
			SortOrder:      plan.SortOrder,
		}

		if plan.BillingPeriod != nil {
			planResp.BillingPeriod = &BillingPeriodResponse{
				ID:           plan.BillingPeriod.ID,
				Code:         plan.BillingPeriod.Code,
				Name:         plan.BillingPeriod.Name,
				DurationDays: plan.BillingPeriod.DurationDays,
			}
		}

		response = append(response, planResp)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetStatus(c echo.Context) error {
	userID := c.Get("user_id").(int)

	status, err := h.subscriptionSvc.GetStatus(userID)
	if err != nil {
		return h.mapServiceError(c, err)
	}

	// Convert service Limits to map[string]int for the response.
	var limits map[string]int
	if status.Limits != nil {
		limits = map[string]int{
			"accounts":     status.Limits.Accounts,
			"budgets":      status.Limits.Budgets,
			"planningDays": status.Limits.PlanningDays,
		}
	}

	// PlanID: the service returns 0 for default free status (no subscription record).
	// The old handler returned nil for PlanID when there was no subscription.
	var planID *int
	if status.PlanID != 0 {
		planID = &status.PlanID
	}

	response := SubscriptionStatusResponse{
		PlanType:              mapPlanType(status.PlanType),
		IsActive:              status.IsActive,
		PlanID:                planID,
		TrialDaysRemaining:    status.TrialDaysRemaining,
		RequiresDowngrade:     status.RequiresDowngrade,
		Limits:                limits,
		SubscribedAt:          status.SubscribedAt,
		ExpiresAt:             status.ExpiresAt,
		PendingPlanID:         status.PendingPlanID,
		PendingPlanName:       status.PendingPlanName,
		HasStripeSubscription: status.HasStripeSubscription,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) Upgrade(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req UpgradeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Subscription upgrade requested",
		"user_id", userID,
		"plan_id", req.PlanID,
	)

	result, err := h.subscriptionSvc.Upgrade(userID, req.PlanID)
	if err != nil {
		return h.mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, UpgradeResponse{
		Message:      result.Message,
		Subscription: result.Subscription,
		ExpiresAt:    result.Subscription.ExpiresAt,
	})
}

func (h *Handler) Downgrade(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req DowngradeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Subscription downgrade requested",
		"user_id", userID,
		"account_ids", req.AccountIDs,
		"budget_id", req.BudgetID,
	)

	result, err := h.subscriptionSvc.ScheduleDowngrade(userID, req.AccountIDs, req.BudgetID)
	if err != nil {
		return h.mapServiceError(c, err)
	}

	return c.JSON(http.StatusOK, DowngradeResponse{
		Message:          result.Message,
		ExpiresAt:        result.ExpiresAt,
		PendingDowngrade: result.Scheduled,
		PendingPlanID:    nil, // The service DowngradeResult does not expose the pending plan ID
	})
}
