# Endpoint #61: POST /export/email

## Route Definition
- **Python**: `@router.post('/email', status_code=status.HTTP_202_ACCEPTED)`
- **Go**: `g.POST("/email", h.EmailExport)`

## Request
- Auth: required (both)
- Body: same as download endpoint
- Python (`ExportTransactionsRequestSchema`): start_date (date), end_date (date)
- Go (`ExportRequest`): start_date (string, required), end_date (string, required)

## Response
- **Python**: 202 Accepted with `{"message": "Export will be sent to your email shortly"}`
- **Go**: 202 Accepted with `{"message": "Export will be sent to your email shortly"}`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Invalid request | 422 (Pydantic validation) | 422 `"Invalid request data"` or `"Validation failed"` |
| Invalid date format | 422 (Pydantic) | 422 `"Invalid start_date format, expected YYYY-MM-DD"` |
| Start after end | 422 (Pydantic model_validator) | 422 `"start_date must be before or equal to end_date"` |
| Range > 366 days | 422 (Pydantic model_validator) | 422 `"Date range must not exceed 366 days"` |
| User fetch error | N/A | 500 `"Failed to get user information"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Email retrieval**: Python gets email from `request.state.user['email']` (JWT token); Go fetches user from DB via `h.userRepo.GetByID()`.
2. **Async processing**: Python generates Excel synchronously, saves to temp file, then dispatches Celery task `export_transactions_email.delay()` for email sending. Go fires a goroutine that generates Excel and sends email, with a 5-minute context timeout.
3. **Task queue**: Python uses Celery with Redis/RabbitMQ broker; Go uses a plain goroutine (no task queue).
4. **Excel generation**: Python generates synchronously before dispatching; Go generates within the background goroutine.
5. **Error handling**: Python has no explicit error handling for generation/email; Go handles errors in the goroutine with logging, plus panic recovery.
6. **Timeout**: Python relies on Celery task timeout; Go uses explicit 5-minute context timeout.

## Notes
- Go's goroutine approach means email failures are only logged, never retried. Celery provides retry capabilities.
- Go has panic recovery in the goroutine, which is good defensive coding.
- Go fetches user from DB for email, which is slightly less efficient than reading from JWT token.
- Both return 202 immediately and process asynchronously.
- Go adds `RequireActiveUser` middleware verification.
- Go uses the same `validateExportRequest()` as the download endpoint for consistent validation.

## Tests
- **Python**: 5 tests (test_email_success, _empty_date_range, _start_after_end, _date_range_exceeds_limit, _unauthorized)
- **Go integration tests**: 5 tests (TestEmailExportSuccess, TestEmailExportEmptyDateRange, TestEmailExportStartAfterEnd, TestEmailExportDateRangeExceedsLimit, TestEmailExportUnauthorized)
- **Go unit tests**: 5 tests (TestEmailExportBindError, TestEmailExportUserError, TestEmailExportSuccess, TestEmailExportStartAfterEnd, TestRegisterRoutes)
