# Endpoint #46: PUT `/budgets/{id}/`

**Status**: DIFF (collected_amount recalculation now matches Python parity as of 2026-05-01; remaining DIFFs are unrelated: budget ID source, subscription access check, categories filtering, comment-None handling)

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /budgets/{id}/` | `PUT /budgets/:id/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| Handler | `app/routes/budgets.py:54` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/services/budget_service.py` | `internal/services/budgets/service.go` (`Update`) |

## Architecture

The Go handler is a thin HTTP adapter that:
1. Parses the budget ID from the path parameter.
2. Binds and validates the JSON request body.
3. Delegates to `service.Update(userID, budgetID, params)`.
4. Maps service sentinel errors to HTTP status codes via `handleServiceError`.

All business logic (date parsing, ownership check, currency validation, period normalization, post-persistence recalculation) is in the service layer.

## Request

**Path parameter**: `id` (int) -- budget ID.

**Body (JSON)**:

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int (required, from body via EditBudgetInputSchema) | `*int` (optional, from body) | DIFF -- Python requires `id` in body; Go reads `id` from path param |
| `name` | str (required) | string (required, `validate:"required"`) | OK |
| `currencyId` / `currency_id` | int (camelCase via alias_generator) | int (`json:"currencyId"`, `validate:"required"`) | DIFF -- Python schema accepts both camelCase and snake_case; Go requires camelCase |
| `targetAmount` / `target_amount` | Decimal (required) | decimal.Decimal (`validate:"required"`) | OK |
| `period` | PeriodEnum (required) | string (`validate:"required"`); normalized to uppercase via `strings.ToUpper` in service | DIFF -- Python uses enum validation; Go accepts any string, normalizes to uppercase |
| `repeat` | bool (required) | bool | OK |
| `startDate` / `start_date` | datetime (required) | string (`validate:"required"`); parsed by service via `dateutil.ParseDate` | OK -- Go accepts multiple formats |
| `endDate` / `end_date` | datetime (required) | string (`validate:"required"`); parsed by service via `dateutil.ParseDate` | OK |
| `categories` | list[int] (required) | []int | OK |
| `comment` | str or None | *string | OK |

## Response

**Success**: 200 OK. Budget object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `name` | str | string | OK |
| `currencyId` | int | int | OK |
| `targetAmount` | float (serialized from Decimal) | float64 (via `InexactFloat64()`) | OK -- both return JSON numbers |
| `collectedAmount` | float (recalculated from matching transactions after update) | float64 (recalculated from matching transactions after update via `recalculateBudget`) | OK -- Go now recalculates on update matching Python |
| `period` | PeriodEnum (e.g., "monthly") | string (lowercase, e.g., "monthly") | OK -- both return lowercase |
| `repeat` | bool | bool | OK |
| `startDate` | datetime | time.Time | OK |
| `endDate` | datetime (minus 1 day in get_user_budgets) | time.Time (minus 1 day in toBudgetResponse) | OK |
| `includedCategories` | str | string | OK |
| `isArchived` | bool | bool | OK |
| `comment` | str or None | *string | OK |
| `currency` | CurrencySchema (required) | *CurrencyDTO (omitempty) | DIFF -- Python always includes currency; Go may omit if nil |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 401 | (middleware) | 401 | (middleware) | OK |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go has is_active check |
| Invalid body | 422 | Pydantic validation | 422 | "Invalid request data" | OK |
| Missing required fields | N/A | Pydantic validation | 422 | "Validation failed" | OK |
| Invalid ID (non-numeric) | N/A (FastAPI path validation) | N/A | 400 | "Invalid budget ID" | DIFF -- Go handles manually |
| Invalid start date | N/A | N/A | 400 | "Invalid start date format" | DIFF -- Go validates date format in service |
| Invalid end date | N/A | N/A | 400 | "Invalid end date format" | DIFF -- Go validates date format in service |
| Budget not found | 500 | "Error updating budget" | 404 | "Budget not found" | DIFF -- Python returns 500 (caught by generic exception); Go returns 404 |
| Other user's budget | 403 | "Access denied" (EntityAccessDeniedError) | 403 | "Access denied" | OK |
| Invalid currency | N/A | N/A | 400 | "Invalid currency" | DIFF -- Go validates currency; Python doesn't at route level |
| DB error on update | 500 | "Error updating budget" | 500 | "Failed to update budget" | OK |
| Recalculation failure after persistence | 500 (would propagate) | "Error updating budget" | 200 | persisted budget returned, error logged | DIFF -- Go returns 200 by design (self-healing); Python would 500 |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES (check_token) | YES (auth middleware) | OK |
| is_active check | NO (only enforce_free_plan_compliance) | YES (RequireActiveUser middleware) | DIFF -- Go checks user is_active |
| Subscription access check | YES (check_entity_access) | NO | DIFF -- Python checks subscription access |
| Budget ID source | From body (EditBudgetInputSchema.id) | From path param `c.Param("id")` | DIFF |
| Ownership check | Implicit (service queries by user_id AND budget_id) | Explicit (service: GetByID then compare UserID) | OK (same result) |
| Parse dates | implicit (Pydantic datetime) | service: `dateutil.ParseDate` (multiple formats) | DIFF -- Go explicitly parses, supports bare dates |
| Validate currency | NO (at route level) | YES (service: `currencyRepo.GetByID`) | DIFF -- Go validates currency exists |
| Period normalization | N/A (PeriodEnum) | service: `strings.ToUpper` | DIFF -- Go normalizes to uppercase |
| Categories filtering | YES (filters against user's actual categories) | NO (uses categories as-is) | DIFF -- Python validates categories belong to user |
| End date +1 day | YES (`budget_dto.end_date + timedelta(days=1)`) | YES (service: `endDate.AddDate(0, 0, 1)`) | OK |
| Reset collected_amount | YES (sets to 0, then recalculates from transactions) | YES (recalculated from scratch via `recalculateBudget` after persistence) | OK -- both recompute from scratch on update |
| Fill with existing transactions | YES (fill_budget_with_existing_transactions) | YES (service: `recalculateBudget` after persistence) | OK -- Go now recalculates synchronously after update |
| Failure handling on recalc | Propagates as 500 | Logs error, returns 200 with persisted budget | DIFF -- Go is more permissive (self-healing on next transaction CRUD) |
| Comment handling | Converts to str (`str(budget_dto.comment)`) | Stores *string as-is | DIFF -- Python converts None to "None" string |

## Notes

- Python uses the same `create_new_budget` service for both create and update, detecting update via `hasattr(budget_dto, "id")`.
- Go separates create and update logic in the service with distinct flows (`Create` and `Update` methods).
- Both Python and Go now recompute `collectedAmount` from scratch from existing transactions on every update (matching newly applied filters: changed `included_categories`, date window, and `currencyId`). Go achieves this via a synchronous call to `recalculateBudget` after persistence; if recalc fails after persistence has succeeded, Go logs and returns 200 with the persisted budget (self-healing on next transaction CRUD or manual recalc).
- Python validates that categories belong to the user; Go passes them through without validation.
- Python checks subscription entity access (premium feature); Go does not implement subscription checks.
- Go adds currency validation and date parsing validation that Python lacks at the route level.
- Go normalizes the period to uppercase before storage.

## Tests

### Python Tests (0 total)

No dedicated test files found for budget endpoints.

### Go Integration Tests (11 total for update)

| Test | Verifies |
|------|----------|
| `TestUpdateBudgetSuccess` | 200, name and targetAmount updated |
| `TestUpdateBudgetDateOnlyFormat` | 200, accepts date-only format |
| `TestUpdateBudgetLowercasePeriod` | 200, lowercase period normalized to uppercase |
| `TestUpdateBudgetNotFound` | 404 for nonexistent budget |
| `TestUpdateBudgetOtherUser` | 403 for another user's budget |
| `TestUpdateBudgetInvalidID` | 400 for non-numeric ID |
| `TestUpdateBudgetUnauthorized` | 401 without auth token |
| `TestUpdateBudgetAddCategoryIncreasesCollectedAmount` | 200, `collectedAmount` grows when a category with pre-existing matching transactions is added |
| `TestUpdateBudgetRemoveCategoryDecreasesCollectedAmount` | 200, `collectedAmount` shrinks when a category is removed |
| `TestUpdateBudgetChangeWindowRecalculates` | 200, `collectedAmount` recalculated when `startDate` / `endDate` change |
| `TestUpdateBudgetChangeCurrencyRecalculatesWithConversion` | 200, `collectedAmount` recalculated with FX conversion when `currencyId` changes |

### Go Unit Tests (11 handler + 2 service-recalc total)

Handler unit tests:

| Test | Verifies |
|------|----------|
| `TestUpdateBudgetInvalidID` | 400 for non-numeric path param |
| `TestUpdateBudgetBindError` | 422 for invalid JSON |
| `TestUpdateBudgetValidateError` | 422 for empty body |
| `TestUpdateBudgetNotFound` | 404 when budget not found in DB |
| `TestUpdateBudgetAccessDenied` | 403 for different user |
| `TestUpdateBudgetInvalidCurrency` | 400 for nonexistent currency |
| `TestUpdateBudgetDBError` | 500 on DB update failure |
| `TestUpdateBudgetSuccess` | 200 success path |
| `TestUpdateBudgetDateOnlyFormat` | 200, date-only format parsed correctly |
| `TestUpdateBudgetInvalidStartDateFormat` | 400 for bad start date |
| `TestUpdateBudgetInvalidEndDateFormat` | 400 for bad end date |

Service unit tests (recalculation contract):

| Test | Verifies |
|------|----------|
| `TestUpdate_RecalculatesCollectedAmount` | Service overwrites stale pre-update `collectedAmount` with the recalculated total |
| `TestUpdate_RecalcFailureReturns200` | Recalc failure after persistence is logged; persisted budget is still returned |
