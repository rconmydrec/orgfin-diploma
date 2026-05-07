# Endpoint #2: POST `/auth/login/`

**Status**: PORTED OK (+ is_deleted bug fix over Python)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|----|
| Path | `POST /auth/login/` | `POST /auth/login/` |
| Auth | None (public) | None (public) |
| File | `app/routes/auth.py:40` | `internal/handlers/auth/handler.go:88` |
| Route reg | `app/main.py:65` | `internal/server/server.go:151` |

## Request Body

| Field | Python Type | Go Type | Required | Validation |
|-------|------------|---------|----------|------------|
| email | `EmailStr` | `string` | Yes | required, valid email |
| password | `str` | `string` | Yes | required, min=3, max=50 |

Both projects accept the same fields with the same validation rules. No differences.

## Response

**Success**: 200 OK (both)

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "bearer"
}
```

Both return a JWT token with `accessToken` (camelCase) and `tokenType: "bearer"`.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Invalid JSON / validation | 422 | Pydantic auto | 422 | "Invalid request data" / "Validation failed" | OK |
| User not found | 401 | "Invalid credentials or user not found" | 401 | "Invalid credentials or user not found" | OK |
| Wrong password | 401 | "Invalid credentials. See logs for details" | 401 | "Invalid credentials or user not found" | DIFFERENT |
| User not activated | 401 | "User not activated" | 401 | "User not activated" | OK |
| Internal error | 500 | "Internal server error. See logs for details" | 500 | "Invalid credentials. See logs for details" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|----|-------|
| Input validation | Pydantic (email, password length 3-50) | go-playground/validator (email, password 3-50) | OK |
| Find user by email | `db.query(User).filter(User.email == ...)` | `userRepo.GetByEmail(email)` | OK |
| Password verification | `bcrypt.checkpw()` | `bcrypt.CompareHashAndPassword()` | OK |
| Check `is_active` | Yes — returns 401 "User not activated" | Yes — returns 401 "User not activated" | OK |
| Check `is_deleted` | **NO** (bug — deleted user can login) | **YES** (filtered in SQL: `is_deleted = false`) | Go BETTER |
| JWT signing algorithm | HS256 | HS256 | OK |
| JWT expiration | Configurable, default 30 min | Configurable (hours from config) | DIFFERENT |
| JWT claims | `id`, `first_name`, `last_name`, `email`, `exp`, `exp_human` | `user_id`, `email`, `exp`, `iat` | DIFFERENT |
| Subscription check | No | No | OK |

## JWT Token Claims Comparison

| Claim | Python | Go | Match |
|-------|--------|----|-------|
| User ID | `id` (int) | `user_id` (int) | DIFFERENT key name |
| Email | `email` | `email` | OK |
| First name | `first_name` | **not included** | MISSING in Go |
| Last name | `last_name` | **not included** | MISSING in Go |
| Expiration | `exp` (standard) | `exp` (standard) | OK |
| Human-readable expiry | `exp_human` (string) | **not included** | MISSING in Go |
| Issued at | **not included** | `iat` (standard) | Go only |

## Issues Found

### INFO - JWT claims differ (no impact)

- **Python** includes `id`, `first_name`, `last_name`, `email`, `exp`, `exp_human` in token
- **Go** includes `user_id`, `email`, `exp`, `iat`
- Key name difference: Python uses `id`, Go uses `user_id`
- Missing in Go: `first_name`, `last_name`, `exp_human`
- **Impact**: NONE. Frontend only decodes JWT to read `exp` for session timer. All user identity data is fetched from `GET /auth/profile/`. Token is otherwise used as opaque bearer credential.

### LOW - Wrong password error message differs

- **Python**: 401 "Invalid credentials. See logs for details"
- **Go**: 401 "Invalid credentials or user not found"
- **Impact**: Cosmetic. Both are generic enough to prevent user enumeration. Go is slightly better (same message for both wrong email and wrong password).

### LOW - 500 error message differs

- **Python**: "Internal server error. See logs for details"
- **Go**: "Invalid credentials. See logs for details"
- **Impact**: Go's 500 message is misleading — it says "Invalid credentials" but the status is 500 (internal error). Should probably be "Internal server error" or similar.

### LOW - JWT expiration unit differs

- **Python**: Configurable in minutes (default 30 min)
- **Go**: Configurable in hours
- **Impact**: Need to verify that actual configured values result in same session duration.

### INFO - `is_deleted` handling differs (Go is BETTER)

- **Python**: `get_jwt_token()` queries `WHERE email = ?` without filtering `is_deleted`. A soft-deleted user with `is_active=True` **can log in** — this is a bug in Python.
- **Go**: `GetByEmail()` queries `WHERE email = $1 AND is_deleted = false`. A soft-deleted user gets 401 "Invalid credentials or user not found" — correct behavior.
- All Go repository methods (`GetByID`, `GetByEmail`, `Update`) consistently filter by `is_deleted = false`.
- Python also has the same gap in OAuth login (`login_or_register()`) and duplicate email check during registration.
- **Not a migration issue** — Go fixed an original Python bug.

## Tests

### Python Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `test_login_success` | `test_auth_endpoints.py:171` | 200 + accessToken + tokenType |
| `test_login_wrong_password` | `:186` | 401 on wrong password |
| `test_login_nonexistent_user` | `:198` | 401 on unknown email |
| `test_login_invalid_email_format` | `:210` | 422 on bad email |
| `test_login_missing_password` | `:222` | 422 without password |
| `test_login_inactive_user` | `:233` | 401 "User not activated" |
| `test_login_internal_error` | `:152` | 500 on mocked exception |

### Go Integration Tests (9 total)

| Test | File | Verifies |
|------|------|----------|
| `TestLoginSuccess` | `handler_test.go:250` | 200 + token response |
| `TestLoginWrongPassword` | `:290` | 401 on wrong password |
| `TestLoginNonexistentUser` | `:318` | 401 on unknown email |
| `TestLoginInvalidEmailFormat` | `:338` | 422 on bad email |
| `TestLoginMissingPassword` | `:358` | 422 without password |
| `TestLoginInactiveUser` | `:377` | 401 "User not activated" |
| `TestLoginEmptyEmail` | `:855` | 422 on empty email |
| `TestLoginEmptyPassword` | `:875` | 422 on empty password |
| `TestLoginInvalidJSON` | `:750` | 422 on malformed JSON |

### Go Unit Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestLoginDBError` | `handler_unit_test.go:190` | DB error → 500 |
| `TestLoginBindError` | `:210` | Bind error → 422 |
| `TestLoginValidationError` | `:225` | Validation error → 422 |
