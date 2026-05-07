# Endpoint #21: PUT `/transactions/templates`

**Status**: OK
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /transactions/templates` | `PUT /transactions/templates` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:114` | `internal/handlers/transactions/handler.go:298` |

## Request

Both: PUT with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int (required) | int (validate:"required") | OK |
| `label` | str (required) | string (validate:"required") | OK |
| `categoryId` | int (required) | *int (optional) | DIFFERENT (Go allows nil) |

## Response

**Success**: 200 OK. Template object with category.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `categoryId` | int | *int | OK |
| `label` | str | string | OK |
| `category` | object | object (omitempty) | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Template not found | 404 | "Template not found" | 404 | "Template not found" | EXACT |
| Access denied | — | — | 403 | "Access denied" | Go BETTER |
| Invalid user | — | — | 400 | "Invalid user" | Go extra |
| User not activated | — | — | 401 | "User not activated" | Go extra |
| Internal error | 500 | "Unable to update template" | 500 | "Failed to update template" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | YES | Go BETTER |
| Validate ownership | YES (filter by user_id) | YES (fetch + check UserID) | OK |
| Update label | YES | YES | OK |
| Update categoryId | YES | YES | OK |
| Validate category exists | NO (FK constraint) | NO (FK constraint) | OK |

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_update_template_internal_error` | 500 on service error |
| `test_update_template_http_exception_passthrough` | 404 passthrough |
| `test_update_template_success` | 200, label updated |

### Go Integration Tests (4 total)

| Test | Verifies |
|------|----------|
| `TestUpdateTemplate` | 200, successful update |
| `TestUpdateTemplateNotFound` | 404 |
| `TestUpdateTemplateUnauthorized` | 401 |
| `TestUpdateTemplateInactiveUser` | 401, inactive user |

### Go Unit Tests (7 total)

| Test | Verifies |
|------|----------|
| `TestUpdateTemplateUserNotActivated` | 401 |
| `TestUpdateTemplateInvalidUser` | 400 |
| `TestUpdateTemplateNotFound` | 404 |
| `TestUpdateTemplateAccessDenied` | 403 |
| `TestUpdateTemplateDBError` | 500 |
| `TestUpdateTemplateBindError` | 422 |
| `TestUpdateTemplateValidateError` | 422 |
