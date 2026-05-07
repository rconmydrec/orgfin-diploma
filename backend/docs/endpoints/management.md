# Management Endpoints

Administrative endpoints for system management operations such as database backups.

## Table of Contents

- [GET /management/backup](#get-managementbackup)

---

## GET /management/backup

**Auth**: Required (JWT) + Admin only
**Handler**: `internal/handlers/management/handler.go`

### Request

No request body or query parameters.

### Response

**Success**: HTTP 200

```json
{ "message": "Backup job created", "jobName": "orgfin-api-go-backup-manual-20260315-140000" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Not an admin | 403 | "Admin access required" |
| Not running in K8s | 503 | "Backup is only available in Kubernetes environment" |
| Job creation failure | 500 | "Failed to create backup job" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is not deleted or inactive.
- `RequireAdmin` middleware (applied at the route group level) verifies the user's email is in the `config.AdminEmails` list. Returns 403 with "Admin access required" if not.
- If the backup service is nil (non-K8s environment), returns 503.
- Creates a one-off Kubernetes Job from the existing `orgfin-api-go-backup` CronJob spec. The Job runs pg_dump + rclone in a separate container and has a TTL of 3600 seconds.

### Tests

Integration tests:
- `TestBackupUnauthorized` — verifies 401 without a token
- `TestBackupNonAdmin` — verifies 403 for authenticated non-admin users
- `TestBackupInvalidToken` — verifies 401 with a malformed token
- `TestBackupEmptyToken` — verifies 401 with an empty token
- `TestBackupMultipleNonAdminUsers` — verifies 403 for multiple non-admin accounts
- `TestBackupRequiresAuth` — verifies auth is enforced
- `TestBackupMethodNotAllowed` — verifies non-GET methods are rejected

Unit tests:
- `TestBackupDBSuccess` — verifies 200 for a valid admin user
- `TestRegisterRoutes` — verifies routes are correctly registered

Note: Admin access checks (email validation, non-admin rejection) are now handled by the `RequireAdmin` middleware and tested in `internal/middleware/admin_test.go`. User-active checks are handled by `RequireActiveUser` middleware and tested separately.
