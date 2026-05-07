package transactions

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	txservice "github.com/go-budget/backend/internal/services/transactions"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	txService TransactionsService
	logger    *slog.Logger
}

func New(txService TransactionsService, logger *slog.Logger) *Handler {
	return &Handler{
		txService: txService,
		logger:    logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/", h.CreateTransaction)
	g.GET("/", h.GetTransactions)
	g.GET("/templates", h.GetTemplates)
	g.PUT("/templates", h.UpdateTemplate)
	g.DELETE("/templates/validate", h.ValidateTemplateIDs)
	g.DELETE("/templates", h.DeleteTemplates)
	g.GET("/:id", h.GetTransactionDetails)
	g.PUT("/", h.UpdateTransaction)
	g.DELETE("/:id", h.DeleteTransaction)
}

func (h *Handler) CreateTransaction(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req CreateTransactionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error",
			"handler", "transactions.CreateTransaction",
			"method", c.Request().Method,
			"path", c.Path(),
			"user_id", userID,
			"error", err,
		)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		resp, failedFields := mapValidationError(err)
		h.logger.Error("validation error",
			"handler", "transactions.CreateTransaction",
			"method", c.Request().Method,
			"path", c.Path(),
			"user_id", userID,
			"errorCode", resp.ErrorCode,
			"fields", failedFields,
		)
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	// Convert FlexInt to *int
	var categoryID *int
	if req.CategoryID != nil {
		categoryID = req.CategoryID.ToIntPtr()
	}

	input := txservice.CreateTransactionInput{
		AccountID:          req.AccountID,
		TargetAccountID:    req.TargetAccountID,
		CategoryID:         categoryID,
		Amount:             req.Amount,
		TargetAmount:       req.TargetAmount,
		Label:              req.Label,
		Notes:              req.Notes,
		DateTime:           req.DateTime,
		IsIncome:           req.IsIncome,
		IsTransfer:         req.IsTransfer,
		IsAdjustment:       req.IsAdjustment,
		ExcludeFromReports: req.ExcludeFromReports,
		IsTemplate:         req.IsTemplate,
	}

	tx, err := h.txService.CreateTransaction(userID, input)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			msg := serviceerrors.Message(err)
			switch msg {
			case "invalid user":
				return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
			case "invalid account":
				return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid account"})
			case "invalid category":
				return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid category"})
			default:
				return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Source and target accounts must be different"})
			}
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("create transaction failed",
				"handler", "transactions.CreateTransaction",
				"method", c.Request().Method,
				"path", c.Path(),
				"user_id", userID,
				"label_len", len([]rune(req.Label)),
				"notes_len", notesLen(req.Notes),
				"error", err,
			)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to create transaction"})
		}
	}

	return c.JSON(http.StatusOK, h.toTransactionResponse(tx))
}

func (h *Handler) GetTransactions(c echo.Context) error {
	userID := c.Get("user_id").(int)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	var types []string
	if typesParam := c.QueryParam("types"); typesParam != "" {
		types = strings.Split(typesParam, ",")
	}

	var accountIDs []int
	if accountsParam := c.QueryParam("accounts"); accountsParam != "" {
		for _, idStr := range strings.Split(accountsParam, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				accountIDs = append(accountIDs, id)
			}
		}
	}

	var categoryIDs []int
	if categoriesParam := c.QueryParam("categories"); categoriesParam != "" {
		for _, idStr := range strings.Split(categoriesParam, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				categoryIDs = append(categoryIDs, id)
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

	input := txservice.GetTransactionsInput{
		Page:        page,
		PerPage:     perPage,
		Types:       types,
		AccountIDs:  accountIDs,
		CategoryIDs: categoryIDs,
		FromDate:    fromDate,
		ToDate:      toDate,
	}

	txs, err := h.txService.GetTransactions(userID, input)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get transactions failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get transactions"})
		}
	}

	response := make([]TransactionResponse, len(txs))
	for i, tx := range txs {
		response[i] = h.toTransactionResponse(tx)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetTransactionDetails(c echo.Context) error {
	userID := c.Get("user_id").(int)

	txID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid transaction ID"})
	}

	tx, err := h.txService.GetTransactionDetails(txID, userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Transaction not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get transaction details failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get transaction"})
		}
	}

	return c.JSON(http.StatusOK, h.toTransactionResponse(tx))
}

func (h *Handler) UpdateTransaction(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req UpdateTransactionRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error",
			"handler", "transactions.UpdateTransaction",
			"method", c.Request().Method,
			"path", c.Path(),
			"user_id", userID,
			"error", err,
		)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		resp, failedFields := mapValidationError(err)
		h.logger.Error("validation error",
			"handler", "transactions.UpdateTransaction",
			"method", c.Request().Method,
			"path", c.Path(),
			"user_id", userID,
			"errorCode", resp.ErrorCode,
			"fields", failedFields,
		)
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	// Convert FlexInt to *int
	var categoryID *int
	if req.CategoryID != nil {
		categoryID = req.CategoryID.ToIntPtr()
	}

	input := txservice.CreateTransactionInput{
		AccountID:          req.AccountID,
		TargetAccountID:    req.TargetAccountID,
		CategoryID:         categoryID,
		Amount:             req.Amount,
		TargetAmount:       req.TargetAmount,
		Label:              req.Label,
		Notes:              req.Notes,
		DateTime:           req.DateTime,
		IsIncome:           req.IsIncome,
		IsTransfer:         req.IsTransfer,
		IsAdjustment:       req.IsAdjustment,
		ExcludeFromReports: req.ExcludeFromReports,
		IsTemplate:         req.IsTemplate,
	}

	tx, err := h.txService.UpdateTransaction(userID, input, req.ID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Transaction not found"})
		case serviceerrors.InvalidInput:
			msg := serviceerrors.Message(err)
			switch msg {
			case "invalid user":
				return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
			case "invalid account":
				return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid account"})
			default:
				return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Source and target accounts must be different"})
			}
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("update transaction failed",
				"handler", "transactions.UpdateTransaction",
				"method", c.Request().Method,
				"path", c.Path(),
				"user_id", userID,
				"label_len", len([]rune(req.Label)),
				"notes_len", notesLen(req.Notes),
				"error", err,
			)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to update transaction"})
		}
	}

	return c.JSON(http.StatusOK, h.toTransactionResponse(tx))
}

func (h *Handler) DeleteTransaction(c echo.Context) error {
	userID := c.Get("user_id").(int)

	txID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid transaction ID"})
	}

	tx, err := h.txService.DeleteTransaction(txID, userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Transaction not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("delete transaction failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to delete transaction"})
		}
	}

	return c.JSON(http.StatusOK, h.toTransactionResponse(tx))
}

func (h *Handler) GetTemplates(c echo.Context) error {
	userID := c.Get("user_id").(int)

	templates, err := h.txService.GetTemplates(userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get templates failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get templates"})
		}
	}

	response := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		response[i] = h.toTemplateResponse(t)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateTemplate(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	template, err := h.txService.UpdateTemplate(userID, req.ID, req.Label, req.CategoryID, req.TargetAccountID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Template not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("update template failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to update template"})
		}
	}

	return c.JSON(http.StatusOK, h.toTemplateResponse(template))
}

func (h *Handler) DeleteTemplates(c echo.Context) error {
	userID := c.Get("user_id").(int)

	idsParam := c.QueryParam("ids")
	if idsParam == "" {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "ids parameter required"})
	}

	var ids []int
	for _, idStr := range strings.Split(idsParam, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid IDs"})
	}

	templates, err := h.txService.DeleteTemplates(userID, ids)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("delete templates failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to delete templates"})
		}
	}

	response := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		response[i] = h.toTemplateResponse(t)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) ValidateTemplateIDs(c echo.Context) error {
	_ = c.Get("user_id").(int)

	idsParam := c.QueryParam("ids")
	if idsParam == "" {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid template IDs format"})
	}

	var ids []int
	for _, idStr := range strings.Split(idsParam, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid template IDs format"})
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid template IDs format"})
	}

	return c.JSON(http.StatusOK, ids)
}

func (h *Handler) toTransactionResponse(tx *txservice.Transaction) TransactionResponse {
	amount, _ := tx.Amount.Float64()

	var newBalance *float64
	if tx.NewBalance != nil {
		nb, _ := tx.NewBalance.Float64()
		newBalance = &nb
	}

	var baseCurrencyAmount *float64
	if tx.BaseCurrencyAmount != nil {
		bca, _ := tx.BaseCurrencyAmount.Float64()
		baseCurrencyAmount = &bca
	}

	resp := TransactionResponse{
		ID:                  tx.ID,
		AccountID:           tx.AccountID,
		TargetAccountID:     nil, // Not in DB, always null
		CategoryID:          tx.CategoryID,
		Amount:              amount,
		TargetAmount:        nil, // Not in DB, always null
		DateTime:            tx.DateTime,
		IsTransfer:          tx.IsTransfer,
		IsIncome:            tx.IsIncome,
		IsAdjustment:        tx.IsAdjustment,
		ExcludeFromReports:  tx.ExcludeFromReports,
		UserID:              tx.UserID,
		BaseCurrencyAmount:  baseCurrencyAmount,
		BaseCurrencyCode:    tx.BaseCurrencyCode,
		NewBalance:          newBalance,
		LinkedTransactionID: tx.LinkedTransactionID,
		Notes:               tx.Notes,
	}

	if tx.Label != nil {
		resp.Label = *tx.Label
	}

	if tx.User != nil {
		resp.User = &UserDTO{
			ID:        tx.User.ID,
			Email:     tx.User.Email,
			FirstName: tx.User.FirstName,
			LastName:  tx.User.LastName,
		}
	}

	if tx.Account != nil {
		resp.Account = h.toAccountDTO(tx.Account)
	}

	if tx.Category != nil {
		resp.Category = h.toCategoryDTO(tx.Category)
	}

	return resp
}

func (h *Handler) toAccountDTO(acc *txservice.TransactionAccount) *AccountDTO {
	balance, _ := acc.Balance.Float64()
	initialBalance, _ := acc.InitialBalance.Float64()

	var creditLimit float64
	if acc.CreditLimit != nil {
		creditLimit, _ = acc.CreditLimit.Float64()
	}

	var balanceInBaseCurrency float64
	if acc.BalanceInBaseCurrency != nil {
		balanceInBaseCurrency, _ = acc.BalanceInBaseCurrency.Float64()
	}

	dto := &AccountDTO{
		ID:                    acc.ID,
		UserID:                acc.UserID,
		Name:                  acc.Name,
		Balance:               balance,
		InitialBalance:        initialBalance,
		AccountTypeID:         acc.AccountTypeID,
		CurrencyID:            acc.CurrencyID,
		CreditLimit:           creditLimit,
		OpeningDate:           acc.OpeningDate,
		Comment:               acc.Comment,
		IsHidden:              acc.IsHidden,
		ShowInReports:         acc.ShowInReports,
		IsDeleted:             acc.IsDeleted,
		IsArchived:            acc.IsArchived,
		BalanceInBaseCurrency: balanceInBaseCurrency,
	}

	if acc.Currency != nil {
		dto.Currency = &CurrencyDTO{
			ID:   acc.Currency.ID,
			Code: acc.Currency.Code,
			Name: acc.Currency.Name,
		}
	}

	if acc.AccountType != nil {
		dto.AccountType = &AccountTypeDTO{
			ID:       acc.AccountType.ID,
			TypeName: acc.AccountType.TypeName,
			IsCredit: acc.AccountType.IsCredit,
		}
	}

	return dto
}

func (h *Handler) toCategoryDTO(cat *txservice.TransactionCategory) *CategoryDTO {
	dto := &CategoryDTO{
		ID:        cat.ID,
		UserID:    cat.UserID,
		Name:      cat.Name,
		ParentID:  cat.ParentID,
		IsIncome:  cat.IsIncome,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
		Children:  []*CategoryDTO{},
	}

	for _, child := range cat.Children {
		dto.Children = append(dto.Children, h.toCategoryDTO(child))
	}

	return dto
}

func (h *Handler) toTemplateResponse(t *txservice.TransactionTemplate) TemplateResponse {
	resp := TemplateResponse{
		ID:              t.ID,
		CategoryID:      t.CategoryID,
		TargetAccountID: t.TargetAccountID,
		Label:           t.Label,
	}

	if t.Category != nil {
		resp.Category = h.toCategoryDTO(t.Category)
	}

	if t.TargetAccount != nil {
		resp.TargetAccountName = &t.TargetAccount.Name
	}

	return resp
}
