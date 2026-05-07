# Auth Endpoints

Authentication and user account management endpoints for registration, login, OAuth, profile retrieval, account activation, and password change.

## Table of Contents

- [POST /auth/register/](#post-authregister)
- [POST /auth/login/](#post-authlogin)
- [GET /auth/profile/](#get-authprofile)
- [GET /auth/activate/:token](#get-authactivatetoken)
- [POST /auth/oauth/](#post-authoauth)
- [POST /auth/change-password/](#post-authchange-password)

---

## POST /auth/register/

**Auth**: Public
**Handler**: `internal/handlers/auth/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| email | string | Yes | Valid email format |
| password | string | Yes | Min 3, max 50 characters |
| first_name | string | No | snake_case only |
| last_name | string | No | snake_case only |

### Response

**Success**: HTTP 200

```json
{
  "id": 123,
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe"
}
```

No JWT token is returned. The user must activate their account via email, then log in.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Invalid JSON | 422 | "Invalid request data" |
| Validation failure | 422 | "Validation failed" |
| Duplicate email | 422 | "User with this email already exists" |
| DB / internal error | 500 | "Registration failed" |

### Business Logic

- Validates email format and password length (3-50 chars) via go-playground/validator
- Checks for duplicate email via `userRepo.GetByEmail`; returns 422 if already registered
- Hashes password with bcrypt at `DefaultCost`
- Creates user with `IsActive=false` and `BaseCurrencyID=USD`
- Creates default user settings via `settingsRepo.CreateDefault`; failure is logged but does not block registration
- Copies default categories via `categoryRepo.CopyDefaultCategories`; failure is logged but does not block registration
- Creates a trial subscription via `subscriptionSvc.CreateTrialSubscription(userID)` (non-blocking: failure is logged but does not prevent registration). The trial plan is found by searching active plans for `PlanType == "trial"`. Trial duration is configured via the service.
- Generates a 16-byte hex activation token (via `crypto/rand`) with a 24-hour expiry
- Enqueues an activation email task via Asynq (`email:activation` -> `email:send` chain). The email contains a styled activation link pointing to `{FRONTEND_URL}/activate/{token}`.
- Request body accepts only snake_case field names (`first_name`, `last_name`)

### Tests

**Integration tests:**
- `TestRegisterSuccess` -- 200 with correct response fields
- `TestRegisterDuplicateEmail` -- 422 on duplicate email
- `TestRegisterInvalidEmail` -- 422 on malformed email
- `TestRegisterWeakPassword` -- 3-character password is accepted (minimum boundary)
- `TestRegisterMissingRequiredFields` -- 422 without password
- `TestRegisterEmptyEmail` -- 422 on empty email
- `TestRegisterWithoutOptionalFields` -- succeeds with only email and password
- `TestRegisterInvalidJSON` -- 422 on malformed JSON body
- `TestRegisterEmptyBody` -- 422 on empty body
- `TestRegisterEmptyPassword` -- 422 on empty password

**Unit tests:**
- `TestRegisterDBError` -- DB error returns 500
- `TestRegisterBindError` -- bind error returns 422
- `TestRegisterValidationError` -- validation error returns 422

---

## POST /auth/login/

**Auth**: Public
**Handler**: `internal/handlers/auth/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| email | string | Yes | Valid email format |
| password | string | Yes | Min 3, max 50 characters |

### Response

**Success**: HTTP 200

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "bearer"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Invalid JSON | 422 | "Invalid request data" |
| Validation failure | 422 | "Validation failed" |
| User not found | 401 | "Invalid credentials or user not found" |
| Wrong password | 401 | "Invalid credentials or user not found" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Invalid credentials. See logs for details" |

### Business Logic

- Validates email and password length (3-50 chars)
- Looks up user by email via `userRepo.GetByEmail`, which filters `is_deleted = false`; soft-deleted users cannot log in
- Verifies password with `bcrypt.CompareHashAndPassword`
- Checks `is_active`; inactive users get 401 "User not activated"
- Signs a JWT (HS256) with claims: `user_id`, `email`, `exp`, `iat`
- JWT expiry is configurable in hours

### Tests

**Integration tests:**
- `TestLoginSuccess` -- 200 with token response
- `TestLoginWrongPassword` -- 401 on wrong password
- `TestLoginNonexistentUser` -- 401 on unknown email
- `TestLoginInvalidEmailFormat` -- 422 on bad email format
- `TestLoginMissingPassword` -- 422 without password field
- `TestLoginInactiveUser` -- 401 "User not activated"
- `TestLoginEmptyEmail` -- 422 on empty email
- `TestLoginEmptyPassword` -- 422 on empty password
- `TestLoginInvalidJSON` -- 422 on malformed JSON

**Unit tests:**
- `TestLoginDBError` -- DB error returns 500
- `TestLoginBindError` -- bind error returns 422
- `TestLoginValidationError` -- validation error returns 422

---

## GET /auth/profile/

**Auth**: Required (JWT via `auth-token` header)
**Handler**: `internal/handlers/auth/handler.go`

### Request

No request body. User is identified via JWT in the `auth-token` header.

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "baseCurrency": "USD",
  "settings": {
    "language": "en",
    "projectionEndDate": null,
    "projectionPeriod": null
  }
}
```

A refreshed JWT is set in the `new_access_token` response header on every successful authenticated request (handled by the `RequireAuth` middleware, not by this handler specifically).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth-token | 422 | "Missing authorization header" |
| Invalid token | 401 | "Invalid token" |
| Expired token | 401 | "Token has expired" |
| User not found | 401 | "User not found" |
| DB error | 500 | "Failed to get profile" |

### Business Logic

- User ID is extracted from JWT claims by `RequireAuth` middleware and stored in Echo context
- All user data is fetched fresh from the DB on every request (not from JWT payload)
- User query includes a JOIN for base currency; `is_deleted = false` and `is_active = true` are enforced
- Settings are fetched via `settingsRepo.GetByUserID`; if no settings record exists the field is omitted (graceful nil handling)
- The `RequireAuth` middleware generates a new JWT and sets it in the `new_access_token` response header on every 2xx response, extending the session on each activity (sliding session). This applies to all authenticated endpoints, not just profile

### Tests

**Integration tests:**
- `TestGetProfileSuccess` -- 200 with email and baseCurrency fields
- `TestGetProfileUnauthorized` -- 401/422 without token
- `TestGetProfileInvalidToken` -- 401 on bad token
- `TestGetProfileExpiredToken` -- 401 on expired token

**Unit tests:**
- `TestGetProfileDBError` -- DB error returns 500
- `TestGetProfileUserNotFound` -- user not found returns 401
- `TestGetProfileSuccessWithoutSettings` -- profile returns 200 when user has no settings record

**Middleware tests (sliding session -- applies to all authenticated endpoints):**
- `TestRequireAuth_ValidTokenSetsNewAccessTokenHeader` -- new_access_token header set on 2xx
- `TestRequireAuth_NoRefreshOnErrorResponse` -- no header on 4xx
- `TestRequireAuth_NoRefreshOn500Response` -- no header on 5xx
- `TestRequireAuth_NoRefreshWhenHandlerReturnsError` -- no header when handler returns error
- `TestRequireAuth_RefreshOnCreatedResponse` -- header set on 201

---

## GET /auth/activate/:token

**Auth**: Public
**Handler**: `internal/handlers/auth/handler.go`

### Request

`token` is a path parameter. No body, no query parameters.

### Response

**Success**: HTTP 200

```json
true
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Token not found | 404 | "Token not found" |
| Token expired | 400 | "Token expired" |
| Internal error | 500 | "Activation failed" |

### Business Logic

- Looks up the activation token via `tokenRepo.GetByToken`; returns 404 on `sql.ErrNoRows`
- Checks token expiry with `time.Now().After(activationToken.ExpiresAt)`; returns 400 if expired
- Fetches the user via `userRepo.GetByID`, which enforces `is_deleted = false`; soft-deleted users cannot be activated (returns 404 "Token not found")
- If the user is already active, skips re-activation, cleans up the token, and returns 200 (idempotent)
- Sets `is_active = true` via `userRepo.Activate`; then deletes the token via `tokenRepo.Delete`
- If token deletion fails the error is logged but activation is still treated as successful (the important operation already completed)

### Tests

**Integration tests:**
- `TestActivateSuccess` -- 200, `is_active` verified in DB
- `TestActivateInvalidToken` -- 404 "Token not found"
- `TestActivateExpiredToken` -- 400 "Token expired"
- `TestActivateDeletedUser` -- soft-deleted user returns 404
- `TestActivateAlreadyActivatedUser` -- second activation with the same token returns 404 (token was deleted)
- `TestActivateEmptyToken` -- empty path returns 401 (route mismatch)

**Unit tests:**
- `TestActivateDBError` -- generic error returns 500
- `TestActivateTokenNotFound` -- `ErrTokenNotFound` returns 404
- `TestActivateTokenExpired` -- `ErrTokenExpired` returns 400

---

## POST /auth/oauth/

**Auth**: Public
**Handler**: `internal/handlers/auth/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| credential | string | Yes | Google ID token |

### Response

**Success**: HTTP 200

```json
{
  "accessToken": "<JWT>",
  "tokenType": "bearer"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing credential | 422 | "Validation failed" |
| Invalid JSON | 422 | "Invalid request data" |
| Invalid Google token | 401 | "Invalid credentials or user not found" |
| No email in payload | 422 | "No email provided" |
| Email not verified | 401 | "Email not verified" |
| User not activated | 401 | "User not activated" |
| DB / service error | 500 | "Invalid credentials. See logs for details" |

### Business Logic

- Validates the Google ID token via `googleValidator.Validate` (interface); invalid tokens return 401 (not an unhandled 500)
- Extracts `email`, `email_verified`, `given_name`, `family_name` from Google payload claims
- Requires `email_verified == true`; returns 401 otherwise
- Looks up user by email via `GetByEmail`, which filters `is_deleted = false`; soft-deleted users are treated as not found
- If existing user is inactive returns 401 "User not activated"
- If existing active user is found, generates and returns a JWT
- If no user found, creates a new user: `IsActive: true`, random 32-char hex password, base currency USD, default settings, default categories; activation token/email are skipped for OAuth users
- Creates a trial subscription for new OAuth users via `subscriptionSvc.CreateTrialSubscription(userID)` (non-blocking: failure is logged but does not prevent login)
- Settings and categories creation failures are logged but do not block the response

### Known Gaps / TODOs

- Existing user's name is not updated from the Google payload on subsequent logins (same gap as original).

### Tests

**Integration tests:**
- `TestOAuthSuccessNewUser` -- 200, new user created with accessToken and tokenType
- `TestOAuthSuccessExistingUser` -- 200, existing active user gets token
- `TestOAuthEmailNotVerified` -- 401 "Email not verified"
- `TestOAuthNoEmail` -- 422 "No email provided"
- `TestOAuthGoogleValidationFails` -- 401 on Google validation error
- `TestOAuthInactiveUser` -- 401 "User not activated"
- `TestOAuthMissingCredential` -- 422 on missing credential field
- `TestOAuthInvalidJSON` -- 422 on malformed JSON
- `TestOAuthEmptyBody` -- 422 on empty body
- `TestOAuthEmptyCredential` -- 422 on empty credential string
- `TestOAuthInvalidToken` -- 401 on invalid token

**Unit tests:**
- `TestOAuthDBError` -- DB error returns 500
- `TestOAuthBindError` -- malformed JSON returns 422
- `TestOAuthValidationError` -- empty body returns 422

---

## POST /auth/change-password/

**Auth**: Required (JWT via `auth-token` header)
**Handler**: `internal/handlers/auth/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| current_password | string | Yes | Min 3, max 50 characters |
| new_password | string | Yes | Min 3, max 50 characters |

### Response

**Success**: HTTP 200

```json
{
  "success": true,
  "message": "Password changed successfully"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth-token | 401 | "Missing authorization header" |
| Invalid token | 401 | "Invalid token" |
| Expired token | 401 | "Token has expired" |
| Invalid body | 422 | "Invalid request data" |
| Missing fields | 422 | "Validation failed" |
| User not found | 401 | "User not found" |
| User not activated | 401 | "User not activated" |
| Wrong current password | 401 | "Current password is incorrect" |
| Internal error | 500 | "Failed to change password" |

### Business Logic

- `RequireAuth` middleware validates JWT and injects `user_id` into context
- Fetches user via `userRepo.GetByID`, which filters `is_deleted = false`; soft-deleted users cannot change password
- Checks `is_active`; inactive users get 401 "User not activated"
- Verifies `current_password` with `bcrypt.CompareHashAndPassword`; returns 401 on mismatch
- Hashes new password with `bcrypt.GenerateFromPassword` at `DefaultCost`
- Updates password via `userRepo.UpdatePassword`
- No check for same-as-current password (allowed)
- No password strength validation beyond length (3-50 chars)

### Tests

**Integration tests:**
- `TestChangePasswordSuccess` -- 200, login with new password succeeds afterward
- `TestChangePasswordWrongCurrent` -- 401 on wrong current password
- `TestChangePasswordUnauthorized` -- 401/422 without auth token
- `TestChangePasswordSameAsCurrent` -- 200 (same password is accepted)
- `TestChangePasswordInvalidJSON` -- 422 on malformed JSON
- `TestChangePasswordEmptyBody` -- 422 on empty body
- `TestChangePasswordMissingCurrentPassword` -- 422 when current_password is absent
- `TestChangePasswordMissingNewPassword` -- 422 when new_password is absent

**Unit tests:**
- `TestChangePasswordDBError` -- DB error returns 500
- `TestChangePasswordUserNotFound` -- `ErrUserNotFound` returns 401
- `TestChangePasswordIncorrect` -- `ErrIncorrectPassword` returns 401
- `TestChangePasswordBindError` -- invalid JSON returns 422
- `TestChangePasswordValidationError` -- missing field returns 422
