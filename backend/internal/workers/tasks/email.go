package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// EmailSendPayload holds the data needed to send a generic email.
type EmailSendPayload struct {
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
	AttachmentData []byte `json:"attachment_data,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
}

// NewEmailSendTask creates a new email:send task with the given payload.
func NewEmailSendTask(payload EmailSendPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal email send payload: %w", err)
	}
	return asynq.NewTask(
		TypeEmailSend,
		data,
		asynq.MaxRetry(10),
		asynq.Queue("critical"),
		asynq.Timeout(2*time.Minute),
	), nil
}

// EmailSendHandler processes email:send tasks by sending an email
// with or without an attachment.
type EmailSendHandler struct {
	emailService EmailService
	logger       *slog.Logger
}

// NewEmailSendHandler creates a new handler for email:send tasks.
func NewEmailSendHandler(emailService EmailService, logger *slog.Logger) *EmailSendHandler {
	return &EmailSendHandler{
		emailService: emailService,
		logger:       logger,
	}
}

// ProcessTask sends an email, optionally with an attachment.
func (h *EmailSendHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload EmailSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal email send payload: %w", err)
	}

	h.logger.Info("processing email send task",
		"to", payload.To,
		"subject", payload.Subject,
	)

	if len(payload.AttachmentData) > 0 && payload.AttachmentName != "" {
		if err := h.emailService.SendWithAttachment(
			payload.To, payload.Subject, payload.Body,
			payload.AttachmentData, payload.AttachmentName,
		); err != nil {
			h.logger.Error("send email with attachment failed",
				"to", payload.To,
				"error", err,
			)
			return fmt.Errorf("send email with attachment: %w", err)
		}
	} else {
		if err := h.emailService.Send(payload.To, payload.Subject, payload.Body); err != nil {
			h.logger.Error("send email failed",
				"to", payload.To,
				"error", err,
			)
			return fmt.Errorf("send email: %w", err)
		}
	}

	return nil
}
