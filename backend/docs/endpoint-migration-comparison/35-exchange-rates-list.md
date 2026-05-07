# Endpoint #35: GET `/exchange-rates/`

**Status**: DIFF - Response structure differs
**Date**: 2026-02-21

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /exchange-rates/` | `GET /exchange-rates/` |
| Auth | `check_token` (router-level dependency) | JWT middleware (`user_id` from context) |
| File | `app/routes/exchange_rates.py:24` | `internal/handlers/exchange_rates/handler.go:59` |

## Request

Both: GET with no parameters. Auth token required.

| Aspect | Python | Go | Match |
|--------|--------|-----|-------|
| Auth header | `Authorization: Bearer <token>` | `auth-token: <token>` | **DIFF** (framework convention) |
| Query params | None | None | OK |

## Response

**Success**: 200 OK.

### Python Response (ExchangeRateHistory model, auto-serialized)

Returns the full SQLAlchemy model for today's exchange rates (single object):

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | Primary key |
| `rates` | dict (JSONB) | Map of currency codes to rates |
| `actual_date` | date (snake_case) | Date of the rates |
| `base_currency_code` | str (snake_case) | Base currency code |
| `service_name` | str | Name of the rate service |
| `is_deleted` | bool | Soft delete flag |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Update timestamp |

### Go Response (ExchangeRateListResponse)

Returns a curated DTO:

| Field | Type | Notes |
|-------|------|-------|
| `id` | int `json:"id"` | Primary key |
| `rates` | map[string]float64 `json:"rates"` | Map of currency codes to rates |
| `actualDate` | string `json:"actualDate"` | Date string (camelCase) |
| `baseCurrencyCode` | string `json:"baseCurrencyCode"` | Base currency code (camelCase) |

### Field Comparison

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int | OK |
| `rates` | dict | map[string]float64 | OK |
| `actual_date` / `actualDate` | date (snake_case) | string (camelCase) | **DIFF** (casing) |
| `base_currency_code` / `baseCurrencyCode` | str (snake_case) | string (camelCase) | **DIFF** (casing) |
| `service_name` | str | N/A | **DIFF** (Go omits) |
| `is_deleted` | bool | N/A | **DIFF** (Go omits) |
| `created_at` | datetime | N/A | **DIFF** (Go omits) |
| `updated_at` | datetime | N/A | **DIFF** (Go omits) |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 422 | Validation error | 401 | Unauthorized | **DIFF** |
| Internal error | 500 | "Unable to get exchange rates" | 500 | "Unable to get exchange rates" | OK |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** |
| User deleted | N/A | N/A | 401 | "User not activated" | **DIFF** |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| Check user active | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Check user deleted | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Date filter | Defaults to `date.today()` | Defaults to `time.Now().Format("2006-01-02")` | OK |
| Query method | `db.query(ExchangeRateHistory).filter(actual_date == today).one()` | `exchangerates.Service.GetRatesForDate(date)` (delegates to repo) | OK |
| Supports "when" param | YES (optional `when` parameter in service) | NO (always today) | **DIFF** |
| Supports "latest" mode | YES (`when='latest'` orders by date desc) | NO | **DIFF** |
| Returns single object | YES (`.one()`) | YES (single response) | OK |

## Notes

- Python returns the full `ExchangeRateHistory` SQLAlchemy model with all columns (including `service_name`, `is_deleted`, `created_at`, `updated_at`). Go returns a curated DTO with only `id`, `rates`, `actualDate`, `baseCurrencyCode`.
- Field naming differs: Python uses snake_case (`actual_date`, `base_currency_code`), Go uses camelCase (`actualDate`, `baseCurrencyCode`).
- Python's `get_exchange_rates()` service supports an optional `when` parameter (date, 'latest', or empty string for today). Go always fetches for today.
- Python uses SQLAlchemy `.one()` which raises `NoResultFound` if no rates exist for today. Go's repository handles this similarly.
- Go adds `RequireActiveUser` middleware guard not present in Python.
- Both Go and Python query the `exchange_rates` table directly, parsing the JSONB `rates` column into a map of currency codes to rates. Go delegates through `exchangerates.Service.GetRatesForDate()` to the repository.

## Tests

### Python Tests (2 total)
| Test | Verifies |
|------|----------|
| `test_get_exchange_rates_success` | 200 with data (list or dict) |
| `test_get_exchange_rates_unauthorized` | 422 without auth |

### Go Integration Tests (4 total)
| Test | Verifies |
|------|----------|
| `TestGetRatesAuthorized` | Not 401 for authenticated user |
| `TestGetRatesUnauthorized` | 401 without auth token |
| `TestGetRatesWithToken` | Not 401 with valid token |
| `TestGetRatesInvalidToken` | 401 with invalid token |

### Go Unit Tests (4 total)
| Test | Verifies |
|------|----------|
| `TestGetRatesDBError` | 500 when repo returns error |
| `TestGetRatesSuccess` | 200 with EUR in response |
| `TestGetRatesUserNotActive` | 401 when user is not active |
| `TestGetRatesUserRepoError` | 401 when user repo returns error |
