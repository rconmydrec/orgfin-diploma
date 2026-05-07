# Endpoint #24: GET `/categories/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /categories/` | `GET /categories/` |
| Auth | `check_token` | `RequireAuth` middleware |
| File | `app/routes/categories.py:23` | `internal/handlers/categories/handler.go:62` |

## Request

Both: GET with no parameters.

## Response

**Success**: 200 OK. Flat array of categories with (+)/(-) prefix and >> notation for hierarchy.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `userId` | int | int | OK |
| `name` | str (prefixed) | string (prefixed) | OK |
| `parentId` | int/None | *int | OK |
| `isIncome` | bool | bool | OK |
| `createdAt` | datetime | time | OK |
| `updatedAt` | datetime | time | OK |
| `children` | [] (always empty) | [] (always empty) | OK |

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
| Build flat list with prefixes | YES | YES | OK |
| Sort children alphabetically | YES | YES | OK |
| Order by is_income, name | YES | YES | OK |

## Architecture Note

Go handler delegates to the categories service (`internal/services/categories/service.go`), which builds the flat list with hierarchy name formatting, income/expense prefixes, and alphabetical sorting. The handler is a thin HTTP adapter.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_categories_success` | 200, list |
| `test_get_categories_with_user_categories` | User's category in response |
| `test_get_categories_unauthorized` | 422 |

### Go Integration Tests (2 total)

| Test | Verifies |
|------|----------|
| `TestGetCategoriesSuccess` | 200, correct structure |
| `TestGetCategoriesUnauthorized` | 401 |

### Go Unit Tests (3 total)

| Test | Verifies |
|------|----------|
| `TestGetCategoriesDBError` | 500 |
| `TestGetCategoriesSuccess` | 200, categories |
| `TestGetCategoriesWithChildrenSorting` | Alphabetical sorting |
