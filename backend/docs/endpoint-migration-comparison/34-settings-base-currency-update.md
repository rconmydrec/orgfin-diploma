# Endpoint #34: PUT `/settings/base-currency/`

**Status**: DIFF - Go lacks planned transaction conversion
**Date**: 2026-02-21
**Last Updated**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `PUT /settings/base-currency/` | `PUT /settings/base-currency/` |
| Auth | `check_token` dependency | JWT middleware (`user_id` from context) |
| File | `app/routes/user_settings.py:105` | `internal/handlers/settings/handler.go:169` |

## Request

Both: PUT with JSON body. Auth token required.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `currency_id` / `currencyId` | int (`currency_id` with camelCase alias `currencyId`) | int `json:"currencyId"` | OK |

**Python schema** (`BaseCurrencyInputSchema`): Pydantic model with `currency_id: int`, alias generator `to_camel` so accepts `currencyId`.

**Go struct** (`BaseCurrencyRequest`): `CurrencyID int` with `json:"currencyId"` and `validate:"required"`.

## Response

**Success**: 200 OK. Updated currency object.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `id` | int | int `json:"id"` | OK |
| `code` | str | string `json:"code"` | OK |
| `name` | str | string `json:"name"` | OK |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 422 | Validation error | 401 | Unauthorized | **DIFF** |
| Missing currencyId | 422 | "currencyId is required" | 422 | "Validation failed" | OK (different wording) |
| Null currencyId | 422 | "currencyId is required" | 422 | "Validation failed" | OK (different wording) |
| Invalid currency ID | 500 | Generic error (sqlalchemy `.one()` raises) | 400 | "Invalid currency" | **DIFF** (Go returns 400) |
| Invalid JSON body | 422 | FastAPI validation | 422 | "Invalid request data" | OK |
| Internal error | 500 | "Unable to update base currency" | 500 | "Failed to update base currency" | OK (different wording) |
| User not active | N/A | N/A | 401 | "User not activated" | **DIFF** |
| User deleted | N/A | N/A | 401 | "User not activated" | **DIFF** |
| HTTPException (422) | 422 | "currencyId is required" | N/A | N/A | **DIFF** (Python re-raises) |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| Check user active | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Check user deleted | NO | YES (RequireActiveUser middleware) | **DIFF** |
| Validate request | Pydantic schema | Echo Bind + Validate | OK |
| Check null currencyId | YES (explicit nil check) | YES (validate:"required") | OK |
| Look up new currency | YES (query Currency by ID) | YES (`service.UpdateBaseCurrency` -> `currencyRepo.GetByID`) | OK |
| Look up user | YES (query User with joinedload) | YES (service -> `userRepo.GetByID`) | OK |
| Get old base currency | YES (user.base_currency) | NO | **DIFF** |
| Convert planned transactions | YES (if currency changed) | NO | **DIFF** (major feature gap) |
| Update user.base_currency_id | YES | YES (service sets `user.BaseCurrencyID`) | OK |
| Commit/persist | YES (db.commit) | YES (service -> `userRepo.Update`) | OK |
| Return new currency | YES | YES | OK |

## Architecture

Go handler is a thin HTTP adapter. It reads `user_id` from context, binds and validates the request, then delegates to `service.UpdateBaseCurrency(userID, currencyID)`. The service validates the currency exists, fetches the user, updates `BaseCurrencyID`, persists via `userRepo.Update`, and returns the currency. Handler maps `ErrInvalidCurrency` to 400, other errors to 500.

## Notes

- **Major difference**: Python's `update_base_currency` service converts all non-deleted, non-executed planned transactions from the old base currency to the new base currency using `calc_amount()`. Go does NOT have this logic -- the service simply updates the `base_currency_id` on the user record.
- Python handles `HTTPException` separately in the except chain, re-raising 422 errors. Go does not have this pattern.
- For invalid currency ID, Python raises a SQLAlchemy `NoResultFound` exception (caught as generic -> 500), while Go returns 400 "Invalid currency" via `ErrInvalidCurrency`. Go's approach is more user-friendly.
- Go adds `RequireActiveUser` middleware guard not present in Python.
- Both validate that `currencyId` is provided, but through different mechanisms (Python: explicit `if currency_id is None` + Pydantic, Go: `validate:"required"` tag).

## Tests

### Python Tests (6 total)
| Test | Verifies |
|------|----------|
| `test_update_base_currency_success` | 200 with EUR currency response |
| `test_update_base_currency_invalid` | 500 for non-existent currency ID |
| `test_update_base_currency_missing_id` | 422 for empty body |
| `test_update_base_currency_unauthorized` | 422 without auth |
| `test_update_base_currency_null_id` | 422 for null currencyId |
| `test_update_base_currency_http_exception` | 422 when HTTPException raised (mocked) |
| `test_update_base_currency_currency_id_none_direct` | 422 direct function call with None currencyId |

### Go Integration Tests (7 total)
| Test | Verifies |
|------|----------|
| `TestUpdateBaseCurrencySuccess` | 200 with EUR currency response |
| `TestUpdateBaseCurrencyInvalid` | 400 for non-existent currency ID |
| `TestUpdateBaseCurrencyMissingID` | 422 for empty body |
| `TestUpdateBaseCurrencyUnauthorized` | 401 without auth token |
| `TestUpdateBaseCurrencyToUSD` | 200 update to USD |
| `TestUpdateBaseCurrencyInvalidBody` | 422 for invalid JSON body |
| `TestUpdateBaseCurrencyVerifyPersistence` | PUT then GET verifies persistence |

### Go Unit Tests — Handler (3 total)
| Test | Verifies |
|------|----------|
| `TestUpdateBaseCurrencyInvalidCurrencyUnit` | 400 when service returns `ErrInvalidCurrency` |
| `TestUpdateBaseCurrencyUpdateError` | 500 when userRepo.Update fails |
| `TestRegisterRoutes` | All 5 routes registered correctly |

### Go Unit Tests — Service (6 total)
| Test | Verifies |
|------|----------|
| `TestUpdateBaseCurrency_Success` | Updates base currency and returns it |
| `TestUpdateBaseCurrency_InvalidCurrency` | Returns `ErrInvalidCurrency` |
| `TestUpdateBaseCurrency_UserFetchError` | Propagates user repo fetch errors |
| `TestUpdateBaseCurrency_UserUpdateError` | Propagates user repo update errors |
| `TestUpdateBaseCurrency_BaseCurrencyIDSetCorrectly` | Correct ID set on user model |
| `TestUpdateBaseCurrency_CorrectCurrencyReturned` | Returns correct currency object |
