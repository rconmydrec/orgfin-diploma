# Endpoint #32: POST `/settings`

**Status**: DIFF - Validation logic differs (path dimension is PARITY)
**Date**: 2026-02-21
**Last Updated**: 2026-04-24

## Route Definition

| Aspect | Python | Go | Match |
|--------|--------|-----|-------|
| Path | `POST /settings` (no trailing slash) | `POST /settings` (no trailing slash) | OK |
| Auth | `check_token` dependency | JWT middleware (`user_id` from context) | OK |
| File | `app/routes/user_settings.py:59` | `internal/handlers/settings/handler.go:113` | n/a |

## Request

Both: POST with JSON body. Auth token required.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `language` | str (required) | string `json:"language"` | OK |
| `projectionEndDate` | str or None (optional) | *string `json:"projectionEndDate"` | OK |
| `projectionPeriod` | str or None (optional) | *string `json:"projectionPeriod"` | OK |

**Python schema** (`UserSettingsSchema`): Pydantic model with `language` (str), `projectionEndDate` (str|None), `projectionPeriod` (str|None).

**Go struct** (`SettingsRequest`): Has same three fields with JSON binding.

## Response

**Success**: 200 OK. Updated user settings object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int `json:"id"` | OK |
| `settings` | dict (JSON column, full model) | `SettingsData` object `json:"settings"` | OK |
| `settings.language` | str | string | OK |
| `settings.projectionEndDate` | str or None | *string | OK |
| `settings.projectionPeriod` | str or None | *string | OK |
| `user_id` | int (snake_case) | int `json:"userId"` (camelCase) | **DIFF** (casing) |
| `created_at` | datetime (snake_case) | string `json:"createdAt"` (camelCase) | **DIFF** (casing, format) |
| `updated_at` | datetime (snake_case) | string `json:"updatedAt"` (camelCase) | **DIFF** (casing, format) |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 422 | Validation error | 401 | Unauthorized | **DIFF** |
| Invalid JSON body | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Unknown settings key | 422 | "Unknown settings key: {key}" | N/A | N/A | **DIFF** (Go has no validation) |
| Missing settings key | 422 | "Missing settings key: {key}" | N/A | N/A | **DIFF** (Go has no validation) |
| Incorrect settings type | 422 | "Incorrect settings type: {key}" | N/A | N/A | **DIFF** (Go has no validation) |
| Error creating defaults | N/A | N/A | 500 | "Failed to create settings" | **DIFF** (Go auto-creates) |
| Internal error | 500 | "Unable to get user settings" | 500 | "Failed to update settings" | OK (different wording) |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** |
| User deleted | N/A | N/A | 401 | "User not activated" | **DIFF** |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| Check user active | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Check user deleted | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Validate settings keys | YES (validate_settings) | NO | **DIFF** |
| Check unknown keys | YES (UnknownSettingsKeyError) | NO | **DIFF** |
| Check missing keys | YES (MissingSettingsKeyError) | NO | **DIFF** |
| Check type correctness | YES (IncorrectSettingsTypeError) | NO | **DIFF** |
| Get existing settings | YES (query by user_id) | YES (via `service.UpdateSettings`) | OK |
| Auto-create default | NO (exception if not found) | YES (service get-or-create pattern) | **DIFF** |
| Update settings | YES (overwrite settings dict) | YES (set individual fields in service) | OK |
| Commit/persist | YES (db.commit) | YES (repo.Update via service) | OK |

## Architecture

Go handler is a thin HTTP adapter. It reads `user_id` from context, binds the request, constructs `UpdateParams`, and delegates to `service.UpdateSettings(userID, params)`. The service contains the get-or-create logic and field assignment. The handler checks for `ErrSettingsCreateFailed` to return a specific error message ("Failed to create settings"), while other errors get "Failed to update settings".

## Notes

- **Major difference**: Python has `validate_settings()` which checks against `existing_settings` template, validating that all keys are known, all required keys are present, and types match. Go has no such validation -- it accepts any values for the three fixed fields in `SettingsRequest`.
- Route path parity restored on 2026-04-24: Go now registers `POST /settings` (no trailing slash) to match the Python contract and the frontend call. Prior to the fix the Go route was `POST /settings/`, causing production 404s on every settings save.
- Go auto-creates default settings if none exist before updating, Python does not (would throw exception).
- Python returns the full model (including `user_id` in snake_case), Go returns a DTO with camelCase fields.
- Go adds `RequireActiveUser` middleware guard not present in Python.

## Tests

### Python Tests (7 total)
| Test | Verifies |
|------|----------|
| `test_save_settings_success` | 200 on valid settings save |
| `test_save_settings_invalid_language` | 422 on UnknownSettingsKeyError (mocked) |
| `test_save_settings_unauthorized` | 422 without auth |
| `test_save_settings_unknown_key_error` | 422 on unknown key (mocked) |
| `test_save_settings_missing_key_error` | 422 on missing key (mocked) |
| `test_save_settings_incorrect_type_error` | 422 on incorrect type (mocked) |
| `test_save_settings_generic_error` | 500 on generic error (mocked) |

### Go Integration Tests (7 total)
| Test | Verifies |
|------|----------|
| `TestUpdateSettingsSuccess` | 200 on valid settings update |
| `TestUpdateSettingsUnauthorized` | 401 without auth token |
| `TestUpdateSettingsWithProjection` | 200 with projectionEndDate and projectionPeriod |
| `TestUpdateSettingsInvalidBody` | 422 on invalid JSON body |
| `TestUpdateSettingsMultipleTimes` | Multiple sequential updates persist correctly |
| `TestUpdateSettingsCreatesDefaultWhenNotExists` | Auto-creates defaults then updates |
| `TestUpdateSettingsClearProjectionFields` | Set then clear projection fields with null |

### Go Unit Tests — Handler (4 total)
| Test | Verifies |
|------|----------|
| `TestUpdateSettingsDBError` | 500 when settings repo returns error |
| `TestUpdateSettingsCreateDefaultError` | 500 when creating default settings fails |
| `TestUpdateSettingsUpdateError` | 500 when repo.Update fails |
| `TestUpdateSettingsGetAfterCreateError` | 500 when re-fetching after default creation fails |

### Go Unit Tests — Service (6 total)
| Test | Verifies |
|------|----------|
| `TestUpdateSettings_Success` | Updates settings with new values |
| `TestUpdateSettings_AutoCreatePath` | Auto-creates defaults, then updates |
| `TestUpdateSettings_NilProjectionFields` | Clears projection fields when nil |
| `TestUpdateSettings_UpdateRepoError` | Propagates repo update errors |
| `TestUpdateSettings_GetOrCreateFailure` | Returns `ErrSettingsCreateFailed` |
| `TestUpdateSettings_FieldsAppliedCorrectly` | Verifies all three fields are set |
