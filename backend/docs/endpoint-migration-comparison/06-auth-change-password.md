# Endpoint #6: POST `/auth/change-password/`

**Status**: PORTED OK (minor differences, Go improvements)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /auth/change-password/` | `POST /auth/change-password/` |
| Auth | `check_token` dependency (JWT) | `RequireAuth` middleware (JWT) |
| Auth header | `auth-token` (custom) | `auth-token` (custom) |
| File | `app/routes/auth.py:124` | `internal/handlers/auth/handler.go:226` |
| Route reg | `app/main.py:65` | `internal/server/server.go:159` |

## Request

Both: POST with JSON body. Auth via `auth-token` header.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `current_password` | str, required, min=3, max=50 | string, required, min=3, max=50 | OK |
| `new_password` | str, required, min=3, max=50 | string, required, min=3, max=50 | OK |
| camelCase aliases | YES (via alias_generator) | NO (snake_case only) | DIFFERENT (minor) |

## Response

**Success**: 200 OK (both)

```json
{
  "success": true,
  "message": "Password changed successfully"
}
```

Field comparison: **EXACT match**.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth-token | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Invalid token | 401 | "Invalid token" | 401 | "Invalid token" | EXACT |
| Expired token | 401 | "Token has expired" | 401 | "Token has expired" | EXACT |
| Invalid body | 422 | Pydantic error | 422 | "Invalid request data" | OK |
| Missing fields | 422 | Pydantic error | 422 | "Validation failed" | OK |
| User not found | 404 | "User not found" | 401 | "User not found" | DIFFERENT |
| Wrong current password | 401 | "Current password is incorrect" | 401 | "Current password is incorrect" | EXACT |
| Internal error | 500 | "Internal server error..." | 500 | "Failed to change password" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth mechanism | JWT in auth-token header | JWT in auth-token header (middleware) | OK |
| Get user by ID | `db.query(User).filter(id)` — NO `is_deleted` filter | `userRepo.GetByID()` — `WHERE is_deleted = false` | Go BETTER |
| Check `is_active` | NO (bug) | YES (returns ErrUserNotActivated) | Go BETTER (fixed) |
| Verify current password | `bcrypt.checkpw` | `bcrypt.CompareHashAndPassword` | OK |
| Hash new password | `bcrypt.hashpw` with gensalt | `bcrypt.GenerateFromPassword` (DefaultCost=10) | OK |
| Update password | `user.password_hash = ...; db.commit()` | `userRepo.UpdatePassword()` SQL UPDATE | OK |
| Same password allowed | YES (no check) | YES (no check) | OK |
| Password strength validation | Length only (3-50) | Length only (3-50) | OK |

## Issues Found

### FIXED (Go) — Soft-deleted user cannot change password

- **Python**: Queries user by `User.id == user_id` without `is_deleted` filter. A soft-deleted user with a valid JWT could change their password.
- **Go**: `GetByID()` has `WHERE is_deleted = false`. Soft-deleted user gets 401 "User not found".
- **Impact**: Go correctly blocks this edge case.

### INFO — Different status code for "user not found"

- **Python**: Returns 404 "User not found".
- **Go**: Returns 401 "User not found".
- **Impact**: Minor. Go's 401 is arguably more appropriate since this is an auth-related endpoint. No frontend impact (both are error states).

### FIXED (Go) — `is_active` now checked before password change

- **Python**: Does NOT check `is_active`. An inactive user with a valid JWT could change their password.
- **Go** (after fix): `ChangePassword` checks `user.IsActive` after `GetByID()`. Inactive user gets 401 "User not activated".
- **Impact**: Go correctly blocks inactive users from changing password.

### INFO — Missing auth-token status code difference

- **Python**: Returns 422 (Pydantic header validation).
- **Go**: Returns 401 "Missing authorization header" (middleware).
- **Impact**: Go's 401 is more semantically correct.

## Tests

### Python Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `test_change_password_success` | `test_auth_endpoints.py:630` | 200, success=true, can login with new password |
| `test_change_password_wrong_current` | `:661` | 401 on wrong current password |
| `test_change_password_weak_new_password` | `:678` | 200 with "123" (3 chars, accepted) |
| `test_change_password_unauthorized` | `:696` | 422 without auth-token |
| `test_change_password_same_as_current` | `:708` | 200 with same password |
| `test_change_password_internal_error` | `:609` | 500 on mocked exception |

### Go Integration Tests (8 total)

| Test | File | Verifies |
|------|------|----------|
| `TestChangePasswordSuccess` | `handler_test.go:687` | 200, success=true, login with new password works |
| `TestChangePasswordWrongCurrent` | `:740` | 401 on wrong current password |
| `TestChangePasswordUnauthorized` | `:770` | 401/422 without auth-token |
| `TestChangePasswordSameAsCurrent` | `:792` | 200 with same password |
| `TestChangePasswordInvalidJSON` | `:859` | 422 on malformed JSON |
| `TestChangePasswordEmptyBody` | `:920` | 422 on empty body |
| `TestChangePasswordMissingCurrentPassword` | `:1051` | 422 missing current_password |
| `TestChangePasswordMissingNewPassword` | `:1080` | 422 missing new_password |

### Go Unit Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `TestChangePasswordDBError` | `handler_unit_test.go:380` | DB error → 500 |
| `TestChangePasswordUserNotFound` | `:402` | ErrUserNotFound → 401 |
| `TestChangePasswordIncorrect` | `:424` | ErrIncorrectPassword → 401 |
| `TestChangePasswordBindError` | `:446` | Invalid JSON → 422 |
| `TestChangePasswordValidationError` | `:462` | Missing field → 422 |
