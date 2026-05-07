# Endpoint #29: GET `/currencies/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /currencies/` | `GET /currencies/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/currencies.py:12` | `internal/handlers/currencies/handler.go:37` |

## Request

Both: GET with no parameters.

## Response

**Success**: 200 OK. Array of currencies.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `code` | str | string | OK |
| `name` | str | string | OK |

Python uses snake_case (no alias generator); Go uses camelCase but fields are simple.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Internal error | 500 | Unhandled | 500 | "Failed to get currencies" | Go BETTER |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter is_deleted | NO (no column) | NO (no column) | OK |
| Order by code | YES | YES | OK |
| Return all currencies | YES | YES | OK |

## Issues Found

### BUG — Missing is_active check

- **Go**: No `is_active` check. Handler calls repo directly.
- **Note**: This is reference data, but for consistency with other authenticated endpoints, is_active should be checked.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_currencies_success` | 200, list |
| `test_get_currencies_includes_major_currencies` | USD, EUR present |
| `test_get_currencies_unauthorized` | 422 |

### Go Integration Tests (10 total)

| Test | Verifies |
|------|----------|
| `TestGetCurrenciesSuccess` | 200, correct fields |
| `TestGetCurrenciesIncludesMajorCurrencies` | USD, EUR |
| `TestGetCurrenciesUnauthorized` | 401 |
| `TestGetCurrenciesHasValidStructure` | All have id, code, name |
| `TestGetCurrenciesReturnsNonEmpty` | >= 5 currencies |
| `TestGetCurrenciesWithGBP` | GBP present |
| `TestGetCurrenciesWithJPY` | JPY present |
| `TestGetCurrenciesWithUAH` | UAH present |
| `TestGetCurrenciesHasIDs` | All IDs positive |
| `TestGetCurrenciesCodeLength` | All codes 3 chars |

### Go Unit Tests (2 total)

| Test | Verifies |
|------|----------|
| `TestGetCurrenciesDBError` | 500 |
| `TestGetCurrenciesSuccess` | 200 |
