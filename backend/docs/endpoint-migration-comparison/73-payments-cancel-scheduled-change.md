# Endpoint #73: POST /payments/cancel-scheduled-change

## Route Definition
- **Python**: `@router.post("/cancel-scheduled-change", dependencies=[Depends(check_token)])`
- **Go**: `protected.POST("/cancel-scheduled-change", h.CancelScheduledChange)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 500 (dependency chain) | 401 |
| User not active | N/A | 401 |
| No subscription | 400 | 404 |
| No pending change | N/A (handled in service) | 400 ("No scheduled change to cancel") |
| SubscriptionChangeError | 400 | N/A |
| Internal error | 500 | 500 |

### Response Body Structure
- Python: `asdict(result)` -- `success` (bool), `message` (string)
- Go: `map[string]interface{}` -- `success` (bool), `message` (string)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 | 401 |
| User not active | N/A | 401 |
| No subscription | 400 | 404 |
| No pending change | 400 (SubscriptionChangeError from service) | 400 |
| Update DB error | 500 | 500 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Get subscription | `user.subscription` (ORM) | `subscriptionRepo.GetByUserID(userID)` |
| 3. No subscription | Returns 400 | Returns 404 |
| 4. Cancel schedule | `checkout_service.cancel_scheduled_plan_change(subscription, db)` -- releases Stripe schedule + re-enables subscription | Clears `PendingPlanID` to nil |
| 5. Stripe operations | Releases Stripe Schedule, re-enables cancel_at_period_end | N/A (no Stripe integration) |
| 6. Clear pending fields | Service clears pending_plan_id, downgrade fields | Only clears PendingPlanID |

## Notes
- **Critical difference**: Python cancels actual Stripe schedules and/or re-enables subscriptions that were set to cancel at period end. Go only clears the `PendingPlanID` field in the database.
- **Different error for no subscription**: Python returns 400, Go returns 404.
- Go explicitly checks `subscription.PendingPlanID == nil` and returns 400 before attempting update. Python delegates this check to the service.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 6 (TestCancelScheduledChangeEndpoint class)
- Tests: success, no_subscription, unauthorized, internal_error, subscription_error, success_mocked

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 3
- Tests: TestCancelScheduledChangeNoSubscription, TestCancelScheduledChangeUnauthorized, TestCancelScheduledChangeNoPendingChange, TestCancelScheduledChangeSuccess

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 4
- Tests: TestCancelScheduledChangeNoSubscription, TestCancelScheduledChangeNoPendingPlan, TestCancelScheduledChangeUpdateError, TestCancelScheduledChangeSuccess
