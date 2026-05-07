# Endpoint #8: GET `/accounts/`

**Status**: PORTED OK (identical behavior)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /accounts/` | `GET /accounts/` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:61` | `internal/handlers/accounts/handler.go:87` |
| Route reg | `app/main.py` | `internal/server/server.go:162-165` |

## Request

Both: GET with query parameters. No body.

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `includeHidden` | bool, default False | bool (strconv.ParseBool), default false | OK |
| `includeArchived` | bool, default False | bool (strconv.ParseBool), default false | OK |
| `archivedOnly` | bool, default False | bool (strconv.ParseBool), default false | OK |

## Response

**Success**: 200 OK (both). JSON array of account objects, or `[]` for empty.

Each account object has the same structure as create response (see #7).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic header validation | 401 | "Missing authorization header" | DIFFERENT |
| Invalid user | 400 | "Failed to get user accounts" | 400 | "Invalid user" | OK |
| Internal error | — | — | 500 | "Failed to get accounts" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Filter by user_id | YES (`Account.user_id`) | YES (`WHERE a.user_id = $1`) | OK |
| Exclude deleted (default) | YES (`is_deleted == False`, hardcoded) | YES (`AND a.is_deleted = false`) | OK |
| Exclude hidden (default) | YES (`is_hidden == False`) | YES (`AND a.is_hidden = false`) | OK |
| Exclude archived (default) | YES (`is_archived == False`) | YES (`AND a.is_archived = false`) | OK |
| `archivedOnly` mode | Only `is_archived == True` | Only `AND a.is_archived = true` | OK |
| `archivedOnly` skips `is_deleted` filter | YES (bug — includes deleted) | NO (fixed — adds `AND is_deleted = false`) | Go BETTER (fixed) |
| Sort order | `asc(Account.name)` | `ORDER BY a.name ASC` | OK |
| Compute `balanceInBaseCurrency` | YES (`calc_amount`) | YES (currency conversion) | OK |
| Empty list returns `[]` | YES | YES | OK |

## Issues Found

### FIXED (Go) — `archivedOnly` now filters `is_deleted`

- **Python**: When `archivedOnly=true`, query only checks `is_archived = true` without filtering `is_deleted`. Soft-deleted archived accounts appear in the list.
- **Go** (after fix): `archivedOnly` branch adds `AND a.is_archived = true AND a.is_deleted = false`.
- **Impact**: Go correctly excludes deleted accounts from archived-only lists.

## Tests

### Python Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `test_get_accounts_success` | `test_accounts_endpoints.py:253` | 200, finds test account |
| `test_get_accounts_empty_list` | `:270` | 200, empty array |
| `test_get_accounts_filter_hidden` | `:293` | Hidden excluded/included |
| `test_get_accounts_filter_archived` | `:324` | Archived excluded/included |
| `test_get_accounts_archived_only` | `:355` | Only archived returned |
| `test_get_accounts_unauthorized` | `:378` | 422 |
| `test_get_accounts_invalid_user` | `:235` | 400 |

### Go Integration Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountsSuccess` | `handler_test.go:298` | 200, finds test account |
| `TestGetAccountsEmptyList` | `:349` | 200, empty array |
| `TestGetAccountsUnauthorized` | `:382` | 401/422 |
| `TestGetAccountsFilterHidden` | `:1248` | Hidden excluded/included |
| `TestGetAccountsFilterArchived` | `:1320` | Archived excluded/included |

### Go Unit Tests (3 total) -- NOTE: counted from analysis

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountsInvalidUser` | `handler_unit_test.go:159` | ErrInvalidUser → 400 |
| `TestGetAccountsDBError` | `:179` | Generic error → 500 |
| `TestGetAccountsArchivedOnly` (integration) | `handler_test.go:1392` | archivedOnly returns only archived |
