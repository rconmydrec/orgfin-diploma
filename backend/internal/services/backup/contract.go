// Package backup provides Kubernetes backup job creation by extracting
// the spec from an existing CronJob and creating a one-off Job.
//
// Invariants:
//   - Requires running inside a Kubernetes cluster (uses in-cluster config).
//   - Falls back to "default" namespace if not specified and not auto-detected.
//   - Created jobs have a 1-hour TTL after completion.
package backup

import "context"

// ServiceInterface defines the public API of the backup service.
type ServiceInterface interface {
	CreateBackupJob(ctx context.Context) (string, error)
}
