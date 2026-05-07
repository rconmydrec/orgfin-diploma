# Planned Transactions Endpoints

CRUD and query endpoints for managing planned (scheduled) transactions, including one-time and recurring entries with recurrence rules.

## Table of Contents

- [POST /planned-transactions/](#post-planned-transactions)
- [GET /planned-transactions/](#get-planned-transactions)
- [GET /planned-transactions/upcoming/occurrences](#get-planned-transactionsupcomingoccurrences)
- [GET /planned-transactions/:id](#get-planned-transactionsid)
- [PUT /planned-transactions/:id](#put-planned-transactionsid)
- [DELETE /planned-transactions/:id](#delete-planned-transactionsid)

---

## POST /planned-transactions/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| amount | decimal.Decimal | Yes | |
| label | string | No | Defaults to empty string |
| notes | *string | No | |
| accountId | *int | No | JSON tag: `accountId` |
| isIncome | bool | Yes | JSON tag: `isIncome` |
| plannedDate | FlexDate | Yes | JSON tag: `plannedDate`; accepts RFC3339, `2006-01-02T15:04:05`, `2006-01-02T15:04`, or `2006-01-02` formats. Type is `common.FlexDate` from `internal/handlers/common/date.go`. |
| isRecurring | bool | No | JSON tag: `isRecurring`; defaults to false |
| recurrenceRule | *RecurrenceRuleDTO | No | Required when isRecurring=true (not enforced by Go) |

**RecurrenceRuleDTO fields**:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| frequency | string | Yes | Any string accepted (daily/weekly/monthly/yearly) |
| interval | int | No | No minimum validation |
| endDate | *time.Time | No | |
| count | *int | No | No minimum validation |
| dayOfMonth | *int | No | No range validation |

### Response

**Success**: HTTP 201

```json
{
  "id": 1,
  "userId": 42,
  "currencyId": 5,
  "amount": 250.00,
  "label": "Rent",
  "notes": null,
  "isIncome": false,
  "plannedDate": "2026-03-01T00:00:00Z",
  "isRecurring": true,
  "recurrenceRule": {
    "frequency": "monthly",
    "interval": 1,
    "endDate": null,
    "count": null,
    "dayOfMonth": 1
  },
  "isExecuted": false,
  "executedTransactionId": null,
  "executionDate": null,
  "isActive": true,
  "createdAt": "2026-02-22T10:00:00Z",
  "updatedAt": "2026-02-22T10:00:00Z"
}
```

Notes:
- `amount` is a JSON number (float64), not a string. Internally uses `decimal.Decimal` but is serialized via `InexactFloat64()`.
- `isActive` is explicitly set to `true` on creation.
- `currencyId` is always derived from `user.BaseCurrencyID` server-side. The client does not send a currency ID. If the user's `BaseCurrencyID` is 0 (not set), the endpoint returns 400 with "Base currency not set".

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| User's base currency not set | 400 | "Base currency not set" |
| Invalid JSON body | 422 | "Invalid request data" |
| Validation failure | 422 | "Invalid request data" |
| Internal DB error | 500 | "Failed to create planned transaction" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted. The handler reads `BaseCurrencyID` from the middleware context primitives via `common.ActiveUserBaseCurrencyID(c)`.
- The handler converts the request DTO to `CreateParams` (using `dtoToRuleParams` for recurrence rule conversion), then calls `service.Create(userID, baseCurrencyID, params)`.
- The service validates that `baseCurrencyID != 0`; returns `ErrBaseCurrencyNotSet` (mapped to 400 by the handler) if not set.
- The service sets `currencyId = baseCurrencyID` and `is_active = true` on the new record.
- The service converts `RecurrenceRuleParams` to a JSON string with snake_case field names via `ruleToStorageJSON` (uses `models.RecurrenceRule` struct for marshaling).
- The service calls `plannedTxRepo.Create()` to persist the record.
- Cross-field validation (e.g., requiring `recurrenceRule` when `isRecurring = true`, or the mutual exclusivity of `endDate` and `count`) is not enforced.
- `accountId` is stored without validating that the account belongs to the user.
- `frequency`, `interval`, `count`, and `dayOfMonth` in the recurrence rule have no value constraints.

### Known Gaps / TODOs

- No validation that `recurrenceRule` is provided when `isRecurring = true`.
- No validation that `endDate` and `count` in `recurrenceRule` are mutually exclusive.
- `accountId` ownership is not validated against the authenticated user.
- `frequency` is not validated against an enum.
- `interval`, `count`, and `dayOfMonth` have no range validation.

### Tests

Integration tests:
- `TestCreatePlannedTxSuccess` -- 201, basic creation succeeds
- `TestCreatePlannedTxWithRecurrenceRule` -- 201, weekly recurrence rule stored
- `TestCreatePlannedTxIncome` -- 201, income type with monthly recurrence and dayOfMonth
- `TestCreatePlannedTxWithNotes` -- 201, notes field stored correctly
- `TestCreatePlannedTxUnauthorized` -- 401 without auth token
- `TestCreatePlannedTxInvalidJSON` -- 422 for malformed JSON body

Unit tests:
- `TestCreateBindError` -- 422 for invalid JSON
- `TestCreateValidateError` -- 422 for empty body
- `TestCreateDBError` -- 500 on DB create failure
- `TestCreateSuccess` -- 201 success path
- `TestCreateBaseCurrencyNotSet` -- 400 when user's base currency is 0
- `TestCreateWithRecurrenceRule` -- 201 with recurrence rule object
- `TestCreateUserNotActive` -- 401 when user is inactive
- `TestCreateUserRepoError` -- 401 when user repo returns error
- `TestCreateUserDeleted` -- 401 when user is soft-deleted
- `TestCreateWithBareDate` -- 201 with bare date format (YYYY-MM-DD)
- `TestCreateRecurrenceRuleStoredAsSnakeCase` -- recurrence rule stored with snake_case keys

---

## GET /planned-transactions/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

Query parameters (all optional):

| Parameter | Type | Notes |
|-----------|------|-------|
| account_ids | string | Comma-separated list of account IDs |
| from_date | string | Start date filter |
| to_date | string | End date filter |
| is_recurring | string | Truthy string to filter recurring |
| is_executed | string | Truthy string to filter executed |
| is_active | string | Truthy string to filter active |
| include_inactive | string | Pass "true" to include inactive records |

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "userId": 42,
    "currencyId": 5,
    "amount": 250.00,
    "label": "Rent",
    "notes": null,
    "isIncome": false,
    "plannedDate": "2026-03-01T00:00:00Z",
    "isRecurring": true,
    "recurrenceRule": { "frequency": "monthly", "interval": 1 },
    "isExecuted": false,
    "executedTransactionId": null,
    "executionDate": null,
    "isActive": true,
    "createdAt": "2026-02-22T10:00:00Z",
    "updatedAt": "2026-02-22T10:00:00Z"
  }
]
```

Notes:
- `amount` is a JSON number (float64), not a string.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Internal DB error | 500 | "Failed to get planned transactions" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler parses `account_ids` by splitting the query string on commas, assembles `ListFilters`, then calls `service.List(userID, filters)`.
- Filter values for `is_recurring`, `is_executed`, and `is_active` are passed as raw strings; truthy evaluation is deferred to the repository layer.
- The service converts `ListFilters` to `repositories.PlannedTxFilters` and calls `plannedTxRepo.GetByUserID()`.
- Results are mapped to response DTOs via `toResponse()` in the handler.

### Tests

Integration tests:
- `TestListPlannedTxSuccess` -- 200, list returned for authenticated user
- `TestListPlannedTxEmpty` -- 200, empty list for user with no records
- `TestListPlannedTxUnauthorized` -- 401 without auth token
- `TestListPlannedTxWithRecurring` -- 200 with is_recurring filter
- `TestListPlannedTxIncome` -- 200 filtering by income type
- `TestListMultiplePlannedTx` -- 200, multiple records returned
- `TestListPlannedTxInvalidToken` -- 401 with invalid token

Unit tests:
- `TestListDBError` -- 500 when repo returns error
- `TestListSuccess` -- 200 success path
- `TestListUserNotActive` -- 401 when user is inactive

---

## GET /planned-transactions/upcoming/occurrences

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

Query parameters:

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| days | string | No | Parsed to int; defaults to 30 if absent or unparseable |
| include_inactive | string | No | Pass "true" to include inactive records |

### Response

**Success**: HTTP 200

```json
[
  {
    "plannedTransactionId": 1,
    "occurrenceDate": "2026-03-01T00:00:00Z",
    "amount": 250.00,
    "isIncome": false,
    "label": "Rent",
    "isActive": true,
    "isRecurring": true
  }
]
```

Notes:
- `amount` is a JSON number (float64), not a string. The handler maps repository results to `OccurrenceResponse` DTOs, converting `decimal.Decimal` to float64 via `InexactFloat64()`.
- Recurring transactions are expanded into individual occurrences by the repository using `generateOccurrences`. A single recurring planned transaction may produce multiple entries in the response.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Internal DB error | 500 | "Failed to get upcoming occurrences" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler parses `days` from query string (silently defaults to 30 on parse failure) and `includeInactive`, then calls `service.GetUpcoming(userID, days, includeInactive)`.
- The service delegates to `plannedTxRepo.GetUpcomingOccurrences()`.
- The repository fetches all non-deleted, non-executed planned transactions for the user, then calls `generateOccurrences` for each one to expand recurring transactions into individual occurrences within the `[today, today + days]` range.
- `generateOccurrences` handles recurrence by frequency (daily, weekly, monthly, yearly), respects `interval`, `count`, `endDate`, and `dayOfMonth` from the recurrence rule.
- Results are sorted by occurrence date. The handler maps repository results to `OccurrenceResponse` DTOs, converting `decimal.Decimal` to float64 via `InexactFloat64()`.

### Tests

Integration tests:
- `TestGetUpcomingSuccess` -- 200, occurrences returned within default 30 days
- `TestGetUpcomingWithDays` -- 200 with custom days parameter
- `TestGetUpcomingUnauthorized` -- 401 without auth token
- `TestGetUpcomingWithInvalidDays` -- 200, falls back to default 30 days
- `TestGetUpcomingInvalidToken` -- 401 with invalid token

Unit tests:
- `TestGetUpcomingDBError` -- 500 when repo returns error
- `TestGetUpcomingSuccess` -- 200 success path
- `TestGetUpcomingUserNotActive` -- 401 when user is inactive

---

## GET /planned-transactions/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

Path parameter: `id` (int) -- planned transaction ID.

No request body.

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "userId": 42,
  "currencyId": 5,
  "amount": 250.00,
  "label": "Rent",
  "notes": null,
  "isIncome": false,
  "plannedDate": "2026-03-01T00:00:00Z",
  "isRecurring": true,
  "recurrenceRule": { "frequency": "monthly", "interval": 1 },
  "isExecuted": false,
  "executedTransactionId": null,
  "executionDate": null,
  "isActive": true,
  "createdAt": "2026-02-22T10:00:00Z",
  "updatedAt": "2026-02-22T10:00:00Z"
}
```

Notes:
- `amount` is a JSON number (float64), not a string.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid ID" |
| Not found or DB error | 404 | "Planned transaction not found" |
| Belongs to another user | 403 | "Access denied" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler parses `id` from the path using `strconv.Atoi`; returns 400 on failure.
- The handler calls `service.GetByID(userID, id)`.
- The service calls `plannedTxRepo.GetByID()` and checks ownership (`tx.UserID != userID`). Returns `ErrNotFound` for repo errors or `ErrAccessDenied` for ownership mismatch.
- The handler maps service errors to HTTP responses via `handleServiceError`.

### Known Gaps / TODOs

- Any repository error (including genuine DB errors) is returned as 404 rather than 500, because the service maps all repo errors to `ErrNotFound`.

### Tests

Integration tests:
- `TestGetByIDSuccess` -- 200, correct record returned
- `TestGetByIDNotFound` -- 404 for nonexistent ID
- `TestGetByIDOtherUser` -- 403 for another user's record
- `TestGetByIDUnauthorized` -- 401 without auth token
- `TestGetByIDInvalidID` -- 400 for non-numeric path ID

Unit tests:
- `TestGetByIDInvalidID` -- 400 for non-numeric path param
- `TestGetByIDNotFound` -- 404 when repo returns error
- `TestGetByIDAccessDenied` -- 403 for different user
- `TestGetByIDSuccess` -- 200 success path
- `TestGetByIDUserNotActive` -- 401 when user is inactive

---

## PUT /planned-transactions/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

Path parameter: `id` (int) -- planned transaction ID.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| amount | decimal.Decimal | Yes | `validate:"required"` |
| label | string | No | |
| notes | *string | No | |
| accountId | *int | No | JSON tag: `accountId` |
| isIncome | bool | No | JSON tag: `isIncome` |
| plannedDate | FlexDate | Yes | JSON tag: `plannedDate`; `validate:"required"`. Type is `common.FlexDate` from `internal/handlers/common/date.go`. |
| isRecurring | bool | No | JSON tag: `isRecurring` |
| recurrenceRule | *RecurrenceRuleDTO | No | |
| isActive | *bool | No | Only updated when non-nil |

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "userId": 42,
  "currencyId": 5,
  "amount": 300.00,
  "label": "Updated Rent",
  "notes": null,
  "isIncome": false,
  "plannedDate": "2026-04-01T00:00:00Z",
  "isRecurring": false,
  "recurrenceRule": null,
  "isExecuted": false,
  "executedTransactionId": null,
  "executionDate": null,
  "isActive": true,
  "createdAt": "2026-02-22T10:00:00Z",
  "updatedAt": "2026-02-22T12:00:00Z"
}
```

Notes:
- `amount` is a JSON number (float64), not a string.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid ID" |
| Invalid JSON body | 422 | "Invalid request data" |
| Not found | 404 | "Planned transaction not found" |
| Belongs to another user | 403 | "Access denied" |
| Internal DB error | 500 | "Failed to update planned transaction" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler parses `id` from the path using `strconv.Atoi`; returns 400 on failure.
- The handler converts the request DTO to `UpdateParams` (using `dtoToRuleParams`), then calls `service.Update(userID, id, params)`.
- The service fetches the existing record (returns `ErrNotFound` if not found), checks ownership (`existing.UserID != userID`, returns `ErrAccessDenied` if mismatched), then applies all fields to the model struct.
- `isActive` is only updated when `params.IsActive` is non-nil (pointer check).
- `recurrenceRule` is converted to a JSON string with snake_case field names via `ruleToStorageJSON` in the service.
- The service calls `plannedTxRepo.Update()` to persist changes.
- `currencyId` is preserved from the existing record (not changed during update).
- No validation that `recurrenceRule` is consistent with `isRecurring`.
- `accountId` ownership is not validated.

### Known Gaps / TODOs

- No validation that `recurrenceRule` is present when `isRecurring = true`.
- `accountId` ownership is not validated against the authenticated user.

### Tests

Integration tests:
- `TestUpdatePlannedTxSuccess` -- 200, fields updated correctly
- `TestUpdatePlannedTxWithRecurrenceRule` -- 200, recurrence rule updated
- `TestUpdatePlannedTxChangeIsActive` -- 200, isActive toggled via pointer field
- `TestUpdatePlannedTxNotFound` -- 404 for nonexistent record
- `TestUpdatePlannedTxUnauthorized` -- 401 without auth token
- `TestUpdatePlannedTxInvalidID` -- 400 for non-numeric path ID
- `TestUpdatePlannedTxInvalidJSON` -- 422 for malformed JSON
- `TestUpdatePlannedTxOtherUser` -- 403 for another user's record

Unit tests:
- `TestUpdateInvalidID` -- 400 for non-numeric path param
- `TestUpdateBindError` -- 422 for invalid JSON
- `TestUpdateNotFound` -- 404 when record not found
- `TestUpdateAccessDenied` -- 403 for different user
- `TestUpdateDBError` -- 500 on DB update failure
- `TestUpdateSuccess` -- 200 success path
- `TestUpdateWithRecurrenceRule` -- 200 with recurrence rule marshaling
- `TestUpdateUserNotActive` -- 401 when user is inactive
- `TestUpdatePreservesCurrencyID` -- currency ID preserved during update
- `TestUpdateWithBareDate` -- 200 with bare date format
- `TestUpdateRecurrenceRuleStoredAsSnakeCase` -- recurrence rule stored with snake_case keys

---

## DELETE /planned-transactions/:id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/planned_transactions/handler.go`
**Service**: `internal/services/planned_transactions/service.go`

### Request

Path parameter: `id` (int) -- planned transaction ID.

No request body.

### Response

**Success**: HTTP 200

```json
{
  "deleted": true
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid ID" |
| Not found | 404 | "Planned transaction not found" |
| Belongs to another user | 403 | "Access denied" |
| Internal DB error | 500 | "Failed to delete planned transaction" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted before the handler runs.
- The handler parses `id` from the path using `strconv.Atoi`; returns 400 on failure.
- The handler calls `service.Delete(userID, id)`.
- The service fetches the existing record (returns `ErrNotFound` if not found), checks ownership (`existing.UserID != userID`, returns `ErrAccessDenied` if mismatched), then calls `plannedTxRepo.Delete()` which performs a soft delete (sets `is_deleted = true` and `is_active = false`).

### Tests

Integration tests:
- `TestDeletePlannedTxSuccess` -- 200, record deleted
- `TestDeletePlannedTxNotFound` -- 404 for nonexistent record
- `TestDeletePlannedTxUnauthorized` -- 401 without auth token
- `TestDeletePlannedTxOtherUser` -- 403 for another user's record
- `TestDeletePlannedTxInvalidID` -- 400 for non-numeric path ID

Unit tests:
- `TestDeleteInvalidID` -- 400 for non-numeric path param
- `TestDeleteNotFound` -- 404 when record not found
- `TestDeleteAccessDenied` -- 403 for different user
- `TestDeleteDBError` -- 500 on DB delete failure
- `TestDeleteSuccess` -- 200 success path
- `TestDeleteUserNotActive` -- 401 when user is inactive

---

## Shared Components

### FlexDate Type

`common.FlexDate` (defined in `internal/handlers/common/date.go`) wraps `time.Time` and handles multiple date formats during JSON unmarshalling. It delegates parsing to `common.ParseDate`, which tries formats in this order:
- Bare date (`2006-01-02`)
- Datetime without timezone (`2006-01-02T15:04:05`)
- Datetime without seconds (`2006-01-02T15:04`)
- RFC3339 (`2006-01-02T15:04:05Z07:00`)

Used in `CreateRequest.PlannedDate` and `UpdateRequest.PlannedDate`.

### Recurrence Rule Storage

Recurrence rules are stored in the database as JSON strings with snake_case field names (via `models.RecurrenceRule`). The conversion happens in two stages:
- **Handler**: `dtoToRuleParams` converts the camelCase API DTO (`RecurrenceRuleDTO`) to domain `RecurrenceRuleParams`; `ruleParamsToDTO` converts back for responses.
- **Service**: `ruleToStorageJSON` converts `RecurrenceRuleParams` to a snake_case JSON string for DB storage; `StorageRuleToDTO` converts a DB JSON string back to `RecurrenceRuleParams`.

### Additional Unit Tests

- `TestToResponseWithRecurrenceRule` -- recurrence rule included in response
- `TestToResponseWithRecurrenceRuleEndDate` -- recurrence rule with endDate
- `TestRegisterRoutes` -- verifies route registration
- `TestFlexDateUnmarshalRFC3339` -- RFC3339 format parsing (in `internal/handlers/common/date_test.go`)
- `TestFlexDateUnmarshalDatetime` -- datetime format parsing (in `internal/handlers/common/date_test.go`)
- `TestFlexDateUnmarshalBareDate` -- bare date format parsing (in `internal/handlers/common/date_test.go`)
- `TestFlexDateUnmarshalInvalid` -- invalid date format error (in `internal/handlers/common/date_test.go`)
- `TestFlexDateUnmarshalBadJSON` -- non-string JSON error (in `internal/handlers/common/date_test.go`)
- `TestFlexDateMarshalJSON` -- FlexDate marshaling (in `internal/handlers/common/date_test.go`)
- `TestDtoToStorageRuleNil` -- nil rule handling
- `TestDtoToStorageRuleBasic` -- basic rule conversion
- `TestDtoToStorageRuleWithEndDate` -- rule with endDate
- `TestStorageRuleToDTOBasic` -- basic storage-to-DTO conversion
- `TestStorageRuleToDTOWithEndDate` -- storage-to-DTO with endDate
- `TestStorageRuleToDTOWithBareDateEndDate` -- bare date endDate parsing
- `TestStorageRuleToDTOInvalidJSON` -- invalid JSON handling
- `TestStorageRuleToDTOEmptyEndDate` -- empty endDate string
- `TestFullRoundTripRecurrenceRule` -- full round-trip conversion test
