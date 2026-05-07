# Endpoint #45: GET `/budgets/`

**Status**: OK
**Date**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /budgets/` | `GET /budgets/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| Handler | `app/routes/budgets.py:73` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/services/budget_service.py` | `internal/services/budgets/service.go` (`GetByUserID`) |

## Architecture

The Go handler is a thin HTTP adapter that reads the `include` query parameter and delegates to `service.GetByUserID(userID, include)`. The service defaults `include` to `"all"` if empty, then passes to the repository.

## Request

**Method**: GET with optional query parameter.

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `include` | str query param (default "all") | string query param (default "all") | OK |

**Valid values for `include`**: `all`, `active`, `archived`.

## Response

**Success**: 200 OK. Array of budget objects.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `name` | str | string | OK |
| `currencyId` | int (camelCase alias) | int | OK |
| `targetAmount` | Decimal (serialized as float) | float64 (via `InexactFloat64()`) | OK -- both return JSON numbers |
| `collectedAmount` | Decimal (serialized as float) | float64 (via `InexactFloat64()`) | OK -- both return JSON numbers |
| `period` | PeriodEnum (e.g., "monthly") | string (lowercase, e.g., "monthly") | OK -- both return lowercase |
| `repeat` | bool | bool | OK |
| `startDate` | datetime | time.Time (RFC3339) | OK |
| `endDate` | datetime (minus 1 day) | time.Time (minus 1 day via toBudgetResponse) | OK |
| `includedCategories` | str (comma-separated IDs) | string (comma-separated IDs) | OK |
| `isArchived` | bool (camelCase alias) | bool | OK |
| `comment` | str or None | *string | OK |
| `currency` | object {id, code, name} (required via joinedload) | object {id, code, name} (omitempty) | DIFF -- Python always includes via eager load; Go uses omitempty |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Unauthorized | 422 (missing header) | FastAPI validation | 401 | JWT middleware | DIFF |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go adds is_active check |
| Invalid include param | 400 | "Invalid include parameter" | N/A | N/A | DIFF -- Python validates, Go passes to repo |
| Internal error | 500 | "Error getting budgets" | 500 | "Failed to get budgets" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | DIFF -- Go is stricter |
| Free plan enforcement | YES | NO | DIFF |
| Default include value | "all" | "all" (defaulted in service if empty) | OK |
| Filter: all | No additional filter | Passed to budgetRepo.GetByUserID | OK |
| Filter: active | `is_archived = False` | Passed to budgetRepo.GetByUserID | OK |
| Filter: archived | `is_archived = True` | Passed to budgetRepo.GetByUserID | OK |
| Invalid include value | Raises ValueError -> 400 | Passed to repo (behavior depends on repo impl) | DIFF -- Python validates explicitly |
| Filter: is_deleted | YES (`is_deleted = False`) | Depends on repo implementation | OK (likely both filter deleted) |
| Ordering | ORDER BY is_archived, end_date ASC, name ASC | Depends on repo implementation | DIFF -- Python defines explicit ordering |
| End date adjustment | Subtracts 1 day in service (before returning) | Subtracts 1 day in toBudgetResponse | OK |
| Currency eager loading | YES (joinedload(Budget.currency)) | Depends on repo implementation | OK |

## Notes

- Both implementations default the `include` parameter to `"all"` when not provided.
- Python validates the `include` parameter and returns 400 for invalid values. Go passes the value directly to the repository; invalid values may return all budgets or cause repo-level errors depending on implementation.
- Python uses `joinedload(Budget.currency)` to eagerly load the currency relationship. Go's response uses `omitempty` on the Currency field, so it may be omitted if not loaded.
- Both subtract 1 day from the stored `endDate` before returning in the response (to reverse the +1 day applied during creation).
- Python's ordering is explicit: `ORDER BY is_archived, end_date ASC, name ASC`. Go's ordering depends on the repository implementation.
- The response model (`BudgetSchema` in Python, `BudgetResponse` in Go) is the same as used in the create endpoint (#44).

## Tests

### Python Tests (8 total)

| Test | Verifies |
|------|----------|
| `test_get_budgets_success` | 200, returns list with at least 1 budget |
| `test_get_budgets_filter_all` | 200 with include=all |
| `test_get_budgets_filter_active` | 200, all returned budgets have isArchived=false |
| `test_get_budgets_filter_archived` | 200, all returned budgets have isArchived=true |
| `test_get_budgets_empty_list` | 200, empty list for user with no budgets |
| `test_get_budgets_unauthorized` | 422 (missing auth header) |
| `test_get_budgets_value_error` | 400 (mocked ValueError for invalid include) |
| `test_get_budgets_internal_error` | 500 (mocked exception) |

### Go Integration Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestGetBudgetsSuccess` | 200, at least 1 budget returned |
| `TestGetBudgetsFilterActive` | 200, all returned budgets have isArchived=false |
| `TestGetBudgetsEmptyList` | 200, empty list for new user |
| `TestGetBudgetsUnauthorized` | 401 (no auth) |
| `TestGetBudgetsFilterArchived` | 200, archived budget present |

### Go Unit Tests (2 total)

| Test | Verifies |
|------|----------|
| `TestGetBudgetsDBError` | 500 (budget repo error) |
| `TestGetBudgetsSuccess` | 200 with include=active, currency included |
