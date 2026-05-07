package email

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/config"
)

// hostnameFunc resolves the local machine hostname. It is a package-level
// variable so tests can override it to exercise the fallback branch without
// touching system state. Tests MUST restore the original value via cleanup.
var hostnameFunc = os.Hostname

// lookupCNAMEFunc resolves the canonical name for a host. It is a package-level
// variable so tests can override it to exercise the FQDN-via-CNAME branch
// without depending on the local resolver state.
var lookupCNAMEFunc = net.LookupCNAME

// generateMessageID creates a unique RFC 5322 Message-ID. The domain part
// MUST contain at least one dot in normal operation to avoid Gmail's bulk
// mail heuristics (RFC 5322 §3.6.4 mandates an FQDN-shaped right-hand side).
//
// Resolution order for the domain:
//  1. config.Config.MessageIDDomain (if non-empty) — the authoritative
//     deployment-configured FQDN, always wins.
//  2. os.Hostname():
//     a. if it already contains a dot, use it as-is
//     b. otherwise, attempt net.LookupCNAME(hostname); if the CNAME contains
//     a dot, use the CNAME (trimmed of trailing ".") as the FQDN
//     c. otherwise, use the bare hostname (single-label fallback — only
//     reached when no FQDN can be derived)
//  3. "localhost" — last-resort fallback when no hostname is available.
func (s *Service) generateMessageID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	domain := s.resolveMessageIDDomain()
	return fmt.Sprintf("<%x.%x@%s>", buf[:8], buf[8:], domain)
}

// resolveMessageIDDomain implements the resolution order documented on
// generateMessageID. Extracted as a separate method so each branch can be
// unit-tested in isolation. The configured domain is sourced from
// s.config.MessageIDDomain (populated in config.New() from MESSAGE_ID_DOMAIN);
// the service intentionally does NOT re-read the env variable directly so that
// configuration remains single-sourced through the config layer.
func (s *Service) resolveMessageIDDomain() string {
	if envDomain := strings.TrimSpace(s.config.MessageIDDomain); envDomain != "" {
		return envDomain
	}

	hostname, err := hostnameFunc()
	if err != nil || hostname == "" {
		return "localhost"
	}

	if strings.Contains(hostname, ".") {
		return hostname
	}

	if cname, cerr := lookupCNAMEFunc(hostname); cerr == nil {
		cname = strings.TrimSuffix(cname, ".")
		if strings.Contains(cname, ".") {
			return cname
		}
	}

	// Single-label hostname with no resolvable FQDN. Returning the bare
	// hostname here keeps tests deterministic; in production deployments
	// MESSAGE_ID_DOMAIN should always be set so this branch is not reached.
	return hostname
}

// rfc2822Date returns the current time formatted per RFC 2822.
func rfc2822Date() string {
	return time.Now().Format(time.RFC1123Z)
}

type Service struct {
	config *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) *Service {
	return &Service{
		config: cfg,
		logger: logger,
	}
}

// fileExtension returns the last extension from a filename (e.g. "gz" from "backup.sql.gz").
func fileExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[idx+1:]
	}
	return ""
}

// sanitizeHeader strips CR and LF characters to prevent SMTP header injection.
func sanitizeHeader(value string) string {
	r := strings.NewReplacer("\r", "", "\n", "")
	return r.Replace(value)
}

// styleStripRE and scriptStripRE remove <style>...</style> and
// <script>...</script> blocks (including the contents) before tag stripping.
// They are split into two regexps because RE2 does not support backreferences,
// so we cannot reuse a single capture for both tag names.
//
// "(?is)" enables case-insensitive matching across newlines.
var (
	styleStripRE  = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</\s*style\s*>`)
	scriptStripRE = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</\s*script\s*>`)
)

// tagStripRE matches any remaining HTML tag.
var tagStripRE = regexp.MustCompile(`(?s)<[^>]+>`)

// whitespaceCollapseRE collapses any run of whitespace (spaces, tabs,
// newlines) into a single space.
var whitespaceCollapseRE = regexp.MustCompile(`\s+`)

// htmlToPlainText derives a minimal plain-text rendering of an HTML body for
// use as the text/plain alternative inside multipart/alternative messages.
// The goal is not pretty rendering — Gmail just needs a non-empty plain part
// to stop classifying the message as "HTML-only bulk mail".
func htmlToPlainText(htmlBody string) string {
	stripped := styleStripRE.ReplaceAllString(htmlBody, " ")
	stripped = scriptStripRE.ReplaceAllString(stripped, " ")
	stripped = tagStripRE.ReplaceAllString(stripped, " ")
	// Decode the few HTML entities the email body actually uses. We rely on
	// html.UnescapeString to be safe for the entities we generate ourselves
	// (&copy;, &lt;, &gt;, &amp;).
	stripped = htmlUnescape(stripped)
	stripped = whitespaceCollapseRE.ReplaceAllString(stripped, " ")
	return strings.TrimSpace(stripped)
}

// htmlUnescape decodes HTML entities (e.g. &amp;, &lt;, &copy;) into their
// runes. Wrapped in a function so the entity decoder can be swapped in tests
// if needed.
func htmlUnescape(s string) string {
	return html.UnescapeString(s)
}

// writeQuotedPrintable writes the input as quoted-printable to the given
// writer. Returns any underlying I/O error.
func writeQuotedPrintable(w *multipart.Writer, header textproto.MIMEHeader, body []byte) error {
	pw, err := w.CreatePart(header)
	if err != nil {
		return err
	}
	qpw := quotedprintable.NewWriter(pw)
	if _, err := qpw.Write(body); err != nil {
		return err
	}
	return qpw.Close()
}

// buildAlternativeBody writes a multipart/alternative payload (plain + html)
// to the given multipart.Writer. The boundary of the inner multipart is
// chosen by an inner writer so it cannot collide with the outer boundary.
// Returns the Content-Type header value used for the alternative part so the
// caller can register it correctly when the alternative is wrapped in an
// outer multipart/mixed for attachments.
func buildAlternativeBody(w *multipart.Writer, htmlBody, plainBody string) error {
	altHeader := textproto.MIMEHeader{}
	altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", w.Boundary()))

	plainHeader := textproto.MIMEHeader{}
	plainHeader.Set("Content-Type", `text/plain; charset="utf-8"`)
	plainHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	if err := writeQuotedPrintable(w, plainHeader, []byte(plainBody)); err != nil {
		return fmt.Errorf("write plain part: %w", err)
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="utf-8"`)
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	if err := writeQuotedPrintable(w, htmlHeader, []byte(htmlBody)); err != nil {
		return fmt.Errorf("write html part: %w", err)
	}
	return nil
}

// writeCommonHeaders writes the headers shared by Send and SendWithAttachment
// to the message buffer. It does NOT write the Content-Type header — that is
// the caller's responsibility because it differs between the two flows.
func (s *Service) writeCommonHeaders(msg *bytes.Buffer, to, subject string) {
	from := s.config.SMTPFrom
	fromName := s.config.SMTPFromName

	if fromName != "" {
		fmt.Fprintf(msg, "From: %s <%s>\r\n", mime.QEncoding.Encode("utf-8", fromName), from)
	} else {
		fmt.Fprintf(msg, "From: %s\r\n", from)
	}
	fmt.Fprintf(msg, "To: %s\r\n", to)
	fmt.Fprintf(msg, "Date: %s\r\n", rfc2822Date())
	fmt.Fprintf(msg, "Message-ID: %s\r\n", s.generateMessageID())
	fmt.Fprintf(msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
}

// sendRaw sends a raw email message via SMTP with proper EHLO hostname.
func (s *Service) sendRaw(from string, to []string, msgBytes []byte) error {
	host := s.config.SMTPHost
	port := s.config.SMTPPort
	user := s.config.SMTPUser
	password := s.config.SMTPPassword

	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Use the machine's actual hostname for EHLO (matching Python's socket.getfqdn() behavior).
	// Sending the SMTP server's hostname (e.g. "smtp.gmail.com") looks like spoofing.
	ehloHost, _ := os.Hostname()
	if ehloHost == "" {
		ehloHost = "localhost"
	}
	if err = client.Hello(ehloHost); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	// Upgrade to TLS (STARTTLS)
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: host}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	// Authenticate
	auth := smtp.PlainAuth("", user, password, host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	// Set sender and recipients
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed: %w", err)
		}
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}
	if _, err = w.Write(msgBytes); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}

	return client.Quit()
}

// buildSendMessage renders a multipart/alternative MIME message for Send.
// Extracted so it can be exercised by unit tests without an SMTP roundtrip.
func (s *Service) buildSendMessage(to, subject, htmlBody string) ([]byte, error) {
	plainBody := htmlToPlainText(htmlBody)

	var msg bytes.Buffer
	s.writeCommonHeaders(&msg, to, subject)

	altWriter := multipart.NewWriter(&msg)
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n", altWriter.Boundary())
	msg.WriteString("\r\n")

	if err := buildAlternativeBody(altWriter, htmlBody, plainBody); err != nil {
		return nil, err
	}
	if err := altWriter.Close(); err != nil {
		return nil, fmt.Errorf("close alternative writer: %w", err)
	}

	return msg.Bytes(), nil
}

// buildSendWithAttachmentMessage renders a multipart/mixed MIME message that
// contains a nested multipart/alternative body part followed by a base64
// attachment. Extracted so it can be exercised by unit tests without an SMTP
// roundtrip.
func (s *Service) buildSendWithAttachmentMessage(to, subject, htmlBody string, attachment []byte, filename string) ([]byte, error) {
	plainBody := htmlToPlainText(htmlBody)

	var msg bytes.Buffer
	s.writeCommonHeaders(&msg, to, subject)

	mixedWriter := multipart.NewWriter(&msg)
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%q\r\n", mixedWriter.Boundary())
	msg.WriteString("\r\n")

	// Nested multipart/alternative for the body. We render it into its own
	// buffer first so we can attach it to the mixed writer with the correct
	// Content-Type header that includes the inner boundary.
	var altBuf bytes.Buffer
	altWriter := multipart.NewWriter(&altBuf)
	if err := buildAlternativeBody(altWriter, htmlBody, plainBody); err != nil {
		return nil, err
	}
	if err := altWriter.Close(); err != nil {
		return nil, fmt.Errorf("close alternative writer: %w", err)
	}

	altPartHeader := textproto.MIMEHeader{}
	altPartHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", altWriter.Boundary()))
	altPart, err := mixedWriter.CreatePart(altPartHeader)
	if err != nil {
		return nil, fmt.Errorf("create alternative wrapper part: %w", err)
	}
	if _, err := altPart.Write(altBuf.Bytes()); err != nil {
		return nil, fmt.Errorf("write alternative wrapper part: %w", err)
	}

	// Attachment part.
	contentType := mime.TypeByExtension("." + fileExtension(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	safeFilename := sanitizeHeader(filename)
	attachHeader := textproto.MIMEHeader{}
	attachHeader.Set("Content-Type", fmt.Sprintf("%s; name=%q", contentType, safeFilename))
	attachHeader.Set("Content-Transfer-Encoding", "base64")
	attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeFilename))

	attachPart, err := mixedWriter.CreatePart(attachHeader)
	if err != nil {
		return nil, fmt.Errorf("create attachment part: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(attachment)
	for i := 0; i < len(encoded); i += 76 {
		end := min(i+76, len(encoded))
		if _, err := attachPart.Write([]byte(encoded[i:end] + "\r\n")); err != nil {
			return nil, fmt.Errorf("write attachment chunk: %w", err)
		}
	}

	if err := mixedWriter.Close(); err != nil {
		return nil, fmt.Errorf("close mixed writer: %w", err)
	}

	return msg.Bytes(), nil
}

// SendWithAttachment sends an email with a file attachment using SMTP.
func (s *Service) SendWithAttachment(to, subject, body string, attachment []byte, filename string) error {
	to = sanitizeHeader(to)
	subject = sanitizeHeader(subject)

	msgBytes, err := s.buildSendWithAttachmentMessage(to, subject, body, attachment, filename)
	if err != nil {
		s.logger.Error("failed to build email message", "to", to, "error", err)
		return fmt.Errorf("failed to build email: %w", err)
	}

	if err := s.sendRaw(s.config.SMTPFrom, []string{to}, msgBytes); err != nil {
		s.logger.Error("failed to send email", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("email sent successfully", "to", to, "subject", subject)
	return nil
}

// Send sends an HTML email (with an auto-generated plain-text alternative)
// without attachments using SMTP. The rendered MIME message is
// multipart/alternative with both text/plain and text/html parts so that
// Gmail does not flag it as HTML-only bulk mail.
func (s *Service) Send(to, subject, body string) error {
	to = sanitizeHeader(to)
	subject = sanitizeHeader(subject)

	msgBytes, err := s.buildSendMessage(to, subject, body)
	if err != nil {
		s.logger.Error("failed to build email message", "to", to, "error", err)
		return fmt.Errorf("failed to build email: %w", err)
	}

	if err := s.sendRaw(s.config.SMTPFrom, []string{to}, msgBytes); err != nil {
		s.logger.Error("failed to send email", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("email sent successfully", "to", to, "subject", subject)
	return nil
}
