# Endpoint #13: PUT `/accounts/set-archive-status`

**Status**: PORTED OK (minor inconsistencies)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /accounts/set-archive-status` | `PUT /accounts/set-archive-status` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:84` | `internal/handlers/accounts/handler.go:259` |

## Request

Both: PUT with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountId` | int, required | int, required (`validate:"required"`) | OK |
| `isArchived` | bool, required | bool | OK |

## Response

**Success**: 200 OK. Returns `true` (both).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Account not found | 500 | Unhandled NotFoundError (BUG) | 500 | "Account not found" | Both bad (500) |
| Access denied | 401 | "Access denied" | 401 | "Access denied" | EXACT |
| Internal error | 500 | "Failed to set archive status" | 500 | "Failed to set archive status" | EXACT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Validate user exists + is_active | NO | YES | Go BETTER (fixed) |
| Check is_deleted on account | NO | YES (`GetByID` filters `is_deleted = false`) | Go BETTER (fixed) |
| Check entity access (subscription) | NO (missing unlike other endpoints) | NO | OK (both skip) |
| Check user ownership | YES | YES | OK |
| Set archived_at on un-archive | YES (bug — sets to now()) | YES (same issue — sets to now()) | Same bug |
| Double get_account_details | YES (redundant second call) | NO (single call) | Go BETTER |

## Issues Found

### INFO — Status code inconsistency in Go

- `ErrAccountNotFound` returns 500 in SetArchiveStatus but 400 in other endpoints.
- `ErrAccessDenied` returns 401 in SetArchiveStatus but 400 in other endpoints.
- **Impact**: Pre-existing, documented. Not a regression.

### INFO — archived_at set on un-archive (both)

- Both Python and Go set `archived_at` to current time when un-archiving (`is_archived = false`).
- Should clear `archived_at` to `NULL` on un-archive.
- **Impact**: Minor data issue. Pre-existing in both.

## Tests

### Python Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `test_archive_status_internal_error` | `test_accounts_endpoints.py:805` | 500 |
| `test_archive_account_success` | `:832` | 200 |
| `test_unarchive_account_success` | `:853` | 200 |
| `test_archive_account_not_found` | `:877` | 500 (documents bug) |
| `test_archive_account_other_user` | `:898` | 401 |
| `test_archive_account_unauthorized` | `:924` | 422 |

### Go Integration Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `TestArchiveAccountSuccess` | `handler_test.go:834` | 200 |
| `TestUnarchiveAccountSuccess` | `:876` | 200 |
| `TestArchiveAccountOtherUser` | `:921` | 401 |
| `TestArchiveAccountUnauthorized` | `:957` | 401/422 |
| `TestArchiveAccountInvalidJSON` | `:1705` | 422 |
| `TestArchiveAccountNotFound` | `:1732` | 401/404/500 |
| `TestSetArchiveStatusInactiveUser` | `:2180` | 401 |

### Go Unit Tests (4 total)

| Test | File | Verifies |
|------|------|----------|
| `TestSetArchiveStatusNotFound` | `handler_unit_test.go:364` | 500 |
| `TestSetArchiveStatusAccessDenied` | `:386` | 401 |
| `TestSetArchiveStatusDBError` | `:408` | 500 |
| `TestSetArchiveStatusValidationError` | `:476` | 422 |
