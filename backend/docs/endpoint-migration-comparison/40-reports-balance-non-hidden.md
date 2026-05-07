# Endpoint #40: POST `/reports/balance/non-hidden/`

**Status**: DIFF (partial implementation)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /reports/balance/non-hidden/` | `POST /reports/balance/non-hidden/` |
| Auth | JWT (check_token) + free plan check | JWT (auth middleware) |
| File | `app/routes/reports.py:94` | `internal/handlers/reports/handler.go:277` |

## Request

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountIds` | `list[int]` (required) | `[]int` | OK |
| `balanceDate` | `date` (required) | `string` (required, `validate:"required"`) | OK |

## Response

**Success**: 200 OK. Array of balance report items (excluding hidden accounts).

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountId` | int | int | OK |
| `accountName` | str | string | OK |
| `accountTypeId` | int | int | OK |
| `currencyCode` | str | string | OK |
| `balance` | float | float64 | **DIFF** -- Python calculates at-date balance, Go returns current balance |
| `baseCurrencyBalance` | float | float64 | **DIFF** -- Python converts via exchange rate, Go returns same as balance |
| `baseCurrencyCode` | str | string | OK |
| `reportDate` | date | string | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Invalid body | 422 | Validation error | 422 | "Invalid request data" | OK |
| Missing balanceDate | 422 | Validation error | 422 | "Invalid request data" | OK |
| AccessDenied | 403 | "Access denied" | N/A | N/A | **DIFF** -- Go has no AccessDenied handling |
| Internal error | 500 | "Error generting report" | 500 | "Error generating report" | OK (Python has typo) |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** -- Go has is_active check |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | Go BETTER |
| Free plan enforcement | YES | NO | **DIFF** |
| Account IDs filter | Passes `account_ids` from request | Respects `accountIds` -- filters if provided | OK |
| Exclude hidden accounts | Via BalanceReportGenerator (uses account_ids to filter) | Via `IncludeHidden: false` in account query + `account.IsHidden` check in loop | OK |
| Balance calculation | BalanceReportGenerator: calculates at-date balance | Returns current `account.Balance` (TODO comment) | **DIFF** |
| Base currency conversion | Via exchange rates | Returns same value as balance (TODO comment) | **DIFF** |
| Exclude archived | Via generator logic | YES (`IncludeArchived: false`) | OK |
| AccessDenied error | Caught, returns 403 | Not implemented | **DIFF** |

## Notes

- This endpoint differs from `/balance/` in that it excludes hidden accounts. Both Python and Go implement this correctly.
- **Key Python behavior**: Unlike `/balance/` which passes `[]`, the `/non-hidden/` endpoint passes the actual `account_ids` from the request to `get_balance_report()`. The `BalanceReportGenerator` uses these IDs to filter which accounts to exclude (hidden ones).
- Go uses a two-layer approach: (1) passes `IncludeHidden: false` to the account repository query to exclude hidden accounts at the DB level, and (2) additionally checks `account.IsHidden` in the response loop as a safeguard.
- Go has the same TODO items as `/balance/`: balance calculation uses current balance instead of at-date balance, and base currency conversion is not implemented.
- Python enforces free plan compliance. Go does not.
- Go shares the `reports.Service.BalanceReport` method with the `/balance/` endpoint, using the `ExcludeHidden` boolean flag to differentiate behavior.

## Tests

### Python Tests (4 total)

| Test | Verifies |
|------|----------|
| `test_balance_report_non_hidden_success` | 200, list response |
| `test_balance_report_non_hidden_unauthorized` | 422 without token |
| `test_balance_report_non_hidden_access_denied` | 403 on AccessDenied |
| `test_balance_report_non_hidden_internal_error` | 500 on exception |

### Go Integration Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestBalanceReportNonHiddenSuccess` | 200 |
| `TestBalanceReportNonHiddenUnauthorized` | 401 without token |
| `TestBalanceReportNonHiddenInvalidBody` | 422 |
| `TestBalanceReportNonHiddenMissingDate` | 422 |
| `TestBalanceReportExcludesHiddenAccounts` | Hidden account not in response |

### Go Unit Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestBalanceReportNonHiddenUserError` | 500 on user DB error |
| `TestBalanceReportNonHiddenWithHiddenAccount` | Hidden account excluded |
| `TestBalanceReportNonHiddenUserRepoError` | 401 on user repo error |
| `TestBalanceReportNonHiddenUserNotActive` | 401 for inactive user |
| `TestBalanceReportNonHiddenUserError` (BalanceReport service path) | 500 |
