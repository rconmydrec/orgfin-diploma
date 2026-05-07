# Currencies Endpoints

HTTP handlers for retrieving the list of supported currencies available to authenticated users.

## Table of Contents

- [GET /currencies/](#get-currencies)

---

## GET /currencies/

**Auth**: Required (JWT)
**Handler**: `internal/handlers/currencies/handler.go`

### Request

No request body.

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "code": "USD",
    "name": "US Dollar"
  },
  {
    "id": 2,
    "code": "EUR",
    "name": "Euro"
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Missing auth token | 401 | "Missing authorization header" |
| Internal error | 500 | "Failed to get currencies" |

### Business Logic

- Returns all currencies from the reference table, ordered by currency code.
- Currencies are global reference data; they are not user-specific.
- The currencies table has no `is_deleted` column — no soft-delete filtering is applied.
- Handler calls the repository directly without a service layer.

### Known Gaps / TODOs

- Missing `is_active` check on the user. Inactive users can still retrieve the currency list. This is a consistency issue relative to other authenticated endpoints.

### Tests

- `TestGetCurrenciesSuccess` — 200 with correct id/code/name fields
- `TestGetCurrenciesIncludesMajorCurrencies` — USD and EUR are present
- `TestGetCurrenciesUnauthorized` — 401 when no auth token provided
- `TestGetCurrenciesHasValidStructure` — all items contain id, code, and name
- `TestGetCurrenciesReturnsNonEmpty` — response contains at least 5 currencies
- `TestGetCurrenciesWithGBP` — GBP is present in the list
- `TestGetCurrenciesWithJPY` — JPY is present in the list
- `TestGetCurrenciesWithUAH` — UAH is present in the list
- `TestGetCurrenciesHasIDs` — all currency IDs are positive integers
- `TestGetCurrenciesCodeLength` — all currency codes are exactly 3 characters
- `TestGetCurrenciesDBError` — 500 when repository returns a database error
- `TestGetCurrenciesSuccess` (unit) — 200 with currencies in response
