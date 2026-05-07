# Endpoint #36: GET `/exchange-rates/update/`

**Status**: MATCH (async approach)
**Date**: 2026-02-23

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /exchange-rates/update` | `GET /exchange-rates/update` |
| Auth | JWT (check_token) | JWT (auth middleware) |
| File | `app/routes/exchange_rates.py:38` | `internal/handlers/exchange_rates/handler.go` |

## Request

Both: GET with no parameters. Auth required.

## Response

**Success**: 200 OK.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| Response | Rates object from DB | `{"message": "Exchange rate update task enqueued"}` | **DIFF** -- Go is async (enqueues task), Python is sync |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| User not active | N/A (no check) | N/A | 401 | "User not activated" | Go BETTER |
| Non-admin | N/A (no check) | N/A | 403 | "You are not authorized" | Go BETTER |
| Task enqueue failure | N/A | N/A | 500 | "Failed to start exchange rate update" | Go only |
| Unauthorized | 422 (missing token) | N/A | 401 | N/A | **DIFF** -- status code differs |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES | Go BETTER |
| Admin check | NO | YES | Go BETTER |
| Calls external API | YES (sync, in handler) | YES (async, via worker task) | OK (different approach) |
| Saves to DB | YES (sync) | YES (async via worker) | OK |
| Sends notification email | NO | YES (worker sends admin email) | Go BETTER |
| Returns real rates | YES (sync response) | NO (returns enqueue confirmation) | **DIFF** -- Go is async |

## Notes

- Go enqueues an `exchange_rate:update` asynq task and returns immediately. The worker task handler delegates to `exchangerates.Service.FetchAndSaveRates()` which fetches rates from the CurrencyBeacon API, saves to DB, and then sends admin notification emails.
- Python calls the API synchronously in the handler and returns the saved rates.
- Go has admin and is_active/is_deleted user checks that Python does not have.
- The handler struct holds `{service, enqueuer, logger}` -- no direct repo or config dependencies.

## Tests

### Go Integration Tests

| Test | Verifies |
|------|----------|
| `TestUpdateRatesEnqueuesTask` | 200, message contains "Exchange rate update task enqueued" |
| `TestUpdateRatesUnauthorized` | 401 without token |
| `TestUpdateRatesNonAdmin` | 403 for non-admin user |
| `TestUpdateRatesInvalidToken` | 401 with invalid token |

### Go Unit Tests

| Test | Verifies |
|------|----------|
| `TestUpdateRatesSuccess` | 200, verifies task type is "exchange_rate:update" |
| `TestUpdateRatesEnqueueError` | 500 when enqueue fails |
| `TestUpdateRatesUserNotActive` | 401, "User not activated" |
| `TestUpdateRatesNotAdmin` | 403 for non-admin email |
| `TestUpdateRatesNoEmail` | 403 when email missing from context |
| `TestUpdateRatesEmailWrongType` | 403 when email is wrong type |
