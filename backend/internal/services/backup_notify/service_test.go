package backup_notify

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskEnqueuer implements TaskEnqueuer for unit tests.
type mockTaskEnqueuer struct {
	EnqueueFunc func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
	calls       int
}

func (m *mockTaskEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.calls++
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task, opts...)
	}
	return &asynq.TaskInfo{}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNotifyBackupComplete_SuccessStatus(t *testing.T) {
	mock := &mockTaskEnqueuer{}
	svc := New(mock, []string{"admin1@test.com", "admin2@test.com"}, "production", testLogger())

	sent, err := svc.NotifyBackupComplete(NotifyParams{
		Status:         "success",
		Filename:       "backup_20260315.sql.gz",
		FileSize:       "42M",
		GdriveUploaded: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, sent)
	assert.Equal(t, 2, mock.calls)
}

func TestNotifyBackupComplete_ErrorStatus(t *testing.T) {
	mock := &mockTaskEnqueuer{}
	svc := New(mock, []string{"admin@test.com"}, "staging", testLogger())

	sent, err := svc.NotifyBackupComplete(NotifyParams{
		Status:         "error",
		ErrorMessage:   "pg_dump failed: connection refused",
		GdriveUploaded: false,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, 1, mock.calls)
}

func TestNotifyBackupComplete_EmptyAdminEmails(t *testing.T) {
	mock := &mockTaskEnqueuer{}
	svc := New(mock, []string{}, "production", testLogger())

	sent, err := svc.NotifyBackupComplete(NotifyParams{
		Status: "success",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, sent)
	assert.Equal(t, 0, mock.calls)
}

func TestNotifyBackupComplete_EnqueueFailure(t *testing.T) {
	mock := &mockTaskEnqueuer{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			return nil, errors.New("redis connection refused")
		},
	}
	svc := New(mock, []string{"admin1@test.com", "admin2@test.com"}, "production", testLogger())

	sent, err := svc.NotifyBackupComplete(NotifyParams{
		Status:   "success",
		Filename: "backup.sql.gz",
	})
	require.NoError(t, err)
	// No emails were successfully sent
	assert.Equal(t, 0, sent)
	// But both were attempted
	assert.Equal(t, 2, mock.calls)
}

func TestNotifyBackupComplete_PartialEnqueueFailure(t *testing.T) {
	callCount := 0
	mock := &mockTaskEnqueuer{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("redis error")
			}
			return &asynq.TaskInfo{}, nil
		},
	}
	svc := New(mock, []string{"admin1@test.com", "admin2@test.com"}, "production", testLogger())

	sent, err := svc.NotifyBackupComplete(NotifyParams{
		Status: "success",
	})
	require.NoError(t, err)
	// Only the second email was sent
	assert.Equal(t, 1, sent)
}

func TestBuildSubject_Success(t *testing.T) {
	svc := New(nil, nil, "production", testLogger())
	subject := svc.buildSubject("success")
	assert.Equal(t, "Database backup created", subject)
}

func TestBuildSubject_Error(t *testing.T) {
	svc := New(nil, nil, "staging", testLogger())
	subject := svc.buildSubject("error")
	assert.Equal(t, "Database backup FAILED", subject)
}

// TestBuildSubject_SuccessNoSpamSignals enforces the spam-filter properties
// of the success subject: no square brackets (`[`/`]`) and no "successfully"
// token (case-insensitive). These were previously the strongest single
// non-encoding spam signal in the rendered message, per the Python-vs-Go
// comparison.
func TestBuildSubject_SuccessNoSpamSignals(t *testing.T) {
	svc := New(nil, nil, "production", testLogger())
	subject := svc.buildSubject("success")

	assert.NotContains(t, subject, "[")
	assert.NotContains(t, subject, "]")
	assert.NotContains(t, strings.ToLower(subject), "successfully")
}

// TestBuildSubject_FailureNoBrackets ensures the failure subject is also
// stripped of brackets. The "FAILED" token is intentionally kept for
// operator visibility (short, ASCII-only, not a known bulk-mail trigger).
func TestBuildSubject_FailureNoBrackets(t *testing.T) {
	svc := New(nil, nil, "staging", testLogger())
	subject := svc.buildSubject("error")

	assert.NotContains(t, subject, "[")
	assert.NotContains(t, subject, "]")
}

// TestBuildEmailBody_StillContainsEnvironment confirms that removing the
// `[env]` prefix from the subject does NOT remove the environment label
// from the body — operators must still be able to tell production from
// staging at a glance.
func TestBuildEmailBody_StillContainsEnvironment(t *testing.T) {
	svc := New(nil, nil, "production", testLogger())
	body := svc.buildEmailBody(NotifyParams{Status: "success"})

	assert.Contains(t, body, "production")
}

func TestBuildEmailBody_SuccessContainsExpectedFields(t *testing.T) {
	svc := New(nil, nil, "production", testLogger())

	body := svc.buildEmailBody(NotifyParams{
		Status:         "success",
		Filename:       "backup_20260315.sql.gz",
		FileSize:       "42M",
		GdriveUploaded: true,
	})

	assert.Contains(t, body, "production")
	assert.Contains(t, body, "Success")
	assert.Contains(t, body, "backup_20260315.sql.gz")
	assert.Contains(t, body, "42M")
	assert.Contains(t, body, "Yes")
	assert.Contains(t, body, "Dear Team,")
	assert.Contains(t, body, "Orgfin Team")
	assert.Contains(t, body, "Backup Notification")
}

func TestBuildEmailBody_ErrorContainsErrorMessage(t *testing.T) {
	svc := New(nil, nil, "staging", testLogger())

	body := svc.buildEmailBody(NotifyParams{
		Status:         "error",
		ErrorMessage:   "pg_dump failed: connection refused",
		GdriveUploaded: false,
	})

	assert.Contains(t, body, "staging")
	assert.Contains(t, body, "FAILED")
	assert.Contains(t, body, "pg_dump failed: connection refused")
	assert.Contains(t, body, "No")
	assert.Contains(t, body, "#dc3545") // error color
}

func TestBuildEmailBody_HTMLEscapesUserInput(t *testing.T) {
	svc := New(nil, nil, "test", testLogger())

	body := svc.buildEmailBody(NotifyParams{
		Status:       "error",
		Filename:     "<script>alert('xss')</script>",
		ErrorMessage: "Error: <b>bold</b>",
	})

	assert.Contains(t, body, "&lt;script&gt;")
	assert.Contains(t, body, "&lt;b&gt;bold&lt;/b&gt;")
	assert.NotContains(t, body, "<script>alert")
}

func TestBuildEmailBody_NoFilenameRow(t *testing.T) {
	svc := New(nil, nil, "test", testLogger())

	body := svc.buildEmailBody(NotifyParams{
		Status: "success",
	})

	assert.NotContains(t, body, "Filename")
}

func TestBuildEmailBody_NoFileSizeRow(t *testing.T) {
	svc := New(nil, nil, "test", testLogger())

	body := svc.buildEmailBody(NotifyParams{
		Status: "success",
	})

	assert.NotContains(t, body, "File Size")
}
