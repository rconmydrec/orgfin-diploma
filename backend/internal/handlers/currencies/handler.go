package currencies

import (
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service CurrencyService
	logger  *slog.Logger
}

func New(service CurrencyService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/", h.GetCurrencies)
}

type CurrencyResponse struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *Handler) GetCurrencies(c echo.Context) error {
	_ = c.Get("user_id").(int)

	currencies, err := h.service.GetAll()
	if err != nil {
		h.logger.Error("get currencies failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get currencies"})
	}

	response := make([]CurrencyResponse, len(currencies))
	for i, curr := range currencies {
		response[i] = CurrencyResponse{
			ID:   curr.ID,
			Code: curr.Code,
			Name: curr.Name,
		}
	}

	return c.JSON(http.StatusOK, response)
}
