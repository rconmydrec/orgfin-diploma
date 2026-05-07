package budgets

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	budgetsservice "github.com/go-budget/backend/internal/services/budgets"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

type Handler struct {
	service BudgetsService
	logger  *slog.Logger
}

func New(service BudgetsService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/add/", h.CreateBudget)
	g.GET("/", h.GetBudgets)
	g.PUT("/:id/", h.UpdateBudget)
	g.DELETE("/:id/", h.DeleteBudget)
	g.PUT("/:id/archive/", h.ArchiveBudget)
}

// DTOs
type BudgetRequest struct {
	ID           *int            `json:"id"`
	Name         string          `json:"name" validate:"required"`
	CurrencyID   int             `json:"currencyId" validate:"required"`
	TargetAmount decimal.Decimal `json:"targetAmount" validate:"required"`
	Period       string          `json:"period" validate:"required"`
	Repeat       bool            `json:"repeat"`
	StartDate    string          `json:"startDate" validate:"required"`
	EndDate      string          `json:"endDate" validate:"required"`
	Categories   []int           `json:"categories"`
	Comment      *string         `json:"comment"`
}

type BudgetResponse struct {
	ID                 int                 `json:"id"`
	Name               string              `json:"name"`
	CurrencyID         int                 `json:"currencyId"`
	TargetAmount       float64             `json:"targetAmount"`
	CollectedAmount    float64             `json:"collectedAmount"`
	Period             string              `json:"period"`
	Repeat             bool                `json:"repeat"`
	StartDate          time.Time           `json:"startDate"`
	EndDate            time.Time           `json:"endDate"`
	IncludedCategories string              `json:"includedCategories"`
	IsArchived         bool                `json:"isArchived"`
	Comment            *string             `json:"comment"`
	Currency           *common.CurrencyDTO `json:"currency,omitempty"`
}

func (h *Handler) CreateBudget(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req BudgetRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	params := budgetsservice.CreateParams{
		Name:         req.Name,
		CurrencyID:   req.CurrencyID,
		TargetAmount: req.TargetAmount,
		Period:       req.Period,
		Repeat:       req.Repeat,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Categories:   req.Categories,
		Comment:      req.Comment,
	}

	created, err := h.service.Create(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to create budget")
	}

	return c.JSON(http.StatusOK, h.toBudgetResponse(created))
}

func (h *Handler) GetBudgets(c echo.Context) error {
	userID := c.Get("user_id").(int)

	include := c.QueryParam("include")

	budgets, err := h.service.GetByUserID(userID, include)
	if err != nil {
		h.logger.Error("get budgets failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get budgets"})
	}

	response := make([]BudgetResponse, len(budgets))
	for i, b := range budgets {
		response[i] = h.toBudgetResponse(b)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateBudget(c echo.Context) error {
	userID := c.Get("user_id").(int)

	budgetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid budget ID"})
	}

	var req BudgetRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	params := budgetsservice.UpdateParams{
		Name:         req.Name,
		CurrencyID:   req.CurrencyID,
		TargetAmount: req.TargetAmount,
		Period:       req.Period,
		Repeat:       req.Repeat,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Categories:   req.Categories,
		Comment:      req.Comment,
	}

	updated, err := h.service.Update(userID, budgetID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to update budget")
	}

	return c.JSON(http.StatusOK, h.toBudgetResponse(updated))
}

func (h *Handler) DeleteBudget(c echo.Context) error {
	userID := c.Get("user_id").(int)

	budgetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid budget ID"})
	}

	if err := h.service.Delete(userID, budgetID); err != nil {
		return h.handleServiceError(c, err, "Failed to delete budget")
	}

	return c.JSON(http.StatusOK, common.MessageResponse{Message: "Budget with id " + strconv.Itoa(budgetID) + " deleted"})
}

func (h *Handler) ArchiveBudget(c echo.Context) error {
	userID := c.Get("user_id").(int)

	budgetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid budget ID"})
	}

	if err := h.service.Archive(userID, budgetID); err != nil {
		return h.handleServiceError(c, err, "Failed to archive budget")
	}

	return c.JSON(http.StatusOK, common.MessageResponse{Message: "Budget with id " + strconv.Itoa(budgetID) + " archived"})
}

type DailyProcessingResponse struct {
	Message   string `json:"message"`
	Processed int    `json:"processed"`
}

func (h *Handler) DailyProcessing(c echo.Context) error {
	result, err := h.service.DailyProcessing()
	if err != nil {
		h.logger.Error("daily processing failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get outdated budgets"})
	}

	return c.JSON(http.StatusOK, DailyProcessingResponse{
		Message:   "Daily processing completed",
		Processed: result.Processed,
	})
}

func (h *Handler) handleServiceError(c echo.Context, err error, context string) error {
	switch serviceerrors.GetKind(err) {
	case serviceerrors.NotFound:
		return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Budget not found"})
	case serviceerrors.AccessDenied:
		return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
	case serviceerrors.InvalidInput:
		// Distinguish between InvalidInput errors by message.
		switch serviceerrors.Message(err) {
		case "invalid start date format":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid start date format"})
		case "invalid end date format":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid end date format"})
		default:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid currency"})
		}
	default:
		h.logger.Error(context, "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: context})
	}
}

func (h *Handler) toBudgetResponse(b *budgetsservice.Budget) BudgetResponse {
	resp := BudgetResponse{
		ID:                 b.ID,
		Name:               b.Name,
		CurrencyID:         b.CurrencyID,
		TargetAmount:       b.TargetAmount.InexactFloat64(),
		CollectedAmount:    b.CollectedAmount.InexactFloat64(),
		Period:             strings.ToLower(b.Period),
		Repeat:             b.Repeat,
		StartDate:          b.StartDate,
		EndDate:            b.EndDate.AddDate(0, 0, -1), // Subtract 1 day for display
		IncludedCategories: b.IncludedCategories,
		IsArchived:         b.IsArchived,
		Comment:            b.Comment,
	}

	if b.Currency != nil {
		resp.Currency = &common.CurrencyDTO{
			ID:   b.Currency.ID,
			Code: b.Currency.Code,
			Name: b.Currency.Name,
		}
	}

	return resp
}
