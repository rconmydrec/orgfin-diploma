# Endpoint #43: POST `/reports/expenses-data/`

**Status**: OK
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /reports/expenses-data/` | `POST /reports/expenses-data/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| File | `app/routes/reports.py:182` | `internal/handlers/reports/handler.go:614` |

## Request

**Method**: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `startDate` | date (required, camelCase alias) | string (required, validate:"required") | DIFF — Python uses `date` type, Go uses plain string |
| `endDate` | date (required, camelCase alias) | string (required, validate:"required") | DIFF — same |
| `categories` | list[int] (optional, default []) | []int (optional) | OK |
| `hideEmptyCategories` | bool (optional, default False) | bool (optional) | OK |

**Note**: Python uses `ExpensesReportInputSchema` (same as #41). Go reuses `ExpensesReportRequest` (same as #41).

## Response

**Success**: 200 OK. Array of aggregated expense data items.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `category_id` / `categoryId` | int | int (`categoryId`) | DIFF — Python uses snake_case key, Go uses camelCase |
| `label` | str | string | OK |
| `amount` | Decimal (number) | float64 | OK |
| `categoryName` | N/A | string | DIFF — Go adds extra field |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Unauthorized | 422 (missing header) | FastAPI validation | 401 | JWT middleware | DIFF |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF — Go adds is_active check |
| Invalid body | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Missing dates | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Internal error | N/A (no try/except in Python) | Unhandled 500 | 500 | "Error generating report" | DIFF — Python has no explicit error handling |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | DIFF — Go is stricter |
| Free plan enforcement | YES | NO | DIFF |
| Get expenses | Calls get_expenses_by_categories (reuses #41 report) | Direct: fetches categories, transactions in handler | OK (different impl, same data source) |
| Aggregation (prepare_data) | Aggregates child categories into parent totals | Aggregates child categories into parent totals | OK |
| "Other" bucket | Yes, <2% threshold via combine_small_categories | Yes, <2% threshold via inline code | OK |
| Sort | Sorted by amount descending | Sorted by amount descending (bubble sort) | OK |
| Currency conversion | Uses calc_amount via ExpensesReportGenerator | Uses pre-stored baseCurrencyAmount | DIFF |
| hide_empty_categories | Always False (hardcoded) | Used from request (but not applied since data is aggregated by parents) | DIFF — Python hardcodes False |
| Zero total handling | Returns all aggregated (no special case) | Returns `aggregated` without "Other" processing | OK |
| Response fields | `category_id`, `label`, `amount` | `categoryId`, `label`, `amount`, `categoryName` | DIFF — Go adds `categoryName`, uses camelCase |

## Notes

- This endpoint returns aggregated expense data suitable for chart rendering on the frontend. Both implementations perform parent-category rollup and small-category merging.
- Python's `get_expenses_diagram_data` calls `get_expenses_by_categories` then `prepare_data` + `combine_small_categories`. Go implements this logic inline in the handler.
- Both use a 2% threshold for combining small categories into "Other".
- Both sort the final result by amount descending.
- Go adds a `categoryName` field that Python does not return.
- Python has no explicit error handling in the route handler (no try/except). Errors propagate as unhandled 500s. Go wraps all repo calls with error returns.
- Go does not use a separate currency for base conversion; it relies on the `baseCurrencyAmount` field pre-calculated on each transaction.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_expenses_data_success` | 200 |
| `test_expenses_data_unauthorized` | 422 (missing auth header) |
| `test_expenses_data_internal_error` | 500 (mocked exception) |

### Go Integration Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestExpensesDataSuccess` | 200 |
| `TestExpensesDataUnauthorized` | 401 (no auth) |
| `TestExpensesDataMissingDates` | 422 (missing required fields) |
| `TestExpensesDataWithTransactions` | 200 with actual expense data |
| `TestExpensesDataInvalidBody` | 422 (malformed JSON) |

### Go Unit Tests (6 total)

| Test | Verifies |
|------|----------|
| `TestExpensesDataCategoryError` | 500 (category repo error) |
| `TestExpensesDataTransactionError` | 500 (transaction repo error) |
| `TestExpensesDataWithZeroAmount` | 200 with zero amount transactions |
| `TestExpensesDataExcludedAndIncomeTransactions` | Excluded/income/nil-category filtered |
| `TestExpensesDataUserRepoError` | 401 (user repo error) |
| `TestExpensesDataUserNotActive` | 401 (user not active) |
