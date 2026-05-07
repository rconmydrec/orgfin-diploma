# Endpoint #66: POST /subscriptions/downgrade

## Route Definition
- **Python**: `@router.post('/downgrade')` (router has `dependencies=[Depends(check_token)]`)
- **Go**: `protected.POST("/downgrade", h.Downgrade)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**:
  - Python: `DowngradeRequest` -- `account_ids` (list[int]), `budget_id` (int, optional)
  - Go: `DowngradeRequest` -- `accountIds` ([]int, json:"accountIds"), `budgetId` (*int, json:"budgetId")

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 422 | 401 |
| User not active | N/A | 401 |
| No subscription | 400 ("already on free") | 404 ("No subscription found") |
| Free plan not found | 500 (service) | 500 |
| DB error | 500 | 500 |
| Invalid body | 422 | 422 |

### Response Body Structure
- Python: Returns `asdict(result)` from `downgrade_to_free` -- `message`, `expires_at`, `pending_downgrade`, `pending_plan_id`
- Go: `DowngradeResult` -- `message`, `expiresAt`, `pendingDowngrade`, `pendingPlanId`

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 422 | 401 |
| User not active | N/A | 401 |
| No subscription | 400 (already free) | 404 |
| Free plan DB error | 500 | 500 |
| Update DB error | 500 | 500 |
| Invalid body | 422 | 422 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token dependency | authMiddleware + RequireActiveUser middleware |
| 2. Parse body | Pydantic model | Bind |
| 3. Validate state | Service checks if user is already on free plan (returns 400) | `subscriptionSvc.ScheduleDowngrade()` checks effective plan type, returns `ErrAlreadyOnFreePlan` |
| 4. Get free plan | Service handles | Service calls `planRepo.GetFreePlan()` |
| 5. Validate entity selection | Service validates account count against free plan limits | Service validates `len(accountIDs) <= freeLimits.Accounts`, returns `ErrInvalidEntitySelection` |
| 6. Schedule downgrade | Service stores account_ids, budget_id, sets pending fields | Service stores `PendingPlanID`, `PendingDowngradeAccountIDs`, `PendingDowngradeBudgetID` via `UpdateSubscriptionFull` |
| 7. Account deactivation | Python service deactivates non-selected accounts | If no Stripe subscription: immediate downgrade + `ApplyDowngradeEntitySelection` (archives accounts/budgets). If Stripe: scheduled for billing period end. |
| 8. Response | Returns DowngradeResult dataclass as dict | Returns DowngradeResult struct (`scheduled`, `message`, `expiresAt`) |

## Notes
- Both Python and Go now fully implement entity selection and archiving on downgrade. Go uses `AccountArchiver.ArchiveByUserExcept` and `BudgetArchiver.ArchiveByUserExcept` interfaces.
- Go differentiates between immediate downgrade (no Stripe subscription) and scheduled downgrade (with Stripe subscription, deferred to billing period end).
- **Different error for no subscription**: Python returns 400 ("already on free plan"), Go returns 400 ("Already on free plan") via `ErrAlreadyOnFreePlan`.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_subscriptions_endpoints.py`
- **Test count**: 3 (TestDowngradeSubscriptionEndpoint class)
- Tests: success, already_free, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_test.go`
- **Test count**: 4
- Tests: TestDowngradeNoSubscription, TestDowngradeUnauthorized, TestDowngradeInvalidBody, TestDowngradeWithAccountIds, TestDowngradeWithSubscription

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/subscriptions/handler_unit_test.go`
- **Test count**: 5
- Tests: TestDowngradeBindError, TestDowngradeNoSubscription, TestDowngradeFreePlanError, TestDowngradeUpdateError, TestDowngradeSuccess
