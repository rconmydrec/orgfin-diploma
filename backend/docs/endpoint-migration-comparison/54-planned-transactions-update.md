# Endpoint #54: PUT /planned-transactions/:id

## Route Definition
- **Python**: `@router.put('/{planned_transaction_id}', response_model=ResponsePlannedTransactionSchema)`
- **Go**: `g.PUT("/:id", h.Update)`

## Request
- Auth: required (both)
- Path params: `planned_transaction_id` (Python) / `id` (Go) -- integer
- Body (Python `UpdatePlannedTransactionSchema`): id, amount, label, notes, account_id, is_income, planned_date, is_recurring, recurrence_rule, is_active
- Body (Go `UpdateRequest`): id, accountId, amount, label, notes, isIncome, plannedDate, isRecurring, recurrenceRule, isActive

## Response
- **Python**: 200 OK with `ResponsePlannedTransactionSchema`
- **Go**: 200 OK with `PlannedTxResponse`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Not found | 404 `"Planned transaction not found"` | 404 `"Planned transaction not found"` |
| Access denied | 403 `"Access denied"` | 403 `"Access denied"` |
| Invalid data | 422 (Pydantic validation) | 422 `"Invalid request data"` |
| Invalid ID | N/A (FastAPI validates) | 400 `"Invalid ID"` |
| Planning date limit | 402 (PlanningDateLimitExceededError) | N/A |
| Entity access denied | 403 (EntityAccessDeniedError) | N/A |
| Internal error | 500 `"Error updating planned transaction"` | 500 `"Failed to update planned transaction"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Date validation**: Python calls `check_planned_transaction_date()` for subscription plan limits; Go does not.
2. **Account access**: Python calls `check_entity_access()` to verify account ownership; Go does not validate account ownership.
3. **Lookup + ownership**: Python delegates both to service; Go also delegates to `service.Update()` which does fetch, ownership check, field mutation, and save.
4. **Update flow**: Both fetch existing, apply changes, and save. Go's service modifies the model struct fields then calls the repository; Python delegates to service similarly.
5. **Recurrence rule**: Go marshals `RecurrenceRuleDTO` to JSON string; Python handles via Pydantic model.
6. **IsActive update**: Go only updates `IsActive` when `req.IsActive != nil` (pointer check); Python schema allows `is_active` as optional.
7. **Validation**: Python validates `recurrence_rule` consistency (required when `is_recurring` is True); Go has no such validation.

## Notes
- Go is missing subscription plan date validation and account access checks present in Python.
- Go is missing recurrence rule consistency validation (e.g., requiring rule when is_recurring is True).
- Python returns 402 for subscription limits; Go has no equivalent.
- Go adds `RequireActiveUser` middleware verification.

## Tests
- **Python**: 7 tests (test_update_planned_transaction_success, _not_found, _other_user, _planning_date_limit_exceeded, _entity_access_denied, _internal_error)
- **Go integration tests**: 8 tests (TestUpdatePlannedTxSuccess, TestUpdatePlannedTxWithRecurrenceRule, TestUpdatePlannedTxChangeIsActive, TestUpdatePlannedTxNotFound, TestUpdatePlannedTxUnauthorized, TestUpdatePlannedTxInvalidID, TestUpdatePlannedTxInvalidJSON, TestUpdatePlannedTxOtherUser)
- **Go unit tests**: 8 tests (TestUpdateInvalidID, TestUpdateBindError, TestUpdateNotFound, TestUpdateAccessDenied, TestUpdateDBError, TestUpdateSuccess, TestUpdateWithRecurrenceRule, TestUpdateUserNotActive)
