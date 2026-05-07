# Endpoint #12: DELETE `/accounts/:id`

**Status**: PORTED OK (Go better)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /accounts/{account_id}` | `DELETE /accounts/:id` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:157` | `internal/handlers/accounts/handler.go:235` |

## Request

Both: DELETE with account ID path parameter. No body.

## Response

**Success**: 200 OK. `{"deleted": true}` (both).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Account not found | 500 | Unhandled NotFoundError (BUG) | 400 | "Account not found" | Go BETTER |
| Access denied | 400 | "Failed to delete account" | 400 | "Access denied" | Go BETTER |
| Already deleted | Returns success (no check) | 400 | "Account not found" (filtered) | Go BETTER |
| Internal error | 500 | Internal server error | 500 | "Failed to delete account" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Validate user exists + is_active | NO | YES | Go BETTER (fixed) |
| Check is_deleted on account | NO (already-deleted can be "re-deleted") | YES (`GetByID` filters `is_deleted = false`) | Go BETTER (fixed) |
| Check entity access (subscription) | YES | NO | DIFFERENT |
| Check user ownership | YES | YES | OK |
| Soft delete | YES (`is_deleted = True`) | YES (`is_deleted = true`) | OK |
| Handle NotFoundError | NO (bug — returns 500) | YES (returns 400) | Go BETTER |

## Issues Found

### FIXED (Go) — Python returns 500 for not-found account

- **Python**: Delete route catches `(InvalidAccount, AccessDenied)` but NOT `NotFoundError`. Non-existent account returns 500.
- **Go**: `ErrAccountNotFound` handled properly with 400.

### FIXED (Go) — is_deleted prevents double-delete

- **Python**: No `is_deleted` filter. Already-deleted accounts can be "deleted" again.
- **Go**: `GetByID` filters `is_deleted = false`. Already-deleted accounts return "Account not found".

## Tests

### Python Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `test_delete_account_entity_access_denied` | `test_accounts_endpoints.py:687` | 403 |
| `test_delete_account_success` | `:706` | 200, is_deleted=True in DB |
| `test_delete_account_not_found` | `:730` | 500 (documents bug) |
| `test_delete_account_other_user` | `:747` | 403 |
| `test_delete_account_unauthorized` | `:774` | 422 |
| `test_delete_account_access_denied_other_user` | `:780` | 400 |

### Go Integration Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `TestDeleteAccountSuccess` | `handler_test.go:553` | 200, is_deleted in DB |
| `TestDeleteAccountUnauthorized` | `:601` | 401/422 |
| `TestDeleteAccountNotFound` | `:615` | 400/404 |
| `TestDeleteAccountOtherUser` | `:639` | 400/403 |
| `TestDeleteAccountInvalidID` | `:1586` | 400/404 |
| `TestDeleteAccountInactiveUser` | `:2145` | 401 |

### Go Unit Tests (1 total)

| Test | File | Verifies |
|------|------|----------|
| `TestDeleteAccountDBError` | `handler_unit_test.go:340` | 500 |
