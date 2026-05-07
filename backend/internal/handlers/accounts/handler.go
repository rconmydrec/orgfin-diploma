package accounts

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	accountsservice "github.com/go-budget/backend/internal/services/accounts"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	accountsService AccountsService
	logger          *slog.Logger
}

func New(accountsService AccountsService, logger *slog.Logger) *Handler {
	return &Handler{
		accountsService: accountsService,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/", h.CreateAccount)
	g.GET("/", h.GetAccounts)
	g.GET("/types/", h.GetAccountTypes)
	g.GET("/:id", h.GetAccountDetails)
	g.PUT("/:id", h.UpdateAccount)
	g.DELETE("/:id", h.DeleteAccount)
	g.PUT("/set-archive-status", h.SetArchiveStatus)
	g.POST("/adjust-balance", h.AdjustBalance)
}

func (h *Handler) CreateAccount(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req CreateAccountRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)

		// Re-validate with a fresh validator to get structured field errors.
		validate := validator.New()
		if valErr := validate.Struct(req); valErr != nil {
			var ve validator.ValidationErrors
			if errors.As(valErr, &ve) {
				fields := make([]ValidationField, len(ve))
				for i, fe := range ve {
					fields[i] = ValidationField{
						Field:   fe.Field(),
						Message: fe.Tag(),
					}
				}
				return c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
					Detail: "Validation failed",
					Errors: fields,
				})
			}
		}

		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	// Default ShowInReports to true if not explicitly set
	showInReports := true
	if req.ShowInReports != nil {
		showInReports = *req.ShowInReports
	}

	input := accountsservice.CreateAccountInput{
		Name:           req.Name,
		CurrencyID:     req.CurrencyID,
		AccountTypeID:  req.AccountTypeID,
		InitialBalance: req.InitialBalance,
		Balance:        req.Balance,
		CreditLimit:    req.CreditLimit,
		IsHidden:       req.IsHidden,
		ShowInReports:  showInReports,
		OpeningDate:    req.OpeningDate,
		Comment:        req.Comment,
	}

	account, err := h.accountsService.CreateAccount(userID, input)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			msg := serviceerrors.Message(err)
			if msg == "invalid account type" {
				return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid account type"})
			}
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid currency"})
		case serviceerrors.LimitExceeded:
			return c.JSON(http.StatusPaymentRequired, common.ErrorResponse{Detail: "Account limit exceeded"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("create account failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to create account"})
		}
	}

	return c.JSON(http.StatusOK, h.toAccountResponse(account))
}

func (h *Handler) GetAccounts(c echo.Context) error {
	userID := c.Get("user_id").(int)

	includeHidden, _ := strconv.ParseBool(c.QueryParam("includeHidden"))
	includeArchived, _ := strconv.ParseBool(c.QueryParam("includeArchived"))
	archivedOnly, _ := strconv.ParseBool(c.QueryParam("archivedOnly"))

	input := accountsservice.GetAccountsInput{
		IncludeHidden:   includeHidden,
		IncludeArchived: includeArchived,
		ArchivedOnly:    archivedOnly,
	}

	accounts, err := h.accountsService.GetUserAccounts(userID, input)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid user"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get accounts failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get accounts"})
		}
	}

	response := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = h.toAccountResponse(acc)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAccountTypes(c echo.Context) error {
	types, err := h.accountsService.GetAccountTypes()
	if err != nil {
		h.logger.Error("get account types failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get account types"})
	}

	response := make([]AccountTypeDTO, len(types))
	for i, t := range types {
		response[i] = AccountTypeDTO{
			ID:       t.ID,
			TypeName: t.TypeName,
			IsCredit: t.IsCredit,
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetAccountDetails(c echo.Context) error {
	userID := c.Get("user_id").(int)

	accountID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid account ID"})
	}

	account, err := h.accountsService.GetAccountDetails(accountID, userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Account not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get account details failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get account"})
		}
	}

	return c.JSON(http.StatusOK, h.toAccountResponse(account))
}

func (h *Handler) UpdateAccount(c echo.Context) error {
	userID := c.Get("user_id").(int)

	accountID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid account ID"})
	}

	var req UpdateAccountRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	// Parse opening date from string
	var openingDate time.Time
	if req.OpeningDate != "" {
		var err error
		openingDate, err = dateutil.ParseDate(req.OpeningDate)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid opening_date format"})
		}
	}

	input := accountsservice.UpdateAccountInput{
		Name:           req.Name,
		CurrencyID:     req.CurrencyID,
		AccountTypeID:  req.AccountTypeID,
		InitialBalance: req.InitialBalance,
		CreditLimit:    req.CreditLimit,
		IsHidden:       req.IsHidden,
		ShowInReports:  req.ShowInReports,
		OpeningDate:    openingDate,
		Comment:        req.Comment,
	}

	account, err := h.accountsService.UpdateAccount(accountID, userID, input)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Account not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.InvalidInput:
			msg := serviceerrors.Message(err)
			if msg == "invalid account type" {
				return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid account type"})
			}
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid currency"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("update account failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to update account"})
		}
	}

	return c.JSON(http.StatusOK, h.toAccountResponse(account))
}

func (h *Handler) DeleteAccount(c echo.Context) error {
	userID := c.Get("user_id").(int)

	accountID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid account ID"})
	}

	if err := h.accountsService.DeleteAccount(accountID, userID); err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Account not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("delete account failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to delete account"})
		}
	}

	return c.JSON(http.StatusOK, common.DeleteResponse{Deleted: true})
}

func (h *Handler) SetArchiveStatus(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req ArchiveStatusRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	if err := h.accountsService.SetArchiveStatus(req.AccountID, req.IsArchived, userID); err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Account not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("set archive status failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to set archive status"})
		}
	}

	return c.JSON(http.StatusOK, true)
}

func (h *Handler) AdjustBalance(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req BalanceAdjustmentRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	tx, err := h.accountsService.AdjustBalance(req.AccountID, req.NewBalance, req.Notes, userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Account not found"})
		case serviceerrors.AccessDenied:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Access denied"})
		case serviceerrors.NoChange:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Balance unchanged"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("adjust balance failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to adjust balance"})
		}
	}

	return c.JSON(http.StatusOK, h.toTransactionResponse(tx))
}

func (h *Handler) toAccountResponse(acc *accountsservice.Account) AccountResponse {
	initialBalance, _ := acc.InitialBalance.Float64()
	balance, _ := acc.Balance.Float64()

	var creditLimit float64
	if acc.CreditLimit != nil {
		creditLimit, _ = acc.CreditLimit.Float64()
	}

	var balanceInBaseCurrency float64
	if acc.BalanceInBaseCurrency != nil {
		balanceInBaseCurrency, _ = acc.BalanceInBaseCurrency.Float64()
	}

	resp := AccountResponse{
		ID:                    acc.ID,
		UserID:                acc.UserID,
		Name:                  acc.Name,
		CurrencyID:            acc.CurrencyID,
		AccountTypeID:         acc.AccountTypeID,
		InitialBalance:        initialBalance,
		Balance:               balance,
		CreditLimit:           creditLimit,
		OpeningDate:           acc.OpeningDate,
		IsHidden:              acc.IsHidden,
		ShowInReports:         acc.ShowInReports,
		IsDeleted:             acc.IsDeleted,
		IsArchived:            acc.IsArchived,
		ArchivedAt:            acc.ArchivedAt,
		BalanceInBaseCurrency: balanceInBaseCurrency,
	}

	if acc.Comment != nil {
		resp.Comment = *acc.Comment
	}

	if acc.Currency != nil {
		resp.Currency = common.CurrencyDTO{
			ID:   acc.Currency.ID,
			Code: acc.Currency.Code,
			Name: acc.Currency.Name,
		}
	}

	if acc.AccountType != nil {
		resp.AccountType = AccountTypeDTO{
			ID:       acc.AccountType.ID,
			TypeName: acc.AccountType.TypeName,
			IsCredit: acc.AccountType.IsCredit,
		}
	}

	return resp
}

func (h *Handler) toTransactionResponse(tx *accountsservice.Transaction) TransactionResponse {
	resp := TransactionResponse{
		ID:                  tx.ID,
		UserID:              tx.UserID,
		AccountID:           tx.AccountID,
		CategoryID:          tx.CategoryID,
		Amount:              tx.Amount.InexactFloat64(),
		DateTime:            tx.DateTime,
		IsTransfer:          tx.IsTransfer,
		IsIncome:            tx.IsIncome,
		IsAdjustment:        tx.IsAdjustment,
		ExcludeFromReports:  tx.ExcludeFromReports,
		LinkedTransactionID: tx.LinkedTransactionID,
		Notes:               tx.Notes,
	}

	if tx.BaseCurrencyAmount != nil {
		v := tx.BaseCurrencyAmount.InexactFloat64()
		resp.BaseCurrencyAmount = &v
	}

	if tx.NewBalance != nil {
		v := tx.NewBalance.InexactFloat64()
		resp.NewBalance = &v
	}

	if tx.Label != nil {
		resp.Label = *tx.Label
	}

	return resp
}
