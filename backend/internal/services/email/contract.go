// Package email provides SMTP email sending with support for plain text/HTML
// emails and file attachments.
//
// Invariants:
//   - All header values are sanitized to prevent SMTP header injection.
//   - STARTTLS is used when available.
//   - EHLO uses the machine's actual hostname (not the SMTP server hostname).
package email

// ServiceInterface defines the public API of the email service.
type ServiceInterface interface {
	Send(to, subject, body string) error
	SendWithAttachment(to, subject, body string, attachment []byte, filename string) error
}
