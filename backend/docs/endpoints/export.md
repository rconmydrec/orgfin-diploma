# Export Endpoints

Endpoints for exporting transaction data as Excel files, either as a direct download or sent via email.

## Table of Contents

- [POST /export/download](#post-exportdownload)
- [POST /export/email](#post-exportemail)

---

## POST /export/download

**Auth**: Required (JWT)
**Handler**: `internal/handlers/export/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| start_date | string | Yes | Format: YYYY-MM-DD |
| end_date | string | Yes | Format: YYYY-MM-DD |

### Response

**Success**: HTTP 200

Returns a binary Excel file attachment.

- Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- Content-Disposition: `attachment; filename="transactions_export.xlsx"`

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| Validation failed | 422 | "Validation failed" |
| Invalid start_date format | 422 | "Invalid start_date format, expected YYYY-MM-DD" |
| Invalid end_date format | 422 | "Invalid end_date format, expected YYYY-MM-DD" |
| start_date after end_date | 422 | "start_date must be before or equal to end_date" |
| Date range exceeds 366 days | 422 | "Date range must not exceed 366 days" |
| Excel generation failure | 500 | "Failed to generate export" |

### Business Logic

- Validates the date range via `validateExportRequest()`: parses both dates, checks ordering, checks max 366-day span.
- Calls `ExportService.GenerateExcel()` which returns the file as an in-memory byte slice.
- Sends the response via `c.Blob()`.
- Uses the `errResponseSent` sentinel pattern to avoid double-writing responses on validation errors.
- `RequireActiveUser` middleware ensures the user is active and not deleted before any processing.

### Tests

Integration tests:
- `TestDownloadExportSuccess` — verifies 200 and Excel content-type header
- `TestDownloadExportEmptyDateRange` — verifies behavior with same start and end date
- `TestDownloadExportStartAfterEnd` — verifies 422 when start_date > end_date
- `TestDownloadExportDateRangeExceedsLimit` — verifies 422 when range > 366 days
- `TestDownloadExportDateRangeAtBoundary` — verifies 200 at exactly 366 days
- `TestDownloadExportExcludesReportExcludedTransactions` — verifies report-excluded transactions are not included
- `TestDownloadExportUnauthorized` — verifies 401 without valid JWT

Unit tests:
- `TestDownloadExportUserNotActive` — verifies 401 for inactive user
- `TestDownloadExportUserRepoError` — verifies error path in user repo lookup
- `TestDownloadExportUserDeleted` — verifies 401 for soft-deleted user
- `TestDownloadExportBindError` — verifies 422 on bind failure
- `TestDownloadExportValidationError` — verifies 422 on general validation failure
- `TestDownloadExportInvalidStartDateFormat` — verifies 422 for bad start_date string
- `TestDownloadExportInvalidEndDateFormat` — verifies 422 for bad end_date string
- `TestDownloadExportStartAfterEnd` — unit-level check for date ordering rule
- `TestDownloadExportDateRangeExceeds366Days` — unit-level check for range limit
- `TestDownloadExportServiceError` — verifies 500 on service failure
- `TestDownloadExportSuccess` — unit-level success path

---

## POST /export/email

**Auth**: Required (JWT)
**Handler**: `internal/handlers/export/handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| start_date | string | Yes | Format: YYYY-MM-DD |
| end_date | string | Yes | Format: YYYY-MM-DD |

### Response

**Success**: HTTP 202

```json
{ "message": "Export will be sent to your email shortly" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| Validation failed | 422 | "Validation failed" |
| Invalid start_date format | 422 | "Invalid start_date format, expected YYYY-MM-DD" |
| Invalid end_date format | 422 | "Invalid end_date format, expected YYYY-MM-DD" |
| start_date after end_date | 422 | "start_date must be before or equal to end_date" |
| Date range exceeds 366 days | 422 | "Date range must not exceed 366 days" |
| User fetch error | 500 | "Failed to get user information" |

### Business Logic

- Uses the same `validateExportRequest()` as the download endpoint for consistent date validation.
- Fetches the user record from the DB via `userRepo.GetByID()` to obtain the email address.
- Fires a background goroutine to generate the Excel file and send the email.
- The goroutine runs with a 5-minute context timeout.
- Panic recovery is included in the goroutine.
- Email and generation errors are logged but not surfaced to the caller (202 is returned immediately).
- `RequireActiveUser` middleware ensures the user is active and not deleted before any processing.

### Tests

Integration tests:
- `TestEmailExportSuccess` — verifies 202 and response message
- `TestEmailExportEmptyDateRange` — verifies behavior with same start and end date
- `TestEmailExportStartAfterEnd` — verifies 422 when start_date > end_date
- `TestEmailExportDateRangeExceedsLimit` — verifies 422 when range > 366 days
- `TestEmailExportUnauthorized` — verifies 401 without valid JWT

Unit tests:
- `TestEmailExportBindError` — verifies 422 on bind failure
- `TestEmailExportUserError` — verifies 500 when user repo returns an error
- `TestEmailExportSuccess` — verifies 202 on the happy path
- `TestEmailExportStartAfterEnd` — unit-level check for date ordering rule
- `TestRegisterRoutes` — verifies routes are correctly registered
