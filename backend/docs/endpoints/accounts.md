# Accounts Endpoints

Account management endpoints for creating, listing, retrieving, updating, deleting, archiving, and adjusting account balances.

## Table of Contents

- [POST /accounts/](#post-accounts)
- [GET /accounts/](#get-accounts)
- [GET /accounts/types/](#get-accountstypes)
- [GET /accounts/:id](#get-accountsid)
- [PUT /accounts/:id](#put-accountsid)
- [DELETE /accounts/:id](#delete-accountsid)
- [PUT /accounts/set-archive-status](#put-accountsset-archive-status)
- [POST /accounts/adjust-balance](#post-accountsadjust-balance)

---

## POST /accounts/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | Yes | Account name |
| currencyId | int | Yes | Must reference a valid currency |
| accountTypeId | int | Yes | Must reference a valid account type |
| initialBalance | decimal.Decimal | No | Defaults to 0 |
| balance | decimal.Decimal | No | Defaults to 0 |
| creditLimit | decimal.Decimal | No | Defaults to 0 |
| openingDate | *time.Time | No | Defaults to `time.Now().UTC()` if omitted |
| comment | string | No | Defaults to empty string |
| isHidden | bool | No | Defaults to false |
| showInReports | bool | No | Defaults to true when omitted |

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "name": "My Account",
  "currencyId": 1,
  "accountTypeId": 2,
  "initialBalance": 0,
  "balance": 0,
  "creditLimit": 0,
  "openingDate": "2026-01-01T00:00:00Z",
  "comment": "",
  "isHidden": false,
  "showInReports": true,
  "isDeleted": false,
  "isArchived": false,
  "balanceInBaseCurrency": 0,
  "archivedAt": null,
  "userId": 1,
  "currency": {"id": 1, "code": "USD", "name": "US Dollar"},
  "accountType": {"id": 2, "type_name": "Cash", "is_credit": false}
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Invalid currency | 400 | "Invalid currency" |
| Invalid account type | 400 | "Invalid account type" |
| Account limit exceeded | 402 | "Account limit exceeded" |
| Missing required fields | 422 | "Validation failed" |
| Internal error | 500 | "Failed to create account" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Validates currency via `currencyRepo.GetByID`; returns 400 "Invalid currency" on failure
- Validates account type via `accountTypeRepo.GetByID`; returns 400 "Invalid account type" on failure
- `openingDate` defaults to `time.Now().UTC()` when not provided
- `showInReports` is a `*bool` pointer; handler sets it to `true` when the field is omitted
- Checks account limit via `CheckAccountLimit` (subscription-based); returns 402 on breach
- Computes `balanceInBaseCurrency` using currency conversion at creation time

### Tests

**Integration tests:**
- `TestCreateAccountSuccess` — 200 with basic fields verified
- `TestCreateCreditAccountSuccess` — 200 with creditLimit
- `TestCreateAccountInvalidCurrency` — 400 on invalid currency
- `TestCreateAccountInvalidType` — 400 on invalid account type
- `TestCreateAccountUnauthorized` — 401/422 without token
- `TestCreateAccountMissingRequiredFields` — 422 without name/currencyId/accountTypeId
- `TestCreateHiddenAccount` — 200 with isHidden=true
- `TestCreateAccountInvalidJSON` — 422 on malformed JSON
- `TestCreateAccountWithZeroBalance` — 200 with zero balance
- `TestCreateAccountWithNegativeBalance` — 200, balance=-500
- `TestCreateAccountWith100CharName` — 200 with max-length name

**Unit tests:**
- `TestCreateAccountDBError` — generic DB error returns 500
- `TestCreateAccountLimitExceeded` — `ErrAccountLimitExceeded` returns 402

---

## GET /accounts/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| includeHidden | bool | No | Default false; include hidden accounts |
| includeArchived | bool | No | Default false; include archived accounts |
| archivedOnly | bool | No | Default false; return only archived accounts |

### Response

**Success**: HTTP 200

JSON array of account objects (same structure as create response). Returns `[]` for an empty list.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Invalid user | 400 | "Invalid user" |
| Internal error | 500 | "Failed to get accounts" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Filters by `user_id` and `is_deleted = false` always
- By default excludes hidden accounts (`is_hidden = false`) and archived accounts (`is_archived = false`)
- `includeHidden=true` lifts the hidden filter
- `includeArchived=true` lifts the archived filter
- `archivedOnly=true` returns only `is_archived = true AND is_deleted = false` (soft-deleted archived accounts are excluded)
- Results are ordered by `name ASC`
- `balanceInBaseCurrency` is computed via currency conversion for each account

### Tests

**Integration tests:**
- `TestGetAccountsSuccess` — 200, test account found in response
- `TestGetAccountsEmptyList` — 200, empty array for user with no accounts
- `TestGetAccountsUnauthorized` — 401/422 without token
- `TestGetAccountsFilterHidden` — hidden accounts excluded/included based on param
- `TestGetAccountsFilterArchived` — archived accounts excluded/included based on param
- `TestGetAccountsArchivedOnly` — only archived accounts returned when archivedOnly=true

**Unit tests:**
- `TestGetAccountsInvalidUser` — `ErrInvalidUser` returns 400
- `TestGetAccountsDBError` — generic error returns 500

---

## GET /accounts/types/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

No request body, no query parameters.

### Response

**Success**: HTTP 200

```json
[
  {"id": 1, "type_name": "Cash", "is_credit": false},
  {"id": 2, "type_name": "Credit Card", "is_credit": true}
]
```

Fields use snake_case (`type_name`, `is_credit`).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| DB error | 500 | "Failed to get account types" |

### Business Logic

- Returns global account types (not user-specific)
- Filters `WHERE is_deleted = false ORDER BY id`; soft-deleted types are excluded
- No error handling was present in the original; Go returns 500 with a message on DB failure

### Tests

**Integration tests:**
- `TestGetAccountTypesSuccess` — 200, non-empty array with id/type_name/is_credit
- `TestGetAccountTypesUnauthorized` — 401/422 without token
- `TestGetAccountTypesInvalidToken` — 401 on bad token

**Unit tests:**
- `TestGetAccountTypesDBError` — DB error returns 500

---

## GET /accounts/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

`id` is a path parameter (integer). No body, no query parameters.

### Response

**Success**: HTTP 200

Single account object (same structure as create response).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Non-numeric ID | 400 | "Invalid account ID" |
| Account not found | 404 | "Account not found" |
| Other user's account | 400 | "Access denied" |
| Internal error | 500 | "Failed to get account" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Parses account ID with `strconv.Atoi`; returns 400 on non-numeric input
- Fetches account via `accountRepo.GetByID`, which enforces `is_deleted = false`; soft-deleted accounts return 404
- Checks `account.UserID == userID`; returns 400 "Access denied" on mismatch
- Computes `balanceInBaseCurrency` via currency conversion

### Tests

**Integration tests:**
- `TestGetAccountDetailsSuccess` — 200 with correct fields
- `TestGetAccountDetailsNotFound` — 404 on unknown ID
- `TestGetAccountDetailsOtherUser` — 400/403 for another user's account
- `TestGetAccountDetailsInvalidID` — 400/404 on non-numeric ID

**Unit tests:**
- `TestGetAccountDetailsDBError` — generic DB error returns 500

---

## PUT /accounts/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | Yes | |
| currency_id | int | Yes | snake_case |
| account_type_id | int | Yes | snake_case |
| initial_balance | decimal.Decimal | No | Always updated if present (even 0) |
| balance | decimal.Decimal | No | Ignored; balance is preserved from DB |
| credit_limit | decimal.Decimal | No | |
| opening_date | string | No | Parsed as datetime |
| comment | string | No | |
| is_hidden | bool | No | |
| show_in_reports | bool | No | |

### Response

**Success**: HTTP 200

Account object with updated fields (same structure as create response).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Invalid date format | 422 | "Invalid opening_date format" |
| Account not found | 400 | "Account not found" |
| Access denied | 400 | "Access denied" |
| Invalid currency | 400 | "Invalid currency" |
| Invalid account type | 400 | "Invalid account type" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to update account" |

### Business Logic

- Parses account ID with `strconv.Atoi`
- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches account via `accountRepo.GetByID`, which filters `is_deleted = false`
- Checks ownership; returns 400 "Access denied" on mismatch
- Validates currency and account type by ID
- Balance from the request body is ignored; current DB balance is preserved
- `initial_balance` is always updated from request (including 0), unlike the original which skipped falsy values
- Computes `balanceInBaseCurrency` after update

### Tests

**Integration tests:**
- `TestUpdateAccountSuccess` — 200, name and comment updated
- `TestUpdateAccountNotFound` — 400 on unknown account
- `TestUpdateAccountOtherUser` — 400/403 for another user's account
- `TestUpdateAccountUnauthorized` — 401/422 without token
- `TestUpdateAccountInvalidID` — 400/404 on non-numeric ID
- `TestUpdateAccountInvalidJSON` — 422 on malformed JSON
- `TestUpdateAccountInactiveUser` — 401 for inactive user

**Unit tests:**
- `TestUpdateAccountInvalidDateFormat` — invalid opening_date returns 422
- `TestUpdateAccountDBError` — DB error returns 500
- `TestUpdateAccountInvalidCurrency` — invalid currency returns 400
- `TestUpdateAccountInvalidAccountType` — invalid account type returns 400

---

## DELETE /accounts/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

`id` is a path parameter. No body.

### Response

**Success**: HTTP 200

```json
{"deleted": true}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Non-numeric ID | 400 | "Invalid account ID" |
| Account not found | 400 | "Account not found" |
| Access denied | 400 | "Access denied" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to delete account" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches account via `accountRepo.GetByID`, which filters `is_deleted = false`; already-deleted accounts return 400 "Account not found" (prevents double-delete)
- Checks ownership; returns 400 "Access denied" on mismatch
- Soft-deletes the account by setting `is_deleted = true`

### Tests

**Integration tests:**
- `TestDeleteAccountSuccess` — 200, is_deleted verified in DB
- `TestDeleteAccountUnauthorized` — 401/422 without token
- `TestDeleteAccountNotFound` — 400/404 on unknown account
- `TestDeleteAccountOtherUser` — 400/403 for another user's account
- `TestDeleteAccountInvalidID` — 400/404 on non-numeric ID
- `TestDeleteAccountInactiveUser` — 401 for inactive user

**Unit tests:**
- `TestDeleteAccountDBError` — DB error returns 500

---

## PUT /accounts/set-archive-status

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| accountId | int | Yes | |
| isArchived | bool | Yes | true to archive, false to unarchive |

### Response

**Success**: HTTP 200

```json
true
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Invalid JSON | 422 | validation message |
| Account not found | 500 | "Account not found" |
| Access denied | 401 | "Access denied" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to set archive status" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches account via `accountRepo.GetByID`, which filters `is_deleted = false`
- Checks ownership; returns 401 "Access denied" on mismatch
- Sets `is_archived` and `archived_at` to current time; uses a single DB call (no redundant second fetch)

### Known Gaps / TODOs

- Status code inconsistency: `ErrAccountNotFound` returns 500 in this endpoint but 400/404 in other endpoints; `ErrAccessDenied` returns 401 here but 400 elsewhere.
- When un-archiving (`isArchived=false`), `archived_at` is set to the current time instead of being cleared to `NULL`. This is a pre-existing data issue.

### Tests

**Integration tests:**
- `TestArchiveAccountSuccess` — 200, account marked as archived
- `TestUnarchiveAccountSuccess` — 200, account marked as unarchived
- `TestArchiveAccountOtherUser` — 401 for another user's account
- `TestArchiveAccountUnauthorized` — 401/422 without token
- `TestArchiveAccountInvalidJSON` — 422 on malformed JSON
- `TestArchiveAccountNotFound` — 401/404/500 on unknown account
- `TestSetArchiveStatusInactiveUser` — 401 for inactive user

**Unit tests:**
- `TestSetArchiveStatusNotFound` — not found returns 500
- `TestSetArchiveStatusAccessDenied` — access denied returns 401
- `TestSetArchiveStatusDBError` — DB error returns 500
- `TestSetArchiveStatusValidationError` — missing field returns 422

---

## POST /accounts/adjust-balance

**Auth**: Required (JWT)
**Handler**: `internal/handlers/accounts/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| accountId | int | Yes | |
| newBalance | decimal.Decimal | Yes | Target balance |
| notes | *string | No | Optional note for the adjustment transaction |

### Response

**Success**: HTTP 200

Transaction response object for the created adjustment transaction.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Invalid JSON | 422 | validation message |
| Account not found | 400 | "Account not found" |
| Access denied | 400 | "Access denied" |
| Balance unchanged | 400 | "Balance unchanged" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to adjust balance" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches account via `accountRepo.GetByID`, which filters `is_deleted = false`
- Checks ownership; returns 400 "Access denied" on mismatch
- Computes the difference between `newBalance` and the current balance; returns 400 "Balance unchanged" if difference is zero
- Creates an adjustment transaction with `label="Balance adjustment"`, `isAdjustment=true`, `excludeFromReports=true`
- Updates the account balance
- `balanceInBaseCurrency` is computed via currency conversion

### Known Gaps / TODOs

- Transaction creation and balance update are two separate DB operations without a wrapping transaction. If the balance update fails after the transaction record is created, the data will be inconsistent.

### Tests

**Integration tests:**
- `TestAdjustBalanceIncreaseSuccess` — 200, isAdjustment and isIncome verified
- `TestAdjustBalanceDecreaseSuccess` — 200, isIncome=false
- `TestAdjustBalanceSameBalanceFails` — 400 "Balance unchanged"
- `TestAdjustBalanceAccountNotFound` — 400 on unknown account
- `TestAdjustBalanceOtherUser` — 400 for another user's account
- `TestAdjustBalanceUnauthorized` — 401/422 without token
- `TestAdjustBalanceNegativeBalance` — 200 with negative target balance
- `TestAdjustBalanceInvalidJSON` — 422 on malformed JSON
- `TestAdjustBalanceInactiveUser` — 401 for inactive user

**Unit tests:**
- `TestAdjustBalanceUnchanged` — unchanged balance returns 400
- `TestAdjustBalanceDBError` — DB error returns 500
- `TestAdjustBalanceValidationError` — missing required field returns 422
- `TestAdjustBalanceNotFound` — account not found returns 400
- `TestAdjustBalanceAccessDenied` — access denied returns 400
