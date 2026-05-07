# Endpoint #56: POST /financial-planning/future-balance

## Route Definition
- **Python**: `@router.post('/future-balance', response_model=FutureBalanceResponseSchema)`
- **Go**: `g.POST("/future-balance", h.CalculateFutureBalance)`

## Request
- Auth: required (both)
- Body (Python `FutureBalanceRequestSchema`): target_date (datetime, required), account_ids (list[int], optional), include_inactive (bool, default False)
- Body (Go `FutureBalanceRequest`): targetDate (time.Time, required via validate tag), accountIds ([]int), includeInactive (bool)

## Response
- **Python**: 200 OK with `FutureBalanceResponseSchema`
- **Go**: 200 OK with `FutureBalanceResponse`
- Both return: targetDate, baseCurrencyCode, totalCurrentBalance, totalProjectedBalance, totalPlannedIncome, totalPlannedExpenses, incomeCount, expensesCount, accounts[]

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Validation error | 400 (ValueError from service) | 422 `"Invalid request data"` |
| Internal error | 500 `"Error calculating future balance"` | 500 `"Error calculating future balance"` |
| User not active | N/A | 401 `"User not activated"` |
| Exchange rates empty | N/A | 500 `"Exchange rates unavailable"` |

## Business Logic Comparison
1. **Architecture**: Python delegates to `fp_service.calculate_future_balance()`; Go delegates to `services/financial_planning.Service.CalculateFutureBalance()`. Both use a service layer pattern.
2. **User settings**: Both retrieve the user's base currency. Go reads `BaseCurrencyID` from the active user context, passes it to the service which calls `currencyRepo.GetByID()`.
3. **Accounts**: Both fetch user accounts (Go with `IncludeHidden: false, IncludeArchived: false`).
4. **Account filtering**: Both filter by account IDs if provided.
5. **Days calculation**: Go calculates `days = int(time.Until(targetDate).Hours() / 24)`, clamps to 0 if negative.
6. **Planned transactions**: Both fetch upcoming occurrences for the calculated time range.
7. **Balance calculation**: Go aggregates planned transactions globally (totalPlannedIncome, totalPlannedExpenses), then builds per-account projections with raw account balances. Global totals are converted to base currency.
8. **Currency conversion**: Go converts all amounts to base currency using `currency.Service.ConvertToBaseCurrency()` with `RateCache`. Falls back to raw balance when conversion fails. Returns 500 on `ErrNoExchangeRates`.

## Notes
- Go has multiple intermediate error points (base currency, accounts, planned transactions) each returning 500.
- Python has subscription plan enforcement; Go does not.
- Go adds `RequireActiveUser` middleware verification.
- Go handler is a thin HTTP adapter; all business logic is in the service layer.

## Tests
- **Python**: 8 tests (test_calculate_future_balance_value_error, _mocked_success, _success, _multiple_accounts, _with_planned_transactions, _past_date, _no_accounts, _unauthorized)
- **Go integration tests**: 7 tests (TestFutureBalanceSuccess, TestFutureBalanceWithAccounts, TestFutureBalanceUnauthorized, TestFutureBalanceMissingDate, TestFutureBalanceInvalidBody, TestFutureBalanceWithMultipleAccounts, TestFutureBalanceWithIncludeInactive)
- **Go handler unit tests**: 24 tests covering bind/validate errors, user lifecycle, service error mapping, account filtering, currency conversion, exchange rate scenarios
- **Go service unit tests**: 15 tests covering business logic (days calculation, planned tx aggregation, currency conversion, account filtering, error paths)
