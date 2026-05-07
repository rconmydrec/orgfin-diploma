package export

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/workers"
	"github.com/go-budget/backend/internal/workers/tasks"
	"github.com/labstack/echo/v4"
)

// errResponseSent is a sentinel error indicating that an HTTP error response
// has already been written to the client. Handlers should return nil when they
// receive this error, since the response is already committed.
var errResponseSent = errors.New("response already sent")

// ExportService defines the interface for export operations.
type ExportService interface {
	GenerateExcel(userID int, startDate, endDate string) ([]byte, error)
}

// EmailService defines the interface for sending emails with attachments.
type EmailService interface {
	SendWithAttachment(to, subject, body string, attachment []byte, filename string) error
}

type Handler struct {
	exportService ExportService
	emailService  EmailService
	enqueuer      workers.TaskEnqueuer
	logger        *slog.Logger
	maxRangeDays  int
}

func New(
	exportService ExportService,
	emailService EmailService,
	enqueuer workers.TaskEnqueuer,
	logger *slog.Logger,
	maxRangeDays int,
) *Handler {
	return &Handler{
		exportService: exportService,
		emailService:  emailService,
		enqueuer:      enqueuer,
		logger:        logger,
		maxRangeDays:  maxRangeDays,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/download", h.DownloadExport)
	g.POST("/email", h.EmailExport)
}

// DTOs

type ExportRequest struct {
	StartDate string `json:"start_date" validate:"required"`
	EndDate   string `json:"end_date" validate:"required"`
}

// respondValidationError writes a JSON error response and returns errResponseSent
// so the caller knows to stop processing. If c.JSON itself fails, that error is returned instead.
func (h *Handler) respondValidationError(c echo.Context, status int, detail string) error {
	if err := c.JSON(status, common.ErrorResponse{Detail: detail}); err != nil {
		return err
	}
	return errResponseSent
}

// validateExportRequest validates the export request and returns an error if validation fails.
// The returned error is either errResponseSent (response already written) or an Echo error.
func (h *Handler) validateExportRequest(c echo.Context) (*ExportRequest, error) {
	var req ExportRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, "Invalid request data")
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, "Validation failed")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, "Invalid start_date format, expected YYYY-MM-DD")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, "Invalid end_date format, expected YYYY-MM-DD")
	}

	if startDate.After(endDate) {
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, "start_date must be before or equal to end_date")
	}

	daysDiff := endDate.Sub(startDate).Hours() / 24
	if daysDiff > float64(h.maxRangeDays) {
		return nil, h.respondValidationError(c, http.StatusUnprocessableEntity, fmt.Sprintf("Date range must not exceed %d days", h.maxRangeDays))
	}

	return &req, nil
}

func (h *Handler) DownloadExport(c echo.Context) error {
	userID := c.Get("user_id").(int)

	req, err := h.validateExportRequest(c)
	if err != nil {
		if errors.Is(err, errResponseSent) {
			return nil
		}
		return err
	}

	excelBytes, err := h.exportService.GenerateExcel(userID, req.StartDate, req.EndDate)
	if err != nil {
		h.logger.Error("generate excel failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to generate export"})
	}

	c.Response().Header().Set("Content-Disposition", `attachment; filename="transactions_export.xlsx"`)
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	return c.Blob(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

func (h *Handler) EmailExport(c echo.Context) error {
	userID := c.Get("user_id").(int)

	req, err := h.validateExportRequest(c)
	if err != nil {
		if errors.Is(err, errResponseSent) {
			return nil
		}
		return err
	}

	email := common.ActiveUserEmail(c)

	task, err := tasks.NewExportEmailTask(tasks.ExportEmailPayload{
		UserID:    userID,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Email:     email,
	})
	if err != nil {
		h.logger.Error("create export email task failed", "userID", userID, "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to schedule export"})
	}

	if _, err := h.enqueuer.Enqueue(task); err != nil {
		h.logger.Error("enqueue export email task failed", "userID", userID, "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to schedule export"})
	}

	return c.JSON(http.StatusAccepted, common.MessageResponse{Message: "Export will be sent to your email shortly"})
}
