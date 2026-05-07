# Endpoint #28: DELETE `/categories/:id/`

**Status**: NEEDS FIX (missing is_active check, error conflation)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /categories/{category_id}/` | `DELETE /categories/:id/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/categories.py:69` | `internal/handlers/categories/handler.go:225` |

## Request

Both: DELETE with category ID in path.

## Response

**Success**: 200 OK. Deleted category object.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Not found | 400 | "Error deleting category" | 404 | "Category not found" | Go BETTER |
| Other user | 400 | "Error deleting category" | 403 | "Access denied" | Go BETTER |
| Invalid ID | — | — | 400 | "Invalid category ID" | Go extra |
| Internal error | 400 | "Error deleting category" | 500 | "Failed to delete category" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter is_deleted | YES (implicit) | YES (WHERE is_deleted = false) | OK |
| Check ownership | YES (.one() with user_id filter) | YES (fetch + check UserID) | OK |
| Soft delete | YES (is_deleted = True) | YES (is_deleted = true) | OK |
| Cascade check (children) | NO | NO | OK (both leave orphans) |

## Issues Found

### BUG — Missing is_active check

- **Go**: No `is_active` check. Handler delegates to the categories service, which does not perform this check.

### BUG — Error conflation in GetByID

- **Go**: Any error from `categoryRepo.GetByID()` returns 404. DB connection errors misreported as "not found".

## Tests

### Python Tests (4 total)

| Test | Verifies |
|------|----------|
| `test_delete_category_success` | 200 |
| `test_delete_category_not_found` | 400 |
| `test_delete_category_other_user` | 400 |
| `test_delete_category_unauthorized` | 422 |

### Go Integration Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestDeleteCategorySuccess` | 200 |
| `TestDeleteCategoryNotFound` | 404 |
| `TestDeleteCategoryOtherUser` | 403 |
| `TestDeleteCategoryUnauthorized` | 401 |

### Go Unit Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestDeleteCategoryInvalidID` | 400 |
| `TestDeleteCategoryNotFound` | 404 |
| `TestDeleteCategoryAccessDenied` | 403 |
| `TestDeleteCategoryDBError` | 500 |
| `TestDeleteCategorySuccess` | 200 |
