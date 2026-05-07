package management

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/labstack/echo/v4"
)

// BackupJobCreator creates Kubernetes Jobs for database backups.
type BackupJobCreator interface {
	CreateBackupJob(ctx context.Context) (jobName string, err error)
}

// backupResponse is the response returned on successful backup job creation.
type backupResponse struct {
	Message string `json:"message"`
	JobName string `json:"jobName"`
}

type Handler struct {
	backupSvc BackupJobCreator
	logger    *slog.Logger
}

func New(backupSvc BackupJobCreator, logger *slog.Logger) *Handler {
	return &Handler{
		backupSvc: backupSvc,
		logger:    logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/backup", h.BackupDB)
}

func (h *Handler) BackupDB(c echo.Context) error {
	email := common.ActiveUserEmail(c)
	h.logger.Info("Backup of the database is requested", "user_email", email)

	if h.backupSvc == nil {
		h.logger.Warn("Backup is only available in Kubernetes environment", "user_email", email)
		return c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Detail: "Backup is only available in Kubernetes environment",
		})
	}

	jobName, err := h.backupSvc.CreateBackupJob(c.Request().Context())
	if err != nil {
		h.logger.Error("failed to create backup job", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Detail: "Failed to create backup job",
		})
	}

	return c.JSON(http.StatusOK, backupResponse{
		Message: "Backup job created",
		JobName: jobName,
	})
}
