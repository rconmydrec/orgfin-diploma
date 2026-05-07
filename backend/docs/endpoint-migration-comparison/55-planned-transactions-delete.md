# Endpoint #55: DELETE /planned-transactions/:id

## Route Definition
- **Python**: `@router.delete('/{planned_transaction_id}')`
- **Go**: `g.DELETE("/:id", h.Delete)`

## Request
- Auth: required (both)
- Path params: `planned_transaction_id` (Python) / `id` (Go) -- integer

## Response
- **Python**: 200 OK with `{"deleted": true}`
- **Go**: 200 OK with `{"deleted": true}`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Not found | 404 `"Planned transaction not found"` | 404 `"Planned transaction not found"` |
| Access denied | 403 `"Access denied"` | 403 `"Access denied"` |
| Invalid ID | N/A (FastAPI validates) | 400 `"Invalid ID"` |
| Internal error | 500 `"Error deleting planned transaction"` | 500 `"Failed to delete planned transaction"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Delete type**: Python calls `pt_service.delete_planned_transaction()` which performs a soft delete; Go calls `service.Delete()` which performs a soft delete via the repository.
2. **Ownership check**: Python delegates to service which raises `AccessDenied`; Go also delegates to service which returns `ErrAccessDenied`.
3. **Flow**: Go's service does fetch-check-delete in 3 steps (similar to Python's service layer architecture).
4. **ID parsing**: Python gets typed int from FastAPI; Go uses `strconv.Atoi()`.

## Notes
- Both return `{"deleted": true}` on success.
- Go performs a soft delete (sets `is_deleted = true` and `is_active = false`) via the repository.
- Go adds `RequireActiveUser` middleware verification.
- Python has subscription plan enforcement.

## Tests
- **Python**: 5 tests (test_delete_planned_transaction_success, _not_found, _other_user, _unauthorized, _internal_error)
- **Go integration tests**: 5 tests (TestDeletePlannedTxSuccess, TestDeletePlannedTxNotFound, TestDeletePlannedTxUnauthorized, TestDeletePlannedTxOtherUser, TestDeletePlannedTxInvalidID)
- **Go unit tests**: 6 tests (TestDeleteInvalidID, TestDeleteNotFound, TestDeleteAccessDenied, TestDeleteDBError, TestDeleteSuccess, TestDeleteUserNotActive)
