# Transactions Endpoints

Transaction management endpoints for creating, listing, retrieving, updating, and deleting transactions, as well as managing transaction templates.

## Table of Contents

- [POST /transactions/](#post-transactions)
- [GET /transactions/](#get-transactions)
- [GET /transactions/:id](#get-transactionsid)
- [PUT /transactions/](#put-transactions)
- [DELETE /transactions/:id](#delete-transactionsid)
- [GET /transactions/templates](#get-transactionstemplates)
- [PUT /transactions/templates](#put-transactionstemplates)
- [DELETE /transactions/templates](#delete-transactionstemplates)
- [DELETE /transactions/templates/validate](#delete-transactionstemplatesvalidate)

---

## POST /transactions/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| accountId | int | Yes | Must belong to authenticated user |
| targetAccountId | *int | No | Required for transfers |
| categoryId | *FlexInt | No | Accepts both string and int values |
| amount | decimal.Decimal | Yes | |
| targetAmount | *decimal.Decimal | No | Used for cross-currency transfers |
| label | string | No | Max 255 characters. Defaults to empty string. |
| notes | *string | No | Max 4000 characters (DTO-layer cap; DB column is unlimited). |
| dateTime | *time.Time | No | Defaults to `time.Now()` |
| isTransfer | bool | Yes | |
| isIncome | bool | Yes | |
| isAdjustment | bool | No | Defaults to false |
| excludeFromReports | bool | No | Defaults to false |
| isTemplate | bool | No | Defaults to false; if true, also creates a template record |

### Response

**Success**: HTTP 200

Transaction response object.

### Error Responses

| Condition | Status | Body |
|-----------|--------|------|
| Missing auth | 401 | `{"detail": "Missing authorization header"}` |
| User inactive or deleted | 401 | `{"detail": "User not activated"}` |
| Label longer than 255 chars | 422 | `{"detail": "Label must be at most 255 characters", "errorCode": "errors.transaction.labelTooLong", "params": {"max": 255}}` |
| Notes longer than 4000 chars | 422 | `{"detail": "Notes must be at most 4000 characters", "errorCode": "errors.transaction.notesTooLong", "params": {"max": 4000}}` |
| Other validation failure | 422 | `{"detail": "Validation failed", "errorCode": "errors.transaction.validationFailed"}` |
| Invalid account | 422 | `{"detail": "Invalid account"}` |
| Invalid category | 422 | `{"detail": "Invalid category"}` |
| Access denied | 403 | `{"detail": "Access denied"}` |
| Internal error | 500 | `{"detail": "Failed to create transaction"}` |

The 422 validation responses follow a stable contract: a human-readable `detail` (English fallback), a machine-readable `errorCode` (i18n key), and an optional `params` object the frontend interpolates into the localized message. The response NEVER carries Go struct field names, validator tag literals (e.g. `max`, `required`), or raw `validator.FieldError` strings — the mapping from validator failure to error code lives server-side in the handler.

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches account via `accountRepo.GetByID`, which filters `is_deleted = false`
- Validates account ownership; returns 403 "Access denied" on mismatch
- For transfers, validates `targetAccountId` and its ownership
- Validates category existence by ID; returns 422 "Invalid category" if not found (category type is not validated against transaction direction — income/expense type check is not enforced)
- `dateTime` defaults to `time.Now()` when not provided
- `categoryId` uses the `FlexInt` type, which accepts both string and numeric JSON values
- Calculates `baseAmount` via currency conversion
- Computes and stores the new running balance for the account
- Updates account balance after transaction creation
- If `isTemplate=true`, creates a corresponding template record. For transfers (`isTransfer=true`), the template's `target_account_id` is set from the transaction's `targetAccountId`.
- Transfer handling creates a linked transaction on the target account

### Known Gaps / TODOs

- Category type validation is not enforced (Go only checks existence; income/expense direction mismatch is allowed).

### Tests

**Integration tests:**
- `TestCreateExpenseTransaction` — 200, expense transaction
- `TestCreateIncomeTransaction` — 200, income transaction
- `TestCreateTransferTransaction` — 200, transfer between accounts
- `TestCreateTransferDifferentCurrencies` — 200, cross-currency transfer
- `TestCreateTransactionWithNotes` — 200, notes persisted
- `TestCreateTransactionWithDateTime` — 200, explicit dateTime used
- `TestCreateTransactionExcludeFromReports` — 200, flag set correctly
- `TestCreateTransactionWithTemplate` — 200, template record created alongside transaction
- `TestCreateTransferTransactionWithTemplate` — 200, transfer template created with `targetAccountId`
- `TestCreateTransactionWithoutCategory` — 200, null category accepted
- `TestCreateTransactionInvalidAccount` — 422 on invalid account
- `TestCreateTransactionOtherUserAccount` — 403 on another user's account
- `TestCreateTransactionMissingAccountId` — 422 without accountId
- `TestCreateTransactionUnauthorized` — 401 without token
- `TestCreateTransactionZeroAmount` — 200 with amount=0
- `TestCreateTransactionDecimalAmount` — 200 with decimal amount
- `TestCreateTransactionCategoryIdAsString` — 200, categoryId sent as string
- `TestCreateTransactionLabel255Boundary` — 200, exactly 255-char label persisted
- `TestCreateTransactionLabelTooLong` — 422 with `errorCode: "errors.transaction.labelTooLong"` for 256-char label, no row inserted, no validator internals leak
- `TestCreateTransactionNotesTooLong` — 422 with `errorCode: "errors.transaction.notesTooLong"` for 4001-char notes
- `TestTransactionsLabelColumnIsVarchar255` — migration smoke test: asserts `transactions.label` is `character varying(255)` after migration 00006

**Unit tests (10+ total):**
Various: invalid account, invalid category, access denied, DB error, bind/validate errors, FlexInt parsing edge cases.
- `TestCreateTransactionValidationLogEnriched` — validation-error log line carries `user_id`, `errorCode`, `handler`, `path`, and the failed JSON field name; response body is free of validator internals
- `TestUpdateTransactionUnknownErrorLogsLengths` — unknown-DB-error log carries `label_len` and `notes_len` (no raw payload)
- `TestMapValidationError*` — direct unit coverage of the (field, tag) -> error code mapping

---

## GET /transactions/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| page | int | No | Default 1 |
| per_page | int | No | Default 30, capped at 50 |
| types | string | No | Comma-separated; only the first value is used |
| categories | string | No | Comma-separated integer IDs |
| accounts | string | No | Comma-separated integer IDs |
| from_date | string | No | Format YYYY-MM-DD |
| to_date | string | No | Format YYYY-MM-DD |

### Response

**Success**: HTTP 200

JSON array of transaction objects.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Internal error | 500 | "Failed to get transactions" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Filters by `user_id` (derived from JWT) and `is_deleted = false`
- Sorted by `date_time DESC`
- Pagination uses offset/limit based on `page` and `per_page`
- Account, category, date range filters are applied when provided
- Unknown query parameters are silently ignored
- `balanceInBaseCurrency` is computed per transaction via currency conversion

### Known Gaps / TODOs

- The `types` filter only uses the first value in a comma-separated list; multiple transaction types cannot be filtered simultaneously.
- The `currencies` filter parameter (supported in the original) is not implemented.

### Tests

**Integration tests:**
- `TestGetTransactionsList` — 200, list returned
- `TestGetTransactionsEmpty` — 200, empty array for user with no transactions
- `TestGetTransactionsPagination` — page and per_page parameters applied
- `TestGetTransactionsFilterByAccount` — account filter applied correctly
- `TestGetTransactionsFilterByCategory` — category filter applied correctly
- `TestGetTransactionsFilterByDateRange` — from_date and to_date filter applied
- `TestGetTransactionsFilterByType` — type filter applied
- `TestGetTransactionsFilterByMultipleAccounts` — multiple account IDs applied
- `TestGetTransactionsUnauthorized` — 401 without token

**Unit tests:**
- `TestGetTransactionsDBError` — DB error returns 500

---

## GET /transactions/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

`id` is a path parameter (integer). No body, no query parameters.

### Response

**Success**: HTTP 200

Single transaction object.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Non-numeric ID | 400 | "Invalid transaction ID" |
| Transaction not found | 404 | "Transaction not found" |
| Other user's transaction | 403 | "Access denied" |
| Internal error | 500 | "Failed to get transaction" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Parses transaction ID with `strconv.Atoi`; returns 400 "Invalid transaction ID" on non-numeric input
- Fetches transaction via repository, which filters `is_deleted = false`; soft-deleted transactions return 404
- Checks ownership; returns 403 "Access denied" on mismatch
- Computes `balanceInBaseCurrency` via currency conversion

### Tests

**Integration tests:**
- `TestGetTransactionDetails` — 200 with correct fields
- `TestGetTransactionDetailsWithAccount` — 200 with account relation
- `TestGetTransactionDetailsNotFound` — 404 on unknown transaction
- `TestGetTransactionDetailsOtherUser` — 403 for another user's transaction
- `TestGetTransactionDetailsInvalidID` — 400 on non-numeric ID
- `TestGetTransactionDetailsUnauthorized` — 401 without token

**Unit tests:**
- `TestGetTransactionDetailsNotFound` — not found returns 404
- `TestGetTransactionDetailsAccessDenied` — access denied returns 403
- `TestGetTransactionDetailsDBError` — DB error returns 500
- `TestGetTransactionDetailsInvalidID` — non-numeric ID returns 400

---

## PUT /transactions/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

Same fields as create (including `label` max=255 and `notes` max=4000), plus:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| id | int | Yes | ID of the transaction to update |

### Response

**Success**: HTTP 200

Updated transaction object.

### Error Responses

| Condition | Status | Body |
|-----------|--------|------|
| Missing auth | 401 | `{"detail": "Missing authorization header"}` |
| User inactive or deleted | 401 | `{"detail": "User not activated"}` |
| Label longer than 255 chars | 422 | `{"detail": "Label must be at most 255 characters", "errorCode": "errors.transaction.labelTooLong", "params": {"max": 255}}` |
| Notes longer than 4000 chars | 422 | `{"detail": "Notes must be at most 4000 characters", "errorCode": "errors.transaction.notesTooLong", "params": {"max": 4000}}` |
| Other validation failure (incl. missing `id`) | 422 | `{"detail": "Validation failed", "errorCode": "errors.transaction.validationFailed"}` |
| Transaction not found | 404 | `{"detail": "Transaction not found"}` |
| Invalid account | 422 | `{"detail": "Invalid account"}` |
| Access denied | 403 | `{"detail": "Access denied"}` |
| Internal error | 500 | `{"detail": "Failed to update transaction"}` |

Same 422 contract notes as POST: stable `errorCode` + optional `params`, no Go struct names or validator tags ever leak.

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches existing transaction via `GetByID`, which filters `is_deleted = false`
- Checks ownership; returns 403 on mismatch
- Validates new account via `accountRepo.GetByID`
- Reverses the previous transaction effect and applies the new values (full balance recalculation via `recalculateBalances`)
- `account_id` is included in the SQL UPDATE, so moving a transaction to a different account is correctly persisted

### Transfer Update Handling

When updating a transfer transaction, the service handles all scenarios:
- **Field changes** (amount, label, notes, datetime, category, excludeFromReports) are mirrored to the linked transaction
- **Target account change** updates the linked transaction's account and recalculates balances for old and new target accounts
- **Cross-currency transfers** use `targetAmount` for the linked transaction when provided
- **Transfer → regular** soft-deletes the linked transaction and recalculates the target account balance
- **Regular → transfer** creates a new linked transaction with proper cross-linking
- **Self-transfer** (source == target) is rejected with HTTP 422

### Known Gaps / TODOs

- Category type validation is not enforced (income/expense direction mismatch is allowed).

### Tests

**Integration tests:**
- `TestUpdateTransaction` — 200, fields updated
- `TestUpdateTransactionChangeCategory` — 200, category changed
- `TestUpdateTransactionAddNotes` — 200, notes added
- `TestUpdateTransactionNotFound` — 404 on unknown transaction
- `TestUpdateTransactionOtherUser` — 403 for another user's transaction
- `TestUpdateTransactionUnauthorized` — 401 without token
- `TestUpdateTransactionMissingID` — 422 when id field is absent
- `TestUpdateTransferAmountUpdatesBothBalances` — 200, linked tx amount changes, both balances correct
- `TestUpdateTransferTargetAccountRecalculatesBalances` — 200, old target restored, new target credited
- `TestConvertTransferToRegularTransaction` — 200, linked tx soft-deleted, target balance restored
- `TestConvertRegularToTransfer` — 200, linked tx created, target balance credited
- `TestUpdateTransferLabelNotesDateTime` — 200, linked tx mirrors label/notes/datetime/category/excludeFromReports
- `TestCrossCurrencyTransferUpdateWithTargetAmount` — 200, linked tx uses targetAmount
- `TestSelfTransferRejected` — 422, source == target rejected

**Unit tests (8 total):**
Various: not found, invalid account, access denied, DB error, bind/validate errors, self-transfer, invalid target account, target access denied.

---

## DELETE /transactions/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

`id` is a path parameter (integer). No body.

### Response

**Success**: HTTP 200

Transaction object as it existed before deletion.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Non-numeric ID | 400 | "Invalid transaction ID" |
| Transaction not found | 404 | "Transaction not found" |
| Access denied | 403 | "Access denied" |
| Internal error | 500 | "Failed to delete transaction" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches transaction via `GetByID`, which filters `is_deleted = false`; already-deleted transactions return 404
- Checks ownership; returns 403 "Access denied" on mismatch
- Soft-deletes transaction by setting `is_deleted = true`
- For transfers, also soft-deletes the linked transaction
- Recalculates running balances via `recalculateBalances`

### Known Gaps / TODOs

None.

### Tests

**Integration tests:**
- `TestDeleteTransaction` — 200, subsequent GET returns 404
- `TestDeleteTransactionNotFound` — 404 on unknown transaction
- `TestDeleteTransactionOtherUser` — 403 for another user's transaction
- `TestDeleteTransactionUnauthorized` — 401 without token
- `TestDeleteTransactionInvalidID` — 400 on non-numeric ID

**Unit tests (3 total):**
Various: not found, access denied, DB error.

---

## GET /transactions/templates

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

No request body, no query parameters.

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "categoryId": 3,
    "targetAccountId": null,
    "targetAccountName": null,
    "label": "Monthly rent",
    "category": {"id": 3, "name": "Housing"}
  },
  {
    "id": 2,
    "categoryId": null,
    "targetAccountId": 5,
    "targetAccountName": "Savings Account",
    "label": "Transfer to savings"
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Internal error | 500 | "Failed to get templates" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Filters templates by `user_id` from JWT
- Results ordered by `label`
- Category relation loaded via LEFT JOIN with `user_categories`
- Target account relation loaded via LEFT JOIN with `accounts` on `target_account_id`
- Templates with `targetAccountId` set are "transfer templates"; those without are regular templates

### Tests

**Integration tests:**
- `TestGetTemplates` — 200, template with category in response
- `TestGetTemplatesEmpty` — 200, empty array for user with no templates
- `TestGetTemplatesUnauthorized` — 401 without token
- `TestGetTemplatesWithTargetAccount` — 200, transfer template with `targetAccountId` and `targetAccountName`

**Unit tests:**
- `TestGetTemplatesDBError` — DB error returns 500
- `TestGetTemplatesWithCategory` — category nested in response

---

## PUT /transactions/templates

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| id | int | Yes | Template ID to update |
| label | string | Yes | |
| categoryId | *int | No | Can be set to null |
| targetAccountId | *int | No | Account ID for transfer templates. Can be set to null. |

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "categoryId": 3,
  "targetAccountId": null,
  "targetAccountName": null,
  "label": "Updated label",
  "category": {"id": 3, "name": "Housing"}
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Template not found | 404 | "Template not found" |
| Access denied | 403 | "Access denied" |
| Invalid user | 400 | "Invalid user" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to update template" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Fetches template and verifies `UserID` ownership; returns 403 "Access denied" on mismatch
- Updates `label`, `categoryId`, and `targetAccountId`; both `categoryId` and `targetAccountId` can be set to null
- Category existence is not validated (relies on FK constraint)
- Target account existence is not validated (relies on FK constraint)
- After update, re-fetches the template to populate the `TargetAccount` relation

### Tests

**Integration tests:**
- `TestUpdateTemplate` — 200, label updated successfully
- `TestUpdateTemplateNotFound` — 404 on unknown template
- `TestUpdateTemplateUnauthorized` — 401 without token
- `TestUpdateTemplateInactiveUser` — 401 for inactive user
- `TestUpdateTemplateWithTargetAccount` — 200, targetAccountId set and targetAccountName populated
- `TestUpdateTemplateRemoveTargetAccount` — 200, targetAccountId removed (set to null)

**Unit tests:**
- `TestUpdateTemplateUserNotActivated` — inactive user returns 401
- `TestUpdateTemplateInvalidUser` — invalid user returns 400
- `TestUpdateTemplateNotFound` — not found returns 404
- `TestUpdateTemplateAccessDenied` — access denied returns 403
- `TestUpdateTemplateDBError` — DB error returns 500
- `TestUpdateTemplateBindError` — invalid JSON returns 422
- `TestUpdateTemplateValidateError` — missing required field returns 422

---

## DELETE /transactions/templates

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| ids | string (query) | Yes | Comma-separated integer IDs |

### Response

**Success**: HTTP 200

Array of REMAINING templates (not the deleted ones).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| Missing ids param | 422 | "ids parameter required" |
| Invalid ids format | 422 | "Invalid IDs" |
| User not activated | 401 | "User not activated" |
| Internal error | 500 | "Failed to delete templates" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Parses `ids` query param manually; invalid (non-integer) values in the comma-separated list are silently skipped
- For each valid ID: fetches the template and checks `UserID` ownership; templates not belonging to the user are silently skipped
- Non-existent template IDs are silently skipped
- Performs a hard delete (not soft delete)
- Returns the remaining templates for the user after deletion

### Tests

**Integration tests:**
- `TestDeleteTemplates` — 200, batch delete, remaining templates returned
- `TestDeleteTemplatesSingle` — 200, single template deleted
- `TestDeleteTemplatesMissingIds` — 422 when ids param is absent
- `TestDeleteTemplatesInvalidIds` — 422 on non-integer ids
- `TestDeleteTemplatesUnauthorized` — 401 without token
- `TestDeleteTemplatesInactiveUser` — 401 for inactive user

**Unit tests:**
- `TestDeleteTemplatesUserNotActivated` — inactive user returns 401
- `TestDeleteTemplatesInvalidUserUnit` — invalid user returns 400
- `TestDeleteTemplatesDBError` — DB error returns 500
- `TestDeleteTemplatesInvalidIDs` — invalid IDs return 422
- `TestDeleteTemplatesMissingIDs` — missing ids param returns 422

---

## DELETE /transactions/templates/validate

**Auth**: Required (JWT)
**Handler**: `internal/handlers/transactions/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| ids | string (query) | Yes | Comma-separated integer IDs |

### Response

**Success**: HTTP 200

```json
[1, 2, 3]
```

Parsed integer array of valid IDs.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth | 401 | "Missing authorization header" |
| User inactive or deleted | 401 | "User not activated" |
| Invalid or empty ids | 400 | "Invalid template IDs format" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not soft-deleted; returns 401 "User not activated" on failure
- Parses `ids` query parameter; IDs must be positive integers
- Returns the parsed list without any DB access
- Any non-positive or non-integer value causes a 400 response

### Known Gaps / TODOs

- This endpoint has 0 tests in Go (no integration tests, no unit tests). Coverage is 0%.

### Tests

No Go tests exist for this endpoint yet. Tests to add: success case, invalid IDs, empty IDs, unauthorized access.
