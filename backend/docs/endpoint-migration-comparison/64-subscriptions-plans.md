# Endpoint #64: GET /subscriptions/plans

## Route Definition
- **Python**: `@router.get('/plans', response_model=list[SubscriptionPlanResponse], dependencies=[])` (note: empty dependencies override, but router-level `check_token` still applies)
- **Go**: `g.GET("/plans", h.GetPlans)` (registered outside the `protected` group -- public route)

## Request
- **Auth**: Python: Required (router-level dependency overrides `dependencies=[]`). Go: Not required (public route).
- **Params**: None (both)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 422 (Python requires auth) | N/A (public) |
| DB error | 500 (unhandled) | 500 |

### Response Body Structure
Both return `list[SubscriptionPlanResponse]`:
- `id` (int)
- `name` (string)
- `translationKey` (string, nullable)
- `planType` (string)
- `billingPeriod` (object, nullable) -- `{id, code, name, durationDays}`
- `price` (decimal)
- `currencyCode` (string)
- `isFeatured` (bool)
- `description` (string, nullable)
- `sortOrder` (int)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth token | 422 | N/A (public) |
| DB error | 500 (unhandled exception) | 500 ("Failed to get subscription plans") |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | Required (router-level dependency) | Not required (public) |
| 2. Get plans | `get_available_plans_service(db)` | `planRepo.GetActivePlans()` |
| 3. Map response | Service returns schema-compatible objects | Manual mapping with plan type switch and billing period conversion |
| 4. Plan type mapping | Automatic via enum | Switch on `strings.ToLower`: free, trial, premium, default->free |

## Notes
- **Critical difference**: In Python, despite `dependencies=[]` at the route level, the router-level `dependencies=[Depends(check_token)]` still applies, making this endpoint require auth. In Go, this is correctly a public endpoint (registered outside the protected group).
- Go has more explicit plan type mapping with an "unknown" fallback to "free".
- Python returns plans directly from the service; Go manually constructs the response DTOs.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_subscriptions_endpoints.py`
- **Test count**: 4 (TestGetPlansEndpoint class)
- Tests: success, includes_required_info, includes_free_and_premium, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_test.go`
- **Test count**: 2
- Tests: TestGetPlansSuccess, TestGetPlansReturnsArray

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_unit_test.go`
- **Test count**: 3
- Tests: TestGetPlansDBError, TestGetPlansWithBillingPeriod, TestGetPlansAllTypes
