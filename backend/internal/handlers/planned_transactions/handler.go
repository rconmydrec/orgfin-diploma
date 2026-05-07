package planned_transactions

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	plannedtransactions "github.com/go-budget/backend/internal/services/planned_transactions"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

type Handler struct {
	service PlannedTransactionsService
	logger  *slog.Logger
}

func New(service PlannedTransactionsService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/", h.Create)
	g.GET("/", h.List)
	g.GET("/upcoming/occurrences", h.GetUpcoming)
	g.GET("/:id", h.GetByID)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// DTOs
type CreateRequest struct {
	AccountID      *int               `json:"accountId"`
	Amount         decimal.Decimal    `json:"amount" validate:"required"`
	Label          string             `json:"label"`
	Notes          *string            `json:"notes"`
	IsIncome       bool               `json:"isIncome"`
	PlannedDate    common.FlexDate    `json:"plannedDate" validate:"required"`
	IsRecurring    bool               `json:"isRecurring"`
	RecurrenceRule *RecurrenceRuleDTO `json:"recurrenceRule"`
}

type UpdateRequest struct {
	ID             int                `json:"id"`
	AccountID      *int               `json:"accountId"`
	Amount         decimal.Decimal    `json:"amount" validate:"required"`
	Label          string             `json:"label"`
	Notes          *string            `json:"notes"`
	IsIncome       bool               `json:"isIncome"`
	PlannedDate    common.FlexDate    `json:"plannedDate" validate:"required"`
	IsRecurring    bool               `json:"isRecurring"`
	RecurrenceRule *RecurrenceRuleDTO `json:"recurrenceRule"`
	IsActive       *bool              `json:"isActive"`
}

type RecurrenceRuleDTO struct {
	Frequency  string           `json:"frequency"`
	Interval   int              `json:"interval"`
	EndDate    *common.DateOnly `json:"endDate,omitempty"`
	Count      *int             `json:"count,omitempty"`
	DayOfMonth *int             `json:"dayOfMonth,omitempty"`
}

type PlannedTxResponse struct {
	ID                    int                `json:"id"`
	UserID                int                `json:"userId"`
	CurrencyID            int                `json:"currencyId"`
	Amount                float64            `json:"amount"`
	Label                 string             `json:"label"`
	Notes                 *string            `json:"notes"`
	IsIncome              bool               `json:"isIncome"`
	PlannedDate           time.Time          `json:"plannedDate"`
	IsRecurring           bool               `json:"isRecurring"`
	RecurrenceRule        *RecurrenceRuleDTO `json:"recurrenceRule"`
	IsExecuted            bool               `json:"isExecuted"`
	ExecutedTransactionID *int               `json:"executedTransactionId"`
	ExecutionDate         *time.Time         `json:"executionDate"`
	IsActive              bool               `json:"isActive"`
	CreatedAt             time.Time          `json:"createdAt"`
	UpdatedAt             time.Time          `json:"updatedAt"`
}

type OccurrenceResponse struct {
	PlannedTransactionID int       `json:"plannedTransactionId"`
	OccurrenceDate       time.Time `json:"occurrenceDate"`
	Amount               float64   `json:"amount"`
	IsIncome             bool      `json:"isIncome"`
	Label                string    `json:"label"`
	IsActive             bool      `json:"isActive"`
	IsRecurring          bool      `json:"isRecurring"`
}

func (h *Handler) Create(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error in Create", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := plannedtransactions.CreateParams{
		AccountID:      req.AccountID,
		Amount:         req.Amount,
		Label:          req.Label,
		Notes:          req.Notes,
		IsIncome:       req.IsIncome,
		PlannedDate:    req.PlannedDate.Time(),
		IsRecurring:    req.IsRecurring,
		RecurrenceRule: dtoToRuleParams(req.RecurrenceRule),
	}

	created, err := h.service.Create(userID, baseCurrencyID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to create planned transaction")
	}

	return c.JSON(http.StatusCreated, h.toResponse(created))
}

func (h *Handler) List(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var accountIDs []int
	if accountsParam := c.QueryParam("account_ids"); accountsParam != "" {
		for _, idStr := range strings.Split(accountsParam, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				accountIDs = append(accountIDs, id)
			}
		}
	}

	var fromDate, toDate *string
	if fd := c.QueryParam("from_date"); fd != "" {
		fromDate = &fd
	}
	if td := c.QueryParam("to_date"); td != "" {
		toDate = &td
	}

	filters := plannedtransactions.ListFilters{
		AccountIDs:      accountIDs,
		FromDate:        fromDate,
		ToDate:          toDate,
		IsRecurring:     c.QueryParam("is_recurring"),
		IsExecuted:      c.QueryParam("is_executed"),
		IsActive:        c.QueryParam("is_active"),
		IncludeInactive: c.QueryParam("include_inactive") == "true",
	}

	txs, err := h.service.List(userID, filters)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to get planned transactions")
	}

	response := make([]PlannedTxResponse, len(txs))
	for i, tx := range txs {
		response[i] = h.toResponse(tx)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetUpcoming(c echo.Context) error {
	userID := c.Get("user_id").(int)

	days := 30
	if daysParam := c.QueryParam("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil {
			days = d
		}
	}

	includeInactive := c.QueryParam("include_inactive") == "true"

	occurrences, err := h.service.GetUpcoming(userID, days, includeInactive)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to get upcoming occurrences")
	}

	response := make([]OccurrenceResponse, len(occurrences))
	for i, occ := range occurrences {
		response[i] = OccurrenceResponse{
			PlannedTransactionID: occ.PlannedTransactionID,
			OccurrenceDate:       occ.OccurrenceDate,
			Amount:               occ.Amount.InexactFloat64(),
			IsIncome:             occ.IsIncome,
			Label:                occ.Label,
			IsActive:             occ.IsActive,
			IsRecurring:          occ.IsRecurring,
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetByID(c echo.Context) error {
	userID := c.Get("user_id").(int)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid ID"})
	}

	tx, err := h.service.GetByID(userID, id)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to get planned transaction")
	}

	return c.JSON(http.StatusOK, h.toResponse(tx))
}

func (h *Handler) Update(c echo.Context) error {
	userID := c.Get("user_id").(int)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid ID"})
	}

	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error in Update", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := plannedtransactions.UpdateParams{
		Amount:         req.Amount,
		Label:          req.Label,
		Notes:          req.Notes,
		IsIncome:       req.IsIncome,
		PlannedDate:    req.PlannedDate.Time(),
		IsRecurring:    req.IsRecurring,
		RecurrenceRule: dtoToRuleParams(req.RecurrenceRule),
		IsActive:       req.IsActive,
	}

	updated, err := h.service.Update(userID, id, params)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to update planned transaction")
	}

	return c.JSON(http.StatusOK, h.toResponse(updated))
}

func (h *Handler) Delete(c echo.Context) error {
	userID := c.Get("user_id").(int)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid ID"})
	}

	if err := h.service.Delete(userID, id); err != nil {
		return h.handleServiceError(c, err, "Failed to delete planned transaction")
	}

	return c.JSON(http.StatusOK, common.DeleteResponse{Deleted: true})
}

// handleServiceError maps service-layer errors to HTTP responses.
func (h *Handler) handleServiceError(c echo.Context, err error, context string) error {
	switch serviceerrors.GetKind(err) {
	case serviceerrors.InvalidInput:
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Base currency not set"})
	case serviceerrors.NotFound:
		return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Planned transaction not found"})
	case serviceerrors.AccessDenied:
		return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
	default:
		h.logger.Error(context, "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: context})
	}
}

// dtoToRuleParams converts a RecurrenceRuleDTO to RecurrenceRuleParams for the service layer.
func dtoToRuleParams(dto *RecurrenceRuleDTO) *plannedtransactions.RecurrenceRuleParams {
	if dto == nil {
		return nil
	}
	var endDate *time.Time
	if dto.EndDate != nil && !dto.EndDate.Time.IsZero() {
		t := dto.EndDate.Time
		endDate = &t
	}
	return &plannedtransactions.RecurrenceRuleParams{
		Frequency:  dto.Frequency,
		Interval:   dto.Interval,
		EndDate:    endDate,
		Count:      dto.Count,
		DayOfMonth: dto.DayOfMonth,
	}
}

// ruleParamsToDTO converts RecurrenceRuleParams from the service back to a RecurrenceRuleDTO for the response.
func ruleParamsToDTO(params *plannedtransactions.RecurrenceRuleParams) *RecurrenceRuleDTO {
	if params == nil {
		return nil
	}
	var endDate *common.DateOnly
	if params.EndDate != nil {
		endDate = &common.DateOnly{Time: *params.EndDate}
	}
	return &RecurrenceRuleDTO{
		Frequency:  params.Frequency,
		Interval:   params.Interval,
		EndDate:    endDate,
		Count:      params.Count,
		DayOfMonth: params.DayOfMonth,
	}
}

func (h *Handler) toResponse(tx *plannedtransactions.PlannedTransaction) PlannedTxResponse {
	resp := PlannedTxResponse{
		ID:                    tx.ID,
		UserID:                tx.UserID,
		CurrencyID:            tx.CurrencyID,
		Amount:                tx.Amount.InexactFloat64(),
		Label:                 tx.Label,
		Notes:                 tx.Notes,
		IsIncome:              tx.IsIncome,
		PlannedDate:           tx.PlannedDate,
		IsRecurring:           tx.IsRecurring,
		IsExecuted:            tx.IsExecuted,
		ExecutedTransactionID: tx.ExecutedTransactionID,
		ExecutionDate:         tx.ExecutionDate,
		IsActive:              tx.IsActive,
		CreatedAt:             tx.CreatedAt,
		UpdatedAt:             tx.UpdatedAt,
	}

	if tx.RecurrenceRule != nil {
		resp.RecurrenceRule = ruleParamsToDTO(plannedtransactions.StorageRuleToDTO(*tx.RecurrenceRule))
	}

	return resp
}
