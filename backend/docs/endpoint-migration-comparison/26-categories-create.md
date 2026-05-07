# Endpoint #26: POST `/categories/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /categories/` | `POST /categories/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/categories.py:54` | `internal/handlers/categories/handler.go:162` |

## Request

Both: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `name` | str (required) | string (validate:"required") | OK |
| `parentId` | int/None | *int | OK |
| `isIncome` | bool (required) | bool | OK |
| `id` | int/None (used for create/update decision) | *int (ignored in create) | DIFFERENT |

## Response

**Success**: Python 201, Go 201. Category object.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Missing name | 422 | Pydantic validation | 422 | "Validation failed" | OK |
| Internal error | 400 | "Error creating category: {e}" | 500 | "Failed to create category" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Create/Update decision by `id` | YES (creates if id falsy) | NO (always creates) | DIFFERENT |
| Validate parent exists | NO | NO | OK |
| Set is_deleted = false | Implicit (model default) | YES (explicit in SQL) | OK |
| Duplicate names allowed | YES | YES | OK |

## Issues Found

### BUG — Missing is_active check

- **Go**: No `is_active` check. Handler delegates to the categories service, which does not perform this check.
- **Impact**: Must be fixed.

### INFO — Python leaks internal error

- **Python**: Returns `"Error creating category: {exception message}"` — leaks internal errors.
- **Go**: Returns generic `"Failed to create category"` — correct behavior.

## Tests

### Python Tests (6 total)

| Test | Verifies |
|------|----------|
| `test_create_expense_category_success` | 201 |
| `test_create_income_category_success` | 201, isIncome=true |
| `test_create_category_duplicate_name` | 201, allowed |
| `test_create_category_missing_name` | 422 |
| `test_create_category_unauthorized` | 422 |
| `test_create_category_internal_error` | 400 |

### Go Integration Tests (8 total)

| Test | Verifies |
|------|----------|
| `TestCreateExpenseCategorySuccess` | 201 |
| `TestCreateIncomeCategorySuccess` | 201 |
| `TestCreateCategoryMissingName` | 422 |
| `TestCreateCategoryUnauthorized` | 401 |
| `TestCreateSubcategorySuccess` | 201, with parentId |
| `TestCreateCategoryInvalidJSON` | 422 |
| `TestCreateCategoryWithLongName` | 201 |
| `TestCreateCategoryWithSpecialCharacters` | 201 |

### Go Unit Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestCreateCategoryBindError` | 422 |
| `TestCreateCategoryValidateError` | 422 |
| `TestCreateCategoryDBError` | 500 |
| `TestCreateCategorySuccess` | 201 |
