# Endpoint #18: PUT `/transactions/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /transactions/` | `PUT /transactions/` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:198` | `internal/handlers/transactions/handler.go:175` |

## Request

Both: PUT with JSON body. Same fields as create + required `id`.

## Response

**Success**: 200 OK. Transaction response.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Transaction not found | 404 | "Transaction not found" | 404 | "Transaction not found" | EXACT |
| Invalid account | 422 | "Invalid account" | 422 | "Invalid account" | EXACT |
| Access denied | 403 | "Access denied" | 403 | "Access denied" | EXACT |
| Invalid category | 422 | "Invalid category" | — | No category validation | DIFFERENT |
| Internal error | 500 | "Unable to update transaction" | 500 | "Failed to update transaction" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Check is_deleted on transaction | NO | YES (`GetByID` filters `is_deleted = false`) | Go BETTER |
| Check entity access (subscription) | YES | NO | DIFFERENT |
| Fetch existing transaction | YES | YES | OK |
| Check ownership | YES | YES | OK |
| Validate account | YES | YES | OK |
| Validate category type | YES (income/expense match) | NO | Python BETTER |
| Handle transfer update | YES (full linked tx update) | NO (no transfer handling) | Python BETTER |
| Correct previous balance | YES (reverses old, applies new) | NO (recalculates all) | Different approach |
| Running balance recalculation | YES | YES (`recalculateBalances`) | OK |
| Update account_id in SQL | YES | YES (fixed in commit 13875e8) | OK |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check.
- **Impact**: Must be fixed.

### RESOLVED — account_id not updated in SQL

- **Go**: Previously `Update` SQL did not include `account_id` in SET clause. **Fixed in commit 13875e8** — `account_id` is now included in the UPDATE SET clause, so moving a transaction to a different account is correctly persisted.
- **Python**: Updates account via the full model save.
- **Impact**: Resolved. No action needed.

### RESOLVED — Transfer update now handled in Go

- **Python**: Full transfer update support (updates both linked transactions, recalculates both accounts).
- **Go**: Full transfer update support added. Handles field mirroring, target account changes, cross-currency transfers, transfer↔regular conversion, and self-transfer rejection.
- **Impact**: Resolved. Feature parity with Python.

## Tests

### Python Tests (11 total)

Various: success, not found, invalid category, access denied, template, entity access.

### Go Integration Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `TestUpdateTransaction` | `handler_test.go:1182` | 200 |
| `TestUpdateTransactionChangeCategory` | `:1225` | Category change |
| `TestUpdateTransactionAddNotes` | `:1274` | Notes |
| `TestUpdateTransactionNotFound` | `:1317` | 404 |
| `TestUpdateTransactionOtherUser` | `:1350` | 403 |
| `TestUpdateTransactionUnauthorized` | `:1397` | 401 |
| `TestUpdateTransactionMissingID` | `:1418` | 422 |

### Go Unit Tests (5 total)

Various: not found, invalid account, access denied, DB error, bind/validate errors.
