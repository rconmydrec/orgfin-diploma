# Endpoint #16: GET `/transactions/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /transactions/` | `GET /transactions/` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:83` | `internal/handlers/transactions/handler.go:91` |

## Request

Both: GET with query parameters.

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `page` | int, default 1 | int, default 1 | OK |
| `per_page` | int, default 30 | int, default 30 (cap 50) | OK |
| `types` | comma-separated | comma-separated (first only) | Go DIFFERENT (first type only) |
| `categories` | comma-separated ints | comma-separated ints | OK |
| `accounts` | comma-separated ints | comma-separated ints | OK |
| `currencies` | comma-separated ints | — | Go MISSING |
| `from_date` | string YYYY-MM-DD | string | OK |
| `to_date` | string YYYY-MM-DD | string | OK |

## Response

**Success**: 200 OK. JSON array of transaction objects.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Unknown filter | 422 | "Incorrect filter: {key}" | — | Silently ignored | DIFFERENT |
| Internal error | 500 | "Unable to get transactions" | 500 | "Failed to get transactions" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter is_deleted transactions | YES (`Transaction.is_deleted == False`) | YES (`t.is_deleted = false`) | OK |
| Filter by account is_active (free plan) | YES (`Account.is_active == True`) | NO | DIFFERENT |
| Sort order | `date_time DESC` | `date_time DESC` | OK |
| Pagination | offset/limit | offset/limit | OK |
| Base currency conversion | YES | YES | OK |
| Type filter: multiple types | YES (OR logic) | NO (first type only) | Python BETTER |
| Currency filter | YES | NO | Python has more |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check. Inactive user can list transactions.
- **Impact**: Must be fixed.

### INFO — Only first type filter used in Go

- **Python**: Supports multiple types with OR logic (e.g., `types=expense,income`).
- **Go**: Uses only the first type value.
- **Impact**: Pre-existing limitation.

## Tests

### Python Tests (5 total)

| Test | Verifies |
|------|----------|
| `test_get_transactions_success` | 200, list |
| `test_get_transactions_filter_by_account` | Account filter |
| `test_get_transactions_filter_by_date_range` | Date filter |
| `test_get_transactions_empty_list` | Empty array |
| `test_get_transactions_unauthorized` | 422 |

### Go Integration Tests (9 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetTransactionsList` | `handler_test.go:679` | 200, list |
| `TestGetTransactionsEmpty` | `:715` | Empty array |
| `TestGetTransactionsPagination` | `:741` | page/per_page |
| `TestGetTransactionsFilterByAccount` | `:793` | Account filter |
| `TestGetTransactionsFilterByCategory` | `:836` | Category filter |
| `TestGetTransactionsFilterByDateRange` | `:883` | Date filter |
| `TestGetTransactionsFilterByType` | `:920` | Type filter |
| `TestGetTransactionsFilterByMultipleAccounts` | `:961` | Multiple accounts |
| `TestGetTransactionsUnauthorized` | `:1011` | 401 |

### Go Unit Tests (1 total)

| Test | Verifies |
|------|----------|
| `TestGetTransactionsDBError` | 500 |
