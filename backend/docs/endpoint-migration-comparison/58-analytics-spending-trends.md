# Endpoint #58: POST /analytics/spending-trends

## Route Definition
- **Python**: `@router.post('/spending-trends', response_model=AnalysisResponseSchema)`
- **Go**: `g.POST("/spending-trends", h.SpendingTrends)`

## Request
- Auth: required (both)
- Body (Python `AnalysisRequestSchema`): start_date (date, required), end_date (date, required), limit (int, optional, default -1)
- Body (Go `AnalysisRequest`): startDate (time.Time, required via validate tag), endDate (time.Time, required via validate tag), limit (*int, optional)

## Response
- **Python**: 404 `"Not found"` (endpoint is disabled)
- **Go**: 404 `"Not found"` (endpoint is disabled)

## Error Responses
| Scenario | Python | Go |
|---|---|---|
| Unauthorized | 401 (via check_token) | 401 (via auth middleware) |
| Not found (disabled) | 404 `"Not found"` | 404 `"Not found"` |

## Business Logic Comparison
1. Both implementations immediately return 404 "Not found".
2. Python has the actual AI-powered implementation code below the `raise HTTPException` (unreachable code), using `ExpenseAnalyzer` service.
3. Go has a simple TODO comment.
4. Python would use OpenAI/AI service for analysis; Go has no such implementation.
5. Neither endpoint is functional.

## Notes
- Both endpoints are intentionally disabled with TODO comments indicating future redesign.
- Python has dead code below the 404 raise showing the intended AI integration.
- Go DTOs (AnalysisRequest, AnalysisResponse) are defined but unused since the handler returns immediately.
- Go now has `RequireActiveUser` middleware applied to all authenticated route groups, including analytics.
- Python has the `check_token` dependency but no `enforce_free_plan_compliance`.

## Tests
- **Python**: 2 tests (test_spending_trends_returns_404, _unauthorized)
- **Go integration tests**: 6 tests in `handler_test.go` (TestSpendingTrendsNotFound, TestSpendingTrendsUnauthorized, TestSpendingTrendsInvalidBody, TestSpendingTrendsInvalidToken, TestSpendingTrendsEmptyBody, TestSpendingTrendsWithAccountID)
- **Go unit tests**: 0 (no unit test file for analytics)
