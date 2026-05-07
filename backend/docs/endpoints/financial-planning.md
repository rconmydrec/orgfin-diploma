# Financial Planning Endpoints

Analytical endpoints for projecting future account balances based on current balances and upcoming planned transactions.

## Table of Contents

- [POST /financial-planning/future-balance](#post-financial-planningfuture-balance)
- [POST /financial-planning/projection](#post-financial-planningprojection)

---

## POST /financial-planning/future-balance

**Auth**: Required (JWT)
**Handler**: `internal/handlers/financial_planning/handler.go` (thin adapter)
**Service**: `internal/services/financial_planning/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| targetDate | time.Time | Yes | The future date to project balance to |
| accountIds | []int | No | Filter to specific accounts; all accounts used if omitted |
| includeInactive | bool | No | Whether to include inactive planned transactions |
| includeHidden | bool | No | Whether to include hidden accounts (default false) |

### Response

**Success**: HTTP 200

```json
{
  "targetDate": "2026-06-01T00:00:00Z",
  "baseCurrencyCode": "USD",
  "totalCurrentBalance": 5000.00,
  "totalProjectedBalance": 4750.00,
  "totalPlannedIncome": 1000.00,
  "totalPlannedExpenses": 1250.00,
  "incomeCount": 2,
  "expensesCount": 3,
  "accounts": [
    {
      "accountId": 1,
      "accountName": "Checking",
      "currencyCode": "USD",
      "currentBalance": 5000.00,
      "projectedBalance": 5000.00,
      "totalPlannedIncome": 0,
      "totalPlannedExpenses": 0
    }
  ]
}
```

Notes:
- All monetary amounts (`totalCurrentBalance`, `totalProjectedBalance`, `totalPlannedIncome`, `totalPlannedExpenses`, `currentBalance`, `projectedBalance`) are JSON numbers (float64), not strings.
- Per-account projections show the account's own currency balance (not converted to base currency). The `totalPlannedIncome` and `totalPlannedExpenses` on individual accounts are always 0 (planned transactions are only aggregated globally).
- The top-level totals (`totalCurrentBalance`, `totalProjectedBalance`, `totalPlannedIncome`, `totalPlannedExpenses`) are converted to the user's base currency.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Validation failure (missing targetDate) | 422 | "Invalid request data" |
| User settings fetch error | 500 | "Error calculating future balance" |
| Base currency fetch error | 500 | "Error calculating future balance" |
| Account fetch error | 500 | "Error calculating future balance" |
| Planned transactions fetch error | 500 | "Error calculating future balance" |
| Exchange rates table empty | 500 | "Exchange rates unavailable" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler binds/validates the request and delegates to `services/financial_planning.Service.CalculateFutureBalance()`.
- The service fetches the base currency via `currencyRepo.GetByID(params.BaseCurrencyID)`.
- Fetches user accounts with `OnlyShowInReports: true`, `IncludeHidden: params.IncludeHidden`, `IncludeArchived: false`. The `includeHidden` request parameter controls whether hidden accounts are included in the calculation.
- Filters accounts by `accountIds` if provided.
- Calculates `days` using date-only math: normalizes both "now" and `targetDate` to midnight, computes `target.Sub(today).Hours()/24 + 0.5`, then clamps to 0 if `targetDate` is in the past.
- Fetches upcoming planned transaction occurrences via the planned transactions repository for the calculated number of days. The repository expands recurring transactions using `generateOccurrences`.
- **Currency conversion**: All planned transaction amounts are converted to the user's base currency using `currency.Service.ConvertToBaseCurrency()` with exchange rates for the occurrence date. If exchange rates are unavailable for a specific currency pair, that transaction is skipped with a warning. If the exchange rates table is completely empty (`currency.ErrNoExchangeRates`), returns 500 with "Exchange rates unavailable".
- Iterates planned transactions once to compute global `totalPlannedIncome`, `totalPlannedExpenses`, `incomeCount`, and `expensesCount`.
- Iterates accounts to build per-account projections (showing raw account balances) and to compute `totalCurrentBalance` by converting each account balance to the user's base currency.
- `totalProjectedBalance = totalCurrentBalance + totalPlannedIncome - totalPlannedExpenses`.
- Uses `currency.RateCache` (from `services/currency`) to avoid redundant exchange rate lookups within a single request.
- If an account's balance cannot be converted to the base currency (missing rate), falls back to using the raw (unconverted) balance.

### Tests

Integration tests (`handlers/financial_planning/`):
- `TestFutureBalanceSuccess` -- 200, basic projection with one account
- `TestFutureBalanceWithAccounts` -- 200, filtered by specific account IDs
- `TestFutureBalanceUnauthorized` -- 401 without auth token
- `TestFutureBalanceMissingDate` -- 422 when targetDate is absent
- `TestFutureBalanceInvalidBody` -- 422 for malformed JSON
- `TestFutureBalanceWithMultipleAccounts` -- 200, multiple accounts in response
- `TestFutureBalanceWithIncludeInactive` -- 200 with includeInactive=true

Handler unit tests (`handlers/financial_planning/`):
- `TestCalculateFutureBalanceUserNotActive` -- 401 when user is inactive
- `TestCalculateFutureBalanceUserRepoError` -- 500 on user repo error
- `TestCalculateFutureBalanceUserDeleted` -- 401 when user is soft-deleted
- `TestCalculateFutureBalanceBindError` -- 422 for invalid JSON
- `TestCalculateFutureBalanceValidationError` -- 422 for missing required field
- `TestCalculateFutureBalanceUserSettingsError` -- 500 on user settings fetch failure
- `TestCalculateFutureBalanceCurrencyError` -- 500 on currency fetch failure
- `TestCalculateFutureBalanceAccountError` -- 500 on account fetch failure
- `TestCalculateFutureBalancePlannedTxError` -- 500 on planned transaction fetch failure
- `TestCalculateFutureBalancePastDate` -- 200, days clamped to 0 for past targetDate
- `TestCalculateFutureBalanceWithTransactions` -- 200, income and expense transactions reflected
- `TestCalculateFutureBalanceWithAccountFilter` -- 200, non-matching accounts excluded
- `TestCalculateFutureBalanceCurrencyErrorInLoop` -- currency lookup failure in account loop
- `TestCalculateFutureBalanceMultipleAccounts` -- 200, multiple accounts with currency conversion
- `TestCalculateFutureBalanceGlobalTotals` -- 200, global totals aggregated correctly
- `TestCalculateFutureBalanceMultiplePlannedTx` -- 200, multiple planned transactions
- `TestCalculateFutureBalanceWithAccountFilter2` -- 200, account filter excludes non-matching
- `TestCalculateFutureBalanceMultiCurrencyConversion` -- 200, multi-currency conversion via exchange rates
- `TestCalculateFutureBalanceExchangeRateEmpty` -- 500 when exchange rates table is empty
- `TestCalculateFutureBalancePlannedTxCurrencyError` -- planned tx currency lookup failure
- `TestCalculateFutureBalancePlannedTxExchangeRateEmpty` -- 500 when exchange rates empty during planned tx conversion
- `TestCalculateFutureBalancePlannedTxSkippedOnMissingRate` -- planned tx skipped when specific rate is missing
- `TestCalculateFutureBalanceAccountFallbackToRawBalance` -- falls back to raw balance when conversion fails
- `TestRegisterRoutes` -- verifies route registration

Service unit tests (`services/financial_planning/`):
- `TestCalculateFutureBalance_BaseCurrencyError` -- currency fetch failure
- `TestCalculateFutureBalance_AccountFetchError` -- account fetch failure
- `TestCalculateFutureBalance_PlannedTxFetchError` -- planned tx fetch failure
- `TestCalculateFutureBalance_FutureDate` -- correct days calculation for future date
- `TestCalculateFutureBalance_PastDate` -- days clamped to 0
- `TestCalculateFutureBalance_PlannedTxCurrencyLookupFailure` -- skips tx on currency lookup error
- `TestCalculateFutureBalance_PlannedTxErrNoExchangeRates` -- fatal error on empty rates
- `TestCalculateFutureBalance_PlannedTxNonFatalConversionError` -- skips tx on non-fatal error
- `TestCalculateFutureBalance_AccountCurrencyLookupFailure` -- skips account on currency error
- `TestCalculateFutureBalance_AccountErrNoExchangeRates` -- fatal error on empty rates in account loop
- `TestCalculateFutureBalance_AccountFallbackToRawBalance` -- falls back to raw balance
- `TestCalculateFutureBalance_MultiCurrencyConversion` -- multi-currency conversion
- `TestCalculateFutureBalance_AccountFiltering` -- filters by account IDs
- `TestCalculateFutureBalance_IncomeExpenseCounting` -- correct income/expense counts
- `TestCalculateFutureBalance_PerAccountZeroPlanned` -- per-account planned amounts are zero

---

## POST /financial-planning/projection

**Auth**: Required (JWT)
**Handler**: `internal/handlers/financial_planning/handler.go` (thin adapter)
**Service**: `internal/services/financial_planning/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| endDate | time.Time | Yes | End of the projection range |
| startDate | time.Time | No | Defaults to now if not provided |
| period | string | No | Defaults to "daily". Accepted: daily, weekly, monthly. Unknown values fall back to "daily" |
| accountIds | []int | No | Filter to specific accounts; all accounts used if omitted |
| includeInactive | bool | No | Whether to include inactive planned transactions |
| includeHidden | bool | No | Whether to include hidden accounts (default false) |

### Response

**Success**: HTTP 200

```json
{
  "startDate": "2026-02-22T00:00:00Z",
  "endDate": "2026-05-22T00:00:00Z",
  "period": "monthly",
  "baseCurrencyCode": "USD",
  "projectionPoints": [
    {
      "date": "2026-02-22T00:00:00Z",
      "balance": 5000.00,
      "income": 1000.00,
      "expenses": 250.00
    }
  ]
}
```

Notes:
- All monetary amounts (`balance`, `income`, `expenses`) are JSON numbers (float64), not strings.
- Each `projectionPoints` entry represents one step in the projection.
- `balance` is a running balance starting from the current total account balance (converted to base currency).
- Planned transactions are assigned to projection points based on which period their occurrence date falls within.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Validation failure (missing endDate) | 422 | "Invalid request data" |
| endDate before startDate | 400 | "End date must be after start date" |
| User settings fetch error | 500 | "Error generating balance projection" |
| Base currency fetch error | 500 | "Error generating balance projection" |
| Account fetch error | 500 | "Error generating balance projection" |
| Planned transactions fetch error | 500 | "Error generating balance projection" |
| Exchange rates table empty | 500 | "Exchange rates unavailable" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler binds/validates the request and delegates to `services/financial_planning.Service.GetBalanceProjection()`.
- The service defaults `startDate` to the current time if not provided.
- Defaults `period` to "daily" if not provided or if an unknown value is given.
- Returns `ErrEndDateBeforeStart` if `endDate` is before `startDate` (handler maps to 400).
- Fetches base currency and user accounts using the same filtering as the future-balance endpoint (`OnlyShowInReports: true`, `IncludeHidden: params.IncludeHidden`, `IncludeArchived: false`). Fetches upcoming planned transactions.
- Filters accounts by `accountIds` if provided.
- **Currency conversion**: Account balances are converted to the user's base currency using `currency.Service.ConvertToBaseCurrency()` to compute the starting running balance. Planned transaction amounts are also converted to the base currency using exchange rates for the occurrence date. If an account balance cannot be converted, falls back to the raw (unconverted) balance. If the exchange rates table is empty (`currency.ErrNoExchangeRates`), returns 500.
- Uses `currency.RateCache` (from `services/currency`) to avoid redundant exchange rate lookups within a single request.
- **Period grouping** uses `generateDatePoints()` (private method in service), a calendar-based algorithm:
  - **daily**: Each day from start to end (inclusive).
  - **weekly**: First point = startDate, then find the next Monday, generate end-of-week (Sunday) dates until endDate, cap the last point at endDate.
  - **monthly**: First point = startDate, then find the first of the next month, generate end-of-month dates until endDate, cap the last point at endDate.
- For each projection point, a period boundary is defined:
  - First point: period starts at the beginning of startDate.
  - Subsequent points: period starts the day after the previous point.
  - Period always ends at end-of-day of the current point.
- Planned transactions whose occurrence date falls within a period boundary are collected, converted to base currency, and applied to the running balance for that projection point.

### Known Gaps / TODOs

- `period` is not validated against an enum; unknown values silently fall back to "daily".

### Tests

Integration tests (`handlers/financial_planning/`):
- `TestBalanceProjectionSuccess` -- 200, daily projection over a date range
- `TestBalanceProjectionWeekly` -- 200, weekly period generates correct step count
- `TestBalanceProjectionMonthly` -- 200, monthly period projection
- `TestBalanceProjectionInvalidDates` -- 400 when endDate is before startDate
- `TestBalanceProjectionUnauthorized` -- 401 without auth token
- `TestBalanceProjectionInvalidBody` -- 422 for malformed JSON
- `TestBalanceProjectionMissingEndDate` -- 422 when endDate is absent
- `TestBalanceProjectionWithAccounts` -- 200, filtered by account IDs
- `TestBalanceProjectionDefaultPeriod` -- 200, defaults to daily when period omitted

Handler unit tests (`handlers/financial_planning/`):
- `TestGetBalanceProjectionBindError` -- 422 for invalid JSON
- `TestGetBalanceProjectionValidationError` -- 422 for missing required field
- `TestGetBalanceProjectionUserSettingsError` -- 500 on user settings fetch failure
- `TestGetBalanceProjectionCurrencyError` -- 500 on currency fetch failure
- `TestGetBalanceProjectionAccountError` -- 500 on account fetch failure
- `TestGetBalanceProjectionEndDateBeforeStart` -- 400 when endDate precedes startDate
- `TestGetBalanceProjectionPlannedTxError` -- 500 on planned transaction fetch failure
- `TestGetBalanceProjectionWeeklyPeriod` -- 200, weekly period applied correctly
- `TestGetBalanceProjectionMonthlyPeriod` -- 200, monthly period applied
- `TestGetBalanceProjectionWithAccountFilter` -- 200, non-matching accounts excluded
- `TestGetBalanceProjectionWithTransactions` -- 200, transactions reflected in running balance
- `TestGetBalanceProjectionUserNotActive` -- 401 when user is inactive
- `TestGetBalanceProjectionCurrencyErrorInAccountLoop` -- currency error fallback in account loop
- `TestGetBalanceProjectionMultiCurrencyConversion` -- 200, multi-currency conversion
- `TestGetBalanceProjectionExchangeRateEmpty` -- 500 when exchange rates table is empty
- `TestGetBalanceProjectionPerPeriodGrouping` -- daily period grouping with transactions
- `TestGetBalanceProjectionWeeklyGrouping` -- weekly period grouping
- `TestGetBalanceProjectionWeeklyGroupingWithAmounts` -- weekly grouping with transaction amounts
- `TestGetBalanceProjectionAccountFallbackToRawBalance` -- falls back to raw balance
- `TestGetBalanceProjectionPlannedTxCurrencyLookupFailure` -- planned tx currency lookup fails
- `TestGetBalanceProjectionPlannedTxFatalNoExchangeRates` -- 500 on empty exchange rates during projection
- `TestGetBalanceProjectionPlannedTxNonFatalMissingRate` -- skips tx when specific rate missing

Service unit tests (`services/financial_planning/`):
- `TestGetBalanceProjection_DefaultStartDate` -- defaults start to now
- `TestGetBalanceProjection_DefaultPeriod` -- defaults period to daily
- `TestGetBalanceProjection_BaseCurrencyError` -- currency fetch failure
- `TestGetBalanceProjection_AccountFetchError` -- account fetch failure
- `TestGetBalanceProjection_EndDateBeforeStart` -- returns ErrEndDateBeforeStart
- `TestGetBalanceProjection_PlannedTxFetchError` -- planned tx fetch failure
- `TestGetBalanceProjection_AccountCurrencyFallbackToRaw` -- falls back to raw balance
- `TestGetBalanceProjection_AccountErrNoExchangeRates` -- fatal error on empty rates
- `TestGetBalanceProjection_CurrentBalanceMultipleAccounts` -- multi-account starting balance
- `TestGetBalanceProjection_DailyPoints` -- daily date points generation
- `TestGetBalanceProjection_WeeklyPoints` -- weekly date points generation
- `TestGetBalanceProjection_MonthlyPoints` -- monthly date points generation
- `TestGetBalanceProjection_PeriodGrouping` -- transactions grouped into periods
- `TestGetBalanceProjection_PlannedTxCurrencyLookupFailure` -- currency lookup failure in period
- `TestGetBalanceProjection_PlannedTxErrNoExchangeRates` -- fatal error during period
- `TestGetBalanceProjection_PlannedTxNonFatalMissingRate` -- skips tx on non-fatal error
- `TestGetBalanceProjection_RunningBalanceTracking` -- running balance across periods
- `TestGetBalanceProjection_AccountBalanceFallback` -- account balance fallback
- `TestGenerateDatePoints_Daily` -- daily date points
- `TestGenerateDatePoints_Weekly` -- weekly date points
- `TestGenerateDatePoints_WeeklyStartOnMonday` -- weekly starting on Monday
- `TestGenerateDatePoints_WeeklyShortRange` -- weekly with short range
- `TestGenerateDatePoints_Monthly` -- monthly date points
- `TestGenerateDatePoints_MonthlySameDay` -- monthly boundary edge case
