# Endpoint #60: POST /export/download

## Route Definition
- **Python**: `@router.post('/download')`
- **Go**: `g.POST("/download", h.DownloadExport)`

## Request
- Auth: required (both)
- Body (Python `ExportTransactionsRequestSchema`): start_date (date), end_date (date). Supports camelCase aliases via alias_generator.
- Body (Go `ExportRequest`): start_date (string, required), end_date (string, required). Uses snake_case JSON tags.

## Response
- **Python**: 200 OK with `StreamingResponse` (Excel file, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`)
- **Go**: 200 OK with `c.Blob()` (Excel file, same MIME type)
- Both set `Content-Disposition: attachment; filename="..."` header
- Python uses dynamic filename `transactions_{start_date}_{end_date}.xlsx`; Go uses static `transactions_export.xlsx`

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Invalid request | 422 (Pydantic validation) | 422 `"Invalid request data"` or `"Validation failed"` |
| Invalid date format | 422 (Pydantic) | 422 `"Invalid start_date format, expected YYYY-MM-DD"` |
| Start after end | 422 (Pydantic model_validator) | 422 `"start_date must be before or equal to end_date"` |
| Range > 366 days | 422 (Pydantic model_validator) | 422 `"Date range must not exceed 366 days"` |
| Generation error | N/A (no explicit catch) | 500 `"Failed to generate export"` |
| User not active | N/A | 401 `"User not activated"` |

## Business Logic Comparison
1. **Validation**: Python validates in schema (Pydantic model_validator for date range); Go validates manually in `validateExportRequest()` with explicit date parsing and range checks.
2. **Date format**: Python uses `date` type (automatic parsing); Go expects `YYYY-MM-DD` string format and parses manually.
3. **Generation**: Python calls `get_export_transactions()` then `generate_excel()` synchronously; Go calls `h.exportService.GenerateExcel()` through a service interface.
4. **Response**: Python uses `StreamingResponse` (streaming); Go uses `c.Blob()` (in-memory byte slice).
5. **Filename**: Python includes date range in filename; Go uses static name.
6. **Architecture**: Go uses a service interface (`ExportService`) for better testability; Python uses direct function calls.

## Notes
- Go has more explicit validation with user-friendly error messages for each failure case.
- Python streams the file; Go loads entire file into memory before sending.
- Go adds `RequireActiveUser` middleware verification.
- Python does not have explicit error handling for Excel generation failure.
- Go uses `errResponseSent` sentinel pattern for validation errors to avoid double-writing responses.

## Tests
- **Python**: 7 tests (test_download_success, _empty_date_range, _excludes_report_excluded, _start_after_end, _date_range_exceeds_limit, _date_range_at_limit, _unauthorized)
- **Go integration tests**: 7 tests (TestDownloadExportSuccess, TestDownloadExportEmptyDateRange, TestDownloadExportStartAfterEnd, TestDownloadExportDateRangeExceedsLimit, TestDownloadExportDateRangeAtBoundary, TestDownloadExportExcludesReportExcludedTransactions, TestDownloadExportUnauthorized)
- **Go unit tests**: 11 tests (TestDownloadExportUserNotActive, _UserRepoError, _UserDeleted, _BindError, _ValidationError, _InvalidStartDateFormat, _InvalidEndDateFormat, _StartAfterEnd, _DateRangeExceeds366Days, _ServiceError, _Success)
