# Endpoint #14: POST `/accounts/adjust-balance`

**Status**: PORTED OK (Go better)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /accounts/adjust-balance` | `POST /accounts/adjust-balance` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/accounts.py:172` | `internal/handlers/accounts/handler.go:289` |

## Request

Both: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountId` | int, required | int, required (`validate:"required"`) | OK |
| `newBalance` | Decimal, required | decimal.Decimal, required | OK |
| `notes` | str/None, optional | *string, optional | OK |

## Response

**Success**: 200 OK. Transaction response object (both).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Account not found | 400 | "Account not found" | 400 | "Account not found" | EXACT |
| Access denied | 400 | "Failed to adjust balance" | 400 | "Access denied" | Go BETTER |
| Balance unchanged | 400 | "Failed to adjust balance" | 400 | "Balance unchanged" | Go BETTER |
| Free plan entity access | 403 | "Access denied" | — | No subscription check | DIFFERENT |
| Internal error | 500 | — | 500 | "Failed to adjust balance" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Validate user exists + is_active | NO | YES | Go BETTER (fixed) |
| Check is_deleted on account | NO | YES (`GetByID` filters `is_deleted = false`) | Go BETTER (fixed) |
| Check entity access (subscription) | YES | NO | DIFFERENT |
| Check ownership | YES | YES | OK |
| Compute difference | YES | YES | OK |
| Balance unchanged check | YES | YES | OK |
| Create adjustment transaction | YES (label="Balance adjustment") | YES (label="Balance adjustment") | OK |
| isAdjustment=true | YES | YES | OK |
| excludeFromReports=true | YES | YES | OK |
| Update account balance | YES | YES | OK |
| Base currency conversion | YES (`calc_amount`) | YES (currencyService) | OK |
| DB transaction wrapping | NO (TOCTOU risk) | NO (TOCTOU risk) | Same issue |

## Issues Found

### INFO — No DB transaction wrapping (both)

- Both Python and Go create the transaction record and then update the balance separately.
- If the second operation fails, data is inconsistent.
- **Impact**: Pre-existing in both. Security review noted this as Medium.

## Tests

### Python Tests (9 total)

| Test | File | Verifies |
|------|------|----------|
| `test_adjust_balance_increase_success` | `test_accounts_endpoints.py:940` | 200, isAdjustment, isIncome, notes |
| `test_adjust_balance_decrease_success` | `:976` | 200, isIncome=false |
| `test_adjust_balance_same_balance_fails` | `:1009` | 400 |
| `test_adjust_balance_not_found` | `:1028` | 400 |
| `test_adjust_balance_other_user_account` | `:1045` | 400 |
| `test_adjust_balance_unauthorized` | `:1070` | 422 |
| `test_adjust_balance_entity_access_denied` | `:1082` | 403 |
| `test_adjust_balance_without_notes` | `:1105` | 200, notes="" |
| `test_adjust_balance_negative_balance` | `:1130` | 200 |

### Go Integration Tests (9 total)

| Test | File | Verifies |
|------|------|----------|
| `TestAdjustBalanceIncreaseSuccess` | `handler_test.go:976` | 200, isAdjustment, isIncome, notes |
| `TestAdjustBalanceDecreaseSuccess` | `:1030` | 200, isIncome=false |
| `TestAdjustBalanceSameBalanceFails` | `:1077` | 400 |
| `TestAdjustBalanceAccountNotFound` | `:1112` | 400 |
| `TestAdjustBalanceOtherUser` | `:1142` | 400 |
| `TestAdjustBalanceUnauthorized` | `:1181` | 401/422 |
| `TestAdjustBalanceNegativeBalance` | `:1201` | 200 |
| `TestAdjustBalanceInvalidJSON` | `:1678` | 422 |
| `TestAdjustBalanceInactiveUser` | `:2218` | 401 |

### Go Unit Tests (5 total)

| Test | File | Verifies |
|------|------|----------|
| `TestAdjustBalanceUnchanged` | `handler_unit_test.go:432` | 400 |
| `TestAdjustBalanceDBError` | `:454` | 500 |
| `TestAdjustBalanceValidationError` | `:495` | 422 |
| `TestAdjustBalanceNotFound` | `:514` | 400 |
| `TestAdjustBalanceAccessDenied` | `:536` | 400 |
