# Endpoint #47: DELETE `/budgets/{id}/`

**Status**: DIFF
**Date**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /budgets/{id}/` | `DELETE /budgets/:id/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| Handler | `app/routes/budgets.py:92` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/services/budget_service.py` | `internal/services/budgets/service.go` (`Delete`) |

## Architecture

The Go handler is a thin HTTP adapter that:
1. Parses the budget ID from the path parameter.
2. Delegates to `service.Delete(userID, budgetID)`.
3. Maps service sentinel errors to HTTP status codes via `handleServiceError`.

The service handles ownership check (GetByID + UserID comparison) and delegates to the repository for the soft delete.

## Request

**Path parameter**: `id` (int) -- budget ID.

Both: DELETE with no body. Auth required.

## Response

**Success**: 200 OK.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `message` | `"Budget with id {id} deleted"` | `"Budget with id {id} deleted"` | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 401 | (middleware) | 401 | (middleware) | OK |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go has is_active check |
| Invalid ID (non-numeric) | N/A (FastAPI path validation) | N/A | 400 | "Invalid budget ID" | DIFF -- Go handles manually |
| Other user's budget | 403 | "Access denied" (EntityAccessDeniedError) | 403 | "Access denied" | OK |
| Budget not found | 404 | "Budget not found" | 404 | "Budget not found" | OK |
| DB error | 500 | "Error deleting budget" | 500 | "Failed to delete budget" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES (RequireActiveUser middleware) | DIFF -- Go checks user is_active |
| Subscription access check | YES (check_entity_access) | NO | DIFF -- Python checks subscription access |
| Parse budget ID | From path (FastAPI typed) | From path (handler: `strconv.Atoi(c.Param("id"))`) | OK |
| Ownership check | In service (query by user_id AND budget_id) | In service (GetByID then compare UserID) | OK (same result) |
| Delete type | Soft delete (`is_deleted = True`) | Soft delete (`is_deleted = true`) | OK |
| Not found check | Service raises NotFoundError (query returns None) | Service returns `ErrNotFound` | OK |
| Response format | `{"message": "Budget with id {id} deleted"}` | `{"message": "Budget with id {id} deleted"}` | OK |

## Notes

- Both implementations use soft delete (set `is_deleted = true`).
- Python checks subscription access before deletion (premium feature); Go does not.
- Go adds user is_active check; Python relies on `enforce_free_plan_compliance` dependency.
- The ownership check is now in the Go service layer (was previously in the handler before refactoring).
- Success response messages are identical in format.

## Tests

### Python Tests (0 total)

No dedicated test files found for budget endpoints.

### Go Integration Tests (4 total for delete)

| Test | Verifies |
|------|----------|
| `TestDeleteBudgetSuccess` | 200, successful deletion |
| `TestDeleteBudgetNotFound` | 404 for nonexistent budget |
| `TestDeleteBudgetOtherUser` | 403 for another user's budget |
| `TestDeleteBudgetInvalidID` | 400 for non-numeric ID |

### Go Unit Tests (5 total for delete)

| Test | Verifies |
|------|----------|
| `TestDeleteBudgetInvalidID` | 400 for non-numeric path param |
| `TestDeleteBudgetNotFound` | 404 when budget not found in DB |
| `TestDeleteBudgetAccessDenied` | 403 for different user |
| `TestDeleteBudgetDBError` | 500 on DB delete failure |
| `TestDeleteBudgetSuccess` | 200 success path |
