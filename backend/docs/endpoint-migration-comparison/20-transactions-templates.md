# Endpoint #20: GET `/transactions/templates`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /transactions/templates` | `GET /transactions/templates` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:100` | `internal/handlers/transactions/handler.go:251` |

## Request

Both: GET with no parameters.

## Response

**Success**: 200 OK. JSON array of template objects.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `categoryId` | int | int | OK |
| `label` | str | string | OK |
| `category` | object | object | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Internal error | 500 | "Unable to get templates" | 500 | "Failed to get templates" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Filter by user_id | YES | YES | OK |
| Order by label | YES | YES | OK |
| Include category relation | YES (eager load) | YES (JOIN) | OK |

## Issues Found

### BUG — Missing is_active check on user

- **Go**: No `is_active` check. Inactive user can get templates.
- **Impact**: Must be fixed for consistency.

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_templates_internal_error` | 500 |
| `test_get_templates_success` | 200, list |
| `test_get_templates_empty_list` | 200, empty |

### Go Integration Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetTemplates` | `handler_test.go:1569` | 200, template with category |
| `TestGetTemplatesEmpty` | `:1609` | Empty array |
| `TestGetTemplatesUnauthorized` | `:1635` | 401 |

### Go Unit Tests (2 total)

| Test | Verifies |
|------|----------|
| `TestGetTemplatesDBError` | 500 |
| `TestGetTemplatesWithCategory` | Category in response |
