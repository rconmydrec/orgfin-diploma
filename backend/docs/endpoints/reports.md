# Reports Endpoints

HTTP handlers for financial reporting: cash flow, account balances, expense breakdowns by category, chart diagram data, and aggregated expense data for frontend visualizations.

## Table of Contents

- [POST /reports/cashflow/](#post-reportscashflow)
- [POST /reports/balance/](#post-reportsbalance)
- [POST /reports/balance/non-hidden/](#post-reportsbalancenon-hidden)
- [POST /reports/expenses-by-categories/](#post-reportsexpenses-by-categories)
- [GET /reports/diagram/:diagram_type/:start_date/:end_date](#get-reportsdiagramdiagram_typestart_dateend_date)
- [POST /reports/expenses-data/](#post-reportsexpenses-data)

---

## POST /reports/cashflow/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| period | string | Yes | `validate:"required"`. Accepted values: `"monthly"`, `"daily"`. Unknown values default to `"monthly"`. |
| startDate | *DateOnly | No | Optional start of range. Accepts `YYYY-MM-DD`, `YYYY-MM-DDThh:mm:ss`, `YYYY-MM-DDThh:mm`, or RFC3339. Type is `common.DateOnly` from `internal/handlers/common/date.go`. |
| endDate | *DateOnly | No | Optional end of range. Accepts `YYYY-MM-DD`, `YYYY-MM-DDThh:mm:ss`, `YYYY-MM-DDThh:mm`, or RFC3339. Type is `common.DateOnly` from `internal/handlers/common/date.go`. |

### Response

**Success**: HTTP 200

```json
{
  "currency": "USD",
  "totalIncome": {
    "2024-01": 5000.00,
    "2024-02": 4800.00
  },
  "totalExpenses": {
    "2024-01": 3200.00,
    "2024-02": 2900.00
  },
  "netFlow": {
    "2024-01": 1800.00,
    "2024-02": 1900.00
  }
}
```

Map keys are period identifiers: `"YYYY-MM"` for monthly, `"YYYY-MM-DD"` for daily.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing period | 422 | "Invalid request data" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Fetches the user's base currency to populate the `currency` field.
- Retrieves all transactions for the user within the requested date range (`NoLimit: true`).
- Aggregates transactions by period key (month or day).
- Transfers (`tx.IsTransfer == true`) are excluded from all aggregations.
- Transactions with `tx.ExcludeFromReports == true` are excluded.
- Converts each transaction amount to the user's base currency using exchange rates from the report's start date (never uses the stored `base_currency_amount` field).
- Computes `netFlow` as `totalIncome - totalExpenses` per period.
- Unknown `period` values default silently to `"monthly"`.

### Error Handling

- When the `exchange_rates` table is completely empty, returns HTTP 500 with `"Exchange rates unavailable, cannot generate report"`.
- When a specific currency code is missing from the rates map for a given date, the transaction is skipped with a warning log, and the report continues.
- Exchange rates use back-fill (nearest past date) and forward-fill (nearest future date) to handle gaps.

### Tests

- `TestCashFlowReportSuccess` — 200 with `currency` present in response
- `TestCashFlowReportDailyPeriod` — 200 with daily period aggregation
- `TestCashFlowReportUnauthorized` — 401 without auth token
- `TestCashFlowReportWithTransactions` — 200 with `totalExpenses` present
- `TestCashFlowReportMissingPeriod` — 422 when period is absent
- `TestCashFlowReportInvalidBody` — 422 for malformed JSON
- `TestCashFlowReportWithIncomeTransaction` — 200 with `totalIncome` present
- `TestCashFlowReportDefaultPeriod` — 200 when unknown period value defaults to monthly
- `TestCashFlowReportWithMixedTransactions` — 200 with all three map fields populated
- `TestCashFlowReportNoDates` — 200 when no date range is specified
- `TestCashFlowReportUserError` — 500 on user DB error
- `TestCashFlowReportCurrencyError` — 500 on currency DB error
- `TestCashFlowReportTransactionError` — 500 on transaction DB error
- `TestCashFlowReportExcludeFromReports` — excluded transactions are not counted
- `TestCashFlowReportUserNotActive` — 401 with "User not activated"

---

## POST /reports/balance/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| balanceDate | string | Yes | `validate:"required"`. Target date for the balance snapshot. |
| accountIds | []int | No | Optional list of account IDs to filter. If empty, all accounts are returned. |

### Response

**Success**: HTTP 200

```json
[
  {
    "accountId": 1,
    "accountName": "Checking",
    "accountTypeId": 1,
    "currencyCode": "USD",
    "balance": 1500.00,
    "baseCurrencyBalance": 1500.00,
    "baseCurrencyCode": "USD",
    "reportDate": "2024-01-15"
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing balanceDate | 422 | "Invalid request data" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Fetches all non-archived accounts for the user, including hidden ones (`IncludeHidden: true`, `IncludeArchived: false`).
- If `accountIds` is non-empty, filters the account list to only those IDs.
- **Historical balance calculation**: the handler starts from the account's current `Balance` and back-calculates the balance at the requested `balanceDate` by fetching all transactions after that date and reversing their effect (subtracting income/incoming transfers, adding back expenses/outgoing transfers). If the `balanceDate` cannot be parsed, the current balance is used as-is.
- `baseCurrencyBalance` is calculated by converting the historical balance to the user's base currency using exchange rates for the `balanceDate`. If a specific currency is missing from the rates map, the raw balance is used as a fallback with a warning log.
- Accounts where the currency cannot be looked up are silently skipped.

### Error Handling

- When the `exchange_rates` table is completely empty and account currency differs from base currency, returns HTTP 500 with `"Exchange rates unavailable, cannot generate report"`.
- When a specific currency is missing from the rates map, `baseCurrencyBalance` falls back to the raw balance with a warning log.
- When transaction fetching for historical balance calculation fails, the error is logged and the current balance is used.

### Tests

- `TestBalanceReportSuccess` — 200 with at least one item
- `TestBalanceReportAllAccounts` — 200 when accountIds is empty (all accounts returned)
- `TestBalanceReportUnauthorized` — 401 without auth token
- `TestBalanceReportMissingDate` — 422 when balanceDate is absent
- `TestBalanceReportInvalidBody` — 422 for malformed JSON
- `TestBalanceReportWithMultipleAccounts` — 200 with two accounts in response
- `TestBalanceReportResponseStructure` — all required fields present in each item
- `TestBalanceReportIncludesHiddenAccounts` — hidden accounts appear in the response
- `TestBalanceReportUserError` — 500 on user DB error
- `TestBalanceReportCurrencyError` — 500 on currency DB error
- `TestBalanceReportAccountError` — 500 on account DB error
- `TestBalanceReportAccountCurrencyError` — 200, account with bad currency is skipped
- `TestBalanceReportAccountIDFilter` — filters results by provided account IDs
- `TestBalanceReportUserRepoError` — 401 on user repository error
- `TestBalanceReportUserNotActive` — 401 for inactive user

---

## POST /reports/balance/non-hidden/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| balanceDate | string | Yes | `validate:"required"`. Target date for the balance snapshot. |
| accountIds | []int | No | Optional list of account IDs to filter. |

### Response

**Success**: HTTP 200

Same structure as [POST /reports/balance/](#post-reportsbalance), but hidden accounts are excluded.

```json
[
  {
    "accountId": 1,
    "accountName": "Checking",
    "accountTypeId": 1,
    "currencyCode": "USD",
    "balance": 1500.00,
    "baseCurrencyBalance": 1500.00,
    "baseCurrencyCode": "USD",
    "reportDate": "2024-01-15"
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing balanceDate | 422 | "Invalid request data" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Shares the `reports.Service.BalanceReport` method with `/reports/balance/` via the `ExcludeHidden: true` flag.
- Excludes hidden accounts at the database level (`IncludeHidden: false` in the account repository query).
- Additionally checks `account.IsHidden` in the result loop as a secondary safeguard.
- Non-archived accounts only (`IncludeArchived: false`).
- If `accountIds` is non-empty, further filters to only those IDs.
- Same historical balance calculation as `/reports/balance/` applies (back-calculated from current balance by reversing transactions after the requested date).

### Error Handling

- Same error handling as [POST /reports/balance/](#post-reportsbalance).

### Tests

- `TestBalanceReportNonHiddenSuccess` — 200 with valid response
- `TestBalanceReportNonHiddenUnauthorized` — 401 without auth token
- `TestBalanceReportNonHiddenInvalidBody` — 422 for malformed JSON
- `TestBalanceReportNonHiddenMissingDate` — 422 when balanceDate is absent
- `TestBalanceReportExcludesHiddenAccounts` — hidden account does not appear in response
- `TestBalanceReportNonHiddenUserError` — 500 on user DB error
- `TestBalanceReportNonHiddenWithHiddenAccount` — hidden account is excluded
- `TestBalanceReportNonHiddenUserRepoError` — 401 on user repository error
- `TestBalanceReportNonHiddenUserNotActive` — 401 for inactive user

---

## POST /reports/expenses-by-categories/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| startDate | string | Yes | `validate:"required"`. Start of the reporting period. |
| endDate | string | Yes | `validate:"required"`. End of the reporting period. |
| categories | []int | No | Optional list of category IDs to filter results. When specified, only matching parent categories and their children (or children whose parent matches) are included. |
| hideEmptyCategories | bool | No | When true, categories with zero total expenses are omitted from the response. |

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "name": "Food",
    "parentId": null,
    "parentName": null,
    "totalExpenses": 450.00,
    "currencyCode": "USD",
    "isParent": true
  },
  {
    "id": 3,
    "name": "Food >> Groceries",
    "parentId": 1,
    "parentName": "Food",
    "totalExpenses": 300.00,
    "currencyCode": "USD",
    "isParent": false
  }
]
```

`totalExpenses` is a JSON number (`float64`).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing startDate or endDate | 422 | "Invalid request data" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Fetches the user's base currency from the currency repository.
- Retrieves all non-deleted expense categories for the user.
- Retrieves expense transactions for the date range. Filters applied:
  - `tx.ExcludeFromReports == true` — excluded
  - `tx.IsIncome == true` — excluded
  - `tx.IsTransfer == true` — excluded
  - `tx.CategoryID == nil` — excluded (uncategorized transactions are skipped)
- Converts each expense amount to the user's base currency using exchange rates from the report's `startDate` (not the current date).
- Category names for child categories are formatted as `"Parent >> Child"`.
- `isParent` is `true` for top-level categories and `false` for subcategories.
- Only top-level parents and their direct children are included; grandchild categories (depth > 2) are skipped.
- When `hideEmptyCategories` is true, categories with `totalExpenses == 0` are omitted.
- If `categories` is non-empty, the category list is filtered: parent categories must be in the filter, and child categories must either be in the filter themselves or have their parent in the filter.
- Results are sorted with **deterministic parent-child grouping**: parents are sorted alphabetically by name, children are grouped under their parent (parent appears first), and children within the same parent are sorted alphabetically by name.

### Known Gaps / TODOs

- Grandchild categories (categories whose parent is itself a child category) are silently skipped.

### Tests

- `TestExpensesByCategoriesSuccess` — 200 with array response
- `TestExpensesByCategoriesHideEmpty` — 200 with `hideEmptyCategories: true`
- `TestExpensesByCategoriesUnauthorized` — 401 without auth token
- `TestExpensesByCategoriesMissingDates` — 422 when dates are absent
- `TestExpensesByCategoriesWithTransactions` — 200 with actual expense data populated
- `TestExpensesByCategoriesWithSpecificCategories` — 200 with categories filter field set
- `TestExpensesByCategoriesInvalidBody` — 422 for malformed JSON
- `TestExpensesByCategoriesCategoryError` — 500 on category repository error
- `TestExpensesByCategoriesTransactionError` — 500 on transaction repository error
- `TestExpensesByCategoriesWithParentCategory` — 200 with parent/child structure
- `TestExpensesByCategoriesNilCategory` — 200 when transaction references a non-existent category ID
- `TestExpensesByCategoriesExcludedAndIncomeTransactions` — excluded/income/nil-category transactions are filtered
- `TestExpensesByCategoriesHideEmptyCategories` — empty categories are absent from response
- `TestExpensesByCategoriesUserRepoError` — 401 on user repository error
- `TestExpensesByCategoriesUserNotActive` — 401 for inactive user

---

## GET /reports/diagram/:diagram_type/:start_date/:end_date

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| diagram_type | string (path) | Yes | Chart type identifier. Logged but not used in aggregation logic. |
| start_date | string (path) | Yes | Start of the reporting period (passed as-is to the DB filter). |
| end_date | string (path) | Yes | End of the reporting period (passed as-is to the DB filter). |

No request body.

### Response

**Success**: HTTP 200

```json
{
  "labels": ["Food", "Transport", "Utilities"],
  "data": [450.50, 120.00, 85.75],
  "currency": "USD"
}
```

`labels` and `data` are parallel arrays; index N in `labels` corresponds to index N in `data`.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Fetches the user's base currency for the `currency` field.
- Retrieves expense transactions in the date range. Same transaction filters as the expenses-by-categories endpoint apply (excludes transfers, excluded-from-reports, income, and uncategorized).
- Aggregates expense amounts by **parent category**: child category amounts are rolled up into their parent totals. Top-level categories without a parent are aggregated independently.
- Categories with zero total amount are skipped.
- Categories representing less than **2% of the total** are grouped into a single **"Other"** bucket.
- Results are **sorted by amount descending**.
- Converts each transaction amount to the user's base currency using exchange rates from the report's `start_date` (not the current date).
- `diagram_type` is read from the path parameter and logged; it has no effect on the aggregation or response structure.
- Date strings are passed directly to the database filter without parsing; invalid date formats may produce empty results rather than an error.

### Known Gaps / TODOs

- Date format is not validated at the handler level; invalid formats cause DB-level errors or silently return empty results.

### Tests

- `TestDiagramSuccess` — 200 with valid response
- `TestDiagramUnauthorized` — 401 without auth token
- `TestDiagramWithTransactions` — 200 with labels and data arrays populated
- `TestDiagramBarChart` — 200 with bar chart type parameter
- `TestDiagramUserError` — 500 on user repository error
- `TestDiagramCurrencyError` — 500 on currency repository error
- `TestDiagramCategoryError` — 500 on category repository error
- `TestDiagramTransactionError` — 500 on transaction repository error
- `TestDiagramWithZeroExpenses` — 200 when all expense amounts are zero
- `TestDiagramExcludedAndIncomeTransactions` — excluded/income/nil-category transactions filtered out
- `TestDiagramUserRepoError` — 401 on user repository error
- `TestDiagramUserNotActive` — 401 for inactive user

---

## POST /reports/expenses-data/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/reports/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| startDate | string | Yes | `validate:"required"`. Start of the reporting period. |
| endDate | string | Yes | `validate:"required"`. End of the reporting period. |
| categories | []int | No | Optional category ID filter. When specified, only transactions belonging to matching categories (or children of matching parents) are included. |
| hideEmptyCategories | bool | No | Accepted in the request struct but hardcoded to `false` internally for this endpoint. |

### Response

**Success**: HTTP 200

```json
[
  {
    "categoryId": 1,
    "label": "Food",
    "amount": 750.50,
    "categoryName": "Food"
  },
  {
    "categoryId": 0,
    "label": "Other",
    "amount": 120.00,
    "categoryName": "Other"
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing startDate or endDate | 422 | "Invalid request data" |
| Internal error | 500 | "Error generating report" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Retrieves expense categories and transactions using the same filters as the expenses-by-categories endpoint.
- If `categories` is non-empty, only transactions belonging to matching categories (or children whose parent is in the filter) are included in aggregation.
- Aggregates child category amounts into their parent totals (parent-level rollup).
- Groups categories whose amount represents less than 2% of the total into a single `"Other"` bucket (`categoryId: 0`, `categoryName: "Other"`).
- Sorts the final result by amount descending (bubble sort).
- `categoryName` is an additional field not present in other expense report endpoints.
- The `hideEmptyCategories` request field is accepted but internally hardcoded to `false`; empty parent categories are never hidden in this endpoint.
- Converts each transaction amount to the user's base currency using exchange rates from the report's `startDate` (not the current date).

### Tests

- `TestExpensesDataSuccess` — 200 with valid response
- `TestExpensesDataUnauthorized` — 401 without auth token
- `TestExpensesDataMissingDates` — 422 when required date fields are absent
- `TestExpensesDataWithTransactions` — 200 with actual aggregated expense data
- `TestExpensesDataInvalidBody` — 422 for malformed JSON
- `TestExpensesDataCategoryError` — 500 on category repository error
- `TestExpensesDataTransactionError` — 500 on transaction repository error
- `TestExpensesDataWithZeroAmount` — 200 when transaction amounts are zero
- `TestExpensesDataExcludedAndIncomeTransactions` — excluded/income/nil-category transactions filtered
- `TestExpensesDataUserRepoError` — 401 on user repository error
- `TestExpensesDataUserNotActive` — 401 for inactive user
