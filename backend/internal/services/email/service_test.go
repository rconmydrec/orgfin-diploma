package email

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/config"
)

// messageIDFormat matches the canonical output of generateMessageID:
// <16 hex chars>.<16 hex chars>@<domain>>
var messageIDFormat = regexp.MustCompile(`^<[0-9a-f]{16}\.[0-9a-f]{16}@[^>]+>$`)

// messageIDFQDNFormat additionally requires the domain part to contain at
// least one dot — the spam-filter-relevant property guarded by acceptance
// criteria.
var messageIDFQDNFormat = regexp.MustCompile(`^<[0-9a-f]{16}\.[0-9a-f]{16}@[^@>]+\.[^@>]+>$`)

// withHostnameFunc swaps hostnameFunc for the duration of a test and
// restores the original implementation via t.Cleanup. This keeps tests
// isolated from one another and from the real machine hostname.
func withHostnameFunc(t *testing.T, fn func() (string, error)) {
	t.Helper()
	original := hostnameFunc
	hostnameFunc = fn
	t.Cleanup(func() {
		hostnameFunc = original
	})
}

// withLookupCNAMEFunc swaps lookupCNAMEFunc for the duration of a test.
func withLookupCNAMEFunc(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	original := lookupCNAMEFunc
	lookupCNAMEFunc = fn
	t.Cleanup(func() {
		lookupCNAMEFunc = original
	})
}

// newServiceWithDomain builds a *Service whose config carries the given
// MessageIDDomain. This is the only knob needed to exercise the
// "configured-domain wins" branch — the env var is intentionally NOT read by
// the service (configuration is single-sourced through config.Config).
func newServiceWithDomain(domain string) *Service {
	cfg := &config.Config{MessageIDDomain: domain}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// ---------------------------------------------------------------------------
// generateMessageID / resolveMessageIDDomain
// ---------------------------------------------------------------------------

func TestGenerateMessageID_UsesEnvDomainWhenSet(t *testing.T) {
	svc := newServiceWithDomain("orgfin")
	// Even if hostname returns a sensible value, the configured domain wins.
	withHostnameFunc(t, func() (string, error) { return "ignored-host", nil })
	withLookupCNAMEFunc(t, func(string) (string, error) {
		t.Fatal("lookupCNAMEFunc must not be called when configured domain is set")
		return "", nil
	})

	got := svc.generateMessageID()

	if !messageIDFQDNFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match FQDN format", got)
	}
	if !strings.HasSuffix(got, "@orgfin>") {
		t.Errorf("Message-ID %q does not end with @orgfin>", got)
	}
}

func TestGenerateMessageID_UsesHostnameWithDot(t *testing.T) {
	svc := newServiceWithDomain("")
	const expectedHost = "test-host.example"
	withHostnameFunc(t, func() (string, error) { return expectedHost, nil })
	withLookupCNAMEFunc(t, func(string) (string, error) {
		t.Fatal("lookupCNAMEFunc must not be called when hostname already contains a dot")
		return "", nil
	})

	got := svc.generateMessageID()

	if !messageIDFQDNFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match FQDN format", got)
	}
	if !strings.HasSuffix(got, "@"+expectedHost+">") {
		t.Errorf("Message-ID %q does not end with @%s>", got, expectedHost)
	}
}

func TestGenerateMessageID_UsesCNAMEWhenHostnameHasNoDot(t *testing.T) {
	svc := newServiceWithDomain("")
	withHostnameFunc(t, func() (string, error) { return "podname", nil })
	withLookupCNAMEFunc(t, func(host string) (string, error) {
		if host != "podname" {
			t.Errorf("expected lookupCNAMEFunc called with %q, got %q", "podname", host)
		}
		return "podname.cluster.local.", nil
	})

	got := svc.generateMessageID()

	if !messageIDFQDNFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match FQDN format", got)
	}
	if !strings.HasSuffix(got, "@podname.cluster.local>") {
		t.Errorf("Message-ID %q does not end with @podname.cluster.local>", got)
	}
}

func TestGenerateMessageID_FallsBackToBareHostnameWhenCNAMEUnusable(t *testing.T) {
	svc := newServiceWithDomain("")
	withHostnameFunc(t, func() (string, error) { return "podname", nil })
	withLookupCNAMEFunc(t, func(string) (string, error) { return "podname.", nil })

	got := svc.generateMessageID()

	if !messageIDFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match canonical format", got)
	}
	// Single-label fallback — does NOT contain a dot in the domain part.
	if !strings.HasSuffix(got, "@podname>") {
		t.Errorf("Message-ID %q does not end with @podname>", got)
	}
}

func TestGenerateMessageID_FallsBackToBareHostnameWhenCNAMEErrors(t *testing.T) {
	svc := newServiceWithDomain("")
	withHostnameFunc(t, func() (string, error) { return "podname", nil })
	withLookupCNAMEFunc(t, func(string) (string, error) {
		return "", errors.New("dns lookup failed")
	})

	got := svc.generateMessageID()

	if !messageIDFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match canonical format", got)
	}
	if !strings.HasSuffix(got, "@podname>") {
		t.Errorf("Message-ID %q does not end with @podname>", got)
	}
}

func TestGenerateMessageID_FallsBackToLocalhost_OnHostnameError(t *testing.T) {
	svc := newServiceWithDomain("")
	withHostnameFunc(t, func() (string, error) {
		return "", errors.New("hostname lookup failed")
	})

	got := svc.generateMessageID()

	if !messageIDFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match expected format", got)
	}
	if !strings.HasSuffix(got, "@localhost>") {
		t.Errorf("Message-ID %q does not end with @localhost>", got)
	}
}

func TestGenerateMessageID_FallsBackToLocalhost_OnEmptyHostname(t *testing.T) {
	svc := newServiceWithDomain("")
	withHostnameFunc(t, func() (string, error) { return "", nil })

	got := svc.generateMessageID()

	if !messageIDFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match expected format", got)
	}
	if !strings.HasSuffix(got, "@localhost>") {
		t.Errorf("Message-ID %q does not end with @localhost>", got)
	}
}

// TestGenerateMessageID_EnvDomainTrimmed ensures whitespace-only configured
// values are treated as empty and trimmed values are used verbatim.
func TestGenerateMessageID_EnvDomainTrimmed(t *testing.T) {
	svc := newServiceWithDomain("  orgfin  ")
	withHostnameFunc(t, func() (string, error) { return "should-be-ignored", nil })

	got := svc.generateMessageID()

	if !strings.HasSuffix(got, "@orgfin>") {
		t.Errorf("Message-ID %q does not end with @orgfin> after trimming configured value", got)
	}
}

func TestGenerateMessageID_NeverUsesGmailDomain(t *testing.T) {
	svc := newServiceWithDomain("")
	const realisticHost = "static.92.248.108.65.clients.your-server.de"
	withHostnameFunc(t, func() (string, error) { return realisticHost, nil })

	got := svc.generateMessageID()

	if strings.Contains(got, "@gmail.com") {
		t.Fatalf("Message-ID %q must not contain @gmail.com", got)
	}
	if !messageIDFQDNFormat.MatchString(got) {
		t.Errorf("Message-ID %q does not match FQDN format", got)
	}
	if !strings.HasSuffix(got, "@"+realisticHost+">") {
		t.Errorf("Message-ID %q does not end with @%s>", got, realisticHost)
	}
}

func TestGenerateMessageID_RealHostname(t *testing.T) {
	// Do not override the resolution chain — exercise the production default.
	svc := newServiceWithDomain("")
	got := svc.generateMessageID()

	if !messageIDFormat.MatchString(got) {
		t.Fatalf("Message-ID %q does not match expected format", got)
	}

	// Whatever the host, the domain portion must be non-empty.
	hn, err := os.Hostname()
	if err == nil && hn != "" {
		// Smoke check: domain ends with the hostname or its FQDN — but we do
		// not assert exact equality because production hostname/CNAME state
		// is not deterministic in test environments. Just confirm the
		// Message-ID is well-formed.
		_ = hn
	}
}

// ---------------------------------------------------------------------------
// htmlToPlainText
// ---------------------------------------------------------------------------

func TestHtmlToPlainText_StripsTagsAndCollapsesWhitespace(t *testing.T) {
	html := `<!DOCTYPE html><html><head>
	<style>body { color: red; } .x { font-size: 12px; }</style>
	<script>alert('x');</script>
	</head><body>
	<h1>Backup Notification</h1>
	<p>Status: <strong>Success</strong></p>
	</body></html>`

	got := htmlToPlainText(html)

	if got == "" {
		t.Fatal("plain-text output must not be empty")
	}
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("plain-text output must not contain raw tag delimiters, got: %q", got)
	}
	if strings.Contains(got, "color: red") {
		t.Errorf("plain-text output must not contain CSS from <style>, got: %q", got)
	}
	if strings.Contains(got, "alert(") {
		t.Errorf("plain-text output must not contain JS from <script>, got: %q", got)
	}
	if !strings.Contains(got, "Backup Notification") {
		t.Errorf("plain-text output must preserve heading text, got: %q", got)
	}
	if !strings.Contains(got, "Status:") || !strings.Contains(got, "Success") {
		t.Errorf("plain-text output must preserve body text, got: %q", got)
	}
	// Whitespace must be collapsed.
	if strings.Contains(got, "  ") {
		t.Errorf("plain-text output must collapse consecutive whitespace, got: %q", got)
	}
}

func TestHtmlToPlainText_DecodesEntities(t *testing.T) {
	got := htmlToPlainText(`<p>Tom &amp; Jerry &copy; 2026</p>`)
	if !strings.Contains(got, "Tom & Jerry") {
		t.Errorf("expected '&amp;' decoded to '&', got %q", got)
	}
	if !strings.Contains(got, "©") {
		t.Errorf("expected '&copy;' decoded to '©', got %q", got)
	}
}

func TestHtmlToPlainText_EmptyInput(t *testing.T) {
	if got := htmlToPlainText(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// MIME structure: Send (multipart/alternative)
// ---------------------------------------------------------------------------

func newTestService() *Service {
	cfg := &config.Config{
		SMTPFrom:        "noreply@example.test",
		SMTPFromName:    "Budget Tracker",
		MessageIDDomain: "example.test",
	}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

const sampleHTML = `<!DOCTYPE html>
<html><head>
<style>body { font-family: Arial; }</style>
</head><body>
<h1>Backup Notification</h1>
<p>Status: <strong>Success</strong></p>
<p>Time: 2026-04-25 12:00:00 UTC</p>
</body></html>`

func TestSend_RendersMultipartAlternative(t *testing.T) {
	svc := newTestService()
	raw, err := svc.buildSendMessage("user@example.com", "Test subject", sampleHTML)
	if err != nil {
		t.Fatalf("buildSendMessage failed: %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("failed to parse rendered message: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse Content-Type: %v", err)
	}
	if mediaType != "multipart/alternative" {
		t.Fatalf("top-level Content-Type = %q, want multipart/alternative", mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		t.Fatal("multipart/alternative boundary must be present")
	}

	parts := readAllParts(t, msg.Body, boundary)
	if len(parts) != 2 {
		t.Fatalf("expected exactly 2 parts (plain + html), got %d", len(parts))
	}

	assertPlainAndHTMLParts(t, parts)

	// Message-ID must contain a dot in the domain.
	mid := msg.Header.Get("Message-ID")
	if !messageIDFQDNFormat.MatchString(mid) {
		t.Errorf("Message-ID %q does not match FQDN format", mid)
	}
}

func TestSendWithAttachment_RendersMultipartMixedWrappingAlternative(t *testing.T) {
	svc := newTestService()
	raw, err := svc.buildSendWithAttachmentMessage(
		"user@example.com",
		"Export ready",
		sampleHTML,
		[]byte("PK\x03\x04 fake xlsx bytes"),
		"export.xlsx",
	)
	if err != nil {
		t.Fatalf("buildSendWithAttachmentMessage failed: %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("failed to parse rendered message: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse Content-Type: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("top-level Content-Type = %q, want multipart/mixed", mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		t.Fatal("multipart/mixed boundary must be present")
	}

	mixedParts := readAllParts(t, msg.Body, boundary)
	if len(mixedParts) != 2 {
		t.Fatalf("expected exactly 2 mixed parts (alternative + attachment), got %d", len(mixedParts))
	}

	// First part: nested multipart/alternative.
	altPart := mixedParts[0]
	altMediaType, altParams, err := mime.ParseMediaType(altPart.header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse inner Content-Type: %v", err)
	}
	if altMediaType != "multipart/alternative" {
		t.Fatalf("inner Content-Type = %q, want multipart/alternative", altMediaType)
	}
	innerBoundary := altParams["boundary"]
	if innerBoundary == "" {
		t.Fatal("inner multipart/alternative boundary must be present")
	}
	if innerBoundary == boundary {
		t.Fatal("inner boundary must differ from outer boundary")
	}

	innerParts := readAllParts(t, strings.NewReader(string(altPart.body)), innerBoundary)
	if len(innerParts) != 2 {
		t.Fatalf("expected exactly 2 inner parts (plain + html), got %d", len(innerParts))
	}
	assertPlainAndHTMLParts(t, innerParts)

	// Second part: attachment.
	attach := mixedParts[1]
	attachMedia, _, err := mime.ParseMediaType(attach.header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse attachment Content-Type: %v", err)
	}
	if !strings.HasPrefix(attachMedia, "application/") && !strings.HasPrefix(attachMedia, "text/") &&
		!strings.HasPrefix(attachMedia, "image/") {
		t.Errorf("unexpected attachment media type: %q", attachMedia)
	}
	if attach.header.Get("Content-Transfer-Encoding") != "base64" {
		t.Errorf("attachment must use base64 encoding, got %q", attach.header.Get("Content-Transfer-Encoding"))
	}
	if disp := attach.header.Get("Content-Disposition"); !strings.Contains(disp, "attachment") {
		t.Errorf("attachment must declare Content-Disposition: attachment, got %q", disp)
	}

	// Message-ID must contain a dot in the domain.
	mid := msg.Header.Get("Message-ID")
	if !messageIDFQDNFormat.MatchString(mid) {
		t.Errorf("Message-ID %q does not match FQDN format", mid)
	}
}

// TestSend_HtmlPartIsQuotedPrintableNot7bit is a focused regression guard
// against the previous "Content-Transfer-Encoding: 7bit" header on HTML
// bodies, which Gmail treats as a forged header.
func TestSend_HtmlPartIsQuotedPrintableNot7bit(t *testing.T) {
	svc := newTestService()
	raw, err := svc.buildSendMessage("user@example.com", "Test", sampleHTML)
	if err != nil {
		t.Fatalf("buildSendMessage failed: %v", err)
	}

	rawStr := string(raw)
	if !strings.Contains(rawStr, "Content-Transfer-Encoding: quoted-printable") {
		t.Errorf("rendered message must declare quoted-printable encoding, raw:\n%s", rawStr)
	}

	// Hard-line regression guard: the rendered message must NEVER declare
	// "Content-Transfer-Encoding: 7bit" — neither on the plain part nor on
	// the HTML part. The previous implementation forced this header on the
	// HTML body and Gmail flagged it as a forged 7bit declaration.
	if strings.Contains(strings.ToLower(rawStr), "content-transfer-encoding: 7bit") {
		t.Errorf("rendered message must not declare 7bit, raw:\n%s", rawStr)
	}
}

// ---------------------------------------------------------------------------
// MIME parsing helpers
// ---------------------------------------------------------------------------

type capturedPart struct {
	header textproto.MIMEHeader
	body   []byte
}

// mime.Header is just a map; here we reuse textproto-style headers by
// converting from multipart.Part.Header (a textproto.MIMEHeader).
func readAllParts(t *testing.T, r io.Reader, boundary string) []capturedPart {
	t.Helper()
	mr := multipart.NewReader(r, boundary)
	var out []capturedPart
	for {
		p, err := mr.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("multipart read error: %v", err)
		}
		body, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("read part body: %v", err)
		}
		out = append(out, capturedPart{header: p.Header, body: body})
		_ = p.Close()
	}
	return out
}

func assertPlainAndHTMLParts(t *testing.T, parts []capturedPart) {
	t.Helper()
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (plain + html), got %d", len(parts))
	}

	plain := parts[0]
	html := parts[1]

	plainMedia, _, err := mime.ParseMediaType(plain.header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse plain Content-Type: %v", err)
	}
	if plainMedia != "text/plain" {
		t.Errorf("first alternative part must be text/plain, got %q", plainMedia)
	}
	if cte := plain.header.Get("Content-Transfer-Encoding"); cte != "quoted-printable" {
		t.Errorf("plain part Content-Transfer-Encoding = %q, want quoted-printable", cte)
	}

	htmlMedia, _, err := mime.ParseMediaType(html.header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("failed to parse html Content-Type: %v", err)
	}
	if htmlMedia != "text/html" {
		t.Errorf("second alternative part must be text/html, got %q", htmlMedia)
	}
	if cte := html.header.Get("Content-Transfer-Encoding"); cte != "quoted-printable" {
		t.Errorf("html part Content-Transfer-Encoding = %q, want quoted-printable", cte)
	}
	if cte := html.header.Get("Content-Transfer-Encoding"); cte == "7bit" {
		t.Errorf("html part must NOT use 7bit encoding")
	}

	// Plain part body must be non-empty after decoding.
	plainText := decodeQuotedPrintable(t, plain.body)
	if strings.TrimSpace(plainText) == "" {
		t.Errorf("text/plain body must be non-empty, got %q", plainText)
	}

	// HTML part body must decode to the original HTML (or a quoted-printable
	// equivalent of it).
	htmlText := decodeQuotedPrintable(t, html.body)
	if !strings.Contains(htmlText, "Backup Notification") {
		t.Errorf("html part body must contain heading, got: %q", htmlText)
	}
}

func decodeQuotedPrintable(t *testing.T, b []byte) string {
	t.Helper()
	r := quotedprintable.NewReader(strings.NewReader(string(b)))
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("quoted-printable decode: %v", err)
	}
	return string(out)
}
