# Endpoint #49: GET `/budgets/daily-processing/`

**Status**: DIFF
**Date**: 2026-02-28

## Route Definition

| Aspect | Python | Go |
|--------|--------|-----|
| Path | `GET /budgets/daily-processing/` | `GET /budgets/daily-processing` |
| Auth | JWT (check_token + enforce_free_plan_compliance) | JWT (auth middleware) + is_active check + RequireAdmin |
| Handler | `app/routes/budgets.py:136` | `internal/handlers/budgets/handler.go` (thin adapter) |
| Service | `app/tasks/celery_tasks.py` (Celery task) | `internal/services/budgets/service.go` (`DailyProcessing`) |

## Architecture

The Go handler is a thin HTTP adapter that delegates to `service.DailyProcessing()`. The service handles all business logic (fetching outdated budgets, creating copies for repeating budgets, archiving).

## Request

Both: GET with no parameters. Auth required. Admin-only (Python: user_id == 1; Go: RequireAdmin middleware).

## Response

**Success**: 200 OK.

| Field | Python | Go | Match |
|-------|--------|-----|-------|
| `message` | `"Daily processing initiated"` | `"Daily processing completed"` | DIFF -- Python returns "initiated", Go returns "completed" |
| `processed` | N/A | int (count of processed budgets) | DIFF -- Go returns count; Python does not |

## Error Responses

| Condition | Python Status | Python Message | Go Status | Go Message | Match |
|-----------|--------------|----------------|-----------|------------|-------|
| No auth token | 401 | (middleware) | 401 | (middleware) | OK |
| User not active | N/A | N/A | 401 | "User not activated" | DIFF -- Go has is_active check |
| Non-admin user | 403 | "Forbidden" | 403 | "Admin access required" | DIFF -- different message text |
| Get outdated budgets error | N/A (async task) | N/A | 500 | "Failed to get outdated budgets" | DIFF -- Go handles inline |

## Business Logic Comparison

| Step | Python | Go | Match |
|------|--------|-----|-------|
| Auth required | YES | YES | OK |
| is_active check | NO | YES (RequireActiveUser middleware) | DIFF -- Go checks user is_active |
| Admin check | `user_id != 1` -> 403 | `RequireAdmin` middleware -> 403 "Admin access required" | DIFF -- Go uses middleware instead of inline check |
| Processing model | ASYNC (Celery task `run_daily_budgets_processing.delay()`) | SYNC (service.DailyProcessing, inline in handler) | DIFF -- Python delegates to async Celery task; Go processes synchronously |
| Get outdated budgets | `end_date < now AND is_archived = false AND is_deleted = false` | `end_date < NOW() AND is_archived = false AND is_deleted = false` | OK |
| Repeat handling | Creates copy of repeating budget, then archives original | Creates copy of repeating budget, then archives original | OK |
| Copy naming | `"{name} (copy)"` | `"{name} (copy)"` | OK |
| Copy collected_amount | Reset to 0 | Reset to 0 (decimal.Zero) | OK |
| Period comparison | lowercase | UPPERCASE (periods stored as uppercase after normalization) | DIFF -- Go compares against uppercase constants (DAILY, WEEKLY, etc.) |
| New date calculation (daily) | `new_start = end_date`, `new_end = end_date + 1 day` | `new_start = start + 1 day`, `new_end = end + 1 day` | DIFF -- Python starts new period from end_date; Go shifts both start and end by 1 day |
| New date calculation (weekly) | `new_start = end_date`, `new_end = end_date + 1 week` | `new_start = start + 7 days`, `new_end = end + 7 days` | DIFF -- Python starts new from end_date; Go shifts both dates |
| New date calculation (monthly) | `new_start = end_date`, `new_end = end_date + 1 month` | `new_start = start + 1 month`, `new_end = end + 1 month` | DIFF -- Python starts new from end_date; Go shifts both dates |
| New date calculation (yearly) | `new_start = end_date`, `new_end = end_date + 1 year` | `new_start = start + 1 year`, `new_end = end + 1 year` | DIFF -- Python starts new from end_date; Go shifts both dates |
| Custom period | Raises `InvalidPeriod` exception | `new_start = end_date`, `new_end = end_date + original_duration` | DIFF -- Python rejects custom; Go handles it using original duration |
| Error handling (create fails) | Exception propagates, stops processing | Logs error, continues to next budget | DIFF -- Go is more resilient |
| Error handling (archive fails) | N/A (always succeeds in loop) | Logs error, continues, budget not counted as processed | DIFF -- Go handles archive failures |
| Response | `{"message": "Daily processing initiated"}` | `{"message": "Daily processing completed", "processed": N}` | DIFF |

## Notes

- **Major architectural difference**: Python dispatches an async Celery task (`run_daily_budgets_processing.delay()`) and immediately returns "initiated". Go processes everything synchronously in the service and returns "completed" with a count.
- **Date calculation difference**: Python starts new budget periods from `end_date` of the old budget (contiguous periods). Go shifts both `start_date` and `end_date` forward by the period duration for standard periods (DAILY/WEEKLY/MONTHLY/YEARLY); for custom/unknown periods, Go starts from the old end date and calculates new end from original duration.
- **Error resilience**: Go continues processing remaining budgets if one fails (create or archive), counting only fully processed budgets. Python's Celery task would stop on the first exception during `create_copy_of_outdated_budget`.
- **Custom period**: Python raises `InvalidPeriod` for custom periods. Go uses the original duration (`endDate - startDate`) for the shift.
- Go includes a `processed` count in the response; Python does not return processing details (the task runs asynchronously).
- Python uses `pendulum` library for date arithmetic; Go uses standard `time.Time.AddDate`.
- Go compares period values against uppercase constants (DAILY, WEEKLY, MONTHLY, YEARLY) since periods are normalized to uppercase on create/update.

## Tests

### Python Tests (0 total)

No dedicated test files found for budget endpoints.

### Go Integration Tests (0 total for daily-processing)

No integration tests found for this endpoint.

### Go Unit Tests (11 total for daily-processing)

| Test | Verifies |
|------|----------|
| `TestDailyProcessingGetOutdatedError` | 500 on DB error |
| `TestDailyProcessingNoOutdatedBudgets` | 200, processed=0 |
| `TestDailyProcessingArchiveNonRepeating` | 200, processed=1, archive called |
| `TestDailyProcessingRepeatMonthly` | 200, creates copy with "(copy)" name |
| `TestDailyProcessingRepeatDaily` | 200, daily period shifting |
| `TestDailyProcessingRepeatWeekly` | 200, weekly period shifting |
| `TestDailyProcessingRepeatYearly` | 200, yearly period shifting |
| `TestDailyProcessingRepeatCustomPeriod` | 200, custom period shifting |
| `TestDailyProcessingCreateError` | 200, processed=0 when create fails |
| `TestDailyProcessingArchiveError` | 200, processed=0 when archive fails |
| `TestDailyProcessingArchiveErrorAfterCopy` | 200, processed=0 when archive fails after copy created |
