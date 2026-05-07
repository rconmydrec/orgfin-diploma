# Endpoint #17: GET `/transactions/:id`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /transactions/{transaction_id}` | `GET /transactions/:id` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:179` | `internal/handlers/transactions/handler.go:152` |

## Request

Both: GET with transaction ID as path parameter.

## Response

**Success**: 200 OK. Single transaction object.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Not found | 500 | "Unable to get transaction" (BUG) | 404 | "Transaction not found" | Go BETTER |
| Other user's tx | 500 | "Unable to get transaction" (BUG) | 403 | "Access denied" | Go BETTER |
| Non-numeric ID | — | (type error) | 400 | "Invalid transaction ID" | Go BETTER |
| Internal error | 500 | "Unable to get transaction" | 500 | "Failed to get transaction" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Check is_deleted on transaction | NO (can retrieve deleted txs) | YES (`WHERE t.is_deleted = false`) | Go BETTER (fixed) |
| Check entity access (subscription) | YES (swallowed as 500) | NO | DIFFERENT |
| Check ownership | YES (returns 403, caught as 500) | YES (returns 403) | Go BETTER |
| Base currency conversion | YES | YES | OK |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check. Inactive user can get transaction details.
- **Impact**: Must be fixed.

### FIXED (Go) — is_deleted filter on transaction

- **Python**: `filter_by(id=transaction_id)` without `is_deleted` check. Deleted transactions can be retrieved.
- **Go**: `WHERE t.is_deleted = false`. Returns 404 for deleted transactions.

### FIXED (Go) — Proper error status codes

- **Python**: Catches all exceptions as 500. Even 404/403 from service are swallowed.
- **Go**: Returns proper 404, 403, 400 for different error conditions.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_transaction_details_success` | 200 |
| `test_get_transaction_details_not_found` | 500 |
| `test_get_transaction_details_other_user` | 500 |

### Go Integration Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetTransactionDetails` | `handler_test.go:1025` | 200 |
| `TestGetTransactionDetailsWithAccount` | `:1060` | 200 with account |
| `TestGetTransactionDetailsNotFound` | `:1096` | 404 |
| `TestGetTransactionDetailsOtherUser` | `:1115` | 403 |
| `TestGetTransactionDetailsInvalidID` | `:1149` | 400 |
| `TestGetTransactionDetailsUnauthorized` | `:1168` | 401 |

### Go Unit Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestGetTransactionDetailsNotFound` | 404 |
| `TestGetTransactionDetailsAccessDenied` | 403 |
| `TestGetTransactionDetailsDBError` | 500 |
| `TestGetTransactionDetailsInvalidID` | 400 |
