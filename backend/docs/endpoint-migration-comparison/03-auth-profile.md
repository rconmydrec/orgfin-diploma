# Endpoint #3: GET `/auth/profile/`

**Status**: PORTED OK (minor differences, all improvements)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|----|
| Path | `GET /auth/profile/` | `GET /auth/profile/` |
| Auth | `check_token` dependency (JWT) | `RequireAuth` middleware (JWT) |
| Auth header | `auth-token` (custom) | `auth-token` (custom) |
| File | `app/routes/auth.py:60` | `internal/handlers/auth/handler.go:118` |
| Route reg | `app/main.py:65` | `internal/server/server.go:156` |

## Request

Both: GET with no query params, no body. User identified via JWT in `auth-token` header.

## Response

**Success**: 200 OK (both)

### Python response:
```json
{
  "id": 1,
  "first_name": "Test",
  "last_name": "User",
  "email": "test@example.com",
  "exp": 1740150600,
  "exp_human": "2026-02-21 14:30:00",
  "settings": {"language": "en", "projectionEndDate": null, "projectionPeriod": null},
  "baseCurrency": "USD"
}
```

### Go response:
```json
{
  "id": 1,
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "baseCurrency": "USD",
  "settings": {"language": "en", "projectionEndDate": null, "projectionPeriod": null}
}
```

### Field comparison:

| Field | Python | Go | Match |
|-------|--------|----|-------|
| `id` | int (from JWT payload) | int (from DB) | OK |
| `email` | string (from JWT) | string (from DB) | OK |
| `first_name` | string (from JWT) | *string (from DB, omitempty) | OK |
| `last_name` | string (from JWT) | *string (from DB, omitempty) | OK |
| `baseCurrency` | string (camelCase) | string (camelCase) | OK |
| `settings` | object | object (omitempty) | OK |
| `exp` | int (Unix timestamp) | **not included** | DIFFERENT |
| `exp_human` | string | **not included** | DIFFERENT |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth-token | 422 | Pydantic validation error | 422 | "Missing authorization header" | OK |
| Invalid token | 401 | "Invalid token" | 401 | "Invalid token" | OK |
| Expired token | 401 | "Token has expired" | 401 | "Token has expired" | OK |
| User not found | 500 | "Internal server error..." | 401 | "User not found" | Go BETTER |
| DB error | 500 | "Internal server error..." | 500 | "Failed to get profile" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|----|-------|
| User identification | JWT payload dict (decoded in `check_token`) | User ID from JWT claims (middleware sets `user_id` in context) | OK |
| User data source | JWT payload + DB query for base_currency | DB query for full user (with currency JOIN) | Go BETTER (fresh data) |
| Settings fetch | `get_user_settings()` — crashes with 500 if no settings (`.one()`) | `settingsRepo.GetByUserID()` — returns nil if no settings (graceful) | Go BETTER |
| Base currency | `fullUser.base_currency.code` (lazy load) | Included via JOIN in user query | OK |
| Check `is_active` | NO | YES (returns 401 "User not activated") | Go BETTER (fixed) |
| Check `is_deleted` | NO (no filter in query) | YES (`WHERE is_deleted = false`) | Go BETTER |
| Subscription info | Not included | Not included | OK |
| Token refresh | No | Yes — `RequireAuth` middleware sets `new_access_token` response header on all 2xx authenticated responses (sliding session) | Go EXTRA |

## Issues Found

### INFO - `exp` and `exp_human` not in Go response (no impact)

- **Python** returns `exp` and `exp_human` because the response IS the JWT payload dict with extra fields added
- **Go** returns a proper DTO built from DB data, does not include token expiration fields
- **Impact**: NONE. Frontend already has the token and decodes `exp` from it directly (confirmed in endpoint #2 analysis). These fields in the profile response are redundant.

### INFO - Go returns fresh data from DB, Python returns stale JWT data

- **Python**: `id`, `first_name`, `last_name`, `email` come from the JWT payload (set at login time). If user updates their name after login, profile still shows old name until re-login.
- **Go**: All user data is fetched fresh from DB on each profile request.
- **Impact**: Go is more correct. Not a regression.

### INFO - Settings handling when missing

- **Python**: Crashes with 500 if `UserSettings` record doesn't exist (uses `.one()` which throws `NoResultFound`)
- **Go**: Returns response without `settings` field (omitempty, graceful nil handling)
- **Impact**: Go is more robust. Not a regression.

### INFO - Token refresh on all authenticated requests (Go only)

- **Go** `RequireAuth` middleware generates a new JWT token on every successful (2xx) authenticated request and sets it in `new_access_token` response header (sliding session). This applies to all authenticated endpoints, not just profile.
- **Python** does not refresh tokens
- **Impact**: Go provides better UX (session extends on activity). Not present in Python — this is a Go enhancement.

## Tests

### Python Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `test_get_profile_success` | `test_auth_endpoints.py:278` | 200 + has `email`, `settings`, `baseCurrency` |
| `test_get_profile_unauthorized` | `:288` | 422 without auth-token header |
| `test_get_profile_invalid_token` | `:294` | 401 on bad token |
| `test_get_profile_expired_token` | `:303` | 401 on expired token |
| `test_get_profile_internal_error` | `:264` | 500 on mocked exception |

### Go Integration Tests (4 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetProfileSuccess` | `handler_test.go:411` | 200 + has `email`, `baseCurrency` |
| `TestGetProfileUnauthorized` | `:447` | 401/422 without token |
| `TestGetProfileInvalidToken` | `:463` | 401 on bad token |
| `TestGetProfileExpiredToken` | `:478` | 401 on expired token |

### Go Unit Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestGetProfileDBError` | `handler_unit_test.go:242` | DB error → 500 |
| `TestGetProfileUserNotFound` | `:262` | User not found → 401 |
| `TestGetProfileSuccessWithoutSettings` | `:274` | Profile returns 200 when user has no settings record |

### Go Middleware Tests (sliding session — 5 total, in `middleware/auth_test.go`)

| Test | Verifies |
|------|----------|
| `TestRequireAuth_ValidTokenSetsNewAccessTokenHeader` | new_access_token header set on 2xx |
| `TestRequireAuth_NoRefreshOnErrorResponse` | No header on 4xx |
| `TestRequireAuth_NoRefreshOn500Response` | No header on 5xx |
| `TestRequireAuth_NoRefreshWhenHandlerReturnsError` | No header when handler returns error |
| `TestRequireAuth_RefreshOnCreatedResponse` | Header set on 201 |
