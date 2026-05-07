# Endpoint #51: GET /planned-transactions/

## Route Definition
- **Python**: `@router.get('/', response_model=list[ResponsePlannedTransactionSchema])`
- **Go**: `g.GET("/", h.List)`

## Request
- Auth: required (both)
- Query params (Python): `account_ids` (list[int]), `from_date` (str), `to_date` (str), `is_recurring` (bool), `is_executed` (bool), `is_active` (bool), `include_inactive` (bool, default False)
- Query params (Go): `account_ids` (comma-separated string), `from_date` (string), `to_date` (string), `is_recurring` (string), `is_executed` (string), `is_active` (string), `include_inactive` (string, "true" to enable)

## Response
- **Python**: 200 OK with `list[ResponsePlannedTransactionSchema]`
- **Go**: 200 OK with `[]PlannedTxResponse`
- Both return the same logical fields (id, userId, currencyId, amount, label, notes, isIncome, plannedDate, isRecurring, recurrenceRule, isExecuted, executedTransactionId, executionDate, isActive, createdAt, updatedAt)
- Python uses camelCase via alias_generator; Go uses explicit camelCase JSON tags

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token dependency) | 401 (via auth middleware) |
| Internal error | 500 `"Error fetching planned transactions"` | 500 `"Failed to get planned transactions"` |
| User not active | N/A (no check) | 401 `"User not activated"` (via RequireActiveUser middleware) |

## Business Logic Comparison
1. **Auth**: Python uses `check_token` dependency; Go uses auth middleware + `RequireActiveUser` middleware.
2. **Filter parsing**: Python receives `account_ids` as native `list[int]` via FastAPI Query; Go splits a comma-separated string manually.
3. **Filter types**: Python passes `is_recurring`/`is_executed`/`is_active` as `bool | None`; Go passes them as raw strings (truthy check deferred to repository).
4. **Service layer**: Python delegates to `pt_service.get_planned_transactions()`; Go delegates to `service.List()` which converts filters and calls the repository.
5. **Response mapping**: Python relies on Pydantic ORM mode; Go manually maps via `toResponse()`.

## Notes
- Python has subscription plan enforcement (`enforce_free_plan_compliance` dependency) which is absent in Go.
- Go adds `RequireActiveUser` middleware (is_active/is_deleted) verification not present in Python.
- Error messages differ slightly ("Error fetching" vs "Failed to get").
- Go parses `account_ids` from a comma-separated string, which differs from Python's native list query param handling.

## Tests
- **Python**: 10 tests in `back-fastapi/app/tests/endpoints/test_planned_transactions_endpoints.py` (test_get_planned_transactions_* and related filter tests)
- **Go integration tests**: 5 tests in `handler_test.go` (TestListPlannedTxSuccess, TestListPlannedTxEmpty, TestListPlannedTxUnauthorized, TestListPlannedTxInvalidToken, TestListPlannedTxWithRecurring, TestListPlannedTxIncome, TestListMultiplePlannedTx)
- **Go unit tests**: 2 tests in `handler_unit_test.go` (TestListDBError, TestListSuccess, TestListUserNotActive)
