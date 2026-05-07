# Endpoint #48: PUT `/budgets/{id}/archive/`

**Status**: DIFF
**Date**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /budgets/{budget_id}/archive/` | `PUT /budgets/:id/archive/` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check |
| Handler | `app/routes/budgets.py:114` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/services/budget_service.py` | `internal/services/budgets/service.go` (`Archive`) |
| Path param name | `budget_id` | `id` | DIFF (cosmetic) |

## Architecture

The Go handler is a thin HTTP adapter that:
1. Parses the budget ID from the path parameter.
2. Delegates to `service.Archive(userID, budgetID)`.
3. Maps service sentinel errors to HTTP status codes via `handleServiceError`.

The service handles ownership check (GetByID + UserID comparison) and delegates to the repository for the archive operation.

## Request

**Path parameter**: `budget_id` (Python) / `id` (Go) -- budget ID.

Both: PUT with no body. Auth required.

## Response

**Success**: 200 OK.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `message` | `"Budget with id {budget_id} archived"` | `"Budget with id {id} archived"` | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 401 | (middleware) | 401 | (middleware) | OK |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go has is_active check |
| Invalid ID (non-numeric) | N/A (FastAPI path validation) | 400 | "Invalid budget ID" | DIFF -- Go handles manually |
| Other user's budget | 403 | "Access denied" (EntityAccessDeniedError) | 403 | "Access denied" | OK |
| Budget not found | 404 | "Budget not found" | 404 | "Budget not found" | OK |
| DB error | 500 | "Error archiving budget" | 500 | "Failed to archive budget" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES (RequireActiveUser middleware) | DIFF -- Go checks user is_active |
| Subscription access check | YES (check_entity_access) | NO | DIFF -- Python checks subscription access |
| Parse budget ID | From path (FastAPI typed, `budget_id`) | From path (handler: `strconv.Atoi(c.Param("id"))`) | OK |
| Ownership check | In service (query by user_id AND budget_id) | In service (GetByID then compare UserID) | OK (same result) |
| Archive action | Sets `is_archived = True` | Sets `is_archived = true` | OK |
| Un-archive support | NO (always sets to True) | NO (always sets to true) | OK |
| Not found check | Service raises NotFoundError | Service returns `ErrNotFound` | OK |

## Notes

- Both implementations only support archiving (setting `is_archived = true`). Neither supports un-archiving via this endpoint.
- Python checks subscription entity access before archiving; Go does not.
- Go adds user is_active check; Python relies on `enforce_free_plan_compliance` dependency.
- The path parameter name differs (`budget_id` in Python, `id` in Go) but this is cosmetic and does not affect API behavior since both are positional path parameters.
- The ownership check is now in the Go service layer (was previously in the handler before refactoring).

## Tests

### Python Tests (0 total)

No dedicated test files found for budget endpoints.

### Go Integration Tests (4 total for archive)

| Test | Verifies |
|------|----------|
| `TestArchiveBudgetSuccess` | 200, successful archive |
| `TestArchiveBudgetNotFound` | 404 for nonexistent budget |
| `TestArchiveBudgetOtherUser` | 403 for another user's budget |
| `TestArchiveBudgetInvalidID` | 400 for non-numeric ID |

### Go Unit Tests (5 total for archive)

| Test | Verifies |
|------|----------|
| `TestArchiveBudgetInvalidID` | 400 for non-numeric path param |
| `TestArchiveBudgetNotFound` | 404 when budget not found in DB |
| `TestArchiveBudgetAccessDenied` | 403 for different user |
| `TestArchiveBudgetDBError` | 500 on DB archive failure |
| `TestArchiveBudgetSuccess` | 200 success path |
