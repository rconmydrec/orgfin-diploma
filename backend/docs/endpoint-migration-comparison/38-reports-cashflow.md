# Endpoint #38: POST `/reports/cashflow/`

**Status**: DIFF (structural differences in logic)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /reports/cashflow/` | `POST /reports/cashflow/` |
| Auth | JWT (check_token) + free plan check | JWT (auth middleware) |
| File | `app/routes/reports.py:37` | `internal/handlers/reports/handler.go:157` |

## Request

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `startDate` | `datetime \| None` (optional) | `*common.DateOnly` (optional, accepts YYYY-MM-DD, datetime, or RFC3339; from `internal/handlers/common/date.go`) | OK |
| `endDate` | `datetime \| None` (optional) | `*common.DateOnly` (optional, accepts YYYY-MM-DD, datetime, or RFC3339; from `internal/handlers/common/date.go`) | OK |
| `period` | `str` (required) | `string` (required, `validate:"required"`) | OK |

## Response

**Success**: 200 OK. Cash flow report object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `currency` | str (base currency code) | string (base currency code) | OK |
| `totalIncome` | dict[str, float] (period -> amount) | map[string]float64 (period -> amount) | OK |
| `totalExpenses` | dict[str, float] (period -> amount) | map[string]float64 (period -> amount) | OK |
| `netFlow` | dict[str, float] (period -> amount) | map[string]float64 (period -> amount) | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Invalid body | 422 | Validation error | 422 | "Invalid request data" | OK |
| Missing period | 422 | Validation error | 422 | "Invalid request data" | OK |
| AccessDenied | 403 | "Access denied" | N/A | N/A | **DIFF** -- Go has no AccessDenied handling |
| Internal error | 500 | "Error generting report" | 500 | "Error generating report" | OK (Python has typo) |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** -- Go has is_active check |
| Free plan limit | Enforced (dependency) | N/A | N/A | N/A | **DIFF** -- Go has no plan enforcement |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | Go BETTER |
| Free plan enforcement | YES | NO | **DIFF** |
| Report generation | CashFlowReportGenerator class | Inline in handler | **DIFF** -- architecture |
| Period aggregation | In generator class | Inline: "monthly" or "daily", default "monthly" | OK (same logic) |
| Base currency amount | Uses service/generator | Uses baseCurrencyAmount field, falls back to amount | OK |
| Excludes transfers | Via generator | `tx.IsTransfer` check | OK |
| Excludes report-excluded | Via generator | `tx.ExcludeFromReports` check | OK |
| Transaction limit | No explicit limit | Hardcoded 10000 | **DIFF** |
| AccessDenied error | Caught, returns 403 | Not implemented | **DIFF** |

## Notes

- Python delegates to `CashFlowReportGenerator` class with dedicated logic. Go implements the aggregation logic inline in the handler.
- Go has a hardcoded limit of 10000 transactions, which could miss data for users with very many transactions. Python has no explicit limit.
- Python enforces free plan compliance via `enforce_free_plan_compliance` dependency. Go does not.
- Python catches `AccessDenied` and returns 403. Go does not have this error handling path.
- Go has an is_active/is_deleted user check that Python does not have.
- Both support "monthly" and "daily" periods. Unknown periods default to "monthly" in Go.
- Python response message has a typo: "Error generting report" (missing 'a').

## Tests

### Python Tests (6 total)

| Test | Verifies |
|------|----------|
| `test_cashflow_report_success` | 200, `currency` in response |
| `test_cashflow_report_different_periods` | 200 for monthly and daily |
| `test_cashflow_report_empty_date_range` | 200 for range with no transactions |
| `test_cashflow_report_unauthorized` | 422 without token |
| `test_cashflow_report_access_denied` | 403 on AccessDenied |
| `test_cashflow_report_internal_error` | 500 on exception |

### Go Integration Tests (10 total)

| Test | Verifies |
|------|----------|
| `TestCashFlowReportSuccess` | 200, `currency` present |
| `TestCashFlowReportDailyPeriod` | 200 with daily period |
| `TestCashFlowReportUnauthorized` | 401 without token |
| `TestCashFlowReportWithTransactions` | 200, `totalExpenses` present |
| `TestCashFlowReportMissingPeriod` | 422 |
| `TestCashFlowReportInvalidBody` | 422 |
| `TestCashFlowReportWithIncomeTransaction` | 200, `totalIncome` present |
| `TestCashFlowReportDefaultPeriod` | 200 with unknown period |
| `TestCashFlowReportWithMixedTransactions` | 200, all fields present |
| `TestCashFlowReportNoDates` | 200 without dates |

### Go Unit Tests (9 total)

| Test | Verifies |
|------|----------|
| `TestCashFlowReportUserError` | 500 on user DB error |
| `TestCashFlowReportCurrencyError` | 500 on currency DB error |
| `TestCashFlowReportTransactionError` | 500 on transaction DB error |
| `TestCashFlowReportExcludeFromReports` | Excluded transactions skipped |
| `TestCashFlowReportDailyPeriod` (unit) | Daily period key format |
| `TestCashFlowReportDefaultPeriod` (unit) | Unknown period defaults to monthly |
| `TestCashFlowReportUserNotActive` | 401, "User not activated" |
| `TestCashFlowReportUserRepoError` | 401 on user repo error |
| `TestCashFlowReportUserDeleted` | 401 for deleted user |
