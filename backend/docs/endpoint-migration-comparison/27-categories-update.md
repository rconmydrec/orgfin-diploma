# Endpoint #27: PUT `/categories/:id/`

**Status**: NEEDS FIX (missing is_active check, missing validation)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /categories/{category_id}/` | `PUT /categories/:id/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/categories.py:34` | `internal/handlers/categories/handler.go:191` |

## Request

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| Path param | `category_id` (IGNORED, uses body `id`) | `:id` (USED) | DIFFERENT (Go is correct) |
| `name` | str (required) | string (no validation on update!) | **Go BUG** |
| `parentId` | int/None | *int | OK |
| `isIncome` | bool | bool | OK |

## Response

**Success**: 200 OK. Updated category object.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Not found | 400 | "Error updating category" | 404 | "Category not found" | Go BETTER |
| Access denied | 400 | "Error updating category" | 403 | "Access denied" | Go BETTER |
| Invalid ID | — | — | 400 | "Invalid category ID" | Go extra |
| Internal error | 400 | "Error updating category" | 500 | "Failed to update category" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter is_deleted | YES (implicit) | YES (WHERE is_deleted = false) | OK |
| Check ownership | YES (.one() with user_id filter) | YES (fetch + check UserID) | OK |
| Validate name required | YES (Pydantic) | NO (c.Validate not called) | **Go BUG** |
| Validate parent | NO | NO | OK |

## Issues Found

### BUG — Missing is_active check

- **Go**: No `is_active` check. Handler delegates to the categories service, which does not perform this check.

### BUG — Missing validation on UpdateCategory

- **Go**: Handler binds request but does NOT call `c.Validate(&req)`. Empty name is accepted.
- **Python**: Pydantic enforces `name` as required.
- **Impact**: Must add validation.

### BUG — Error conflation in GetByID

- **Go**: Any error from `categoryRepo.GetByID()` (including DB connection errors) returns 404 "Category not found".
- **Impact**: Should distinguish `sql.ErrNoRows` from other errors.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_update_category_success` | 200, name updated |
| `test_update_category_not_found` | 400 |
| `test_update_category_unauthorized` | 422 |

### Go Integration Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestUpdateCategorySuccess` | 200 |
| `TestUpdateCategoryNotFound` | 404 |
| `TestUpdateCategoryOtherUser` | 403 |
| `TestUpdateCategoryUnauthorized` | 401 |
| `TestUpdateCategoryInvalidID` | 400 |

### Go Unit Tests (6 total)

| Test | Verifies |
|------|----------|
| `TestUpdateCategoryInvalidID` | 400 |
| `TestUpdateCategoryBindError` | 422 |
| `TestUpdateCategoryNotFound` | 404 |
| `TestUpdateCategoryAccessDenied` | 403 |
| `TestUpdateCategoryDBError` | 500 |
| `TestUpdateCategorySuccess` | 200 |
