package backup_notify

import (
	"fmt"
	"html"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/workers/tasks"
)

// Service handles backup notification business logic, including email
// composition and enqueueing email tasks for each admin.
type Service struct {
	enqueuer    TaskEnqueuer
	adminEmails []string
	environment string
	logger      *slog.Logger
}

// New creates a new backup notification Service.
func New(enqueuer TaskEnqueuer, adminEmails []string, environment string, logger *slog.Logger) *Service {
	return &Service{
		enqueuer:    enqueuer,
		adminEmails: adminEmails,
		environment: environment,
		logger:      logger,
	}
}

// NotifyBackupComplete builds the notification email and enqueues it for all
// admin emails. Individual enqueue failures are logged and skipped. Returns
// an error only if there is no admin email configured.
func (s *Service) NotifyBackupComplete(params NotifyParams) (int, error) {
	if len(s.adminEmails) == 0 {
		s.logger.Info("backup notification received but no admin emails configured", "status", params.Status)
		return 0, nil
	}

	subject := s.buildSubject(params.Status)
	body := s.buildEmailBody(params)

	sent := 0
	for _, email := range s.adminEmails {
		task, err := tasks.NewEmailSendTask(tasks.EmailSendPayload{
			To:      email,
			Subject: subject,
			Body:    body,
		})
		if err != nil {
			s.logger.Error("failed to create email task for backup notification",
				"email", email,
				"error", err,
			)
			continue
		}

		if _, err := s.enqueuer.Enqueue(task); err != nil {
			s.logger.Error("failed to enqueue backup notification email",
				"email", email,
				"error", err,
			)
			continue
		}
		sent++
	}

	return sent, nil
}

// buildSubject returns the email subject line based on backup status.
//
// Subjects are intentionally short, ASCII-only, and free of square brackets
// or the word "successfully" to avoid Gmail's bulk-mail spam heuristics. The
// environment label is still rendered in the email body for operator
// visibility.
func (s *Service) buildSubject(status string) string {
	if status == "success" {
		return "Database backup created"
	}
	return "Database backup FAILED"
}

// buildEmailBody generates an HTML email body for backup notifications.
func (s *Service) buildEmailBody(params NotifyParams) string {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	statusLabel := "Success"
	if params.Status == "error" {
		statusLabel = "FAILED"
	}

	gdriveStatus := "No"
	if params.GdriveUploaded {
		gdriveStatus = "Yes"
	}

	// Build optional detail lines (escape user-supplied fields for HTML safety)
	filenameDetail := ""
	if params.Filename != "" {
		filenameDetail = `            <p><strong>Filename:</strong> ` + html.EscapeString(params.Filename) + `</p>
`
	}

	fileSizeDetail := ""
	if params.FileSize != "" {
		fileSizeDetail = `            <p><strong>File size:</strong> ` + html.EscapeString(params.FileSize) + `</p>
`
	}

	errorDetail := ""
	if params.Status == "error" && params.ErrorMessage != "" {
		errorDetail = `            <p style="color: #dc3545;"><strong>Error:</strong> ` + html.EscapeString(params.ErrorMessage) + `</p>
`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Backup Notification</title>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 0; }
        .container { max-width: 600px; margin: 40px auto; background: #ffffff; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); padding: 30px; }
        h1 { color: #333333; font-size: 22px; }
        p { color: #666666; line-height: 1.6; }
        .details { margin: 20px 0; }
        .details p { margin: 4px 0; }
        .footer { text-align: center; margin-top: 30px; font-size: 12px; color: #aaaaaa; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Backup Notification</h1>
        <p>Dear Team,</p>
        <p>This is to inform you that a database backup has been completed in the <strong>` + html.EscapeString(s.environment) + `</strong> environment.</p>
        <div class="details">
            <p><strong>Status:</strong> ` + statusLabel + `</p>
            <p><strong>Time:</strong> ` + timestamp + `</p>
` + filenameDetail + fileSizeDetail + `            <p><strong>GDrive uploaded:</strong> ` + gdriveStatus + `</p>
` + errorDetail + `        </div>
        <p>If you have any questions or need further assistance, please contact the support team.</p>
        <p>Best regards,<br>The Orgfin Team</p>
        <div class="footer">
            <p>&copy; ` + fmt.Sprintf("%d", time.Now().Year()) + ` Orgfin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
}
