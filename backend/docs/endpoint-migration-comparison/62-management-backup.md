# Endpoint #62: GET /management/backup

**Status**: DIVERGED (Go uses K8s Jobs instead of async task queue)
**Date**: 2026-03-15

## Route Definition
- **Python**: `@router.get('/backup/')`
- **Go**: `g.GET("/backup", h.BackupDB)`

## Request
- Auth: required (both)
- No body or query params
- Requires admin privileges (email must be in admin list)

## Response
- **Python**: 200 OK with `{"message": "Backup of the database is requested"}`
- **Go**: 200 OK with `{"message": "Backup job created", "jobName": "<k8s-job-name>"}`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Not admin | 403 `"You are not authorized"` | 403 `"Admin access required"` |
| Not in K8s | N/A | 503 `"Backup is only available in Kubernetes environment"` |
| Internal error | 500 `"Unable to request a backup of the database"` | 500 `"Failed to create backup job"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Admin check**: Python checks `request.state.user['email'] in settings.ADMINS_NOTIFICATION_EMAILS`; Go uses `RequireAdmin` middleware that checks against `config.AdminEmails`.
2. **Backup execution**: Python dispatches a Celery task `make_db_backup.delay()`; Go creates a one-off Kubernetes Job from the existing `orgfin-api-go-backup` CronJob spec via `services/backup/`. This is a fundamental architectural difference — Go does not use an Asynq task for backups.
3. **K8s dependency**: Go requires a Kubernetes environment; returns 503 if the backup service is nil (non-K8s). Python has no such restriction.
4. **Error handling**: Python wraps backup dispatch in try/except; Go checks job creation error and returns 500 on failure.

## Notes
- Python dispatches backup via Celery (async task queue); Go creates a K8s Job directly.
- Go's backup runs in a separate K8s container (pg_dump + rclone), not in the application process.
- Go returns the created job name in the response for observability.
- Go adds `RequireActiveUser` middleware verification.
- Admin check in Go is handled by dedicated `RequireAdmin` middleware rather than inline logic.

## Tests
- **Python**: 5 tests (test_backup_success_admin, _forbidden_non_admin, _unauthorized, _regular_user_cannot_access, _internal_error)
- **Go integration tests**: 7 tests (TestBackupUnauthorized, TestBackupNonAdmin, TestBackupInvalidToken, TestBackupEmptyToken, TestBackupMultipleNonAdminUsers, TestBackupRequiresAuth, TestBackupMethodNotAllowed)
- **Go unit tests**: 8 tests (TestBackupDBUserNotActive, _UserRepoError, _UserDeleted, _NoEmail, _EmailWrongType, _NotAdmin, _Success, TestRegisterRoutes)
