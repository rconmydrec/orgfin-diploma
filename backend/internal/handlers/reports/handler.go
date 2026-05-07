package reports

import (
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	reportsservice "github.com/go-budget/backend/internal/services/reports"
	"github.com/labstack/echo/v4"
)

// Handler is a thin HTTP adapter for the reports service.
type Handler struct {
	service ReportsService
	logger  *slog.Logger
}

// New creates a new reports Handler.
func New(service ReportsService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all report routes on the given group.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/cashflow/", h.CashFlowReport)
	g.POST("/balance/", h.BalanceReport)
	g.POST("/balance/non-hidden/", h.BalanceReportNonHidden)
	g.POST("/expenses-by-categories/", h.ExpensesByCategories)
	g.GET("/diagram/:diagram_type/:start_date/:end_date", h.Diagram)
	g.POST("/expenses-data/", h.ExpensesData)
}

// DTOs

// CashFlowReportRequest is the request DTO for the cash flow report endpoint.
type CashFlowReportRequest struct {
	StartDate *common.DateOnly `json:"startDate"`
	EndDate   *common.DateOnly `json:"endDate"`
	Period    string           `json:"period" validate:"required"`
}

// CashFlowReportResponse is the response DTO for the cash flow report endpoint.
type CashFlowReportResponse struct {
	Currency      string             `json:"currency"`
	TotalIncome   map[string]float64 `json:"totalIncome"`
	TotalExpenses map[string]float64 `json:"totalExpenses"`
	NetFlow       map[string]float64 `json:"netFlow"`
}

// BalanceReportRequest is the request DTO for the balance report endpoint.
type BalanceReportRequest struct {
	AccountIDs  []int  `json:"accountIds"`
	BalanceDate string `json:"balanceDate" validate:"required"`
}

// BalanceReportResponse is the response DTO for the balance report endpoint.
type BalanceReportResponse struct {
	AccountID           int     `json:"accountId"`
	AccountName         string  `json:"accountName"`
	AccountTypeID       int     `json:"accountTypeId"`
	CurrencyCode        string  `json:"currencyCode"`
	Balance             float64 `json:"balance"`
	BaseCurrencyBalance float64 `json:"baseCurrencyBalance"`
	BaseCurrencyCode    string  `json:"baseCurrencyCode"`
	ReportDate          string  `json:"reportDate"`
}

// ExpensesReportRequest is the request DTO for expense report endpoints.
type ExpensesReportRequest struct {
	StartDate           string `json:"startDate" validate:"required"`
	EndDate             string `json:"endDate" validate:"required"`
	Categories          []int  `json:"categories"`
	HideEmptyCategories bool   `json:"hideEmptyCategories"`
}

// ExpensesReportItem is the response DTO for the expenses-by-categories endpoint.
type ExpensesReportItem struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	ParentID      *int    `json:"parentId"`
	ParentName    *string `json:"parentName"`
	TotalExpenses float64 `json:"totalExpenses"`
	CurrencyCode  *string `json:"currencyCode"`
	IsParent      bool    `json:"isParent"`
}

// DiagramResponse is the response DTO for the diagram endpoint.
type DiagramResponse struct {
	Labels   []string  `json:"labels"`
	Data     []float64 `json:"data"`
	Currency string    `json:"currency"`
}

// ExpenseDataItem is the response DTO for the expenses-data endpoint.
type ExpenseDataItem struct {
	Label        string  `json:"label"`
	Amount       float64 `json:"amount"`
	CategoryID   int     `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
}

// handleServiceError maps service errors to HTTP responses.
func (h *Handler) handleServiceError(c echo.Context, err error, context string) error {
	if serviceerrors.IsServiceError(err) {
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Exchange rates unavailable, cannot generate report"})
	}
	h.logger.Error(context, "error", err)
	return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Error generating report"})
}

// CashFlowReport handles POST /reports/cashflow/
func (h *Handler) CashFlowReport(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req CashFlowReportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	// Map DTO to service params
	params := reportsservice.CashFlowParams{
		Period:         req.Period,
		BaseCurrencyID: baseCurrencyID,
	}
	if req.StartDate != nil && !req.StartDate.IsZero() {
		t := req.StartDate.Time
		params.StartDate = &t
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		t := req.EndDate.Time
		params.EndDate = &t
	}

	result, err := h.service.CashFlowReport(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "cash flow report error")
	}

	return c.JSON(http.StatusOK, CashFlowReportResponse{
		Currency:      result.CurrencyCode,
		TotalIncome:   result.TotalIncome,
		TotalExpenses: result.TotalExpenses,
		NetFlow:       result.NetFlow,
	})
}

// BalanceReport handles POST /reports/balance/
func (h *Handler) BalanceReport(c echo.Context) error {
	var req BalanceReportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	return h.generateBalanceResponse(c, req, false)
}

// BalanceReportNonHidden handles POST /reports/balance/non-hidden/
func (h *Handler) BalanceReportNonHidden(c echo.Context) error {
	var req BalanceReportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	return h.generateBalanceResponse(c, req, true)
}

// generateBalanceResponse maps the balance request to service params, calls the
// service, and maps the result to the HTTP response.
func (h *Handler) generateBalanceResponse(c echo.Context, req BalanceReportRequest, excludeHidden bool) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	params := reportsservice.BalanceParams{
		AccountIDs:     req.AccountIDs,
		BalanceDate:    req.BalanceDate,
		ExcludeHidden:  excludeHidden,
		BaseCurrencyID: baseCurrencyID,
	}

	items, err := h.service.BalanceReport(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "balance report error")
	}

	// Map service result to response DTOs
	response := make([]BalanceReportResponse, 0, len(items))
	for _, item := range items {
		response = append(response, BalanceReportResponse{
			AccountID:           item.AccountID,
			AccountName:         item.AccountName,
			AccountTypeID:       item.AccountTypeID,
			CurrencyCode:        item.CurrencyCode,
			Balance:             item.Balance,
			BaseCurrencyBalance: item.BaseCurrencyBalance,
			BaseCurrencyCode:    item.BaseCurrencyCode,
			ReportDate:          item.ReportDate,
		})
	}

	return c.JSON(http.StatusOK, response)
}

// ExpensesByCategories handles POST /reports/expenses-by-categories/
func (h *Handler) ExpensesByCategories(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req ExpensesReportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := reportsservice.ExpensesParams{
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		Categories:          req.Categories,
		HideEmptyCategories: req.HideEmptyCategories,
		BaseCurrencyID:      baseCurrencyID,
	}

	items, err := h.service.ExpensesByCategories(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "expenses by categories error")
	}

	// Map service result to response DTOs
	response := make([]ExpensesReportItem, 0, len(items))
	for _, item := range items {
		currCode := item.CurrencyCode
		response = append(response, ExpensesReportItem{
			ID:            item.ID,
			Name:          item.Name,
			ParentID:      item.ParentID,
			ParentName:    item.ParentName,
			TotalExpenses: item.TotalExpenses,
			CurrencyCode:  &currCode,
			IsParent:      item.IsParent,
		})
	}

	return c.JSON(http.StatusOK, response)
}

// Diagram handles GET /reports/diagram/:diagram_type/:start_date/:end_date
func (h *Handler) Diagram(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	diagramType := c.Param("diagram_type")
	startDate := c.Param("start_date")
	endDate := c.Param("end_date")

	params := reportsservice.DiagramParams{
		StartDate:      startDate,
		EndDate:        endDate,
		DiagramType:    diagramType,
		BaseCurrencyID: baseCurrencyID,
	}

	result, err := h.service.DiagramData(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "diagram error")
	}

	return c.JSON(http.StatusOK, DiagramResponse{
		Labels:   result.Labels,
		Data:     result.Data,
		Currency: result.CurrencyCode,
	})
}

// ExpensesData handles POST /reports/expenses-data/
func (h *Handler) ExpensesData(c echo.Context) error {
	userID := common.ActiveUserID(c)
	baseCurrencyID := common.ActiveUserBaseCurrencyID(c)

	var req ExpensesReportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := reportsservice.ExpensesParams{
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		Categories:          req.Categories,
		HideEmptyCategories: req.HideEmptyCategories,
		BaseCurrencyID:      baseCurrencyID,
	}

	entries, err := h.service.ExpensesData(userID, params)
	if err != nil {
		return h.handleServiceError(c, err, "expenses data error")
	}

	// Map service result to response DTOs
	response := make([]ExpenseDataItem, 0, len(entries))
	for _, entry := range entries {
		response = append(response, ExpenseDataItem{
			Label:        entry.Label,
			Amount:       entry.Amount,
			CategoryID:   entry.CategoryID,
			CategoryName: entry.CategoryName,
		})
	}

	return c.JSON(http.StatusOK, response)
}
