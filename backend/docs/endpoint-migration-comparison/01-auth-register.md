# Endpoint #1: POST `/auth/register/`

**Status**: Reviewed
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|----|
| Path | `POST /auth/register/` | `POST /auth/register/` |
| Auth | None (public) | None (public) |
| File | `app/routes/auth.py:34` | `internal/handlers/auth/handler.go:60` |
| Route reg | `app/main.py:65` | `internal/server/server.go:150` |

## Request Body

| Field | Python Type | Go Type | Required | Validation |
|-------|------------|---------|----------|------------|
| email | `EmailStr` | `string` | Yes | required, valid email |
| password | `str` | `string` | Yes | required, min=3, max=50 |
| first_name | `str` (default `''`) | `string` | No | none |
| last_name | `str` (default `''`) | `string` | No | none |
| id | `int \| None` (default `None`) | **not accepted** | No | Python only |

**Note**: Python accepts both camelCase and snake_case field names (via `alias_generator=to_camel` + `populate_by_name=True`). Go accepts only snake_case (`json:"first_name"`, `json:"last_name"`).

## Response

**Success**: 200 OK (both)

```json
{
  "id": 123,
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe"
}
```

No JWT token returned. User must activate via email, then login.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message |
|-----------|--------------|----------------|-----------|------------|
| Invalid JSON | 422 | Pydantic auto | 422 | "Invalid request data" |
| Validation fail | 422 | Pydantic auto | 422 | "Validation failed" |
| Duplicate email | 422 | "User with this email already exists" | 422 | "User with this email already exists" |
| DB/internal error | 500 | Unhandled exception | 500 | "Registration failed" |
| TRIAL plan missing | 500 | "TRIAL plan not found in database" | Logged, non-blocking | OK (Go is better) |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|----|-------|
| Input validation | Pydantic (email, password length) | go-playground/validator (email, password length) | OK |
| Duplicate email check | `db.query(User).filter(User.email == ...)` | `userRepo.GetByEmail(email)` | OK |
| Password hashing | bcrypt with `gensalt()` | bcrypt with `DefaultCost` | OK |
| User creation | `is_active=False`, `base_currency=USD` | `IsActive=false`, `BaseCurrencyID=USD` | OK |
| Trial subscription | Creates Subscription (TRIAL, 45 days) | Creates via `subscriptionSvc.CreateTrialSubscription(userID)` (14 days, non-blocking) | OK (different duration) |
| Default settings | `generate_initial_settings()` | `settingsRepo.CreateDefault()` | OK |
| Default categories | `copy_all_categories()` (recursive) | `categoryRepo.CopyDefaultCategories()` | OK |
| Activation token | `secrets.token_hex(16)`, 24hr expiry | `crypto/rand` 16 bytes hex, 24hr expiry | OK |
| Activation email | Celery task `send_activation_email.delay()` | **TODO - NOT IMPLEMENTED** | FAIL |
| Settings/categories error handling | Exception propagates (500) | Logged, registration continues | DIFFERENT (Go is better) |

## Issues Found

### RESOLVED - Trial subscription now created

- **Python**: Creates a `Subscription` record with TRIAL plan (45-day trial period)
- **Go**: Creates trial subscription via `subscriptionSvc.CreateTrialSubscription(userID)` (14-day trial, non-blocking — errors logged, not propagated)
- **Difference**: Go uses 14-day trial (configurable) vs Python's 45-day. Go creation is non-blocking.

### HIGH - Activation email not sent

- **Python**: Sends activation email via Celery task with activation link
- **Go**: Has `// TODO: Queue activation email` comment at `service.go:127`
- **Impact**: Users cannot activate their account (unless using a workaround)

### MEDIUM - Request field name format

- **Python**: Accepts both `camelCase` (`firstName`) and `snake_case` (`first_name`)
- **Go**: Accepts only `snake_case` (`first_name`, `last_name`)
- **Impact**: Frontend sending camelCase names will have them ignored

### LOW - `id` field in request

- **Python**: Accepts optional `id` field to set custom user ID
- **Go**: Does not accept `id` in request body
- **Impact**: Likely intentional security improvement in Go

## Tests

### Python Tests (6 total)

| Test | File | Verifies |
|------|------|----------|
| `test_register_success` | `test_auth_endpoints.py:31` | 200 + correct response fields |
| `test_register_duplicate_email` | `:55` | 422 on duplicate |
| `test_register_invalid_email` | `:69` | 422 on bad email |
| `test_register_weak_password` | `:83` | 3-char password accepted |
| `test_register_missing_required_fields` | `:103` | 422 without password |
| `test_register_empty_email` | `:115` | 422 on empty email |

### Go Integration Tests (10 total)

| Test | File | Verifies |
|------|------|----------|
| `TestRegisterSuccess` | `handler_test.go:47` | 200 + correct response |
| `TestRegisterDuplicateEmail` | `:95` | 422 on duplicate |
| `TestRegisterInvalidEmail` | `:126` | 422 on bad email |
| `TestRegisterWeakPassword` | `:148` | 3-char password accepted |
| `TestRegisterMissingRequiredFields` | `:175` | 422 without password |
| `TestRegisterEmptyEmail` | `:195` | 422 on empty email |
| `TestRegisterWithoutOptionalFields` | `:217` | Only email+password works |
| `TestRegisterInvalidJSON` | `:733` | Malformed JSON → 422 |
| `TestRegisterEmptyBody` | `:794` | Empty body → 422 |
| `TestRegisterEmptyPassword` | `:895` | Empty password → 422 |

### Go Unit Tests (3 total)

| Test | File | Verifies |
|------|------|----------|
| `TestRegisterDBError` | `handler_unit_test.go:134` | DB error → 500 |
| `TestRegisterBindError` | `:155` | Bind error → 422 |
| `TestRegisterValidationError` | `:171` | Validation error → 422 |
