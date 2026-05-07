# Endpoint #7: POST `/accounts/`

**Status**: PORTED OK (minor differences)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /accounts/` | `POST /accounts/` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:48` | `internal/handlers/accounts/handler.go:37` |
| Route reg | `app/main.py` | `internal/server/server.go:162-165` |

## Request

Both: POST with JSON body.

| Field | Python (camelCase alias) | Go (json tag) | Match |
|-------|--------------------------|---------------|-------|
| `name` | str, required | string, required | OK |
| `currencyId` / `currency_id` | int, required | int (`currencyId`), required | OK |
| `accountTypeId` / `account_type_id` | int, required | int (`accountTypeId`), required | OK |
| `initialBalance` | Decimal, default 0 | decimal.Decimal | OK |
| `balance` | Decimal, default 0 | decimal.Decimal | OK |
| `creditLimit` | Decimal, default 0 (None→0) | decimal.Decimal | OK |
| `openingDate` | datetime/null, default None | *time.Time | OK |
| `comment` | str, default "" | string | OK |
| `isHidden` | bool, default False | bool | OK |
| `showInReports` | bool, default True | bool (see issue) | DIFFERENT |

## Response

**Success**: 200 OK (both). Same `AccountResponse` structure with nested `currency` and `accountType`.

### Field comparison:

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `name` | string | string | OK |
| `currencyId` | int | int | OK |
| `accountTypeId` | int | int | OK |
| `initialBalance` | float | float64 | OK |
| `balance` | float | float64 | OK |
| `creditLimit` | float | float64 | OK |
| `openingDate` | datetime string/null | RFC3339 string/null | OK |
| `comment` | string | string | OK |
| `isHidden` | bool | bool | OK |
| `showInReports` | bool | bool | OK |
| `currency` | {id, code, name} | {id, code, name} | OK |
| `accountType` | {id, type_name, is_credit} (snake_case) | {id, type_name, is_credit} (snake_case) | OK |
| `isDeleted` | bool | bool | OK |
| `isArchived` | bool | bool | OK |
| `balanceInBaseCurrency` | float (0.0 on create) | float64 (computed) | Go BETTER |
| `archivedAt` | datetime/null | RFC3339/null | OK |
| `userId` | int/null (in response) | int (in response) | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic header validation | 401 | "Missing authorization header" | DIFFERENT |
| Invalid currency | 400 | "Failed to create account" | 400 | "Invalid currency" | Go BETTER (specific) |
| Invalid account type | 400 | "Failed to create account" | 400 | "Invalid account type" | Go BETTER (specific) |
| Account limit exceeded | 402 | JSON with limits info | 402 | "Account limit exceeded" | OK |
| Missing required fields | 422 | Pydantic error | 422 | "Validation failed" | OK |
| Internal error | 400 | "Failed to create account" | 500 | "Failed to create account" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Validate user exists | YES | YES (GetByID) | OK |
| Validate currency | YES | YES (GetByID) | OK |
| Validate account type | YES | YES (GetByID) | OK |
| Default openingDate | `datetime.now(UTC)` if None | `time.Now().UTC()` if nil | OK |
| Default showInReports | True (schema default) | Bug — see issue | DIFFERENT |
| Account limit check | `check_account_limit` (subscription-based) | `CheckAccountLimit` (subscription-based) | OK |
| balanceInBaseCurrency on create | NOT computed (returns 0.0) | Computed via currency conversion | Go BETTER |
| Trial subscription check | YES (enforce_free_plan_compliance) | YES (subscription middleware) | OK |

## Issues Found

### FIXED (Go) — `showInReports` default logic

- **Python**: Schema has `default=True` for `showInReports`. If field is omitted, it's `True`.
- **Go** (after fix): `ShowInReports` in DTO changed to `*bool` (pointer). Handler checks `if req.ShowInReports != nil` — if omitted, defaults to `true`. If explicitly set, uses the provided value.
- **Impact**: Fixed. Both now correctly default to `true` when field is omitted.

### INFO — More specific error messages in Go

- Python returns generic "Failed to create account" for invalid currency/type.
- Go returns specific "Invalid currency" or "Invalid account type".
- **Impact**: Better developer experience in Go.

### INFO — balanceInBaseCurrency computed on create in Go

- Python does not compute `balanceInBaseCurrency` on create (returns 0.0).
- Go computes it using currency conversion.
- **Impact**: Go provides more complete data on create.

## Tests

### Python Tests (8 total)

| Test | File | Verifies |
|------|------|----------|
| `test_create_account_success` | `test_accounts_endpoints.py:60` | 200, name/balance/currencyId |
| `test_create_credit_account_success` | `:90` | 200, creditLimit |
| `test_create_account_invalid_currency` | `:122` | 400 |
| `test_create_account_invalid_type` | `:145` | 400 |
| `test_create_account_unauthorized` | `:168` | 422 |
| `test_create_account_missing_required_fields` | `:186` | 422 |
| `test_create_hidden_account` | `:203` | 200, isHidden=true |
| `test_create_account_limit_exceeded` | `:32` | 402 |

### Go Integration Tests (11 total)

| Test | File | Verifies |
|------|------|----------|
| `TestCreateAccountSuccess` | `handler_test.go:16` | 200, basic fields |
| `TestCreateCreditAccountSuccess` | `:69` | 200, creditLimit |
| `TestCreateAccountInvalidCurrency` | `:125` | 400 |
| `TestCreateAccountInvalidType` | `:160` | 400 |
| `TestCreateAccountUnauthorized` | `:195` | 401/422 |
| `TestCreateAccountMissingRequiredFields` | `:222` | 422 |
| `TestCreateHiddenAccount` | `:251` | 200, isHidden=true |
| `TestCreateAccountInvalidJSON` | `:1443` | 422 |
| `TestCreateAccountWithZeroBalance` | `:1470` | 200 |
| `TestCreateAccountWithNegativeBalance` | `:1516` | 200, balance=-500 |
| `TestCreateAccountWith100CharName` | `:1829` | 200 |

### Go Unit Tests (2 total)

| Test | File | Verifies |
|------|------|----------|
| `TestCreateAccountDBError` | `handler_unit_test.go:113` | Generic error → 500 |
| `TestCreateAccountLimitExceeded` | `:135` | ErrAccountLimitExceeded → 402 |
