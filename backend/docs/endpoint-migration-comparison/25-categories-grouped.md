# Endpoint #25: GET `/categories/grouped/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /categories/grouped/` | `GET /categories/grouped/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/categories.py:28` | `internal/handlers/categories/handler.go:135` |

## Request

Both: GET with no parameters.

## Response

**Success**: 200 OK. Object with `income` and `expenses` arrays (tree structure).

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `income` | list[dict] | []*CategoryResponse | OK |
| `expenses` | list[dict] | []*CategoryResponse | OK |

Category names are NOT prefixed (unlike flat list endpoint).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Internal error | 500 | Unhandled | 500 | "Failed to get categories" | Go BETTER |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter is_deleted | YES | YES | OK |
| Build tree structure | YES | YES | OK |
| Separate income/expenses | YES | YES | OK |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check. Handler delegates to the categories service, which does not perform this check.
- **Impact**: Must be fixed for consistency with other endpoints.

### INFO — Null vs empty array

- **Go**: When no categories of a type exist, `income`/`expenses` may be `null` instead of `[]`.
- **Python**: Returns `[]`.
- **Impact**: Minor — frontend should handle both.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_grouped_categories_success` | 200, income/expenses structure |
| `test_get_grouped_categories_has_both_types` | Both arrays non-empty |
| `test_get_grouped_categories_unauthorized` | 422 |

### Go Integration Tests (2 total)

| Test | Verifies |
|------|----------|
| `TestGetGroupedCategoriesSuccess` | 200, income/expenses |
| `TestGetGroupedCategoriesUnauthorized` | 401 |

### Go Unit Tests (3 total)

| Test | Verifies |
|------|----------|
| `TestGetGroupedCategoriesDBError` | 500 |
| `TestGetGroupedCategoriesSuccess` | 200, structure |
| `TestGetGroupedCategoriesWithChildren` | Children included |
