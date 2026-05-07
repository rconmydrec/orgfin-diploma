package financial_planning

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	financialplanning "github.com/go-budget/backend/internal/services/financial_planning"
	"github.com/labstack/echo/v4"
)

// Handler is a thin HTTP adapter for the financial planning service.
type Handler struct {
	service FinancialPlanningService
	logger  *slog.Logger
}

// New creates a new financial planning Handler.
func New(service FinancialPlanningService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the financial planning routes.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/future-balance", h.CalculateFutureBalance)
	g.POST("/projection", h.GetBalanceProjection)
}

// DTOs

// FutureBalanceRequest is the request DTO for CalculateFutureBalance.
type FutureBalanceRequest struct {
	TargetDate      time.Time `json:"targetDate" validate:"required"`
	AccountIDs      []int     `json:"accountIds"`
	IncludeInactive bool      `json:"includeInactive"`
	IncludeHidden   bool      `json:"includeHidden"`
}

// AccountBalanceProjection is the response DTO for a single account projection.
type AccountBalanceProjection struct {
	AccountID            int     `json:"accountId"`
	AccountName          string  `json:"accountName"`
	CurrencyCode         string  `json:"currencyCode"`
	CurrentBalance       float64 `json:"currentBalance"`
	ProjectedBalance     float64 `json:"projectedBalance"`
	TotalPlannedIncome   float64 `json:"totalPlannedIncome"`
	TotalPlannedExpenses float64 `json:"totalPlannedExpenses"`
}

// FutureBalanceResponse is the response DTO for CalculateFutureBalance.
type FutureBalanceResponse struct {
	TargetDate            time.Time                  `json:"targetDate"`
	BaseCurrencyCode      string                     `json:"baseCurrencyCode"`
	TotalCurrentBalance   float64                    `json:"totalCurrentBalance"`
	TotalProjectedBalance float64                    `json:"totalProjectedBalance"`
	TotalPlannedIncome    float64                    `json:"totalPlannedIncome"`
	TotalPlannedExpenses  float64                    `json:"totalPlannedExpenses"`
	IncomeCount           int                        `json:"incomeCount"`
	ExpensesCount         int                        `json:"expensesCount"`
	Accounts              []AccountBalanceProjection `json:"accounts"`
}

// BalanceProjectionRequest is the request DTO for GetBalanceProjection.
type BalanceProjectionRequest struct {
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate" validate:"required"`
	Period          string    `json:"period"` // daily, weekly, monthly
	AccountIDs      []int     `json:"accountIds"`
	IncludeInactive bool      `json:"includeInactive"`
	IncludeHidden   bool      `json:"includeHidden"`
}

// BalanceProjectionPoint is the response DTO for a single projection point.
type BalanceProjectionPoint struct {
	Date     time.Time `json:"date"`
	Balance  float64   `json:"balance"`
	Income   float64   `json:"income"`
	Expenses float64   `json:"expenses"`
}

// BalanceProjectionResponse is the response DTO for GetBalanceProjection.
type BalanceProjectionResponse struct {
	StartDate        time.Time                `json:"startDate"`
	EndDate          time.Time                `json:"endDate"`
	Period           string                   `json:"period"`
	BaseCurrencyCode string                   `json:"baseCurrencyCode"`
	ProjectionPoints []BalanceProjectionPoint `json:"projectionPoints"`
}

// CalculateFutureBalance handles POST /financial-planning/future-balance.
func (h *Handler) CalculateFutureBalance(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req FutureBalanceRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Calculating future balance",
		"user_id", userID,
		"target_date", req.TargetDate,
		"account_ids", req.AccountIDs,
	)

	params := financialplanning.FutureBalanceParams{
		TargetDate:      req.TargetDate,
		AccountIDs:      req.AccountIDs,
		IncludeInactive: req.IncludeInactive,
		BaseCurrencyID:  baseCurrencyID,
		IncludeHidden:   req.IncludeHidden,
	}

	result, err := h.service.CalculateFutureBalance(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Error calculating future balance")
	}

	return c.JSON(http.StatusOK, FutureBalanceResponse{
		TargetDate:            result.TargetDate,
		BaseCurrencyCode:      result.BaseCurrencyCode,
		TotalCurrentBalance:   result.TotalCurrentBalance,
		TotalProjectedBalance: result.TotalProjectedBalance,
		TotalPlannedIncome:    result.TotalPlannedIncome,
		TotalPlannedExpenses:  result.TotalPlannedExpenses,
		IncomeCount:           result.IncomeCount,
		ExpensesCount:         result.ExpensesCount,
		Accounts:              mapAccountProjections(result.Accounts),
	})
}

// GetBalanceProjection handles POST /financial-planning/projection.
func (h *Handler) GetBalanceProjection(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req BalanceProjectionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	h.logger.Info("Getting balance projection",
		"user_id", userID,
		"start_date", req.StartDate,
		"end_date", req.EndDate,
		"period", req.Period,
	)

	params := financialplanning.BalanceProjectionParams{
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Period:          req.Period,
		AccountIDs:      req.AccountIDs,
		IncludeInactive: req.IncludeInactive,
		BaseCurrencyID:  baseCurrencyID,
		IncludeHidden:   req.IncludeHidden,
	}

	result, err := h.service.GetBalanceProjection(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Error generating balance projection")
	}

	return c.JSON(http.StatusOK, BalanceProjectionResponse{
		StartDate:        result.StartDate,
		EndDate:          result.EndDate,
		Period:           result.Period,
		BaseCurrencyCode: result.BaseCurrencyCode,
		ProjectionPoints: mapProjectionPoints(result.ProjectionPoints),
	})
}

// handleServiceError maps service-layer errors to appropriate HTTP responses.
func (h *Handler) handleServiceError(c echo.Context, err error, context string) error {
	switch serviceerrors.GetKind(err) {
	case serviceerrors.InvalidInput:
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "End date must be after start date"})
	default:
		h.logger.Error(context, "error", err)
		// ErrNoExchangeRates (Other kind) and unexpected errors both return 500.
		// For ErrNoExchangeRates, use a specific message.
		if serviceerrors.IsServiceError(err) {
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Exchange rates unavailable"})
		}
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: context})
	}
}

// mapAccountProjections converts service-level AccountProjection slice to handler DTOs.
func mapAccountProjections(accounts []financialplanning.AccountProjection) []AccountBalanceProjection {
	result := make([]AccountBalanceProjection, len(accounts))
	for i, a := range accounts {
		result[i] = AccountBalanceProjection{
			AccountID:            a.AccountID,
			AccountName:          a.AccountName,
			CurrencyCode:         a.CurrencyCode,
			CurrentBalance:       a.CurrentBalance,
			ProjectedBalance:     a.ProjectedBalance,
			TotalPlannedIncome:   a.TotalPlannedIncome,
			TotalPlannedExpenses: a.TotalPlannedExpenses,
		}
	}
	return result
}

// mapProjectionPoints converts service-level ProjectionPoint slice to handler DTOs.
func mapProjectionPoints(points []financialplanning.ProjectionPoint) []BalanceProjectionPoint {
	result := make([]BalanceProjectionPoint, len(points))
	for i, p := range points {
		result[i] = BalanceProjectionPoint{
			Date:     p.Date,
			Balance:  p.Balance,
			Income:   p.Income,
			Expenses: p.Expenses,
		}
	}
	return result
}
