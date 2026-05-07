# Analytics Endpoints

AI-powered spending analysis endpoints. **Both endpoints are currently disabled and return 404.**

## Table of Contents

- [POST /analytics/spending-trends](#post-analyticsspending-trends)
- [POST /analytics/expense-categorization](#post-analyticsexpense-categorization)

---

> **WARNING: Both analytics endpoints are intentionally disabled stubs. They return 404 for all requests. The handler DTOs exist but the business logic is not implemented (TODO comment in code).**

---

## POST /analytics/spending-trends

**Auth**: Required (JWT)
**Handler**: `internal/handlers/analytics/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| startDate | time.Time | Yes | validate:"required" |
| endDate | time.Time | Yes | validate:"required" |
| limit | *int | No | Optional result limit; defaults to -1 (all) |

### Response

**Success**: HTTP 404 — this endpoint is disabled and always returns Not Found.

```json
{ "message": "Not found" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| Endpoint disabled | 404 | "Not found" |

### Business Logic

- Handler immediately returns 404 with no further processing.
- DTOs `AnalysisRequest` and `AnalysisResponse` are defined but unused.
- `RequireActiveUser` middleware ensures the user is active and not deleted before the handler runs.
- Future intent: AI-powered spending trend analysis over a date range.

### Known Gaps / TODOs

- Endpoint is not implemented. The handler contains a TODO comment indicating a future redesign.

### Tests

- `TestSpendingTrendsNotFound` — verifies endpoint returns 404
- `TestSpendingTrendsUnauthorized` — verifies 401 without valid JWT
- `TestSpendingTrendsInvalidBody` — verifies behavior with malformed request body
- `TestSpendingTrendsInvalidToken` — verifies 401 with bad token
- `TestSpendingTrendsEmptyBody` — verifies behavior with empty body
- `TestSpendingTrendsWithAccountID` — verifies 404 even with valid account context

---

## POST /analytics/expense-categorization

**Auth**: Required (JWT)
**Handler**: `internal/handlers/analytics/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| startDate | time.Time | Yes | validate:"required" |
| endDate | time.Time | Yes | validate:"required" |

### Response

**Success**: HTTP 404 — this endpoint is disabled and always returns Not Found.

```json
{ "message": "Not found" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| Endpoint disabled | 404 | "Not found" |

### Business Logic

- Handler immediately returns 404 with no further processing.
- DTOs `ExpenseCategorizationRequest` and `AnalysisResponse` are defined but unused.
- `RequireActiveUser` middleware ensures the user is active and not deleted before the handler runs.
- Future intent: AI-powered categorization of expenses over a date range.

### Known Gaps / TODOs

- Endpoint is not implemented. The handler contains a TODO comment indicating a future redesign.
- Response DTO does not include a `status` field that the future design may require.

### Tests

- `TestExpenseCategorizationNotFound` — verifies endpoint returns 404
- `TestExpenseCategorizationUnauthorized` — verifies 401 without valid JWT
- `TestExpenseCategorizationInvalidBody` — verifies behavior with malformed request body
- `TestExpenseCategorizationInvalidToken` — verifies 401 with bad token
- `TestExpenseCategorizationEmptyBody` — verifies behavior with empty body
- `TestExpenseCategorizationWithAccountID` — verifies 404 even with valid account context
