# Endpoint #33: GET `/settings/base-currency/`

**Status**: OK
**Date**: 2026-02-21
**Last Updated**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /settings/base-currency/` | `GET /settings/base-currency/` |
| Auth | `check_token` dependency | JWT middleware (`user_id` from context) |
| File | `app/routes/user_settings.py:92` | `internal/handlers/settings/handler.go:150` |

## Request

Both: GET with no parameters. Auth token required.

| Aspect | Python | Go | Match |
|--------|--------|-----|-------|
| Auth header | `Authorization: Bearer <token>` | `auth-token: <token>` | **DIFF** (framework convention) |

## Response

**Success**: 200 OK. Currency object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int `json:"id"` | OK |
| `code` | str | string `json:"code"` | OK |
| `name` | str | string `json:"name"` | OK |

Both use a currency response schema with the same three fields.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 422 | Validation error (FastAPI) | 401 | Unauthorized (middleware) | **DIFF** |
| Internal error | 500 | "Unable to get base currency" | 500 | "Failed to get base currency" | OK (different wording) |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** |
| User deleted | N/A | N/A | 401 | "User not activated" | **DIFF** |
| Base currency not set | Exception (500) | "Unable to get base currency" | 404 | "Base currency not set" | **DIFF** |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| Check user active | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Check user deleted | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Get base currency | JOIN Currency+User query | `service.GetBaseCurrency(userID)` -> `userRepo.GetByID` | **DIFF** (implementation, same result) |
| Handle no base currency | Exception (sqlalchemy `.one()` raises) | Service returns `ErrBaseCurrencyNotSet`, handler returns 404 | **DIFF** (Go returns explicit 404) |

## Architecture

Go handler is a thin HTTP adapter. It reads `user_id` from context and delegates to `service.GetBaseCurrency(userID)`. The service fetches the user via `userRepo.GetByID` and checks if `BaseCurrency` is nil. Handler maps `ErrBaseCurrencyNotSet` to 404, other errors to 500 with "Failed to get base currency".

## Notes

- Python uses a single SQL query with JOIN (`Currency JOIN User`) and `.one()` which raises `NoResultFound` if user has no base currency, caught as generic Exception -> 500.
- Go service gets the user (with preloaded BaseCurrency), then checks if `BaseCurrency` is nil. If nil, returns `ErrBaseCurrencyNotSet` which the handler maps to 404 with "Base currency not set". This is a more user-friendly approach.
- Go adds `RequireActiveUser` middleware guard not present in Python.
- Response schema is identical between both (id, code, name).

## Tests

### Python Tests (3 total)
| Test | Verifies |
|------|----------|
| `test_get_base_currency_success` | 200 with id, code, name fields |
| `test_get_base_currency_unauthorized` | 422 without auth |
| `test_get_base_currency_error` | 500 on internal error (mocked) |

### Go Integration Tests (4 total)
| Test | Verifies |
|------|----------|
| `TestGetBaseCurrencySuccess` | 200 with id, code, name fields |
| `TestGetBaseCurrencyNewUser` | 200 for new user (default currency) |
| `TestGetBaseCurrencyUnauthorized` | 401 without auth token |
| `TestUpdateBaseCurrencyVerifyPersistence` | GET after PUT returns updated currency |

### Go Unit Tests — Handler (2 total)
| Test | Verifies |
|------|----------|
| `TestGetBaseCurrencyDBError` | 500 when service returns error |
| `TestGetBaseCurrencyNotSet` | 404 when user has no base currency |

### Go Unit Tests — Service (4 total)
| Test | Verifies |
|------|----------|
| `TestGetBaseCurrency_Success` | Returns user's base currency |
| `TestGetBaseCurrency_NotSet` | Returns `ErrBaseCurrencyNotSet` when nil |
| `TestGetBaseCurrency_UserFetchError` | Propagates user repo errors |
| `TestGetBaseCurrency_CorrectUserIDPassed` | Correct user ID forwarded |
