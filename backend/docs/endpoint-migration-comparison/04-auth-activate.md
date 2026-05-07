# Endpoint #4: GET `/auth/activate/:token`

**Status**: PORTED OK (Go fixes Python bug: blocks deleted user activation)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /auth/activate/{token}` | `GET /auth/activate/:token` |
| Auth | None (public) | None (public) |
| File | `app/routes/auth.py:74` | `internal/handlers/auth/handler.go:159` |
| Route reg | `app/main.py:65` | `internal/server/server.go:152` |

## Request

Both: GET with `token` as path parameter. No body, no query params.

## Response

**Success**: 200 OK, body: `true` (JSON boolean) — both identical.

### Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Token not found | 404 | "Token not found" | 404 | "Token not found" | EXACT |
| Token expired | 400 | "Token expired" | 400 | "Token expired" | EXACT |
| User not found | 404 | "Token not found" (remapped from "User not found") | N/A (not checked) | N/A | OK (see notes) |
| Internal error | 500 | "Internal server error. See logs for details" | 500 | "Activation failed" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Look up token | `db.query(ActivationToken).filter(...).first()` | `tokenRepo.GetByToken(token)` SQL query | OK |
| Token not found | 404 "Token not found" | 404 via `sql.ErrNoRows` → `ErrTokenNotFound` | OK |
| Expiration check | `expires_at < datetime.now(UTC)` | `time.Now().After(activationToken.ExpiresAt)` | OK |
| Look up user before activation | YES — queries `users` table by `user_id` | NO — directly calls `Activate(userID)` | DIFFERENT (see notes) |
| User not found check | YES — 404 if user missing | NO — `UPDATE` affects 0 rows silently | DIFFERENT (see notes) |
| Set is_active | `user.is_active = True` | `UPDATE users SET is_active = true WHERE id = $1` | OK |
| Delete token after activation | YES — `db.delete(activation_token)` | YES — `tokenRepo.Delete(activationToken.ID)` | OK |
| Delete expired tokens | NO (expired tokens stay in DB) | NO (expired tokens stay in DB) | OK (same gap) |
| Atomicity | Single `db.commit()` (atomic) | Two separate operations (activate then delete) | DIFFERENT (see notes) |
| Check `is_active` before activation | NO | YES (idempotent — skips re-activation, cleans up token) | Go BETTER |
| Check `is_deleted` before activation | NO (bug) | YES (GetByID filters `is_deleted = false`) | Go BETTER (fixed) |
| Token deletion on failure | Token NOT deleted if expired/not found | Token NOT deleted if expired/not found | OK |

## Issues Found

### FIXED — Go now checks `is_deleted` before activation (Python bug)

- **Python**: Queries user but does NOT check `is_deleted`. A soft-deleted user can be activated.
- **Go** (after fix): Calls `GetByID()` which has `WHERE is_deleted = false`. Soft-deleted user → `sql.ErrNoRows` → 404 "Token not found".
- **Impact**: Go correctly prevents activation of soft-deleted users. Python has the bug.

### FIXED — Go now checks `is_active` for idempotent handling

- **Go** (after fix): If user is already active, skips re-activation, cleans up the token, returns 200. More intentional than blindly running UPDATE.

### INFO — Atomicity difference (Go is arguably better)

- **Python**: `user.is_active = True` + `db.delete(activation_token)` happen in a single `db.commit()`. If anything fails, both are rolled back.
- **Go**: `userRepo.Activate()` runs first, then `tokenRepo.Delete()` runs separately. If token deletion fails, activation has already succeeded — the error is logged but not returned to the client.
- **Impact**: NONE in practice. The important operation (user activation) always completes. If token deletion fails, the orphan token would return 404 on retry (correct behavior since user is already active). Go's approach is arguably more resilient.

## Tests

### Python Tests (4 total)

| Test | File | Verifies |
|------|------|----------|
| `test_activate_success` | `test_auth_endpoints.py:332` | 200, body `true`, user `is_active` set to true |
| `test_activate_invalid_token` | `:370` | 404 on non-existent token |
| `test_activate_expired_token` | `:376` | 400 on expired token |
| `test_activate_internal_error` | `:319` | 500 on mocked exception |

### Go Integration Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `TestActivateSuccess` | `handler_test.go:557` | 200, body `true`, user `is_active` verified in DB |
| `TestActivateInvalidToken` | `:605` | 404 "Token not found" |
| `TestActivateExpiredToken` | `:619` | 400 "Token expired" |
| `TestActivateDeletedUser` | `:654` | Soft-deleted user → 404 "Token not found" |
| `TestActivateAlreadyActivatedUser` | `:1125` | First activation 200, second with same token 404 (token deleted) |

### Go Unit Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestActivateDBError` | `handler_unit_test.go:315` | Generic error → 500 "Activation failed" |
| `TestActivateTokenNotFound` | `:336` | `ErrTokenNotFound` → 404 |
| `TestActivateTokenExpired` | `:357` | `ErrTokenExpired` → 400 |

### Additional Go Test

| Test | File | Verifies |
|------|------|----------|
| `TestActivateEmptyToken` | `handler_test.go:1077` | Empty path → 401 (route mismatch) |
