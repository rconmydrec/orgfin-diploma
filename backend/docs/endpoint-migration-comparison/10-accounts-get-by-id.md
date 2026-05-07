# Endpoint #10: GET `/accounts/:id`

**Status**: PORTED OK (minor differences)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /accounts/{account_id}` | `GET /accounts/:id` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:111` | `internal/handlers/accounts/handler.go:136` |
| Route reg | `app/main.py` | `internal/server/server.go:162-165` |

## Request

Both: GET with account ID as path parameter. No body, no query params.

## Response

**Success**: 200 OK (both). Single account object (same structure as #7/#8).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Account not found | 404 | "Account not found" | 404 | "Account not found" | EXACT |
| Other user's account | 400 | "Failed to get account details" | 400 | "Access denied" | OK |
| Free plan + inactive entity | 403 | "Access to account is restricted..." | — | No subscription check | DIFFERENT |
| Non-numeric ID | — | (type error) | 400 | "Invalid account ID" | Go BETTER |
| Internal error | 500 | "Internal server error" | 500 | "Failed to get account" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Parse account ID | Path param (int, auto-validated) | `strconv.Atoi` (manual) | OK |
| Check entity access (subscription) | YES (`check_entity_access` — free plan check) | NO | DIFFERENT |
| Get account from DB | `filter_by(id=account_id)` — NO `is_deleted` filter | `WHERE a.id = $1 AND a.is_deleted = false` | Go BETTER (fixed) |
| Check user ownership | YES (`account.user_id != user_id` → AccessDenied) | YES (`account.UserID != userID` → ErrAccessDenied) | OK |
| Compute `balanceInBaseCurrency` | NO (not computed in detail view) | YES (computed via currency conversion) | Go BETTER |
| Error for ownership violation | 400 (generic "Failed to get account details") | 400 "Access denied" | Go BETTER (specific) |

## Issues Found

### FIXED (Go) — `is_deleted` now filtered on single account

- **Python**: `GET /accounts/{account_id}` does NOT filter `is_deleted`. A soft-deleted account can be retrieved by ID.
- **Go** (after fix): Repository `GetByID` adds `AND a.is_deleted = false`. Deleted accounts return 404 "Account not found".
- **Impact**: Go correctly prevents access to deleted accounts. Test added: `TestGetAccountDetailsDeletedAccount`.

### INFO — `balanceInBaseCurrency` computed in Go but not Python

- **Python**: Does NOT call `calc_amount()` in `get_account_details`. Returns 0.0.
- **Go**: Computes `BalanceInBaseCurrency` via currency conversion.
- **Impact**: Go provides more complete data. Not a regression.

### INFO — Subscription/entity access check not in Go

- **Python**: Calls `check_entity_access` which, for FREE plan users, checks if the account is `is_active`. Returns 403 if not.
- **Go**: No subscription-level entity access check for single account view.
- **Impact**: In Go, free plan users can view inactive accounts. This is less restrictive.

## Tests

### Python Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `test_get_account_details_success` | `test_accounts_endpoints.py:449` | 200, correct fields |
| `test_get_account_details_not_found` | `:468` | 404 |
| `test_get_account_details_access_denied_other_user` | `:481` | 400 |
| `test_get_account_details_other_user` | `:502` | 403 |
| `test_get_account_details_unauthorized` | `:530` | 422 |
| `test_get_account_entity_access_denied` | `:414` | 403 |
| `test_get_account_internal_error` | `:428` | 500 |

### Go Integration Tests (4 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountDetailsSuccess` | `handler_test.go:444` | 200, correct fields |
| `TestGetAccountDetailsNotFound` | `:491` | 404 |
| `TestGetAccountDetailsOtherUser` | `:515` | 400/403 |
| `TestGetAccountDetailsInvalidID` | `:1562` | 400/404 |

### Go Unit Tests (1 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountDetailsDBError` | `handler_unit_test.go:222` | Generic error → 500 |
