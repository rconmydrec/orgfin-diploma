# Settings Endpoints

HTTP handlers for user application settings, including language preferences, projection settings, and the user's base currency. The handler is a thin HTTP adapter that delegates business logic to `services/settings.Service`.

## Table of Contents

- [GET /settings/languages](#get-settingslanguages)
- [GET /settings](#get-settings)
- [POST /settings](#post-settings)
- [GET /settings/base-currency/](#get-settingsbase-currency)
- [PUT /settings/base-currency/](#put-settingsbase-currency)

---

## GET /settings/languages

**Auth**: Public
**Handler**: `internal/handlers/settings/handler.go`

### Request

No request body. No authentication required.

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "code": "en",
    "name": "English",
    "isDeleted": false
  },
  {
    "id": 4,
    "code": "bg",
    "name": "Български",
    "isDeleted": false
  },
  {
    "id": 2,
    "code": "ru",
    "name": "Русский",
    "isDeleted": false
  },
  {
    "id": 3,
    "code": "uk",
    "name": "Українська",
    "isDeleted": false
  }
]
```

Returns all 4 supported languages (English, Russian, Ukrainian, Bulgarian).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Internal error | 500 | "Failed to get languages" |

### Business Logic

- Public endpoint; no authentication or `is_active` check is performed.
- Handler calls `languageRepo.GetAll()` directly (languages are not part of the settings service).
- Returns only non-deleted languages (`WHERE is_deleted = false`).
- Results are ordered by name.
- The `isDeleted` field in the response is always `false` (filtered at the query level, so the field is redundant but included for schema completeness).

### Tests

**Integration tests:**
- `TestGetLanguagesSuccess` — 200 with correct language structure
- `TestGetLanguagesIncludesCommonLanguages` — "en" is present in the list
- `TestGetLanguagesNonDeleted` — all returned items have `isDeleted: false`

**Unit tests (handler):**
- `TestGetLanguagesDBError` — 500 when repository returns a database error

---

## GET /settings

**Auth**: Required (JWT)
**Handler**: `internal/handlers/settings/handler.go`
**Service**: `internal/services/settings/service.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{
  "id": 10,
  "userId": 42,
  "settings": {
    "language": "en",
    "projectionEndDate": null,
    "projectionPeriod": null
  },
  "createdAt": "2024-01-15T10:00:00",
  "updatedAt": "2024-01-15T10:00:00"
}
```

Timestamps are formatted as `2006-01-02T15:04:05` (no timezone offset).

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Internal error | 500 | "Failed to get settings" |
| Error creating default settings | 500 | "Failed to get settings" |

### Business Logic

- `RequireActiveUser` middleware verifies the user exists, is active (`is_active = true`), and is not soft-deleted (`is_deleted = false`). Returns 401 if any check fails.
- Handler reads `user_id` from context and delegates to `service.GetSettings(userID)`.
- The service uses a get-or-create pattern: fetches settings by `user_id`, and if no settings record exists (`sql.ErrNoRows`), automatically creates a default settings record with `language = "en"` and no projection fields, then re-fetches and returns the new record.
- Handler maps the returned model to a `SettingsResponse` DTO with camelCase JSON fields.

### Tests

**Integration tests:**
- `TestGetSettingsSuccess` — 200 with settings data for authenticated user
- `TestGetSettingsUnauthorized` — 401 without auth token
- `TestGetSettingsReturnsDefaults` — default settings structure for new user
- `TestGetSettingsCreatesDefaultWhenNotExists` — auto-creates defaults when settings record is absent
- `TestGetSettingsResponseStructure` — all fields (id, userId, settings, createdAt, updatedAt) present
- `TestGetSettingsNoTrailingSlashReturns200` — regression: `GET /settings` (no trailing slash) returns 200 with valid auth
- `TestGetSettingsNoTrailingSlashUnauthorized` — regression: `GET /settings` (no trailing slash) returns 401 without auth

**Unit tests (handler):**
- `TestGetSettingsDBError` — 500 when settings repository returns an error
- `TestGetSettingsCreateDefaultError` — 500 when creating default settings fails
- `TestGetSettingsGetAfterCreateError` — 500 when re-fetching after default creation fails

**Unit tests (service):**
- `TestGetSettings_Success` — returns settings for a user
- `TestGetSettings_AutoCreate` — auto-creates defaults when settings not found
- `TestGetSettings_CreateDefaultFails` — returns `ErrSettingsCreateFailed` when creation fails
- `TestGetSettings_ReFetchAfterCreateFails` — returns `ErrSettingsNotFound` when re-fetch fails
- `TestGetSettings_NonErrNoRowsDBError` — propagates non-ErrNoRows DB errors
- `TestGetSettings_CorrectUserIDPassed` — verifies correct user ID is forwarded to the repository

---

## POST /settings

**Auth**: Required (JWT)
**Handler**: `internal/handlers/settings/handler.go`
**Service**: `internal/services/settings/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| language | string | Yes | Language code (e.g., "en") |
| projectionEndDate | *string | No | End date for financial projection |
| projectionPeriod | *string | No | Period identifier for projection |

### Response

**Success**: HTTP 200

```json
{
  "id": 10,
  "userId": 42,
  "settings": {
    "language": "en",
    "projectionEndDate": "2024-12-31",
    "projectionPeriod": "monthly"
  },
  "createdAt": "2024-01-15T10:00:00",
  "updatedAt": "2024-01-15T10:30:00"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Invalid JSON body | 422 | "Invalid request data" |
| Error creating default settings | 500 | "Failed to create settings" |
| Internal error | 500 | "Failed to update settings" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Handler reads `user_id` from context, binds the request body, constructs `UpdateParams`, and delegates to `service.UpdateSettings(userID, params)`.
- The service uses the same get-or-create pattern as GET: if no settings record exists, auto-creates a default record before updating.
- Overwrites the three settings fields (language, projectionEndDate, projectionPeriod) with the request values.
- Does not validate that the `language` value corresponds to a known language code.
- If the service returns `ErrSettingsCreateFailed`, the handler returns 500 with "Failed to create settings". Other errors return 500 with "Failed to update settings".

### Known Gaps / TODOs

- No validation of settings field values: unknown language codes, invalid date formats in `projectionEndDate`, and invalid period values in `projectionPeriod` are all silently accepted.

### Tests

**Integration tests:**
- `TestUpdateSettingsSuccess` — 200 on valid settings update
- `TestUpdateSettingsUnauthorized` — 401 without auth token
- `TestUpdateSettingsWithProjection` — 200 with projectionEndDate and projectionPeriod
- `TestUpdateSettingsInvalidBody` — 422 on invalid JSON body
- `TestUpdateSettingsMultipleTimes` — multiple sequential updates persist correctly
- `TestUpdateSettingsCreatesDefaultWhenNotExists` — auto-creates defaults then updates
- `TestUpdateSettingsClearProjectionFields` — sets then clears projection fields with null
- `TestPostSettingsNoTrailingSlashReturns200` — regression: `POST /settings` (no trailing slash) returns 200 with valid auth
- `TestPostSettingsNoTrailingSlashUnauthorized` — regression: `POST /settings` (no trailing slash) returns 401 without auth

**Unit tests (handler):**
- `TestUpdateSettingsDBError` — 500 when settings repository returns an error
- `TestUpdateSettingsCreateDefaultError` — 500 when creating default settings fails
- `TestUpdateSettingsUpdateError` — 500 when repo Update fails
- `TestUpdateSettingsGetAfterCreateError` — 500 when re-fetching after default creation fails

**Unit tests (service):**
- `TestUpdateSettings_Success` — updates settings with new values
- `TestUpdateSettings_AutoCreatePath` — auto-creates defaults, then updates
- `TestUpdateSettings_NilProjectionFields` — clears projection fields when nil
- `TestUpdateSettings_UpdateRepoError` — propagates repo update errors
- `TestUpdateSettings_GetOrCreateFailure` — returns `ErrSettingsCreateFailed` on creation failure
- `TestUpdateSettings_FieldsAppliedCorrectly` — verifies all three fields are set on the saved model

---

## GET /settings/base-currency/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/settings/handler.go`
**Service**: `internal/services/settings/service.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{
  "id": 1,
  "code": "USD",
  "name": "US Dollar"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Base currency not configured | 404 | "Base currency not set" |
| Internal error | 500 | "Failed to get base currency" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Handler reads `user_id` from context and delegates to `service.GetBaseCurrency(userID)`.
- The service fetches the user record (with preloaded `BaseCurrency`) via `userRepo.GetByID`. If `user.BaseCurrency` is `nil` (no base currency assigned), returns `ErrBaseCurrencyNotSet`.
- Handler maps `ErrBaseCurrencyNotSet` to 404 with "Base currency not set". Other errors return 500 with "Failed to get base currency".
- Returns only the currency's id, code, and name.

### Tests

**Integration tests:**
- `TestGetBaseCurrencySuccess` — 200 with id, code, name fields
- `TestGetBaseCurrencyNewUser` — 200 for a user who has a default currency set
- `TestGetBaseCurrencyUnauthorized` — 401 without auth token
- `TestUpdateBaseCurrencyVerifyPersistence` — GET after PUT returns the updated currency

**Unit tests (handler):**
- `TestGetBaseCurrencyDBError` — 500 when service returns an error
- `TestGetBaseCurrencyNotSet` — 404 when user has no base currency assigned

**Unit tests (service):**
- `TestGetBaseCurrency_Success` — returns user's base currency
- `TestGetBaseCurrency_NotSet` — returns `ErrBaseCurrencyNotSet` when nil
- `TestGetBaseCurrency_UserFetchError` — propagates user repo errors
- `TestGetBaseCurrency_CorrectUserIDPassed` — verifies correct user ID is forwarded

---

## PUT /settings/base-currency/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/settings/handler.go`
**Service**: `internal/services/settings/service.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| currencyId | int | Yes | `validate:"required"` |

### Response

**Success**: HTTP 200

```json
{
  "id": 2,
  "code": "EUR",
  "name": "Euro"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | Unauthorized (middleware) |
| User inactive or deleted | 401 | "User not activated" |
| Missing or null currencyId | 422 | "Validation failed" |
| Invalid JSON body | 422 | "Invalid request data" |
| Currency ID does not exist | 400 | "Invalid currency" |
| Internal error | 500 | "Failed to update base currency" |

### Business Logic

- `RequireActiveUser` middleware verifies the user is active and not deleted.
- Handler reads `user_id` from context, binds and validates the request body (`validate:"required"` on `currencyId`; zero value fails), then delegates to `service.UpdateBaseCurrency(userID, currencyID)`.
- The service validates the currency exists via `currencyRepo.GetByID`; returns `ErrInvalidCurrency` if not found. Then fetches the user record, updates `base_currency_id`, persists via `userRepo.Update`, and returns the currency.
- Handler maps `ErrInvalidCurrency` to 400 with "Invalid currency". Other errors return 500 with "Failed to update base currency".

### Known Gaps / TODOs

- Does not convert existing planned transactions to the new base currency when the base currency changes. This conversion logic exists in the original implementation but is not implemented in the Go service.

### Tests

**Integration tests:**
- `TestUpdateBaseCurrencySuccess` — 200 with updated currency object
- `TestUpdateBaseCurrencyInvalid` — 400 for a non-existent currency ID
- `TestUpdateBaseCurrencyMissingID` — 422 for empty request body
- `TestUpdateBaseCurrencyUnauthorized` — 401 without auth token
- `TestUpdateBaseCurrencyToUSD` — 200 when updating to USD
- `TestUpdateBaseCurrencyInvalidBody` — 422 for invalid JSON body
- `TestUpdateBaseCurrencyVerifyPersistence` — PUT then GET confirms the new currency is saved

**Unit tests (handler):**
- `TestUpdateBaseCurrencyInvalidCurrencyUnit` — 400 for invalid currency ID (service returns `ErrInvalidCurrency`)
- `TestUpdateBaseCurrencyUpdateError` — 500 when userRepo.Update fails

**Unit tests (service):**
- `TestUpdateBaseCurrency_Success` — updates base currency and returns it
- `TestUpdateBaseCurrency_InvalidCurrency` — returns `ErrInvalidCurrency` for non-existent currency
- `TestUpdateBaseCurrency_UserFetchError` — propagates user repo fetch errors
- `TestUpdateBaseCurrency_UserUpdateError` — propagates user repo update errors
- `TestUpdateBaseCurrency_BaseCurrencyIDSetCorrectly` — verifies correct ID is set on user model
- `TestUpdateBaseCurrency_CorrectCurrencyReturned` — verifies returned currency matches the looked-up one
