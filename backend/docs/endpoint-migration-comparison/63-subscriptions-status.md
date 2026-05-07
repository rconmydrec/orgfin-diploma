# Endpoint #63: GET /subscriptions/status

## Route Definition
- **Python**: `@router.get('/status', response_model=SubscriptionStatusResponse)` (router has `dependencies=[Depends(check_token)]`)
- **Go**: `protected.GET("/status", h.GetStatus)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Params**: None (both)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 422 (missing header) | 401 |
| User not active | N/A (no explicit check) | 401 ("User not activated") |
| No subscription | 200 (returns status) | 200 (returns free status) |

### Response Body Structure
Both return `SubscriptionStatusResponse`:
- `planType` (string) -- Python uses enum, Go uses string constants
- `isActive` (bool)
- `planId` (int, nullable)
- `trialDaysRemaining` (int, nullable)
- `requiresDowngrade` (bool)
- `limits` (map/dict, nullable)
- `subscribedAt` (datetime, nullable)
- `expiresAt` (datetime, nullable)
- `pendingPlanId` (int, nullable)
- `pendingPlanName` (string, nullable)
- `hasStripeSubscription` (bool)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth token | 422 | 401 |
| User not active/deleted | N/A | 401 |
| DB error getting user | N/A | 401 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token dependency | authMiddleware |
| 2. User active check | Not explicitly checked | RequireActiveUser middleware (is_active, is_deleted) |
| 3. Get subscription | `get_subscription_status_data(user_id, db)` service call | `subscriptionSvc.GetStatus(userID)` via SubscriptionService |
| 4. No subscription | Returns status via service (handles internally) | Returns free status with PlanLimits from service |
| 5. Plan type mapping | Uses Python enum (PlanType) | Service uses `GetEffectivePlanType` (accounts for expired trials/premium) |
| 6. Trial days calc | Handled in service layer | Service: `math.Ceil(time.Until(*trialEndsAt).Hours() / 24)`, clamps to 0 |
| 7. Limits | Returned from service | Set from `PlanLimits` map (free={accounts:2, budgets:1, planningDays:14}, trial/premium=nil) |
| 8. Pending plan | Returned from service | Service fetches pending plan name via `planRepo.GetByID` |
| 9. RequiresDowngrade | Checked in service | Service checks account/budget counts against free plan limits |

## Notes
- Both Python and Go now delegate all logic to a service layer (`SubscriptionService`).
- **Key difference**: Go adds `RequireActiveUser` middleware guard that Python lacks at this endpoint level.
- **Key difference**: Python returns 422 for missing auth header (FastAPI validation); Go returns 401 (middleware).
- `RequiresDowngrade` is fully implemented — checks `AccountCounter.CountActiveByUserID` and `BudgetCounter.CountActiveByUserID` against free plan limits.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_subscriptions_endpoints.py`
- **Test count**: 4 (TestGetSubscriptionStatusEndpoint class)
- Tests: success_free, success_premium, with_limits, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_test.go`
- **Test count**: 6
- Tests: TestGetStatusSuccess, TestGetStatusFreeUser, TestGetStatusUnauthorized, TestGetStatusHasLimits, TestGetStatusIsActive, TestGetStatusWithPremiumSubscription, TestGetStatusWithTrialSubscription, TestGetStatusWithPendingDowngrade

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_unit_test.go`
- **Test count**: 8
- Tests: TestGetStatusUserNotActive, TestGetStatusUserRepoError, TestGetStatusUserDeleted, TestGetStatusNoSubscription, TestGetStatusWithTrial, TestGetStatusTrialExpired, TestGetStatusWithPendingPlan, TestGetStatusPendingPlanError, TestGetStatusAllPlanTypes
