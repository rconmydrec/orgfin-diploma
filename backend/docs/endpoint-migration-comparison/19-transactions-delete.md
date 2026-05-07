# Endpoint #19: DELETE `/transactions/:id`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /transactions/{transaction_id}` | `DELETE /transactions/:id` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:242` | `internal/handlers/transactions/handler.go:228` |

## Request

Both: DELETE with transaction ID path parameter.

## Response

**Success**: 200 OK. Transaction response (Python returns deleted tx; Go returns pre-deletion tx).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Not found | 500 | "Unable to delete transaction" (swallowed 404) | 404 | "Transaction not found" | Go BETTER |
| Access denied | 403 | "Access denied" | 403 | "Access denied" | EXACT |
| Invalid transaction | 422 | "Invalid transaction" | — | — | — |
| Internal error | 500 | "Unable to delete transaction" | 500 | "Failed to delete transaction" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Check is_deleted on transaction | NO (can delete already-deleted) | YES (`GetByID` filters `is_deleted = false`) | Go BETTER |
| Check entity access (subscription) | YES | NO | DIFFERENT |
| Check ownership | YES | YES | OK |
| Soft delete | YES (`is_deleted = True`) | YES (`is_deleted = true`) | OK |
| Delete linked tx (transfer) | YES | YES | OK |
| Reverse balance | YES | YES (via recalculateBalances) | OK |
| Running balance recalculation | NO (Python doesn't recalculate after delete) | YES (`recalculateBalances`) | Go BETTER |
| Target account balance recalc | YES (reverses directly) | NO (known issue — linked tx already deleted) | Python BETTER |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check.
- **Impact**: Must be fixed.

### RESOLVED — Linked transaction balance recalculation

- **Go**: Target account balance is now correctly recalculated via `recalculateBalances` using the saved `linkedAccountID` (fetched before deletion). Budget update is also enqueued for the target account.
- **Python**: Reverses balance directly without re-fetching.
- **Impact**: Resolved. Feature parity with Python.

## Tests

### Python Tests (7 total)

Various: success, not found, access denied, entity access, other user.

### Go Integration Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `TestDeleteTransaction` | `handler_test.go:1447` | 200, GET returns 404 |
| `TestDeleteTransactionNotFound` | `:1483` | 404 |
| `TestDeleteTransactionOtherUser` | `:1502` | 403 |
| `TestDeleteTransactionUnauthorized` | `:1536` | 401 |
| `TestDeleteTransactionInvalidID` | `:1548` | 400 |

### Go Unit Tests (3 total)

Various: not found, access denied, DB error.
