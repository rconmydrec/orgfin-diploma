# Endpoint #30: GET `/settings/languages`

**Status**: OK
**Date**: 2026-02-21
**Last Updated**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /settings/languages` | `GET /settings/languages` |
| Auth | **NONE** (public) | **NONE** (public) |
| File | `app/routes/user_settings.py:31` | `internal/handlers/settings/handler.go:72` |

## Request

Both: GET with no parameters. No auth required.

## Response

**Success**: 200 OK. Array of languages.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `code` | str | string | OK |
| `name` | str | string | OK |
| `isDeleted` | bool | bool | OK (but always false in Go) |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Internal error | 500 | "Unable to get languages" | 500 | "Failed to get languages" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | NO | NO | OK |
| Filter is_deleted | NO | YES (WHERE is_deleted = false) | Go BETTER |
| Order | None specified | ORDER BY name | Go BETTER |

## Notes

- Public endpoint — no is_active check needed.
- Go filters deleted languages (Python doesn't) — Go is stricter.
- `isDeleted` field in Go response is redundant (always false since filtered).
- Languages are fetched directly via `languageRepo` in the handler (not part of the settings service).

## Tests

### Python Tests (3 total)

| Test | Verifies |
|------|----------|
| `test_get_languages_success_public` | 200, no auth needed |
| `test_get_languages_includes_common_languages` | "en" present |
| `test_get_languages_error` | 500 |

### Go Integration Tests (3 total)

| Test | Verifies |
|------|----------|
| `TestGetLanguagesSuccess` | 200, structure |
| `TestGetLanguagesIncludesCommonLanguages` | "en" present |
| `TestGetLanguagesNonDeleted` | All isDeleted=false |

### Go Unit Tests (1 total)

| Test | Verifies |
|------|----------|
| `TestGetLanguagesDBError` | 500 |
