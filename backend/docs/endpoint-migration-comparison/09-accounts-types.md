# Endpoint #9: GET `/accounts/types/`

**Status**: PORTED OK (Go improvement: filters deleted types)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /accounts/types/` | `GET /accounts/types/` |
| Auth | `check_token` (router-level) | `RequireAuth` middleware |
| File | `app/routes/accounts.py:106` | `internal/handlers/accounts/handler.go:117` |
| Route reg | `app/main.py` | `internal/server/server.go:162-165` |

## Request

Both: GET with no query parameters, no body. Only requires auth header.

## Response

**Success**: 200 OK (both). JSON array of account type objects.

```json
[
  {"id": 1, "type_name": "Cash", "is_credit": false},
  {"id": 2, "type_name": "Credit Card", "is_credit": true}
]
```

Both use **snake_case** for `type_name` and `is_credit`.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| DB error | (unhandled) | — | 500 | "Failed to get account types" | Go BETTER |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Query | `db.query(AccountType).all()` — NO filter | `WHERE is_deleted = false ORDER BY id` | Go BETTER |
| Filter `is_deleted` | NO | YES | Go BETTER |
| User-specific data | NO (global) | NO (global) | OK |
| Error handling | No try/except | Returns 500 on error | Go BETTER |

## Issues Found

### FIXED (Go) — Deleted account types filtered

- **Python**: Returns ALL account types including soft-deleted ones.
- **Go**: Filters `WHERE is_deleted = false`.
- **Impact**: Go correctly excludes deleted types from the list.

### INFO — No error handling in Python

- **Python**: No try/except around the DB query. If the DB fails, it would be an unhandled 500.
- **Go**: Properly returns 500 with "Failed to get account types".

## Tests

### Python Tests (2 total)

| Test | File | Verifies |
|------|------|----------|
| `test_get_account_types_success` | `test_accounts_endpoints.py:388` | 200, non-empty, has id/type_name/is_credit |
| `test_get_account_types_unauthorized` | `:403` | 422 |

### Go Integration Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountTypesSuccess` | `handler_test.go:397` | 200, non-empty, has id/type_name/is_credit |
| `TestGetAccountTypesUnauthorized` | `:1760` | 401/422 |
| `TestGetAccountTypesInvalidToken` | `:1774` | 401 |

### Go Unit Tests (1 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetAccountTypesDBError` | `handler_unit_test.go:201` | Error → 500 |
