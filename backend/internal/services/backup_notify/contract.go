// Package backup_notify provides business logic for sending backup completion
// notifications to admin users via email.
//
// Invariants:
//   - Individual enqueue failures are logged and skipped; not propagated.
//   - Returns 0 sent (not an error) if no admin emails are configured.
package backup_notify

import "github.com/hibiken/asynq"

// ServiceInterface defines the public API of the backup notification service.
type ServiceInterface interface {
	NotifyBackupComplete(params NotifyParams) (int, error)
}

// TaskEnqueuer is the interface for enqueuing async tasks.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// NotifyParams holds the parameters for a backup notification.
type NotifyParams struct {
	Status         string
	Filename       string
	FileSize       string
	ErrorMessage   string
	GdriveUploaded bool
}
