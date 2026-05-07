package analytics

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/spending-trends", h.SpendingTrends)
	g.POST("/expense-categorization", h.ExpenseCategorization)
}

// DTOs
type AnalysisRequest struct {
	StartDate time.Time `json:"startDate" validate:"required"`
	EndDate   time.Time `json:"endDate" validate:"required"`
	Limit     *int      `json:"limit"`
}

type ExpenseCategorizationRequest struct {
	StartDate time.Time `json:"startDate" validate:"required"`
	EndDate   time.Time `json:"endDate" validate:"required"`
}

type AnalysisResponse struct {
	Analysis string `json:"analysis"`
}

func (h *Handler) SpendingTrends(c echo.Context) error {
	// TODO: think about better implementation
	// This endpoint is currently disabled in the Python version
	return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Not found"})
}

func (h *Handler) ExpenseCategorization(c echo.Context) error {
	// TODO: think about better implementation
	// This endpoint is currently disabled in the Python version
	return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Not found"})
}
