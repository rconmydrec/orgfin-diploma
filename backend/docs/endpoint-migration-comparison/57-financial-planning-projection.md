# Endpoint #57: POST /financial-planning/projection

## Route Definition
- **Python**: `@router.post('/projection', response_model=BalanceProjectionResponseSchema)`
- **Go**: `g.POST("/projection", h.GetBalanceProjection)`

## Request
- Auth: required (both)
- Body (Python `BalanceProjectionRequestSchema`): start_date (datetime, default now), end_date (datetime, required), period (literal daily|weekly|monthly, default daily), account_ids (list[int], optional), include_inactive (bool, default False)
- Body (Go `BalanceProjectionRequest`): startDate (time.Time), endDate (time.Time, required via validate tag), period (string), accountIds ([]int), includeInactive (bool)

## Response
- **Python**: 200 OK with `BalanceProjectionResponseSchema`
- **Go**: 200 OK with `BalanceProjectionResponse`
- Both return: startDate, endDate, period, baseCurrencyCode, projectionPoints[] (date, balance, income, expenses)

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Validation error | 400 (ValueError) | 422 `"Invalid request data"` |
| End before start | 400 (ValueError from service) | 400 `"End date must be after start date"` |
| Internal error | 500 `"Error generating balance projection"` | 500 `"Error generating balance projection"` |
| User not active | N/A | 401 `"User not activated"` |
| Exchange rates empty | N/A | 500 `"Exchange rates unavailable"` |

## Business Logic Comparison
1. **Architecture**: Python delegates to `fp_service.get_balance_projection()`; Go delegates to `services/financial_planning.Service.GetBalanceProjection()`. Both use a service layer pattern.
2. **Defaults**: Both default start_date to now and period to "daily".
3. **Date validation**: Go service returns `ErrEndDateBeforeStart`, handler maps to 400; Python raises ValueError from service.
4. **Period mapping**: Python uses Literal type validation; Go accepts any string but defaults unknown periods to "daily".
5. **Period grouping**: Go uses `generateDatePoints()`, a calendar-based algorithm: daily generates each day, weekly uses Monday-to-Sunday boundaries, monthly uses actual month boundaries. This aligns with calendar periods correctly.
6. **Projection generation**: Go iterates projection points, collects planned transactions falling within each period boundary, converts to base currency, and applies to running balance.
7. **Currency conversion**: Go converts all amounts using `currency.Service.ConvertToBaseCurrency()` with `RateCache`. Falls back to raw balance when conversion fails. Returns 500 on `ErrNoExchangeRates`.
8. **Running balance**: Both maintain a running balance starting from current total account balance (converted to base currency).

## Notes
- Python validates period strictly via Literal type; Go accepts any string and defaults unknown to "daily".
- Go adds `RequireActiveUser` middleware verification; Python has subscription plan enforcement.
- Go handler is a thin HTTP adapter; all business logic is in the service layer.

## Tests
- **Python**: 8 tests (test_get_projection_value_error, _internal_error, _daily_success, _weekly_success, _monthly_success, _invalid_date_range, _unauthorized)
- **Go integration tests**: 9 tests (TestBalanceProjectionSuccess, TestBalanceProjectionWeekly, TestBalanceProjectionMonthly, TestBalanceProjectionInvalidDates, TestBalanceProjectionUnauthorized, TestBalanceProjectionInvalidBody, TestBalanceProjectionMissingEndDate, TestBalanceProjectionWithAccounts, TestBalanceProjectionDefaultPeriod)
- **Go handler unit tests**: 22 tests covering bind/validate errors, user lifecycle, service error mapping, period handling, currency conversion, exchange rate scenarios
- **Go service unit tests**: 23 tests covering business logic (date defaults, period grouping, generateDatePoints for daily/weekly/monthly, currency conversion, running balance, error paths)
