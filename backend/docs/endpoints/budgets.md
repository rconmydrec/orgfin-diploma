# Budgets Endpoints

Budget management endpoints for creating, listing, updating, deleting, archiving budgets, and running daily processing to roll over expired repeating budgets.

The handler (`internal/handlers/budgets/handler.go`) is a thin HTTP adapter. All business logic (currency validation, date parsing, period normalization, ownership checks) is delegated to the budget service (`internal/services/budgets/service.go`). The handler maps service sentinel errors to appropriate HTTP status codes via `handleServiceError`.

## Table of Contents

- [POST /budgets/add/](#post-budgetsadd)
- [GET /budgets/](#get-budgets)
- [PUT /budgets/:id/](#put-budgetsid)
- [DELETE /budgets/:id/](#delete-budgetsid)
- [PUT /budgets/:id/archive/](#put-budgetsidarchive)
- [GET /budgets/daily-processing](#get-budgetsdaily-processing)

---

## POST /budgets/add/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`Create`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | Yes | Budget name |
| currencyId | int | Yes | Currency ID |
| targetAmount | decimal.Decimal | Yes | Target amount |
| period | string | Yes | Any string accepted; normalized to uppercase in service via `strings.ToUpper` |
| repeat | bool | Yes | Whether to repeat after expiry |
| startDate | string | Yes | Date string; parsed by service via `dateutil.ParseDate` (accepts `YYYY-MM-DD`, `YYYY-MM-DDThh:mm:ss`, `YYYY-MM-DDThh:mm`, RFC3339) |
| endDate | string | Yes | Date string; parsed by service via `dateutil.ParseDate`; 1 day is added before storage |
| categories | []int | No | Category IDs to include |
| comment | *string | No | Optional comment |

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "name": "Monthly Groceries",
  "currencyId": 1,
  "targetAmount": 500,
  "collectedAmount": 123.45,
  "period": "monthly",
  "repeat": true,
  "startDate": "2026-01-01T00:00:00Z",
  "endDate": "2026-01-31T00:00:00Z",
  "includedCategories": "3,7",
  "isArchived": false,
  "comment": null,
  "currency": {
    "id": 1,
    "code": "USD",
    "name": "US Dollar"
  }
}
```

Notes:
- `targetAmount` and `collectedAmount` are serialized as JSON numbers (float64), not strings.
- `period` is always returned in lowercase (e.g., `"monthly"`, `"daily"`), regardless of the case sent in the request.
- `endDate` is returned as stored minus 1 day (reversing the +1 day added on creation).
- `currency` is included when available (`omitempty`).
- `collectedAmount` is recalculated from existing matching transactions immediately after creation; the value in the example is illustrative and depends on the user's pre-existing data. It will be `0` if no matching transactions exist within the budget's date window.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing required fields | 422 | "Validation failed" |
| Invalid start date format | 400 | "Invalid start date format" |
| Invalid end date format | 400 | "Invalid end date format" |
| Invalid currency ID | 400 | "Invalid currency" |
| Internal DB error | 500 | "Failed to create budget" |

### Business Logic

- `RequireActiveUser` middleware verifies user `is_active = true` and `is_deleted = false`.
- Handler binds and validates the request, then delegates to `service.Create(userID, params)`.
- Service parses `startDate` and `endDate` using `dateutil.ParseDate` (supports multiple formats: bare date `YYYY-MM-DD`, datetime without timezone, datetime without seconds, and RFC3339).
- Service validates that the specified currency exists via `currencyRepo.GetByID`; returns `ErrInvalidCurrency` on failure.
- Service normalizes `period` to uppercase via `strings.ToUpper` before storage.
- Service adds 1 day to `endDate` before storing (to include the full end day); handler subtracts 1 day in the response via `toBudgetResponse`.
- Service converts category IDs to a comma-separated string via `categoriesToCSV`.
- Category IDs are stored as-is without validating that they belong to the user.
- After persistence, the service synchronously recalculates `collectedAmount` by querying expense transactions that fall within the budget's date window and match its `included_categories`, summing them with currency conversion via `currencySvc.ConvertAmount`. The recalculated total is written via `budgetRepo.UpdateCollectedAmount` and reflected in the response. If recalculation fails after persistence has succeeded, the error is logged with `budgetID` and `userID` and the request still returns 200 with the persisted budget (the stale `collectedAmount` is self-healing on the next transaction CRUD or manual recalc).
- `comment` is stored as a nullable `*string`; null is preserved as null (not converted to a string).
- Handler maps service errors via `handleServiceError`: `ErrNotFound` -> 404, `ErrAccessDenied` -> 403, `ErrInvalidCurrency` -> 400, `ErrInvalidStartDate` -> 400, `ErrInvalidEndDate` -> 400, all others -> 500.

### Known Gaps / TODOs

- Category IDs are not validated against the user's actual categories.
- **Synchronous recalc has no row/range bounds.** `recalculateBudget` scans all matching expense transactions in the budget's date window in-request. For users with very large transaction histories or extremely wide date ranges this is a theoretical DoS surface (info-level note from security review 2026-04-30). Same risk profile already exists in the `budget:user_update` worker path; consider a bound or async path if this becomes a real concern.
- **Test coverage gaps for the SQL recalc filter.** No dedicated tests exist that prove the recalc query at the SQL layer correctly excludes `is_transfer = true` and `exclude_from_reports = true` transactions. (QA suggestion 2026-04-30.)
- **Empty `included_categories` semantics not asserted.** Behavior when `included_categories` is empty (string == "" — does it match all categories or none?) is currently relied on in code but not pinned by a test. (QA suggestion 2026-04-30.)
- **Behavior on missing FX rate during recalc not tested.** `currencySvc.ConvertAmount` can fail when no exchange rate exists for a transaction's date/currency pair. The recalc-failure-returns-200 contract is exercised via a generic mock failure but not specifically the missing-FX-rate path. (QA suggestion 2026-04-30.)
- **`TestCreate_RecalcFailureReturns200` does not assert the returned `collectedAmount` value.** The test confirms 200 + persisted budget, but does not assert that the response carries the pre-recalc value (i.e., 0 for create) rather than a sentinel. (QA suggestion 2026-04-30.)

### Tests

Integration tests (11):
- `TestCreateBudgetSuccess` -- 200, correct name and targetAmount
- `TestCreateBudgetDateOnlyFormat` -- 200, accepts date-only format (`YYYY-MM-DD`)
- `TestCreateBudgetLowercasePeriod` -- 200, period returned as lowercase in response
- `TestCreateBudgetInvalidCurrency` -- 400 for invalid currency ID
- `TestCreateBudgetUnauthorized` -- 401 without auth token
- `TestCreateBudgetMissingName` -- 422 on validation failure
- `TestCreateBudgetWithRepeat` -- 200 with repeat=true
- `TestCreateBudgetWeeklyPeriod` -- 200 with WEEKLY period string
- `TestCreateBudgetYearlyPeriod` -- 200 with YEARLY period string
- `TestCreateBudgetWithComment` -- 200 with comment field set
- `TestCreateBudgetRecalculatesFromExistingTransactions` -- 200, `collectedAmount` reflects the sum of pre-existing matching expense transactions

Handler unit tests (8):
- `TestCreateBudgetBindError` -- 422 for malformed JSON
- `TestCreateBudgetValidateError` -- 422 for empty body
- `TestCreateBudgetInvalidCurrency` -- 400 when currency repo returns error
- `TestCreateBudgetDBError` -- 500 when budget repo returns error
- `TestCreateBudgetSuccess` -- 200 success path with categories
- `TestCreateBudgetDateOnlyFormat` -- 200, date-only format parsed correctly
- `TestCreateBudgetInvalidStartDateFormat` -- 400 for unparseable start date
- `TestCreateBudgetInvalidEndDateFormat` -- 400 for unparseable end date

Service unit tests (recalculation contract):
- `TestCreate_RecalculatesCollectedAmount` -- service returns recalculated `collectedAmount` after successful create
- `TestCreate_RecalcFailureReturns200` -- recalculation failure after persistence succeeds is logged and the persisted budget is still returned (no error propagated)

---

## GET /budgets/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`GetByUserID`)

### Request

Query parameters:

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| include | string | No | Default: "all". Valid values: all, active, archived |

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "name": "Monthly Groceries",
    "currencyId": 1,
    "targetAmount": 500,
    "collectedAmount": 123.45,
    "period": "monthly",
    "repeat": true,
    "startDate": "2026-01-01T00:00:00Z",
    "endDate": "2026-01-31T00:00:00Z",
    "includedCategories": "3,7",
    "isArchived": false,
    "comment": null,
    "currency": {
      "id": 1,
      "code": "USD",
      "name": "US Dollar"
    }
  }
]
```

Notes:
- Returns an empty array when the user has no budgets.
- `endDate` is returned as stored minus 1 day.
- `period` is always lowercase (e.g., `"monthly"`).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Internal DB error | 500 | "Failed to get budgets" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Handler delegates to `service.GetByUserID(userID, include)`.
- Service defaults `include` to `"all"` if empty, then passes to the repository which handles filtering by `is_archived`.
- Invalid values for `include` are passed through to the repository without a handler-level error.
- `endDate` is adjusted by subtracting 1 day in `toBudgetResponse`.

### Known Gaps / TODOs

- Invalid `include` values (anything other than all/active/archived) are not rejected at the handler level.

### Tests

Integration tests (5):
- `TestGetBudgetsSuccess` -- 200, at least one budget returned
- `TestGetBudgetsFilterActive` -- 200, all returned budgets have isArchived=false
- `TestGetBudgetsEmptyList` -- 200, empty array for new user
- `TestGetBudgetsUnauthorized` -- 401 without auth token
- `TestGetBudgetsFilterArchived` -- 200, archived budget present in results

Unit tests (2):
- `TestGetBudgetsDBError` -- 500 when budget repo returns error
- `TestGetBudgetsSuccess` -- 200 with include=active, currency object included

---

## PUT /budgets/:id/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`Update`)

### Request

Path parameter: `id` (int) -- budget ID.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | string | Yes | |
| currencyId | int | Yes | camelCase JSON tag |
| targetAmount | decimal.Decimal | Yes | |
| period | string | Yes | Any string accepted; normalized to uppercase in service |
| repeat | bool | Yes | |
| startDate | string | Yes | Date string; parsed by service via `dateutil.ParseDate` |
| endDate | string | Yes | Date string; parsed by service via `dateutil.ParseDate`; 1 day is added before storage |
| categories | []int | No | |
| comment | *string | No | |

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "name": "Updated Budget",
  "currencyId": 1,
  "targetAmount": 600,
  "collectedAmount": 123.45,
  "period": "monthly",
  "repeat": true,
  "startDate": "2026-02-01T00:00:00Z",
  "endDate": "2026-02-28T00:00:00Z",
  "includedCategories": "3,7",
  "isArchived": false,
  "comment": null,
  "currency": {
    "id": 1,
    "code": "USD",
    "name": "US Dollar"
  }
}
```

Notes:
- `collectedAmount` is recalculated from existing matching transactions immediately after the update is persisted; the value in the example above is illustrative.
- `period` is always returned in lowercase.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid budget ID" |
| Invalid JSON body | 422 | "Invalid request data" |
| Missing required fields | 422 | "Validation failed" |
| Invalid start date format | 400 | "Invalid start date format" |
| Invalid end date format | 400 | "Invalid end date format" |
| Budget not found | 404 | "Budget not found" |
| Budget belongs to another user | 403 | "Access denied" |
| Invalid currency ID | 400 | "Invalid currency" |
| Internal DB error | 500 | "Failed to update budget" |

### Business Logic

- Handler parses `id` from the path parameter using `strconv.Atoi`; returns 400 on parse failure.
- Handler binds and validates the request body, then delegates to `service.Update(userID, budgetID, params)`.
- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Service parses `startDate` and `endDate` using `dateutil.ParseDate`; returns `ErrInvalidStartDate` or `ErrInvalidEndDate` on failure.
- Service fetches the budget by ID via `budgetRepo.GetByID`; returns `ErrNotFound` if not found.
- Service compares the budget's `UserID` to the authenticated user's ID; returns `ErrAccessDenied` if mismatched.
- Service validates the currency exists via `currencyRepo.GetByID`; returns `ErrInvalidCurrency` on failure.
- Service normalizes `period` to uppercase via `strings.ToUpper` before storage.
- Service adds 1 day to `endDate` before storing; handler subtracts 1 day in the response.
- After persistence, the service synchronously recalculates `collectedAmount` against the (possibly changed) date window, `included_categories`, and `currencyId` by querying matching expense transactions, summing them with currency conversion via `currencySvc.ConvertAmount`. The recalculated total is written via `budgetRepo.UpdateCollectedAmount` and reflected in the response. If recalculation fails after persistence has succeeded, the error is logged with `budgetID` and `userID` and the request still returns 200 with the persisted budget; the previous `collectedAmount` (the value loaded by `GetByID`) is returned in that case and will be self-healing on the next transaction CRUD or manual recalc.
- Category IDs are stored without validating that they belong to the user.

### Known Gaps / TODOs

- Category IDs are not validated against the user's actual categories.
- **Synchronous recalc has no row/range bounds.** `recalculateBudget` scans all matching expense transactions in the budget's date window in-request. Theoretical DoS surface for users with very large transaction histories or extremely wide date ranges (info-level note from security review 2026-04-30). Same risk profile applies to `Service.Create` and to the existing `budget:user_update` worker path.
- **Test coverage gaps for the SQL recalc filter.** No dedicated tests exist that prove the recalc query at the SQL layer correctly excludes `is_transfer = true` and `exclude_from_reports = true` transactions. (QA suggestion 2026-04-30.)
- **Empty `included_categories` semantics not asserted.** Behavior when `included_categories` is empty (string == "") is currently relied on in code but not pinned by a test. (QA suggestion 2026-04-30.)
- **Behavior on missing FX rate during recalc not tested.** The recalc-failure-returns-200 contract is exercised via a generic mock failure but not specifically the missing-exchange-rate path through `currencySvc.ConvertAmount`. (QA suggestion 2026-04-30.)
- **DB-read consistency assertion missing in one integration test.** At least one of the recalc integration tests should also re-read the budget via `GET /budgets/` (or directly from DB) and assert the persisted `collectedAmount` matches the value returned in the `PUT` response, to prove the `UpdateCollectedAmount` write actually landed. (QA suggestion 2026-04-30.)

### Tests

Integration tests (11):
- `TestUpdateBudgetSuccess` -- 200, name and targetAmount updated correctly
- `TestUpdateBudgetDateOnlyFormat` -- 200, accepts date-only format
- `TestUpdateBudgetLowercasePeriod` -- 200, period returned as lowercase in response
- `TestUpdateBudgetNotFound` -- 404 for nonexistent budget
- `TestUpdateBudgetOtherUser` -- 403 for another user's budget
- `TestUpdateBudgetInvalidID` -- 400 for non-numeric path ID
- `TestUpdateBudgetUnauthorized` -- 401 without auth token
- `TestUpdateBudgetAddCategoryIncreasesCollectedAmount` -- 200, `collectedAmount` grows when a category with pre-existing matching transactions is added
- `TestUpdateBudgetRemoveCategoryDecreasesCollectedAmount` -- 200, `collectedAmount` shrinks when a category is removed
- `TestUpdateBudgetChangeWindowRecalculates` -- 200, `collectedAmount` is recalculated when `startDate`/`endDate` change to include or exclude transactions
- `TestUpdateBudgetChangeCurrencyRecalculatesWithConversion` -- 200, `collectedAmount` is recalculated with FX conversion when `currencyId` changes

Handler unit tests (11):
- `TestUpdateBudgetInvalidID` -- 400 for non-numeric path param
- `TestUpdateBudgetBindError` -- 422 for invalid JSON
- `TestUpdateBudgetValidateError` -- 422 for empty body
- `TestUpdateBudgetNotFound` -- 404 when budget not found in DB
- `TestUpdateBudgetAccessDenied` -- 403 for different user's budget
- `TestUpdateBudgetInvalidCurrency` -- 400 for nonexistent currency
- `TestUpdateBudgetDBError` -- 500 on DB update failure
- `TestUpdateBudgetSuccess` -- 200 success path
- `TestUpdateBudgetDateOnlyFormat` -- 200, date-only format parsed correctly
- `TestUpdateBudgetInvalidStartDateFormat` -- 400 for unparseable start date
- `TestUpdateBudgetInvalidEndDateFormat` -- 400 for unparseable end date

Service unit tests (recalculation contract):
- `TestUpdate_RecalculatesCollectedAmount` -- service overwrites stale pre-update `collectedAmount` with the recalculated total
- `TestUpdate_RecalcFailureReturns200` -- recalculation failure after persistence succeeds is logged and the persisted budget is still returned (no error propagated)

---

## DELETE /budgets/:id/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`Delete`)

### Request

Path parameter: `id` (int) -- budget ID.

No request body.

### Response

**Success**: HTTP 200

```json
{
  "message": "Budget with id 1 deleted"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid budget ID" |
| Budget not found | 404 | "Budget not found" |
| Budget belongs to another user | 403 | "Access denied" |
| Internal DB error | 500 | "Failed to delete budget" |

### Business Logic

- Handler parses `id` from the path parameter using `strconv.Atoi`; returns 400 on parse failure.
- Handler delegates to `service.Delete(userID, budgetID)`.
- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Service fetches the budget by ID; returns `ErrNotFound` if not found.
- Service compares `UserID` to the authenticated user's ID; returns `ErrAccessDenied` if mismatched.
- Service delegates to `budgetRepo.Delete(budgetID)` which performs a soft delete by setting `is_deleted = true`.

### Tests

Integration tests (4):
- `TestDeleteBudgetSuccess` -- 200, successful soft deletion
- `TestDeleteBudgetNotFound` -- 404 for nonexistent budget
- `TestDeleteBudgetOtherUser` -- 403 for another user's budget
- `TestDeleteBudgetInvalidID` -- 400 for non-numeric ID

Unit tests (5):
- `TestDeleteBudgetInvalidID` -- 400 for non-numeric path param
- `TestDeleteBudgetNotFound` -- 404 when budget not found in DB
- `TestDeleteBudgetAccessDenied` -- 403 for different user's budget
- `TestDeleteBudgetDBError` -- 500 on DB delete failure
- `TestDeleteBudgetSuccess` -- 200 success path

---

## PUT /budgets/:id/archive/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`Archive`)

### Request

Path parameter: `id` (int) -- budget ID.

No request body.

### Response

**Success**: HTTP 200

```json
{
  "message": "Budget with id 1 archived"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-numeric path ID | 400 | "Invalid budget ID" |
| Budget not found | 404 | "Budget not found" |
| Budget belongs to another user | 403 | "Access denied" |
| Internal DB error | 500 | "Failed to archive budget" |

### Business Logic

- Handler parses `id` from the path parameter using `strconv.Atoi`; returns 400 on parse failure.
- Handler delegates to `service.Archive(userID, budgetID)`.
- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Service fetches the budget by ID; returns `ErrNotFound` if not found.
- Service compares `UserID` to the authenticated user's ID; returns `ErrAccessDenied` if mismatched.
- Service delegates to `budgetRepo.Archive(budgetID)` which sets `is_archived = true`.
- Un-archiving is not supported by this endpoint.

### Tests

Integration tests (4):
- `TestArchiveBudgetSuccess` -- 200, successful archive
- `TestArchiveBudgetNotFound` -- 404 for nonexistent budget
- `TestArchiveBudgetOtherUser` -- 403 for another user's budget
- `TestArchiveBudgetInvalidID` -- 400 for non-numeric ID

Unit tests (5):
- `TestArchiveBudgetInvalidID` -- 400 for non-numeric path param
- `TestArchiveBudgetNotFound` -- 404 when budget not found in DB
- `TestArchiveBudgetAccessDenied` -- 403 for different user's budget
- `TestArchiveBudgetDBError` -- 500 on DB archive failure
- `TestArchiveBudgetSuccess` -- 200 success path

---

## GET /budgets/daily-processing

**Auth**: Required (JWT) + Admin only (RequireAdmin middleware)
**Handler**: `internal/handlers/budgets/handler.go` (thin adapter)
**Service**: `internal/services/budgets/service.go` (`DailyProcessing`)

### Request

No query parameters. No request body.

### Response

**Success**: HTTP 200

```json
{
  "message": "Daily processing completed",
  "processed": 3
}
```

Notes:
- `processed` is the count of budgets successfully rolled over in this run.

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| No auth token | 401 | JWT middleware |
| User not active or deleted | 401 | "User not activated" |
| Non-admin user | 403 | "Admin access required" |
| DB error fetching outdated budgets | 500 | "Failed to get outdated budgets" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted. `RequireAdmin` middleware checks that the user's email is in the admin emails list; returns 403 with "Admin access required" otherwise.
- Handler delegates to `service.DailyProcessing()`.
- Service fetches all budgets where `end_date < NOW() AND is_archived = false AND is_deleted = false`.
- For each outdated budget:
  - If `repeat = false`: sets `is_archived = true`.
  - If `repeat = true`: creates a copy with the suffix " (copy)" appended to the name, `collectedAmount` reset to `decimal.Zero`, and dates shifted forward. Period values are compared in uppercase:
    - DAILY: both `startDate` and `endDate` shifted +1 day
    - WEEKLY: both shifted +7 days
    - MONTHLY: both shifted +1 month
    - YEARLY: both shifted +1 year
    - default (including CUSTOM): new start = old end date, new end = old end date + original duration
  - After creating the copy, archives the original budget.
  - If creating the copy fails, logs the error and continues to the next budget (does not count as processed).
  - If archiving fails after a copy was created, logs the error and does not count as processed.
- Processing is synchronous; the response is returned only after all budgets are handled.
- Errors in individual budgets are logged and skipped; processing continues for remaining budgets.

### Known Gaps / TODOs

- No integration tests exist for this endpoint; coverage is via unit tests only.
- Date shifting logic differs from the legacy approach: for standard periods (DAILY/WEEKLY/MONTHLY/YEARLY), both `startDate` and `endDate` are shifted by the period duration; for custom/unknown periods, the new start is the old end date and the new end is calculated from the original duration.

### Tests

Unit tests (11):
- `TestDailyProcessingGetOutdatedError` -- 500 on DB error fetching outdated budgets
- `TestDailyProcessingNoOutdatedBudgets` -- 200, processed=0 when nothing is outdated
- `TestDailyProcessingArchiveNonRepeating` -- 200, non-repeating budget archived, processed=1
- `TestDailyProcessingRepeatMonthly` -- 200, copy created with "(copy)" suffix for monthly period
- `TestDailyProcessingRepeatDaily` -- 200, daily period date shifting
- `TestDailyProcessingRepeatWeekly` -- 200, weekly period date shifting
- `TestDailyProcessingRepeatYearly` -- 200, yearly period date shifting
- `TestDailyProcessingRepeatCustomPeriod` -- 200, custom period duration shifting
- `TestDailyProcessingCreateError` -- 200, processed=0 when copy creation fails
- `TestDailyProcessingArchiveError` -- 200, processed=0 when archiving fails
- `TestDailyProcessingArchiveErrorAfterCopy` -- 200, processed=0 when archive fails after copy is created

Additional unit tests (shared):
- `TestRegisterRoutes` -- verifies all budget routes are registered correctly
