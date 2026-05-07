package workers

import (
	"github.com/go-budget/backend/internal/workers/tasks"
)

// TaskEnqueuer is a type alias for the canonical interface defined in the
// tasks package. This alias exists so that consumers (server.go, handlers,
// services) can reference workers.TaskEnqueuer without importing the tasks
// subpackage directly.
type TaskEnqueuer = tasks.TaskEnqueuer
