# Endpoint #53: GET /planned-transactions/:id

## Route Definition
- **Python**: `@router.get('/{planned_transaction_id}', response_model=ResponsePlannedTransactionSchema)`
- **Go**: `g.GET("/:id", h.GetByID)`

## Request
- Auth: required (both)
- Path params: `planned_transaction_id` (Python) / `id` (Go) -- integer

## Response
- **Python**: 200 OK with `ResponsePlannedTransactionSchema`
- **Go**: 200 OK with `PlannedTxResponse`
- Both return the same fields: id, userId, currencyId, amount, label, notes, isIncome, plannedDate, isRecurring, recurrenceRule, isExecuted, executedTransactionId, executionDate, isActive, createdAt, updatedAt

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Not found | 404 `"Planned transaction not found"` | 404 `"Planned transaction not found"` |
| Access denied (wrong user) | 403 `"Access denied"` | 403 `"Access denied"` |
| Internal error | 500 `"Error fetching planned transaction"` | N/A (not explicitly handled) |
| Invalid ID | N/A (FastAPI validates path param type) | 400 `"Invalid ID"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **ID parsing**: Python gets typed `int` from FastAPI path param; Go manually parses with `strconv.Atoi()` and returns 400 on failure.
2. **Lookup**: Python calls `pt_service.get_planned_transaction_by_id()` which raises `NoResultFound` or `AccessDenied`; Go calls `service.GetByID()` which does repo call + ownership check.
3. **Ownership check**: Python delegates to service layer; Go also delegates to service layer (service checks `tx.UserID != userID` and returns `ErrAccessDenied`).
4. **Error separation**: Python catches `NoResultFound` separately from `AccessDenied`; Go service returns `ErrNotFound` or `ErrAccessDenied`, and the handler maps these to HTTP status codes.

## Notes
- Go returns 404 for any repository error (not just "not found"), which could mask DB errors.
- Python has a generic 500 catch-all; Go does not have one for this endpoint.
- Go adds `RequireActiveUser` middleware verification.
- Python has subscription plan enforcement.

## Tests
- **Python**: 4 tests (test_get_planned_transaction_success, _not_found, _other_user, _internal_error)
- **Go integration tests**: 5 tests (TestGetByIDSuccess, TestGetByIDNotFound, TestGetByIDOtherUser, TestGetByIDUnauthorized, TestGetByIDInvalidID)
- **Go unit tests**: 4 tests (TestGetByIDInvalidID, TestGetByIDNotFound, TestGetByIDAccessDenied, TestGetByIDSuccess, TestGetByIDUserNotActive)
