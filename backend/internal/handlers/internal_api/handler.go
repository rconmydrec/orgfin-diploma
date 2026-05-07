package internal_api

import (
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	backupnotify "github.com/go-budget/backend/internal/services/backup_notify"
	"github.com/labstack/echo/v4"
)

// BackupNotifyService defines the interface for the backup notification service.
type BackupNotifyService interface {
	NotifyBackupComplete(params backupnotify.NotifyParams) (int, error)
}

// BackupNotifyRequest is the request body for the backup notification endpoint.
type BackupNotifyRequest struct {
	Status         string `json:"status" validate:"required,oneof=success error"`
	Filename       string `json:"filename"`
	FileSize       string `json:"fileSize"`
	ErrorMessage   string `json:"errorMessage"`
	GdriveUploaded bool   `json:"gdriveUploaded"`
}

// Handler handles internal API requests.
type Handler struct {
	backupNotifySvc BackupNotifyService
	logger          *slog.Logger
}

// New creates a new internal API handler.
func New(backupNotifySvc BackupNotifyService, logger *slog.Logger) *Handler {
	return &Handler{
		backupNotifySvc: backupNotifySvc,
		logger:          logger,
	}
}

// RegisterRoutes registers internal API routes on the given group.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/backup/notify", h.BackupNotify)
}

// BackupNotify handles backup completion notifications and sends emails to admins.
func (h *Handler) BackupNotify(c echo.Context) error {
	var req BackupNotifyRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind backup notify request", "error", err)
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid request data"})
	}

	if err := c.Validate(&req); err != nil {
		h.logger.Debug("backup notify validation failed", "error", err)
		return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Invalid request data"})
	}

	params := backupnotify.NotifyParams{
		Status:         req.Status,
		Filename:       req.Filename,
		FileSize:       req.FileSize,
		ErrorMessage:   req.ErrorMessage,
		GdriveUploaded: req.GdriveUploaded,
	}

	_, err := h.backupNotifySvc.NotifyBackupComplete(params)
	if err != nil {
		h.logger.Error("backup notification failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to send notification"})
	}

	return c.JSON(http.StatusOK, common.MessageResponse{Message: "Notification sent"})
}
