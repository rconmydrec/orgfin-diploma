# Endpoint #11: PUT `/accounts/:id`

**Status**: PORTED OK (Go better)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /accounts/{account_id}` | `PUT /accounts/:id` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:130` | `internal/handlers/accounts/handler.go:167` |
| Route reg | `app/main.py` | `internal/server/server.go:162-165` |

## Request

Both: PUT with JSON body + account ID path parameter.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `name` | str, required | string | OK |
| `currencyId` / `currency_id` | int, required | int (`currency_id`) | OK |
| `accountTypeId` / `account_type_id` | int, required | int (`account_type_id`) | OK |
| `initialBalance` / `initial_balance` | Decimal, default 0 | decimal.Decimal | OK |
| `balance` | Decimal, required | decimal.Decimal (ignored) | OK |
| `creditLimit` | Decimal, default 0 | decimal.Decimal | OK |
| `openingDate` / `opening_date` | datetime, required | string (parsed) | OK |
| `comment` | str, required | string | OK |
| `isHidden` / `is_hidden` | bool, required | bool | OK |
| `showInReports` / `show_in_reports` | bool, required | bool | OK |

**Note**: Both preserve balance from DB and ignore balance in request.

## Response

**Success**: 200 OK (both). Same AccountResponse structure.

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Account not found | 400 | "Failed to update account" | 400 | "Account not found" | Go BETTER |
| Access denied | 400 | "Failed to update account" | 400 | "Access denied" | Go BETTER |
| Invalid currency | 400 | "Failed to update account" | 400 | "Invalid currency" | Go BETTER |
| Invalid account type | 400 | "Failed to update account" | 400 | "Invalid account type" | Go BETTER |
| Invalid date format | — | (Pydantic auto) | 422 | "Invalid opening_date format" | OK |
| Free plan entity access | 403 | "Access to account is restricted..." | — | No subscription check | DIFFERENT |
| Internal error | 400 | "Failed to update account" | 500 | "Failed to update account" | DIFFERENT |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Parse account ID | Path param (int, auto-validated) | `strconv.Atoi` (manual) | OK |
| Validate user exists + is_active | NO | YES (`userRepo.GetByID` + `IsActive` check) | Go BETTER (fixed) |
| Check is_deleted on account | NO | YES (`GetByID` filters `is_deleted = false`) | Go BETTER (fixed) |
| Check entity access (subscription) | YES (`check_entity_access`) | NO | DIFFERENT |
| Validate currency | YES | YES | OK |
| Validate account type | YES | YES | OK |
| Preserve balance from DB | YES (explicit override) | YES (balance not in UpdateAccountInput) | OK |
| Check user ownership | YES | YES | OK |
| Compute balanceInBaseCurrency | NO | YES | Go BETTER |

## Issues Found

### FIXED (Go) — is_deleted filtered on account update

- **Python**: `get_account_details` does NOT filter `is_deleted`. A deleted account can be updated.
- **Go**: `accountRepo.GetByID` has `WHERE a.is_deleted = false`. Deleted accounts return 400.

### FIXED (Go) — is_active checked before update

- **Python**: No `is_active` check on user.
- **Go**: Service checks `user.IsActive` before proceeding. Inactive users get 401.

### INFO — Python initial_balance conditional update

- **Python**: `initial_balance` only updated if truthy — sending `initial_balance: 0` does NOT update it.
- **Go**: Always updates `initial_balance` from request.
- **Impact**: Go behavior is more predictable.

## Tests

### Python Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `test_update_account_entity_access_denied` | `test_accounts_endpoints.py:540` | 403 |
| `test_update_account_success` | `:571` | 200, name/comment updated |
| `test_update_account_not_found` | `:600` | 400 |
| `test_update_account_other_user` | `:630` | 403 |
| `test_update_account_unauthorized` | `:668` | 422 |

### Go Integration Tests (7 total)

| Test | File | Verifies |
|------|------|----------|
| `TestUpdateAccountSuccess` | `handler_test.go:674` | 200, fields updated |
| `TestUpdateAccountNotFound` | `:728` | 400 |
| `TestUpdateAccountOtherUser` | `:767` | 400/403 |
| `TestUpdateAccountUnauthorized` | `:809` | 401/422 |
| `TestUpdateAccountInvalidID` | `:1610` | 400/404 |
| `TestUpdateAccountInvalidJSON` | `:1646` | 422 |
| `TestUpdateAccountInactiveUser` | `:2097` | 401 |

### Go Unit Tests (4 total)

| Test | File | Verifies |
|------|------|----------|
| `TestUpdateAccountInvalidDateFormat` | `handler_unit_test.go:246` | 422 |
| `TestUpdateAccountDBError` | `:266` | 500 |
| `TestUpdateAccountInvalidCurrency` | `:290` | 400 |
| `TestUpdateAccountInvalidAccountType` | `:314` | 400 |
