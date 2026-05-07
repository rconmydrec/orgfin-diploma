package exchange_rates

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/workers/tasks"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service  ExchangeRatesService
	enqueuer tasks.TaskEnqueuer
	logger   *slog.Logger
}

func New(service ExchangeRatesService, enqueuer tasks.TaskEnqueuer, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		enqueuer: enqueuer,
		logger:   logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/", h.GetRates)
}

func (h *Handler) GetRates(c echo.Context) error {
	_ = c.Get("user_id").(int)

	today := time.Now().Format("2006-01-02")
	rates, err := h.service.GetRatesForDate(today)
	if err != nil {
		h.logger.Error("failed to get exchange rates", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Unable to get exchange rates"})
	}

	return c.JSON(http.StatusOK, toExchangeRateListResponse(rates))
}

func (h *Handler) UpdateRates(c echo.Context) error {
	h.logger.Info("Exchange rates update requested via endpoint")

	task := tasks.NewExchangeRateUpdateTask()
	if _, err := h.enqueuer.Enqueue(task); err != nil {
		h.logger.Error("failed to enqueue exchange rate update task", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to start exchange rate update"})
	}

	return c.JSON(http.StatusOK, common.MessageResponse{Message: "Exchange rate update task enqueued"})
}

// UpdateRatesRangeResponse is the response for the date-range update endpoint.
type UpdateRatesRangeResponse struct {
	DatesProcessed int    `json:"datesProcessed"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate"`
}

func (h *Handler) UpdateRatesRange(c echo.Context) error {
	startDateStr := c.Param("start_date")
	endDateStr := c.Param("end_date")

	h.logger.Info("Exchange rates range update requested", "start_date", startDateStr, "end_date", endDateStr)

	result, err := h.service.FetchAndSaveRatesRange(startDateStr, endDateStr)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, UpdateRatesRangeResponse{
		DatesProcessed: result.DatesProcessed,
		StartDate:      result.StartDate,
		EndDate:        result.EndDate,
	})
}

func (h *Handler) handleServiceError(c echo.Context, err error) error {
	// Check for DateError first (structured error, not a sentinel).
	var dateErr *exchangerates.DateError
	if errors.As(err, &dateErr) {
		if dateErr.Operation == "fetch" {
			h.logger.Error("failed to fetch rates for date", "date", dateErr.Date, "error", dateErr.Err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Detail: fmt.Sprintf("Unable to fetch exchange rates for %s", dateErr.Date),
			})
		}
		h.logger.Error("failed to save rates for date", "date", dateErr.Date, "error", dateErr.Err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Detail: fmt.Sprintf("Unable to save exchange rates for %s", dateErr.Date),
		})
	}

	if serviceerrors.IsKind(err, serviceerrors.InvalidInput) {
		// Distinguish between InvalidInput errors by message.
		switch serviceerrors.Message(err) {
		case "end date must not be before start date":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "end_date must not be before start_date"})
		case "date range exceeds maximum allowed days":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Detail: fmt.Sprintf("Date range must not exceed %d days", exchangerates.MaxDateRangeDays),
			})
		case "invalid start_date format":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid start_date format"})
		case "invalid end_date format":
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid end_date format"})
		default:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid input"})
		}
	}

	h.logger.Error("exchange rate operation failed", "error", err)
	return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Exchange rate operation failed"})
}
