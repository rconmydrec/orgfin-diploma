# Endpoint #15: POST `/transactions/`

**Status**: NEEDS FIX (missing is_active check)
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `POST /transactions/` | `POST /transactions/` |
| Auth | `check_token` + `enforce_free_plan_compliance` | `RequireAuth` middleware |
| File | `app/routes/transations.py:45` | `internal/handlers/transactions/handler.go:38` |
| Route reg | `app/main.py` | `internal/server/server.go:168-171` |

## Request

Both: POST with JSON body.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `accountId` | int, required | int, required | OK |
| `targetAccountId` | int/None, optional | *int, optional | OK |
| `categoryId` | int/None, optional | *FlexInt, optional | Go BETTER (accepts string/int) |
| `amount` | Decimal, required | decimal.Decimal, required | OK |
| `targetAmount` | Decimal/None, optional | *decimal.Decimal, optional | OK |
| `label` | str, default "" | string | OK |
| `notes` | str/None, default "" | *string | OK |
| `dateTime` | datetime/None | *time.Time | OK |
| `isTransfer` | bool, required | bool | OK |
| `isIncome` | bool, required | bool | OK |
| `isAdjustment` | bool, default False | bool | OK |
| `excludeFromReports` | bool, default False | bool | OK |
| `isTemplate` | bool, default False | bool | OK |

## Response

**Success**: 200 OK. Transaction response (both).

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Missing auth | 422 | Pydantic validation | 401 | "Missing authorization header" | DIFFERENT |
| Invalid account | 422 | "Invalid account" | 422 | "Invalid account" | EXACT |
| Invalid category | 422 | "Invalid category" | 422 | "Invalid category" | EXACT |
| Access denied | 403 | "Access denied" | 403 | "Access denied" | EXACT |
| Free plan entity access | 403 | "Access denied" | — | No subscription check | DIFFERENT |
| Internal error | 500 | "Unable to create transaction" | 500 | "Failed to create transaction" | OK |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Check user is_active | NO | NO | **BOTH MISSING — FIX NEEDED** |
| Check account is_deleted | NO | YES (`GetByID` filters `is_deleted = false`) | Go BETTER |
| Check entity access (subscription) | YES | NO | DIFFERENT |
| Validate account + ownership | YES | YES | OK |
| Validate target account (transfer) | YES | YES | OK |
| Validate category | YES (with income/expense type check) | YES (weak — only existence) | Python BETTER |
| Default dateTime to now() | YES | YES | OK |
| Calculate base currency amount | YES | YES | OK |
| Calculate new balance | YES | YES | OK |
| Create transaction | YES | YES | OK |
| Update account balance | YES | YES | OK |
| Handle transfer (linked tx) | YES | YES | OK |
| Create template if isTemplate | YES | YES | OK |
| Trigger budget update | YES (Celery) | NO (not implemented) | DIFFERENT |
| Running balance recalculation | YES (all account txs) | NO (inline calc only) | DIFFERENT |

## Issues Found

### BUG — Missing is_active check on user

- **Python**: No `is_active` check on user (only account `is_active` for free plan).
- **Go**: No `is_active` check on user either.
- **Impact**: Inactive user with valid JWT can create transactions. **Must be fixed in Go.**

### INFO — Category type validation weaker in Go

- **Python**: Validates category type matches transaction direction (income category for income tx).
- **Go**: Only checks category existence, not type.
- **Impact**: Go allows using expense category for income transactions. Pre-existing.

### INFO — No running balance recalculation on create in Go

- **Python**: Calls `update_transactions_new_balances()` which recalculates ALL transaction balances.
- **Go**: Calculates `new_balance` inline for the new transaction only.
- **Impact**: Different approach but functionally equivalent for creation.

## Tests

### Python Tests (10 total)

| Test | File | Verifies |
|------|------|----------|
| `test_create_entity_access_denied` | `test_transactions_endpoints.py` | 403 |
| `test_create_access_denied` | | 403 |
| `test_create_expense_success` | | 200 |
| `test_create_income_success` | | 200 |
| `test_create_transfer_success` | | 200 |
| `test_create_transfer_different_currencies` | | 200 |
| `test_create_transaction_invalid_account` | | 422 |
| `test_create_transaction_invalid_category` | | 422 |
| `test_create_transaction_unauthorized` | | 422 |
| `test_create_transaction_with_template` | | 200 |

### Go Integration Tests (17 total)

| Test | File | Verifies |
|------|------|----------|
| `TestCreateExpenseTransaction` | `handler_test.go:26` | 200, expense |
| `TestCreateIncomeTransaction` | `:72` | 200, income |
| `TestCreateTransferTransaction` | `:115` | 200, transfer |
| `TestCreateTransferDifferentCurrencies` | `:158` | 200, cross-currency |
| `TestCreateTransactionWithNotes` | `:204` | 200, notes |
| `TestCreateTransactionWithDateTime` | `:243` | 200, dateTime |
| `TestCreateTransactionExcludeFromReports` | `:283` | 200, excludeFromReports |
| `TestCreateTransactionWithTemplate` | `:322` | 200, template created |
| `TestCreateTransactionWithoutCategory` | `:381` | 200, null category |
| `TestCreateTransactionInvalidAccount` | `:419` | 422 |
| `TestCreateTransactionOtherUserAccount` | `:446` | 403 |
| `TestCreateTransactionMissingAmount` | `:487` | OK (defaults 0) |
| `TestCreateTransactionMissingAccountId` | `:528` | 422 |
| `TestCreateTransactionUnauthorized` | `:554` | 401 |
| `TestCreateTransactionZeroAmount` | `:574` | 200 |
| `TestCreateTransactionDecimalAmount` | `:608` | 200 |
| `TestCreateTransactionCategoryIdAsString` | `:640` | 200 |

### Go Unit Tests (10 total)

Various: invalid account, invalid category, access denied, DB error, bind/validate errors, FlexInt tests.
