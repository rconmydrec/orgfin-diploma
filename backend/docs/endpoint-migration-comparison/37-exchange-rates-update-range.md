# Endpoint #37: GET `/exchange-rates/update/from/{start}/to/{end}/`

**Status**: MATCH (fully implemented via service)
**Date**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /exchange-rates/update/from/{start_date}/to/{end_date}` | `GET /exchange-rates/update/from/:start_date/to/:end_date` |
| Auth | JWT (check_token) | JWT (auth middleware) |
| File | `app/routes/exchange_rates.py:58` | `internal/handlers/exchange_rates/handler.go` (handler) + `internal/services/exchange_rates/service.go` (business logic) |

## Request

Both: GET with path parameters for start and end dates. Auth required.

| Parameter | Python | Go | Match |
|-----------|--------|-----|-------|
| `start_date` | `date` (path, auto-parsed by FastAPI) | `string` (path param, manually parsed) | OK |
| `end_date` | `date` (path, auto-parsed by FastAPI) | `string` (path param, manually parsed) | OK |

## Response

**Success**: 200 OK.

Python returns the last updated exchange rate record. Go returns an operation summary:

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| Response format | Single rate object (last day) | Summary: `{datesProcessed, startDate, endDate}` | **DIFF** -- Go returns operation summary |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| Invalid date format | 422 (FastAPI auto) | Validation error | 400 | "Invalid start_date format" / "Invalid end_date format" | **DIFF** -- different status codes |
| ErrorFetchingData | 500 | "Error while fetching data" | 500 | "Unable to fetch exchange rates for {date}" | OK |
| Generic exception | 500 | "Unable to update exchange rates" | 500 | "Unable to save exchange rates for {date}" | OK |
| End before start | 500 (returns None, fails validation) | N/A | 400 | "end_date must not be before start_date" | Go BETTER (explicit validation) |
| User not active | N/A (no check) | N/A | 401 | "User not activated" | Go BETTER |
| Non-admin | N/A (no check) | N/A | 403 | "Admin access required" | Go BETTER |
| Unauthorized | 422 (missing token) | N/A | 401 | N/A | **DIFF** -- status code differs |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES (RequireActiveUser middleware) | Go BETTER |
| Admin check | NO | YES (RequireAdmin middleware) | Go BETTER |
| Validate date format | YES (auto by FastAPI type) | YES (service uses time.Parse) | OK |
| Validate end >= start | NO (implicit, loop skips) | YES (explicit ErrEndBeforeStart) | Go BETTER |
| Iterate day-by-day | YES (loop from start to end) | YES (service iterates start to end inclusive) | OK |
| Calls external API per day | YES (CurrencyBeaconService) | YES (service calls CurrencyBeacon API) | OK |
| Saves to DB per day | YES (delete + insert per day) | YES (UPSERT via repository) | OK |
| Returns real rates | YES (last day's result) | NO (returns summary: datesProcessed, dates) | **DIFF** |
| Handles end < start | Returns None -> 500 (bug) | Returns 400 with message | Go BETTER |

## Notes

- Go endpoint is **fully implemented** via `services/exchange_rates.Service.FetchAndSaveRatesRange()`. The handler delegates all business logic (date parsing, validation, API calls, DB persistence) to the service.
- Both Python and Go iterate day-by-day from start_date to end_date, calling the CurrencyBeacon API and saving each day's rates to DB. Python returns the last updated record; Go returns an operation summary (`datesProcessed`, `startDate`, `endDate`).
- Python has a bug: when end_date < start_date, the loop doesn't execute and `updated_rates` is empty, causing `updated_rates[-1]` to return `None`, which fails Pydantic validation and returns 500. Go explicitly validates this and returns 400.
- Go validates dates explicitly and returns 400 for invalid formats. Python relies on FastAPI's automatic path parameter type conversion and returns 422.
- Go has RequireActiveUser and RequireAdmin middleware guards that Python does not have.
- The same `exchangerates.Service` is shared by the handler and the `exchange_rate:update` worker task, eliminating code duplication.

## Tests

### Python Tests (5 total for this endpoint)

| Test | Verifies |
|------|----------|
| `test_update_rates_date_range_success` | 200, `rates` in response (mocked) |
| `test_update_rates_invalid_date_range` | 500 when end < start |
| `test_update_rates_future_dates` | 200 with future dates |
| `test_update_rates_date_range_error_fetching_data` | 500 on ErrorFetchingData |
| `test_update_rates_date_range_generic_exception` | 500 on generic error |

### Go Integration Tests

| Test | Verifies |
|------|----------|
| `TestUpdateRatesRangeNoAPIKey` | 500 when API key is not configured |
| `TestUpdateRatesRangeInvalidStartDate` | 400 for bad start_date |
| `TestUpdateRatesRangeInvalidEndDate` | 400 for bad end_date |
| `TestUpdateRatesRangeEndBeforeStartIntegration` | 400 when end < start |
| `TestUpdateRatesRangeUnauthorized` | 401 without token |
| `TestUpdateRatesRangeNotAdmin` | 403 when user is not in admin list |
| `TestUpdateRatesRangeInvalidDateFormat` | 400 for DD-MM-YYYY format |

### Go Unit Tests

| Test | Verifies |
|------|----------|
| `TestUpdateRatesRangeSuccess` | 200 for 3-day range, verifies 3 API calls and datesProcessed=3 |
| `TestUpdateRatesRangeSingleDay` | 200 when start_date equals end_date, datesProcessed=1 |
| `TestUpdateRatesRangeInvalidStartDate` | 400, "Invalid start_date format" |
| `TestUpdateRatesRangeInvalidEndDate` | 400, "Invalid end_date format" |
| `TestUpdateRatesRangeEndBeforeStart` | 400 with "end_date must not be before start_date" |
| `TestUpdateRatesRangeAPIFailure` | 500 when second API call fails |
| `TestUpdateRatesRangeDBSaveFailure` | 500 when second DB save fails |
| `TestUpdateRatesRangeAPIKeyNotConfigured` | 500 when API key is empty |
| `TestUpdateRatesRangeUserNotActive` | 401, "User not activated" |
