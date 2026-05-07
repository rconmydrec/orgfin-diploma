# Endpoint #31: GET `/settings`

**Status**: OK
**Date**: 2026-02-21
**Last Updated**: 2026-04-24

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /settings` (no trailing slash) | `GET /settings` (no trailing slash) |
| Auth | `check_token` dependency | JWT middleware (`user_id` from context) |
| File | `app/routes/user_settings.py:45` | `internal/handlers/settings/handler.go:92` |

Route path parity restored on 2026-04-24. Prior to the fix the Go route was registered as `GET /settings/` (trailing slash), while the frontend and Python contract used `GET /settings` (no slash). The Go registration now matches the Python contract.

## Request

Both: GET with no body. Auth token required.

| Aspect | Python | Go | Match |
|--------|--------|-----|-------|
| Auth header | `Authorization: Bearer <token>` | `auth-token: <token>` | **DIFF** (framework convention) |

## Response

**Success**: 200 OK. User settings object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int (from model) | int `json:"id"` | OK |
| `settings` | dict (JSON column) | `SettingsData` object `json:"settings"` | OK |
| `settings.language` | str | string `json:"language"` | OK |
| `settings.projectionEndDate` | str or None | *string `json:"projectionEndDate"` | OK |
| `settings.projectionPeriod` | str or None | *string `json:"projectionPeriod"` | OK |
| `user_id` | int (from model, snake_case) | int `json:"userId"` (camelCase) | **DIFF** (casing) |
| `created_at` | datetime (from model, snake_case) | string `json:"createdAt"` (camelCase, formatted) | **DIFF** (casing, format) |
| `updated_at` | datetime (from model, snake_case) | string `json:"updatedAt"` (camelCase, formatted) | **DIFF** (casing, format) |

Python returns the full SQLAlchemy model directly (auto-serialized with all columns). Go returns a curated `SettingsResponse` DTO with explicit field mapping and `2006-01-02T15:04:05` time format.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 422 | Validation error (FastAPI) | 401 | Unauthorized (middleware) | **DIFF** |
| Internal error | 500 | "Unable to get user settings" | 500 | "Failed to get settings" | OK (different wording) |
| User not active | N/A (no check) | N/A | 401 | "User not activated" | **DIFF** (Go adds is_active check) |
| User deleted | N/A (no check) | N/A | 401 | "User not activated" | **DIFF** (Go adds is_deleted check) |
| No settings exist | Exception (500) | "Unable to get user settings" | 200 | Creates default and returns | **DIFF** (Go auto-creates defaults) |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| Check user active | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Check user deleted | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Get settings by user ID | YES | YES (via `service.GetSettings`) | OK |
| Auto-create default settings | NO (separate endpoint/logic) | YES (service get-or-create pattern) | **DIFF** |
| Default language | N/A | "en" (via CreateDefault) | **DIFF** |

## Architecture

Go handler is a thin HTTP adapter. It reads `user_id` from context and delegates to `service.GetSettings(userID)`. The service contains the get-or-create logic (auto-create defaults on `sql.ErrNoRows`). The handler only maps the returned model to a `SettingsResponse` DTO.

## Notes

- Python returns the raw SQLAlchemy model which auto-serializes with snake_case fields. Go returns a structured DTO with camelCase fields.
- Go adds `RequireActiveUser` middleware guard that Python does not have. This means inactive/deleted users get 401 in Go but can still access settings in Python.
- Go auto-creates default settings when none exist (returning 200), while Python would raise an exception (returning 500 or relying on separate initialization).
- Time format in Go is `2006-01-02T15:04:05` (no timezone), Python returns full datetime with timezone.

## Tests

### Python Tests (4 total)
| Test | Verifies |
|------|----------|
| `test_get_settings_success` | 200 with settings data |
| `test_get_settings_default_values` | Default settings for new user |
| `test_get_settings_unauthorized` | 422 without auth |
| `test_get_settings_error` | 500 on internal error (mocked) |

### Go Integration Tests (5 total)
| Test | Verifies |
|------|----------|
| `TestGetSettingsSuccess` | 200 with settings data for authenticated user |
| `TestGetSettingsUnauthorized` | 401 without auth token |
| `TestGetSettingsReturnsDefaults` | Default settings structure for new user |
| `TestGetSettingsCreatesDefaultWhenNotExists` | Auto-creates defaults when settings deleted |
| `TestGetSettingsResponseStructure` | All response fields present (id, userId, settings, createdAt, updatedAt) |

### Go Unit Tests — Handler (3 total)
| Test | Verifies |
|------|----------|
| `TestGetSettingsDBError` | 500 when settings repo returns error |
| `TestGetSettingsCreateDefaultError` | 500 when creating default settings fails |
| `TestGetSettingsGetAfterCreateError` | 500 when re-fetching after default creation fails |

### Go Unit Tests — Service (6 total)
| Test | Verifies |
|------|----------|
| `TestGetSettings_Success` | Returns settings for a user |
| `TestGetSettings_AutoCreate` | Auto-creates defaults on `sql.ErrNoRows` |
| `TestGetSettings_CreateDefaultFails` | Returns `ErrSettingsCreateFailed` |
| `TestGetSettings_ReFetchAfterCreateFails` | Returns `ErrSettingsNotFound` |
| `TestGetSettings_NonErrNoRowsDBError` | Propagates non-ErrNoRows errors |
| `TestGetSettings_CorrectUserIDPassed` | Correct user ID forwarded to repo |
