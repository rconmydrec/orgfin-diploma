package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// TaskEnqueuer is the interface for enqueueing async tasks.
// Defined here to avoid an import cycle between tasks and workers packages.
// This is satisfied by *asynq.Client and any mock implementing the Enqueue method.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// ActivationEmailPayload holds the data needed to send an activation email.
type ActivationEmailPayload struct {
	UserID      int    `json:"user_id"`
	Email       string `json:"email"`
	Token       string `json:"token"`
	AppURL      string `json:"app_url"`
	FrontendURL string `json:"frontend_url"`
}

// NewActivationEmailTask creates a new email:activation task with the given payload.
func NewActivationEmailTask(payload ActivationEmailPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal activation email payload: %w", err)
	}
	return asynq.NewTask(
		TypeEmailActivation,
		data,
		asynq.MaxRetry(10),
		asynq.Queue("critical"),
		asynq.Timeout(2*time.Minute),
	), nil
}

// ActivationEmailHandler processes email:activation tasks by building the
// activation email content and chaining to an email:send task.
type ActivationEmailHandler struct {
	enqueuer TaskEnqueuer
	logger   *slog.Logger
}

// NewActivationEmailHandler creates a new handler for email:activation tasks.
func NewActivationEmailHandler(enqueuer TaskEnqueuer, logger *slog.Logger) *ActivationEmailHandler {
	return &ActivationEmailHandler{
		enqueuer: enqueuer,
		logger:   logger,
	}
}

// ProcessTask builds the activation email and enqueues an email:send task (task chaining).
func (h *ActivationEmailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ActivationEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal activation email payload: %w", err)
	}

	h.logger.Info("processing activation email task",
		"userID", payload.UserID,
		"email", payload.Email,
	)

	activationLink := payload.FrontendURL + "/activate/" + payload.Token

	subject := "Activate Your Budget Tracker Account"
	body := buildActivationEmailBody(activationLink)

	// Chain to email:send task
	emailTask, err := NewEmailSendTask(EmailSendPayload{
		To:      payload.Email,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return fmt.Errorf("create email send task: %w", err)
	}

	if _, err := h.enqueuer.Enqueue(emailTask); err != nil {
		h.logger.Error("failed to enqueue email send task",
			"userID", payload.UserID,
			"email", payload.Email,
			"error", err,
		)
		return fmt.Errorf("enqueue email send task: %w", err)
	}

	h.logger.Info("activation email task chained to email:send",
		"userID", payload.UserID,
		"email", payload.Email,
	)

	return nil
}

// buildActivationEmailBody generates the HTML body for the activation email.
func buildActivationEmailBody(activationLink string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Account Activation</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2c3e50;">Welcome to Budget Tracker!</h2>
        <p>Thank you for registering. Please click the link below to activate your account:</p>
        <p style="text-align: center; margin: 30px 0;">
            <a href="` + activationLink + `"
               style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; font-size: 16px;">
                Activate Account
            </a>
        </p>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #3498db;">` + activationLink + `</p>
        <p>This link will expire in 24 hours.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #999;">If you did not create an account, please ignore this email.</p>
    </div>
</body>
</html>`
}
