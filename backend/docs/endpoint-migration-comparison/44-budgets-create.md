# Endpoint #44: POST `/budgets/add/`

**Status**: OK
**Date**: 2026-02-28 (last updated 2026-05-01: Go now recalculates `collectedAmount` from existing matching transactions on create, matching Python)

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /budgets/add/` | `POST /budgets/add/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| Handler | `app/routes/budgets.py:35` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/services/budget_service.py` | `internal/services/budgets/service.go` (`Create`) |

## Architecture

The Go handler is a thin HTTP adapter that:
1. Binds and validates the JSON request body.
2. Passes raw string dates and params to `service.Create(userID, params)`.
3. Maps service sentinel errors to HTTP status codes via `handleServiceError`.

All business logic (date parsing, currency validation, period normalization, categories encoding, post-persistence recalculation) is in the service layer.

## Request

**Method**: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `name` | str (required) | string (required, validate:"required") | OK |
| `currencyId` | int (required, camelCase alias from `currency_id`) | int (required, validate:"required") | OK |
| `targetAmount` | Decimal (required, camelCase alias) | decimal.Decimal (required, validate:"required") | OK |
| `period` | PeriodEnum (required, values: daily/weekly/monthly/yearly/custom) | string (required, validate:"required"); normalized to uppercase via `strings.ToUpper` in service | DIFF -- Python validates against enum, Go accepts any string but normalizes to uppercase |
| `repeat` | bool (required) | bool | OK |
| `startDate` | datetime (required, camelCase alias) | string (required, validate:"required"); parsed by service via `dateutil.ParseDate` | OK -- Go accepts multiple formats including bare date `YYYY-MM-DD` |
| `endDate` | datetime (required, camelCase alias) | string (required, validate:"required"); parsed by service via `dateutil.ParseDate` | OK |
| `categories` | list[int] (required) | []int | OK |
| `comment` | str or None (optional) | *string (optional) | OK |

## Response

**Success**: 200 OK. Created budget object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `name` | str | string | OK |
| `currencyId` | int (camelCase alias) | int | OK |
| `targetAmount` | Decimal (serialized as float) | float64 (via `InexactFloat64()`) | OK -- both return JSON numbers |
| `collectedAmount` | Decimal (serialized as float; recalculated from matching transactions) | float64 (via `InexactFloat64()`; recalculated from matching transactions) | OK -- Go now recalculates on create matching Python |
| `period` | PeriodEnum (e.g., "monthly") | string (lowercase, e.g., "monthly") | OK -- both return lowercase |
| `repeat` | bool | bool | OK |
| `startDate` | datetime | time.Time (RFC3339) | OK |
| `endDate` | datetime (minus 1 day in get_user_budgets) | time.Time (minus 1 day in toBudgetResponse) | OK |
| `includedCategories` | str (comma-separated IDs) | string (comma-separated IDs) | OK |
| `isArchived` | bool | bool | OK |
| `comment` | str or None | *string | OK |
| `currency` | object {id, code, name} (required) | object {id, code, name} (omitempty) | DIFF -- Python always includes, Go uses omitempty |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Unauthorized | 422 (missing header) | FastAPI validation | 401 | JWT middleware | DIFF |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go adds is_active check |
| Budget limit exceeded | 402 | BudgetLimitExceededError message | N/A | N/A | DIFF -- Go has no subscription limit check |
| Invalid body | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Missing required fields | 422 | FastAPI validation | 422 | "Validation failed" | OK |
| Invalid start date | N/A | N/A | 400 | "Invalid start date format" | DIFF -- Go validates date format in service |
| Invalid end date | N/A | N/A | 400 | "Invalid end date format" | DIFF -- Go validates date format in service |
| Invalid currency | N/A (not explicitly checked) | 500 (DB error) | 400 | "Invalid currency" | DIFF -- Go validates currency explicitly |
| Internal error | 500 | "Error generting report" | 500 | "Failed to create budget" | DIFF -- different messages (Python has typo) |
| Recalculation failure after persistence | 500 (would propagate) | "Error generting report" | 200 | persisted budget returned, error logged | DIFF -- Go returns 200 by design (self-healing); Python would 500 |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | DIFF -- Go is stricter |
| Free plan enforcement | YES | NO | DIFF |
| Subscription budget limit | YES (check_budget_limit) | NO | DIFF -- Go lacks subscription check |
| Validate currency | NO (implicit via DB FK) | YES (service: `currencyRepo.GetByID`) | DIFF -- Go validates explicitly |
| Parse dates | implicit (Pydantic datetime) | service: `dateutil.ParseDate` (multiple formats) | DIFF -- Go explicitly parses, supports bare dates |
| Period normalization | N/A (PeriodEnum) | service: `strings.ToUpper` | DIFF -- Go normalizes to uppercase |
| Filter categories | YES (filters against user's actual categories) | NO (accepts all category IDs as-is) | DIFF -- Python validates categories belong to user |
| End date +1 day | YES (timedelta(days=1)) | YES (service: `AddDate(0, 0, 1)`) | OK |
| Fill with existing transactions | YES (fill_budget_with_existing_transactions) | YES (service: `recalculateBudget` after persistence) | OK -- Go now recalculates synchronously after create |
| Failure handling on recalc | Propagates as 500 | Logs error, returns 200 with persisted budget | DIFF -- Go is more permissive (self-healing on next transaction CRUD) |
| Comment handling | `str(budget_dto.comment)` -- converts None to "None" string | Stored as-is (*string, nullable) | DIFF -- Python bug: stores "None" string for null comments |
| Period validation | YES (PeriodEnum: daily/weekly/monthly/yearly/custom) | NO (any string accepted, normalized to uppercase) | DIFF |
| Response includes currency | YES (from ORM relationship) | YES (set from currencyRepo result in service) | OK |

## Notes

- Python's `create_new_budget` service also handles updates (checks if `id` is provided). Go has a separate `UpdateBudget` handler.
- Python validates that provided category IDs belong to the user's categories by filtering against the DB. Go stores whatever category IDs are sent without validation.
- Both Python and Go now recalculate `collectedAmount` from existing transactions immediately after persistence. The Go implementation calls `recalculateBudget` (single-budget) directly from `Service.Create`; if this fails after persistence has succeeded, Go logs the error and still returns 200 with the persisted budget — the stale `collectedAmount` is self-healing on the next transaction CRUD or manual recalc.
- Python converts `comment` with `str()` which turns `None` into the string `"None"` -- this is likely a bug. Go stores the raw `*string` pointer, preserving null.
- Python validates `period` against `PeriodEnum` (daily, weekly, monthly, yearly, custom). Go accepts any string value but normalizes it to uppercase before storage.
- Both add 1 day to `endDate` before storing (to include the full end date), and subtract 1 day when returning in responses.

## Tests

### Python Tests (5 total)

| Test | Verifies |
|------|----------|
| `test_create_budget_success` | 200, correct name and targetAmount |
| `test_create_budget_limit_exceeded` | 402 (mocked BudgetLimitExceededError) |
| `test_create_budget_invalid_category` | 500 (invalid category ID) |
| `test_create_budget_negative_amount` | 200 (no validation on negative amounts) |
| `test_create_budget_unauthorized` | 422 (missing auth header) |

### Go Integration Tests (11 total)

| Test | Verifies |
|------|----------|
| `TestCreateBudgetSuccess` | 200, correct name and targetAmount |
| `TestCreateBudgetDateOnlyFormat` | 200, accepts date-only format |
| `TestCreateBudgetLowercasePeriod` | 200, lowercase period normalized to uppercase |
| `TestCreateBudgetInvalidCurrency` | 400 (invalid currency ID) |
| `TestCreateBudgetUnauthorized` | 401 (no auth) |
| `TestCreateBudgetMissingName` | 422 (validation failure) |
| `TestCreateBudgetWithRepeat` | 200 with repeat=true |
| `TestCreateBudgetWeeklyPeriod` | 200 with WEEKLY period |
| `TestCreateBudgetYearlyPeriod` | 200 with YEARLY period |
| `TestCreateBudgetWithComment` | 200 with comment field |
| `TestCreateBudgetRecalculatesFromExistingTransactions` | 200, `collectedAmount` reflects sum of pre-existing matching expense transactions |

### Go Unit Tests (8 handler + 2 service-recalc total)

Handler unit tests:

| Test | Verifies |
|------|----------|
| `TestCreateBudgetBindError` | 422 (malformed JSON) |
| `TestCreateBudgetValidateError` | 422 (empty body) |
| `TestCreateBudgetInvalidCurrency` | 400 (currency repo error) |
| `TestCreateBudgetDBError` | 500 (budget repo error) |
| `TestCreateBudgetSuccess` | 200 (with categories) |
| `TestCreateBudgetDateOnlyFormat` | 200 (date-only format) |
| `TestCreateBudgetInvalidStartDateFormat` | 400 (bad start date) |
| `TestCreateBudgetInvalidEndDateFormat` | 400 (bad end date) |

Service unit tests (recalculation contract):

| Test | Verifies |
|------|----------|
| `TestCreate_RecalculatesCollectedAmount` | Service returns recalculated `collectedAmount` after successful create |
| `TestCreate_RecalcFailureReturns200` | Recalc failure after persistence is logged; persisted budget is still returned |
