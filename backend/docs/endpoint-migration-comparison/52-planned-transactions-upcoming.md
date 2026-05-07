# Endpoint #52: GET /planned-transactions/upcoming/occurrences

## Route Definition
- **Python**: `@router.get('/upcoming/occurrences', response_model=list[PlannedTransactionOccurrenceSchema])`
- **Go**: `g.GET("/upcoming/occurrences", h.GetUpcoming)`

## Request
- Auth: required (both)
- Query params (Python): `days` (int, default 30), `include_inactive` (bool, default False)
- Query params (Go): `days` (string, parsed to int, default 30), `include_inactive` (string, "true" to enable)

## Response
- **Python**: 200 OK with `list[PlannedTransactionOccurrenceSchema]`
- **Go**: 200 OK with `[]OccurrenceResponse` (mapped through DTOs in the handler)
- Python schema fields: plannedTransactionId, occurrenceDate, amount, isIncome, label, isActive, isRecurring
- Go maps repository results through `OccurrenceResponse` DTOs, converting `decimal.Decimal` to float64 via `InexactFloat64()`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Internal error | 500 `"Error fetching upcoming occurrences"` | 500 `"Failed to get upcoming occurrences"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Date calculation**: Python computes `end_date = datetime.now() + timedelta(days=days)` and passes it to the service; Go passes `days` directly to the repository.
2. **Service vs repository**: Python calls `pt_service.get_upcoming_occurrences()` with `end_date`; Go calls `service.GetUpcoming()` which delegates to the repository with `days`.
3. **Response**: Python serializes through `PlannedTransactionOccurrenceSchema`; Go maps through `OccurrenceResponse` DTOs in the handler.
4. **Invalid days**: Both silently default to 30 if parsing fails (Go) or use default (Python).

## Notes
- Go now maps occurrences through `OccurrenceResponse` DTOs, similar to Python's schema-based serialization.
- Python has subscription plan enforcement; Go does not.
- Go adds `RequireActiveUser` middleware verification.

## Tests
- **Python**: 4 tests (test_get_upcoming_occurrences_success, _custom_days, _include_inactive, _unauthorized, _internal_error)
- **Go integration tests**: 4 tests (TestGetUpcomingSuccess, TestGetUpcomingWithDays, TestGetUpcomingUnauthorized, TestGetUpcomingWithInvalidDays, TestGetUpcomingInvalidToken)
- **Go unit tests**: 3 tests (TestGetUpcomingDBError, TestGetUpcomingSuccess, TestGetUpcomingUserNotActive)
