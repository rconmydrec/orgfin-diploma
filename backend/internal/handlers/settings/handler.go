package settings

import (
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	settingsservice "github.com/go-budget/backend/internal/services/settings"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service SettingsService
	logger  *slog.Logger
}

func New(
	service SettingsService,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/languages", h.GetLanguages)
	g.GET("/", h.GetSettings)
	g.POST("/", h.UpdateSettings)
	g.GET("/base-currency/", h.GetBaseCurrency)
	g.PUT("/base-currency/", h.UpdateBaseCurrency)
}

// DTOs
type LanguageResponse struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDeleted bool   `json:"isDeleted"`
}

type SettingsData struct {
	Language          string  `json:"language"`
	ProjectionEndDate *string `json:"projectionEndDate"`
	ProjectionPeriod  *string `json:"projectionPeriod"`
}

type SettingsResponse struct {
	ID        int          `json:"id"`
	UserID    int          `json:"userId"`
	Settings  SettingsData `json:"settings"`
	CreatedAt string       `json:"createdAt"`
	UpdatedAt string       `json:"updatedAt"`
}

type SettingsRequest struct {
	Language          string  `json:"language"`
	ProjectionEndDate *string `json:"projectionEndDate"`
	ProjectionPeriod  *string `json:"projectionPeriod"`
}

type BaseCurrencyRequest struct {
	CurrencyID int `json:"currencyId" validate:"required"`
}

func (h *Handler) GetLanguages(c echo.Context) error {
	languages, err := h.service.GetLanguages()
	if err != nil {
		h.logger.Error("get languages failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get languages"})
	}

	response := make([]LanguageResponse, len(languages))
	for i, lang := range languages {
		response[i] = h.toLanguageResponse(lang)
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) GetSettings(c echo.Context) error {
	userID := c.Get("user_id").(int)

	settings, err := h.service.GetSettings(userID)
	if err != nil {
		return h.handleServiceError(c, err, "get settings failed", "Failed to get settings")
	}

	return c.JSON(http.StatusOK, h.toSettingsResponse(settings))
}

func (h *Handler) UpdateSettings(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req SettingsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := settingsservice.UpdateParams{
		Language:          req.Language,
		ProjectionEndDate: req.ProjectionEndDate,
		ProjectionPeriod:  req.ProjectionPeriod,
	}

	settings, err := h.service.UpdateSettings(userID, params)
	if err != nil {
		if serviceerrors.IsServiceError(err) && serviceerrors.GetKind(err) == serviceerrors.Other {
			h.logger.Error("create default settings failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to create settings"})
		}
		return h.handleServiceError(c, err, "update settings failed", "Failed to update settings")
	}

	return c.JSON(http.StatusOK, h.toSettingsResponse(settings))
}

func (h *Handler) GetBaseCurrency(c echo.Context) error {
	userID := c.Get("user_id").(int)

	cur, err := h.service.GetBaseCurrency(userID)
	if err != nil {
		if serviceerrors.IsKind(err, serviceerrors.NotFound) {
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Base currency not set"})
		}
		h.logger.Error("get base currency failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get base currency"})
	}

	return c.JSON(http.StatusOK, common.CurrencyDTO{
		ID:   cur.ID,
		Code: cur.Code,
		Name: cur.Name,
	})
}

func (h *Handler) UpdateBaseCurrency(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req BaseCurrencyRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	cur, err := h.service.UpdateBaseCurrency(userID, req.CurrencyID)
	if err != nil {
		if serviceerrors.IsKind(err, serviceerrors.InvalidInput) {
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid currency"})
		}
		h.logger.Error("update base currency failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to update base currency"})
	}

	return c.JSON(http.StatusOK, common.CurrencyDTO{
		ID:   cur.ID,
		Code: cur.Code,
		Name: cur.Name,
	})
}

// handleServiceError maps service errors to HTTP responses for settings-related operations.
func (h *Handler) handleServiceError(c echo.Context, err error, logMsg, clientMsg string) error {
	h.logger.Error(logMsg, "error", err)
	return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: clientMsg})
}

// toSettingsResponse converts a service UserSettings to the handler response DTO.
func (h *Handler) toSettingsResponse(s *settingsservice.UserSettings) SettingsResponse {
	return SettingsResponse{
		ID:     s.ID,
		UserID: s.UserID,
		Settings: SettingsData{
			Language:          s.Settings.Language,
			ProjectionEndDate: s.Settings.ProjectionEndDate,
			ProjectionPeriod:  s.Settings.ProjectionPeriod,
		},
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

// toLanguageResponse converts a service Language to the handler response DTO.
func (h *Handler) toLanguageResponse(l *settingsservice.Language) LanguageResponse {
	return LanguageResponse{
		ID:        l.ID,
		Code:      l.Code,
		Name:      l.Name,
		IsDeleted: l.IsDeleted,
	}
}
