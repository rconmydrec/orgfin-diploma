# Endpoint #42: GET `/reports/diagram/{type}/{start}/{end}`

**Status**: OK
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /reports/diagram/{diagram_type}/{start_date}/{end_date}` | `GET /reports/diagram/:diagram_type/:start_date/:end_date` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| File | `app/routes/reports.py:149` | `internal/handlers/reports/handler.go:519` |

## Request

**Method**: GET with path parameters. No body.

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `diagram_type` | str (path param) | string (path param via c.Param) | OK |
| `start_date` | str (path param, parsed to date) | string (path param, used as-is for filter) | DIFF — Python parses to date, Go passes as string |
| `end_date` | str (path param, parsed to date) | string (path param, used as-is for filter) | DIFF — same |

## Response

**Success**: 200 OK. Chart.js-compatible diagram data.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `labels` | list[str] | []string | OK |
| `data` | list[float] | []float64 | OK |
| `currency` | str | string | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Unauthorized | 422 (missing header) | FastAPI validation | 401 | JWT middleware | DIFF |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF — Go adds is_active check |
| Invalid dates | 500 (ValueError from strptime) | "Error generting report" | N/A (no date parsing) | N/A | DIFF — Go does not validate date format |
| Internal error (diagram) | 500 | "Error generting report" | 500 | "Error generating report" | OK (Python has typo) |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | DIFF — Go is stricter |
| Free plan enforcement | YES | NO | DIFF — Go lacks subscription check |
| Date parsing | datetime.strptime (raises ValueError on bad format) | No parsing, strings passed to DB filter | DIFF — Python validates, Go does not |
| Get expenses | Calls get_expenses_by_categories (full report) | Direct: gets categories, transactions, aggregates in handler | DIFF — Python reuses report service, Go does inline |
| hide_empty_categories | Always False | N/A (not applicable, zero amounts skipped later) | OK |
| Aggregation | prepare_data aggregates children into parents, combine_small_categories groups <2% | Go aggregates by individual category (no parent rollup), skips zero amounts | DIFF — Python groups children under parents and merges small categories; Go does not |
| "Other" bucket | Yes, categories <2% of total combined into "Other" | No | DIFF — Go does not combine small categories |
| Sort | Sorted by amount descending | No sorting (map iteration order) | DIFF |
| diagram_type param | Passed to get_diagram (historically used for image type) | Logged but not used in logic | OK (effectively unused in both) |
| Currency | From user.base_currency | From user.BaseCurrencyID -> currencyRepo.GetByID | OK |
| Currency conversion | Uses calc_amount via ExpensesReportGenerator | Uses pre-stored baseCurrencyAmount | DIFF |

## Notes

- Python has a two-step process: first calls `get_expenses_by_categories` to get the full expenses report, then calls `get_diagram` which calls `prepare_data` + `combine_small_categories` to aggregate children into parents and group small (<2%) categories into "Other".
- Go does a simpler aggregation: fetches expense transactions, groups by individual category ID, and returns labels/data directly. No parent-child rollup and no "Other" bucket.
- The `diagram_type` path parameter is effectively unused in both implementations (Python's old matplotlib code was removed; Go logs it but ignores it).
- Python validates date format via `datetime.strptime` and returns 500 on invalid dates. Go passes strings directly to the DB filter, so invalid dates would cause DB-level errors or empty results.
- The response structure (labels, data, currency) is identical between Python and Go.

## Tests

### Python Tests (4 total)

| Test | Verifies |
|------|----------|
| `test_diagram_success` | 200 |
| `test_diagram_invalid_dates` | 500 (ValueError from date parsing) |
| `test_diagram_unauthorized` | 422 (missing auth header) |
| `test_diagram_internal_error` | 500 (mocked get_diagram exception) |

### Go Integration Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestDiagramSuccess` | 200 |
| `TestDiagramUnauthorized` | 401 (no auth) |
| `TestDiagramWithTransactions` | 200 with labels and data populated |
| `TestDiagramBarChart` | 200 with bar chart type |

### Go Unit Tests (8 total)

| Test | Verifies |
|------|----------|
| `TestDiagramUserError` | 500 (user repo error on second call) |
| `TestDiagramCurrencyError` | 500 (currency repo error) |
| `TestDiagramCategoryError` | 500 (category repo error) |
| `TestDiagramTransactionError` | 500 (transaction repo error) |
| `TestDiagramWithZeroExpenses` | 200 with zero amounts |
| `TestDiagramExcludedAndIncomeTransactions` | Excluded/income/nil-category filtered |
| `TestDiagramUserRepoError` | 401 (user repo error) |
| `TestDiagramUserNotActive` | 401 (user not active) |
