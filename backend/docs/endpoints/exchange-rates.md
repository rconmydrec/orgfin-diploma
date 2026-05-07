# Exchange Rates Endpoints

HTTP handlers for retrieving and updating foreign exchange rate data. The handler delegates all business logic (API calls, DB persistence, date iteration) to `services/exchange_rates.Service`. The update endpoints fetch rates from the CurrencyBeacon API and save them to the `exchange_rates` table.

## Table of Contents

- [GET /exchange-rates/](#get-exchange-rates)
- [GET /exchange-rates/update](#get-exchange-ratesupdate)
- [GET /exchange-rates/updatefrom/:start_date/to/:end_date](#get-exchange-ratesupdatefromstart_datetoendate)

---

## GET /exchange-rates/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/exchange_rates/handler.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{
  "id": 101,
  "rates": {
    "EUR": 0.92,
    "GBP": 0.79,
    "UAH": 37.5
  },
  "actualDate": "2024-01-15",
  "baseCurrencyCode": "USD"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Internal error | 500 | "Unable to get exchange rates" |

### Business Logic

- **Service**: `services/exchange_rates.Service.GetRatesForDate()`
- `RequireActiveUser` middleware verifies the user is active and not deleted.
- The handler calls `Service.GetRatesForDate(today)` which delegates to the repository.
- The repository queries the `exchange_rates` table and parses the JSONB `rates` column into a `map[string]float64`.
- Implements back-fill (nearest past date) and forward-fill (nearest future date) when no exact match exists.
- Returns a curated DTO with `id`, `rates`, `actualDate`, and `baseCurrencyCode`. Internal fields such as `service_name`, `is_deleted`, `created_at`, and `updated_at` are not exposed.
- Only the current day's rates are returned; historical lookups and "latest" mode are not supported.

### Tests

- `TestGetRatesAuthorized` -- response is not 401 for an authenticated user
- `TestGetRatesUnauthorized` -- 401 without auth token
- `TestGetRatesWithToken` -- not 401 with a valid token
- `TestGetRatesInvalidToken` -- 401 with an invalid token
- `TestGetRatesDBError` -- 500 when repository returns an error
- `TestGetRatesSuccess` -- 200 with EUR present in the rates map
- `TestGetRatesUserNotActive` -- 401 when user is not active
- `TestGetRatesUserRepoError` -- 401 when user repository returns an error
- `TestGetRatesUserDeleted` -- 401 when user is soft-deleted

---

## GET /exchange-rates/update

**Auth**: Required (JWT) + Admin only
**Handler**: `internal/handlers/exchange_rates/handler.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{ "message": "Exchange rate update task enqueued" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Non-admin user | 403 | "Admin access required" |
| Task enqueue failure | 500 | "Failed to start exchange rate update" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- `RequireAdmin` middleware verifies the user's email is in the `ADMINS_NOTIFICATION_EMAILS` list. Returns 403 with "Admin access required" if not.
- Enqueues an `exchange_rate:update` async task via asynq. The actual API call, DB save, and admin email notification happen in the worker task handler (see `docs/workers.md`).
- Returns immediately with a success message. The update runs asynchronously.

### Tests

- `TestUpdateRatesSuccess` (unit) -- 200, verifies task is enqueued with correct type
- `TestUpdateRatesEnqueueError` (unit) -- 500 when task enqueue fails
- `TestUpdateRatesEnqueuesTask` (integration) -- 200 with "Exchange rate update task enqueued"
- `TestUpdateRatesUnauthorized` -- 401 without auth token
- `TestUpdateRatesInvalidToken` -- 401 with invalid auth token
- `TestUpdateRatesNotAdmin` -- 403 when user is not in admin list
- `TestUpdateRatesUserNotActive` -- 401 with "User not activated"

Note: Admin email validation edge cases (missing email, wrong type) are now handled by the `RequireAdmin` middleware and tested in `internal/middleware/admin_test.go`.

---

## GET /exchange-rates/updatefrom/:start_date/to/:end_date

**Auth**: Required (JWT) + Admin only
**Handler**: `internal/handlers/exchange_rates/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| start_date | string (path) | Yes | Date in `YYYY-MM-DD` format |
| end_date | string (path) | Yes | Date in `YYYY-MM-DD` format, must not be before start_date |

### Response

**Success**: HTTP 200

```json
{
  "datesProcessed": 31,
  "startDate": "2024-01-01",
  "endDate": "2024-01-31"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Non-admin user | 403 | "Admin access required" |
| Invalid start_date format | 400 | "Invalid start_date format" |
| Invalid end_date format | 400 | "Invalid end_date format" |
| end_date before start_date | 400 | "end_date must not be before start_date" |
| API key not configured | 500 | "Unable to fetch exchange rates for {date}" |
| CurrencyBeacon API failure | 500 | "Unable to fetch exchange rates for {date}" |
| Database save failure | 500 | "Unable to save exchange rates for {date}" |

### Business Logic

- **Service**: `services/exchange_rates.Service.FetchAndSaveRatesRange()`
- `RequireActiveUser` middleware verifies the user is active and not deleted.
- `RequireAdmin` middleware verifies the user's email is in the `ADMINS_NOTIFICATION_EMAILS` list. Returns 403 with "Admin access required" if not.
- The handler delegates to `Service.FetchAndSaveRatesRange(startDateStr, endDateStr)` which:
  - Parses `start_date` and `end_date` using `time.Parse("2006-01-02", ...)`. Returns an error if either date is not in `YYYY-MM-DD` format.
  - Validates that `end_date` is not before `start_date`.
  - Iterates day-by-day from `start_date` to `end_date` (inclusive), calling the CurrencyBeacon API for each date.
  - Saves each day's rates to the `exchange_rates` table using UPSERT on `(service_name, actual_date)`.
  - Returns a result with `DatesProcessed` count, `StartDate`, and `EndDate`.
  - If any date fails (API error or DB save error), stops and returns a `DateError`.
- The handler maps service errors to HTTP responses: `DateError` with "fetch" operation to 500, `DateError` with "save" operation to 500, `ErrEndBeforeStart` to 400, `ErrDateRangeExceeded` to 400, and date parse errors to 400.

### Tests

- `TestUpdateRatesRangeSuccess` (unit) -- 200 for 3-day range, verifies 3 API calls and datesProcessed=3
- `TestUpdateRatesRangeSingleDay` (unit) -- 200 when start_date equals end_date, datesProcessed=1
- `TestUpdateRatesRangeNoAPIKey` (integration) -- 500 when API key is not configured
- `TestUpdateRatesRangeInvalidStartDate` -- 400 for a malformed start_date (unit and integration)
- `TestUpdateRatesRangeInvalidEndDate` -- 400 for a malformed end_date (unit and integration)
- `TestUpdateRatesRangeEndBeforeStart` (unit) -- 400 with "end_date must not be before start_date"
- `TestUpdateRatesRangeEndBeforeStartIntegration` -- 400 integration test
- `TestUpdateRatesRangeNotAdmin` -- 403 when user is not in admin list
- `TestUpdateRatesRangeUnauthorized` -- 401 without auth token
- `TestUpdateRatesRangeInvalidDateFormat` (integration) -- 400 for DD-MM-YYYY format
- `TestUpdateRatesRangeAPIFailure` (unit) -- 500 when second API call fails
- `TestUpdateRatesRangeDBSaveFailure` (unit) -- 500 when second DB save fails
- `TestUpdateRatesRangeAPIKeyNotConfigured` (unit) -- 500 when API key is empty
- `TestUpdateRatesRangeUserNotActive` -- 401 with "User not activated"
