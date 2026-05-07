# Endpoint #23: DELETE `/transactions/templates/validate`

**Status**: NEEDS FIX (0 tests in Go)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /transactions/templates/validate` | `DELETE /transactions/templates/validate` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:166` | `internal/handlers/transactions/handler.go:371` |

## Request

Both: DELETE with `ids` query parameter (comma-separated integers).

## Response

**Success**: 200 OK. Parsed integer array (e.g., `[1, 2, 3]`).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Invalid ids | 400 | "Invalid template IDs format" | 400 | "Invalid template IDs format" | EXACT |
| Empty ids | 400 | "Invalid template IDs format" | 400 | "Invalid template IDs format" | EXACT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | Both missing (but this is a validation-only endpoint, no DB access) |
| Parse IDs | Pydantic validation (strict, positive ints) | Manual parsing (strict, > 0) | OK |
| DB access | NONE | NONE | OK |
| Return parsed IDs | YES | YES | OK |

## Issues Found

### BUG — Zero tests for ValidateTemplateIDs in Go

- **Go**: No integration tests, no unit tests. This endpoint has 0% coverage.
- **Impact**: Must add tests for: success case, invalid IDs, empty IDs, unauthorized.

## Tests

### Python Tests (2 total)

| Test | Verifies |
|------|----------|
| `test_validate_template_ids_success` | 200, [1,2,3,4,5] |
| `test_validate_template_ids_invalid_format` | 400 |

### Go Integration Tests (0 total)

NONE

### Go Unit Tests (0 total)

NONE
