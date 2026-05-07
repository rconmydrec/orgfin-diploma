# Endpoint #50: POST `/planned-transactions/`

**Status**: DIFF
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /planned-transactions/` | `POST /planned-transactions/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) |
| File | `app/routes/planned_transactions.py:29` | `internal/handlers/planned_transactions/handler.go` (handler) + `internal/services/planned_transactions/service.go` (service) |
| Success status | 201 Created | 201 Created | OK |

## Request

**Body (JSON)**:

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `amount` | Decimal (required) | decimal.Decimal (`validate:"required"`) | OK |
| `label` | str (default `""`) | string | OK |
| `notes` | str or None (default `""`) | *string | OK |
| `accountId` / `account_id` | int or None (optional) | *int (`json:"accountId"`) | OK |
| `isIncome` / `is_income` | bool (required) | bool (`json:"isIncome"`) | OK |
| `plannedDate` / `planned_date` | datetime (required) | time.Time (`validate:"required"`) | OK |
| `isRecurring` / `is_recurring` | bool (default `False`) | bool (`json:"isRecurring"`) | OK |
| `recurrenceRule` / `recurrence_rule` | RecurrenceRuleSchema or None | *RecurrenceRuleDTO | OK |

**RecurrenceRule fields**:

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `frequency` | RecurrenceFrequencyEnum (required) | string | DIFF -- Python validates enum (daily/weekly/monthly/yearly); Go accepts any string |
| `interval` | int (>= 1, default 1) | int | DIFF -- Python validates >= 1; Go has no validation |
| `endDate` / `end_date` | datetime or None | *time.Time | OK |
| `count` | int or None (>= 1) | *int | DIFF -- Python validates >= 1; Go has no validation |
| `dayOfMonth` / `day_of_month` | int or None (1-31) | *int | DIFF -- Python validates 1-31; Go has no validation |

## Response

**Success**: 201 Created. Planned transaction object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `userId` | int | int | OK |
| `currencyId` | int | int | OK |
| `amount` | float (serialized from Decimal) | decimal.Decimal (string in JSON) | DIFF -- Python returns float, Go returns string |
| `label` | str | string | OK |
| `notes` | str or None | *string | OK |
| `isIncome` | bool | bool | OK |
| `plannedDate` | datetime | time.Time | OK |
| `isRecurring` | bool | bool | OK |
| `recurrenceRule` | RecurrenceRuleSchema or None | *RecurrenceRuleDTO | OK |
| `isExecuted` | bool | bool | OK |
| `executedTransactionId` | int or None | *int | OK |
| `executionDate` | datetime or None | *time.Time | OK |
| `isActive` | bool | bool | OK |
| `createdAt` | datetime | time.Time | OK |
| `updatedAt` | datetime | time.Time | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 401 | (middleware) | 401 | (middleware) | OK |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go has is_active check |
| Invalid body | 422 | Pydantic validation | 422 | "Invalid request data" | OK |
| Validation failure | 422 | Pydantic details | 422 | "Invalid request data" | OK |
| Planning date limit exceeded | 402 | PlanningDateLimitExceededError message | N/A | N/A | DIFF -- Python has subscription date limit check; Go does not |
| Account access denied | 403 | EntityAccessDeniedError message | N/A | N/A | DIFF -- Python checks account access via subscription; Go does not |
| Invalid account | 400 | InvalidAccount message | N/A | N/A | DIFF -- Python validates account; Go does not |
| DB error | 500 | "Error creating planned transaction" | 500 | "Failed to create planned transaction" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES (RequireActiveUser middleware) | DIFF -- Go checks user is_active |
| Subscription date check | YES (check_planned_transaction_date) | NO | DIFF -- Python checks planning date against subscription limits |
| Account access check | YES (check_entity_access for account) | NO | DIFF -- Python validates account ownership via subscription service |
| Currency determination | From user's `base_currency_id` | From user's `BaseCurrencyID`; returns 400 if not set | OK -- Both use user's base currency |
| User existence check | YES (queries User, raises ValueError) | Via RequireActiveUser middleware | DIFF -- Python checks user exists and has base_currency; Go checks is_active |
| Base currency validation | YES (raises if user.base_currency not set) | YES (service returns ErrBaseCurrencyNotSet → 400) | OK -- Both validate base currency is set |
| Recurrence rule validation | YES (validates is_recurring matches recurrence_rule presence) | NO | DIFF -- Python has cross-field validation |
| End_date/count exclusivity | YES (cannot have both end_date and count) | NO | DIFF -- Python validates |
| RecurrenceRule storage | JSON dict (model_dump) | JSON string (json.Marshal) | OK (both stored as JSON in DB) |
| is_active default | True (model default) | True (explicitly set in service) | OK |
| accountId handling | Stored on model (optional, validated) | Stored on model (optional, no validation) | DIFF -- Python validates account exists and belongs to user |
| Response status | 201 Created | 201 Created | OK |

## Notes

- **Currency handling**: Both Python and Go use the user's `base_currency_id`. Go validates it is set (service returns 400 if not), matching Python's behavior. The Go handler delegates to a service layer (`services/planned_transactions/`), similar to Python's architecture.
- Python has extensive subscription-related checks (planning date limits, account access) that Go does not implement.
- Python validates recurrence rule consistency (is_recurring must match recurrence_rule presence, end_date and count are mutually exclusive). Go accepts any combination.
- Python validates that `account_id` belongs to the user via subscription service. Go does not validate account ownership.
- Both use 201 Created status for successful creation (matching Python's explicit `status_code=status.HTTP_201_CREATED`).
- Amount serialization differs: Python serializes Decimal to float; Go serializes Decimal to string.

## Tests

### Python Tests (0 total)

No dedicated test files found for planned transaction endpoints.

### Go Integration Tests (7 total for create)

| Test | Verifies |
|------|----------|
| `TestCreatePlannedTxSuccess` | 201, basic creation |
| `TestCreatePlannedTxWithRecurrenceRule` | 201, with weekly recurrence |
| `TestCreatePlannedTxIncome` | 201, income with monthly recurrence and dayOfMonth |
| `TestCreatePlannedTxWithNotes` | 201, notes field |
| `TestCreatePlannedTxUnauthorized` | 401 without auth token |
| `TestCreatePlannedTxInvalidJSON` | 422 for invalid JSON body |

### Go Unit Tests (7 total for create)

| Test | Verifies |
|------|----------|
| `TestCreateBindError` | 422 for invalid JSON |
| `TestCreateValidateError` | 422 for empty body |
| `TestCreateDBError` | 500 on DB create failure |
| `TestCreateSuccess` | 201 success path |
| `TestCreateWithRecurrenceRule` | 201 with recurrence rule |
| `TestCreateUserNotActive` | 401 when user is inactive |
| `TestCreateUserRepoError` | 401 when user repo returns error |
