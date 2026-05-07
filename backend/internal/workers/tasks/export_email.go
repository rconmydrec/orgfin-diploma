package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// ExportEmailPayload holds the data needed to generate and email an export.
type ExportEmailPayload struct {
	UserID    int    `json:"user_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Email     string `json:"email"`
}

// NewExportEmailTask creates a new export:email task with the given payload.
func NewExportEmailTask(payload ExportEmailPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal export email payload: %w", err)
	}
	return asynq.NewTask(
		TypeExportEmail,
		data,
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
		asynq.Queue("export"),
	), nil
}

// ExportService defines the interface for generating Excel exports.
type ExportService interface {
	GenerateExcel(userID int, startDate, endDate string) ([]byte, error)
}

// EmailService defines the interface for sending emails.
type EmailService interface {
	Send(to, subject, body string) error
	SendWithAttachment(to, subject, body string, attachment []byte, filename string) error
}

// ExportEmailHandler processes export:email tasks by generating an Excel file
// and sending it to the user via email.
type ExportEmailHandler struct {
	exportService ExportService
	emailService  EmailService
	logger        *slog.Logger
}

// NewExportEmailHandler creates a new handler for export:email tasks.
func NewExportEmailHandler(exportService ExportService, emailService EmailService, logger *slog.Logger) *ExportEmailHandler {
	return &ExportEmailHandler{
		exportService: exportService,
		emailService:  emailService,
		logger:        logger,
	}
}

// ProcessTask generates an Excel export and emails it to the user.
func (h *ExportEmailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ExportEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal export email payload: %w", err)
	}

	h.logger.Info("processing export email task",
		"userID", payload.UserID,
		"startDate", payload.StartDate,
		"endDate", payload.EndDate,
		"email", payload.Email,
	)

	excelBytes, err := h.exportService.GenerateExcel(payload.UserID, payload.StartDate, payload.EndDate)
	if err != nil {
		h.logger.Error("generate excel for email failed",
			"userID", payload.UserID,
			"error", err,
		)
		return fmt.Errorf("generate excel: %w", err)
	}

	subject := "Transactions Export (" + payload.StartDate + " - " + payload.EndDate + ")"
	body := "Please find your transactions export attached.\n\nDate range: " + payload.StartDate + " to " + payload.EndDate
	filename := "transactions_export.xlsx"

	if err := h.emailService.SendWithAttachment(payload.Email, subject, body, excelBytes, filename); err != nil {
		h.logger.Error("send export email failed",
			"userID", payload.UserID,
			"email", payload.Email,
			"error", err,
		)
		return fmt.Errorf("send export email: %w", err)
	}

	return nil
}
