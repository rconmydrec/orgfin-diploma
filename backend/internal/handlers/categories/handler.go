package categories

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	categoriesservice "github.com/go-budget/backend/internal/services/categories"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service CategoriesService
	logger  *slog.Logger
}

func New(service CategoriesService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/", h.GetCategories)
	g.GET("/grouped/", h.GetGroupedCategories)
	g.POST("/", h.CreateCategory)
	g.PUT("/:id/", h.UpdateCategory)
	g.DELETE("/:id/", h.DeleteCategory)
}

// Request/Response DTOs
type CategoryRequest struct {
	ID       *int   `json:"id"`
	Name     string `json:"name" validate:"required"`
	ParentID *int   `json:"parentId"`
	IsIncome bool   `json:"isIncome"`
}

type CategoryResponse struct {
	ID        int                 `json:"id"`
	UserID    int                 `json:"userId"`
	Name      string              `json:"name"`
	ParentID  *int                `json:"parentId"`
	IsIncome  bool                `json:"isIncome"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Children  []*CategoryResponse `json:"children"`
}

type GroupedCategoriesResponse struct {
	Income   []*CategoryResponse `json:"income"`
	Expenses []*CategoryResponse `json:"expenses"`
}

func (h *Handler) GetCategories(c echo.Context) error {
	userID := c.Get("user_id").(int)

	flatCategories, err := h.service.GetFlat(userID)
	if err != nil {
		h.logger.Error("get categories failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get categories"})
	}

	result := make([]*CategoryResponse, len(flatCategories))
	for i, fc := range flatCategories {
		result[i] = &CategoryResponse{
			ID:        fc.ID,
			UserID:    fc.UserID,
			Name:      fc.Name,
			ParentID:  fc.ParentID,
			IsIncome:  fc.IsIncome,
			CreatedAt: fc.CreatedAt,
			UpdatedAt: fc.UpdatedAt,
			Children:  []*CategoryResponse{},
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (h *Handler) GetGroupedCategories(c echo.Context) error {
	userID := c.Get("user_id").(int)

	income, expenses, err := h.service.GetGrouped(userID)
	if err != nil {
		h.logger.Error("get grouped categories failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get categories"})
	}

	incomeResp := make([]*CategoryResponse, len(income))
	for i, cat := range income {
		incomeResp[i] = h.toCategoryResponse(cat)
	}
	expensesResp := make([]*CategoryResponse, len(expenses))
	for i, cat := range expenses {
		expensesResp[i] = h.toCategoryResponse(cat)
	}

	return c.JSON(http.StatusOK, GroupedCategoriesResponse{
		Income:   incomeResp,
		Expenses: expensesResp,
	})
}

func (h *Handler) CreateCategory(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req CategoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	params := categoriesservice.CreateParams{
		Name:     req.Name,
		ParentID: req.ParentID,
		IsIncome: req.IsIncome,
	}

	created, err := h.service.Create(userID, params)
	if err != nil {
		h.logger.Error("create category failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to create category"})
	}

	return c.JSON(http.StatusCreated, h.toCategoryResponse(created))
}

func (h *Handler) UpdateCategory(c echo.Context) error {
	userID := c.Get("user_id").(int)

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid category ID"})
	}

	var req CategoryRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validate error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	params := categoriesservice.UpdateParams{
		Name:     req.Name,
		ParentID: req.ParentID,
		IsIncome: req.IsIncome,
	}

	updated, err := h.service.Update(userID, categoryID, params)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to update category")
	}

	return c.JSON(http.StatusOK, h.toCategoryResponse(updated))
}

func (h *Handler) DeleteCategory(c echo.Context) error {
	userID := c.Get("user_id").(int)

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid category ID"})
	}

	deleted, err := h.service.Delete(userID, categoryID)
	if err != nil {
		return h.handleServiceError(c, err, "Failed to delete category")
	}

	return c.JSON(http.StatusOK, h.toCategoryResponse(deleted))
}

func (h *Handler) handleServiceError(c echo.Context, err error, context string) error {
	switch serviceerrors.GetKind(err) {
	case serviceerrors.NotFound:
		return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Category not found"})
	case serviceerrors.AccessDenied:
		return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
	default:
		h.logger.Error(context, "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: context})
	}
}

func (h *Handler) toCategoryResponse(cat *categoriesservice.Category) *CategoryResponse {
	resp := &CategoryResponse{
		ID:        cat.ID,
		UserID:    cat.UserID,
		Name:      cat.Name,
		ParentID:  cat.ParentID,
		IsIncome:  cat.IsIncome,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
		Children:  []*CategoryResponse{},
	}

	for _, child := range cat.Children {
		resp.Children = append(resp.Children, h.toCategoryResponse(child))
	}

	return resp
}
