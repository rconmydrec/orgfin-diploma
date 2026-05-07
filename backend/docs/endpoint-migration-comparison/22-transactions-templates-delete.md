# Endpoint #22: DELETE `/transactions/templates`

**Status**: OK
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `DELETE /transactions/templates` | `DELETE /transactions/templates` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:140` | `internal/handlers/transactions/handler.go:331` |

## Request

Both: DELETE with `ids` query parameter (comma-separated integers).

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `ids` | Query string, validated by Pydantic (positive ints) | Query string, parsed manually (skips invalid) | DIFFERENT (Python strict, Go lenient) |

## Response

**Success**: 200 OK. Array of REMAINING templates (not deleted ones).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Missing ids | 422 | Pydantic validation | 422 | "ids parameter required" | OK |
| Invalid ids | 400 | "Invalid request parameters" | 422 | "Invalid IDs" | DIFFERENT |
| User not activated | — | — | 401 | "User not activated" | Go extra |
| Internal error | 500 | "Unable to delete templates" | 500 | "Failed to delete templates" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | YES | Go BETTER |
| Parse IDs | Pydantic validation (strict) | Manual parsing (lenient, skips invalid) | DIFFERENT |
| Check ownership per template | YES (WHERE user_id) | YES (fetch + check UserID) | OK |
| Silent skip non-existent | YES | YES | OK |
| Silent skip other user's | YES | YES | OK |
| Return remaining templates | YES | YES | OK |
| Hard delete | YES | YES | OK |

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_delete_templates_internal_error` | 500 |
| `test_delete_templates_success` | 200, deletion works |
| `test_delete_templates_invalid_ids` | 400 |

### Go Integration Tests (6 total)

| Test | Verifies |
|------|----------|
| `TestDeleteTemplates` | 200, batch delete |
| `TestDeleteTemplatesSingle` | 200, single delete |
| `TestDeleteTemplatesMissingIds` | 422 |
| `TestDeleteTemplatesInvalidIds` | 422 |
| `TestDeleteTemplatesUnauthorized` | 401 |
| `TestDeleteTemplatesInactiveUser` | 401 |

### Go Unit Tests (5 total)

| Test | Verifies |
|------|----------|
| `TestDeleteTemplatesUserNotActivated` | 401 |
| `TestDeleteTemplatesInvalidUserUnit` | 400 |
| `TestDeleteTemplatesDBError` | 500 |
| `TestDeleteTemplatesInvalidIDs` | 422 |
| `TestDeleteTemplatesMissingIDs` | 422 |
