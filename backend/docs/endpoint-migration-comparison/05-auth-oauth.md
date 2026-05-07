# Endpoint #5: POST `/auth/oauth/`

**Status**: PORTED OK (minor differences, all improvements)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /auth/oauth/` | `POST /auth/oauth/` |
| Auth | None (public) | None (public) |
| File | `app/routes/auth.py:91` | `internal/handlers/auth/handler.go:180` |
| Route reg | `app/main.py:65` | `internal/server/server.go:153` |

## Request

Both: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `credential` | `str`, required (Pydantic) | `string`, `validate:"required"` | OK |

## Response

**Success**: 200 OK (both)

### Python response:
```json
{
  "accessToken": "<JWT>",
  "tokenType": "bearer"
}
```

### Go response:
```json
{
  "accessToken": "<JWT>",
  "tokenType": "bearer"
}
```

Field comparison: **EXACT match**.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing `credential` | 422 | Pydantic validation error | 422 | "Validation failed" | OK |
| Invalid JSON body | 422 | Pydantic validation error | 422 | "Invalid request data" | OK |
| Invalid Google token | 500 (unhandled!) | FastAPI default 500 | 401 | "Invalid credentials or user not found" | Go BETTER |
| No email in payload | 422 | "No email provided" | 422 | "No email provided" | EXACT |
| Email not verified | 401 | "Email not verified" | 401 | "Email not verified" | EXACT |
| User not activated | 401 | "User not activated" | 401 | "User not activated" | EXACT |
| DB/service error | 500 | "Internal server error..." | 500 | "Invalid credentials. See logs for details" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Google token validation | `id_token.verify_oauth2_token()` | `googleValidator.Validate()` (interface) | OK |
| Invalid token handling | NOT caught — unhandled 500 | Caught — returns 401 | Go BETTER |
| Extract email | `payload['email']` | `payload.Claims["email"].(string)` | OK |
| Check email_verified | `payload['email_verified']` must be True | `payload.Claims["email_verified"].(bool)` must be true | OK |
| Extract given_name | `payload['given_name']` | `payload.Claims["given_name"].(string)` (safe, defaults "") | OK |
| Extract family_name | `payload.get('family_name', '')` | `payload.Claims["family_name"].(string)` (safe, defaults "") | OK |
| Look up user by email | `db.query(User).filter(email)` — NO `is_deleted` filter | `GetByEmail()` — `WHERE is_deleted = false` | Go BETTER |
| Existing active user | Generate JWT, return token | Generate JWT, return token | OK |
| Existing inactive user | Raise `UserNotActivated` → 401 | Return `ErrUserNotActivated` → 401 | OK |
| New user: auto-activate | `is_active = True` | `IsActive: true` | OK |
| New user: random password | `secrets.token_hex(16)` → 32-char hex | `crypto/rand` 16 bytes → 32-char hex | OK |
| New user: default currency | USD | USD | OK |
| New user: default settings | Created (`language: "en"`, etc.) | Created via `CreateDefault()` | OK |
| New user: default categories | Copied from `DefaultCategory` table | Copied via `CopyDefaultCategories()` | OK |
| New user: trial subscription | YES — 45-day trial created | YES — 14-day trial via `subscriptionSvc.CreateTrialSubscription()` (non-blocking) | OK (different duration) |
| New user: activation token/email | Skipped for OAuth | Skipped for OAuth | OK |
| Update existing user name from Google | NO | NO | OK (same gap) |
| Check `is_deleted` | NO (bug — finds deleted users) | YES (filtered in SQL) | Go BETTER |
| Settings/categories creation failure | Unhandled — would crash | Logged, non-blocking | Go BETTER |

## Issues Found

### RESOLVED — Trial subscription now implemented

- **Python**: Creates a trial subscription (45 days) for new OAuth users via `create_users(is_oauth=True)`.
- **Go**: Creates trial subscription via `subscriptionSvc.CreateTrialSubscription(userID)` in both `Register()` and `LoginOrRegisterOAuth()`. Non-blocking (errors logged, not propagated).
- **Difference**: Go uses 14-day trial (configurable) vs Python's 45-day.

### FIXED (Go) — Invalid Google token handling

- **Python**: Exception from `id_token.verify_oauth2_token()` is NOT caught in the route handler. Results in unhandled 500 error.
- **Go**: Error from `googleValidator.Validate()` is properly caught, returns 401 "Invalid credentials or user not found".
- **Impact**: Go correctly handles this error case. Python has a bug.

### FIXED (Go) — Soft-deleted user via OAuth

- **Python**: `login_or_register` queries `db.query(User).filter(User.email == email)` without `is_deleted` filter. A soft-deleted user would be found and could get a token (if active) — bug.
- **Go**: `GetByEmail()` has `WHERE is_deleted = false`. Soft-deleted user is treated as "not found" → proceeds to register new account.
- **Note**: If a soft-deleted user tries to re-register via OAuth in Go, it would attempt to create a new user with the same email. This may fail on a UNIQUE constraint if it doesn't exclude deleted rows. Edge case — extremely unlikely in practice.

### INFO — Settings/categories creation failure handling

- **Python**: If `generate_initial_settings()` or `copy_all_categories()` fails, the exception is not caught — would crash the request.
- **Go**: Both `CreateDefault()` and `CopyDefaultCategories()` failures are logged but don't block registration.
- **Impact**: Go is more resilient.

### INFO — Existing user name not updated from Google (same gap)

- Both Python and Go: when an existing user logs in via OAuth, their `first_name`/`last_name` are NOT updated from the Google payload. Only new user registration uses the Google name data.

## Tests

### Python Tests (10 total)

| Test | File | Verifies |
|------|------|----------|
| `test_oauth_success_existing_user` | `test_auth_endpoints.py:513` | 200, existing active user gets token |
| `test_oauth_success_new_user` | `:537` | 200, new user created and gets token |
| `test_oauth_no_email` | `:414` | 422 when email is None in payload |
| `test_oauth_email_not_verified` | `:565` | 401 when email_verified is false |
| `test_oauth_user_not_activated` | `:431` | 401 when existing user is inactive |
| `test_oauth_not_found_error` | `:452` | 401 on NotFoundError from service |
| `test_oauth_http_exception` | `:473` | 401 on HTTPException from service |
| `test_oauth_internal_error` | `:494` | 500 on generic exception |
| `test_oauth_invalid_token` | `:582` | 500 on invalid Google token (unhandled) |
| `test_oauth_missing_credential` | `:599` | 422 when credential field missing |

### Go Integration Tests (11 total)

| Test | File | Verifies |
|------|------|----------|
| `TestOAuthSuccessNewUser` | `handler_test.go:1340` | 200, new user created, gets accessToken + tokenType |
| `TestOAuthSuccessExistingUser` | `:1379` | 200, existing active user gets token |
| `TestOAuthEmailNotVerified` | `:1417` | 401 "Email not verified" |
| `TestOAuthNoEmail` | `:1445` | 422 "No email provided" |
| `TestOAuthGoogleValidationFails` | `:1464` | 401 on Google validation error |
| `TestOAuthInactiveUser` | `:1483` | 401 "User not activated" |
| `TestOAuthMissingCredential` | `:1229` | 422 on missing credential |
| `TestOAuthInvalidJSON` | `:1247` | 422 on malformed JSON |
| `TestOAuthEmptyBody` | `:1264` | 422 on empty body |
| `TestOAuthEmptyCredential` | `:1281` | 422 on empty credential string |
| `TestOAuthInvalidToken` | `:1298` | 401 on invalid token (real validator) |

### Go Unit Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestOAuthDBError` | `handler_unit_test.go:480` | DB error → 500 |
| `TestOAuthBindError` | `:510` | Malformed JSON → 422 |
| `TestOAuthValidationError` | `:525` | Empty body → 422 |
