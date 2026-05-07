# Endpoint #39: POST `/reports/balance/`

**Status**: DIFF (partial implementation)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /reports/balance/` | `POST /reports/balance/` |
| Auth | JWT (check_token) + free plan check | JWT (auth middleware) |
| File | `app/routes/reports.py:69` | `internal/handlers/reports/handler.go:259` |

## Request

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountIds` | `list[int]` (required) | `[]int` | OK |
| `balanceDate` | `date` (required) | `string` (required, `validate:"required"`) | OK |

## Response

**Success**: 200 OK. Array of balance report items.

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
| Account IDs filter | **IGNORES** `account_ids` (passes `[]`) | Respects `accountIds` -- filters if provided | **DIFF** -- Python bug? |
| Balance calculation | BalanceReportGenerator: calculates at-date balance | Returns current `account.Balance` (TODO comment) | **DIFF** |
| Base currency conversion | Via exchange rates | Returns same value as balance (TODO comment) | **DIFF** |
| Include hidden accounts | YES (all accounts) | YES (`IncludeHidden: true`) | OK |
| Exclude archived | Via generator logic | YES (`IncludeArchived: false`) | OK |
| AccessDenied error | Caught, returns 403 | Not implemented | **DIFF** |

## Notes

- **Critical Python behavior**: The `/balance/` endpoint in Python passes an **empty list `[]`** to `get_balance_report()` regardless of what `account_ids` the client sends (line 81: `get_balance_report(request.state.user['id'], db, [], input_data.balance_date)`). This means `/balance/` always returns all accounts. The Go version respects the `accountIds` field and filters accordingly.
- Go has two TODO items: (1) "Calculate balance at specific date using transaction history" and (2) "Convert using exchange rate". Currently it returns the account's current balance for both `balance` and `baseCurrencyBalance`.
- Python uses `BalanceReportGenerator` which calculates the actual balance at the specified date by examining transaction history.
- Python enforces free plan compliance. Go does not.
- Python catches `AccessDenied` and returns 403. Go does not have this error path.
- Go includes hidden accounts in the `/balance/` endpoint (matching Python behavior for this route).

## Tests

### Python Tests (5 total)

| Test | Verifies |
|------|----------|
| `test_balance_report_success` | 200, list response |
| `test_balance_report_all_accounts` | 200 with empty account_ids |
| `test_balance_report_unauthorized` | 422 without token |
| `test_balance_report_access_denied` | 403 on AccessDenied |
| `test_balance_report_internal_error` | 500 on exception |

### Go Integration Tests (8 total)

| Test | Verifies |
|------|----------|
| `TestBalanceReportSuccess` | 200, at least 1 item |
| `TestBalanceReportAllAccounts` | 200 with empty account_ids |
| `TestBalanceReportUnauthorized` | 401 without token |
| `TestBalanceReportMissingDate` | 422 |
| `TestBalanceReportInvalidBody` | 422 |
| `TestBalanceReportWithMultipleAccounts` | 200, 2 accounts in response |
| `TestBalanceReportResponseStructure` | All required fields present |
| `TestBalanceReportIncludesHiddenAccounts` | Hidden account in response |

### Go Unit Tests (7 total)

| Test | Verifies |
|------|----------|
| `TestBalanceReportUserError` | 500 on user DB error |
| `TestBalanceReportCurrencyError` | 500 on currency DB error |
| `TestBalanceReportAccountError` | 500 on account DB error |
| `TestBalanceReportAccountCurrencyError` | 200, skips account with bad currency |
| `TestBalanceReportAccountIDFilter` | Filters by account IDs |
| `TestBalanceReportUserRepoError` | 401 on user repo error |
| `TestBalanceReportUserNotActive` | 401 for inactive user |
