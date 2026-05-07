# Endpoint #65: POST /subscriptions/upgrade

## Route Definition
- **Python**: `@router.post('/upgrade')` (router has `dependencies=[Depends(check_token)]`)
- **Go**: `protected.POST("/upgrade", h.Upgrade)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**:
  - Python: `UpgradeRequest` -- `plan_id` (int, required), `adjusted_price` (float, optional)
  - Go: `UpgradeRequest` -- `planId` (int, required via `validate:"required"`), `adjustedPrice` (decimal, optional)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 422 | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 (via service exception) | 404 |
| Validation error | 422 | 422 |
| DB error | 500 | 500 |

### Response Body Structure
Both return:
- `message` (string)
- `subscription` (object)
- `expiresAt` / `expires_at` (datetime, nullable)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 422 | 401 |
| User not active | N/A | 401 |
| Invalid body | 422 | 422 |
| Missing planId | 422 | 422 |
| Plan not found | 404 | 404 |
| Create/Update DB error | 500 (unhandled) | 500 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token dependency | authMiddleware + RequireActiveUser middleware |
| 2. Parse body | Pydantic model validation | Bind + Validate |
| 3. Upgrade | `upgrade_subscription(user_id, plan_id, adjusted_price, db)` service | Inline: get plan, get/create subscription, update |
| 4. No existing sub | Service handles internally | Creates new subscription via `subscriptionRepo.Create` |
| 5. Existing sub | Service handles credit/expiry logic | Updates plan_id, plan_type, subscribedAt |
| 6. Response | Returns message + subscription + expires_at | Returns UpgradeResponse with message + subscription + expiresAt |

## Notes
- **Key difference**: Python's `upgrade_subscription` service handles credit calculation and expiry recalculation. Go's implementation is simpler -- it just updates the plan without credit/price adjustments.
- **Key difference**: Python accepts `adjusted_price` and factors it into the upgrade logic. Go accepts it in the DTO but does not use it.
- Go sets `SubscribedAt` to `time.Now()` on update; Python handles this in the service layer.
- Go response key is `expiresAt` (camelCase); Python uses `expires_at` (snake_case).

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_subscriptions_endpoints.py`
- **Test count**: 3 (TestUpgradeSubscriptionEndpoint class)
- Tests: success, invalid_plan, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_test.go`
- **Test count**: 5
- Tests: TestUpgradeInvalidPlan, TestUpgradeMissingPlanID, TestUpgradeUnauthorized, TestUpgradeInvalidBody, TestUpgradeToExistingPlan, TestUpgradeExistingSubscription

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_unit_test.go`
- **Test count**: 6
- Tests: TestUpgradeBindError, TestUpgradeValidateError, TestUpgradePlanNotFound, TestUpgradeNewSubscription, TestUpgradeNewSubscriptionCreateError, TestUpgradeExistingSubscription, TestUpgradeExistingSubscriptionUpdateError
