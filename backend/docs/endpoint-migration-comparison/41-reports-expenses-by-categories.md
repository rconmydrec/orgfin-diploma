# Endpoint #41: POST `/reports/expenses-by-categories/`

**Status**: OK
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /reports/expenses-by-categories/` | `POST /reports/expenses-by-categories/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| File | `app/routes/reports.py:124` | `internal/handlers/reports/handler.go:371` |

## Request

**Method**: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `startDate` | date (required, camelCase alias) | string (required, validate:"required") | DIFF — Python uses `date` type, Go uses plain string |
| `endDate` | date (required, camelCase alias) | string (required, validate:"required") | DIFF — same as above |
| `categories` | list[int] (optional, default []) | []int (optional) | OK |
| `hideEmptyCategories` | bool (optional, default False) | bool (optional) | OK |

## Response

**Success**: 200 OK. Array of expense items by category.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `name` | str (with "Parent >> Child" format) | string (with "Parent >> Child" format) | OK |
| `parentId` | int or None | *int (nullable) | OK |
| `parentName` | str or None | *string (nullable) | OK |
| `totalExpenses` | Decimal (serialized as float) | decimal.Decimal (JSON string) | DIFF — Python serializes as float, Go as JSON string |
| `currencyCode` | str or None | *string (nullable) | OK |
| `isParent` | bool | bool | OK |

**Note on `isParent` key**: Python parent categories use `'isParent': True` but child categories use `'is_parent': False` (inconsistent key casing in internal dict). Go is consistent: always `isParent` in the JSON response.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Unauthorized | 422 (missing header) | FastAPI validation | 401 | JWT middleware | DIFF |
| User not active | N/A (not checked) | N/A | 401 | "User not activated" | DIFF — Go adds is_active check |
| Invalid body | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Missing dates | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Internal error | 500 | "Error generting report" | 500 | "Error generating report" | OK (Python has typo) |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | DIFF — Go is stricter |
| Free plan enforcement | YES (enforce_free_plan_compliance) | NO | DIFF — Go lacks subscription check |
| Get categories | SQL query with JOIN (expense only, not deleted) | categoryRepo.GetByUserID (all, then filter in code) | DIFF — Python filters at DB level, Go filters in memory |
| Build flat structure | Parent >> Child naming, SQL-based | Parent >> Child naming, in-memory from flat list | OK (same result) |
| Get transactions | SQL query with JOINs in ExpensesReportGenerator | transactionRepo.GetByUserID with filters | OK |
| Transaction filters | is_income=False, is_deleted=False, is_transfer=False, exclude_from_reports=False | Type="expense" filter + in-code: ExcludeFromReports, IsIncome, IsTransfer, CategoryID nil | OK |
| Currency conversion | Uses calc_amount (exchange rate service) | Uses baseCurrencyAmount field (pre-calculated) | DIFF — Python converts at query time, Go uses pre-stored base amount |
| Base currency | From user.base_currency | From user.BaseCurrencyID -> currencyRepo.GetByID | OK |
| Hide empty categories | Filter in Python service | Filter in Go handler | OK |
| Grandchild categories | Included (all children of all parents) | Skipped (only top-level parents and direct children) | DIFF — Go skips grandchildren |
| AccessDenied handling | Not handled for this endpoint | Not handled (no AccessDenied path) | OK |

## Notes

- Python uses `ExpensesReportGenerator` class with SQL queries joining Transaction, UserCategory, Account, Currency tables. Go does this in-handler with repository calls.
- The `categories` field in the request is accepted by both but Python uses it to filter DB query; Go currently does not filter by the `categories` request field (it gets all expense categories regardless).
- Python converts amounts using exchange rates at query time via `calc_amount`. Go uses the pre-calculated `baseCurrencyAmount` field on transactions.
- Go skips grandchild categories (categories whose parent is not a top-level category). Python includes all depth levels.

## Tests

### Python Tests (4 total)

| Test | Verifies |
|------|----------|
| `test_expenses_by_categories_success` | 200, returns list |
| `test_expenses_by_categories_hide_empty` | 200 with hideEmptyCategories=true |
| `test_expenses_by_categories_unauthorized` | 422 (missing auth header) |
| `test_expenses_by_categories_internal_error` | 500 (mocked exception) |

### Go Integration Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestExpensesByCategoriesSuccess` | 200, returns array |
| `TestExpensesByCategoriesHideEmpty` | 200 with hideEmptyCategories=true |
| `TestExpensesByCategoriesUnauthorized` | 401 (no auth) |
| `TestExpensesByCategoriesMissingDates` | 422 (missing required fields) |
| `TestExpensesByCategoriesWithTransactions` | 200 with actual expense transactions |
| `TestExpensesByCategoriesWithSpecificCategories` | 200 with categories filter |
| `TestExpensesByCategoriesInvalidBody` | 422 (malformed JSON) |

### Go Unit Tests (10 total)

| Test | Verifies |
|------|----------|
| `TestExpensesByCategoriesCategoryError` | 500 (category repo error) |
| `TestExpensesByCategoriesTransactionError` | 500 (transaction repo error) |
| `TestExpensesByCategoriesWithParentCategory` | 200 with parent/child categories |
| `TestExpensesByCategoriesNilCategory` | 200 with non-existent category ID |
| `TestExpensesByCategoriesExcludedAndIncomeTransactions` | Excluded/income/nil-category transactions filtered |
| `TestExpensesByCategoriesHideEmptyCategories` | Empty categories hidden |
| `TestExpensesByCategoriesUserRepoError` | 401 (user repo error) |
| `TestExpensesByCategoriesUserNotActive` | 401 (user not active) |
| `TestExpensesByCategoriesUserError` (via ExpensesByCategories flow) | 500 (user lookup for base currency fails) |
| `TestExpensesByCategoriesCurrencyError` (via base currency lookup) | 500 (currency repo error) |
