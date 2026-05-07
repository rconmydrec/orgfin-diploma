# Endpoint #72: POST /payments/change-plan

## Route Definition
- **Python**: `@router.post("/change-plan", response_model=ChangePlanResponse | CheckoutResponse, dependencies=[Depends(check_token)])`
- **Go**: `protected.POST("/change-plan", h.ChangePlan)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**:
  - Python: `ChangePlanRequest` -- `plan_id` (int)
  - Go: `ChangePlanRequest` -- `planId` (int, `validate:"required"`)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success (plan changed, upgrade) | 200 (ChangePlanResponse) | 200 (ChangePlanResponse) |
| Success (plan changed, downgrade) | 200 (ChangePlanResponse) | 200 (ChangePlanResponse with scheduleId) |
| Unauthorized | 500 (dependency chain) | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| No Stripe subscription | N/A | 400 |
| Invalid price | 400 | 400 |
| Validation error | 422 | 422 |
| Internal error | 500 | 500 |

### Response Body Structure (ChangePlanResponse)
- `success` (bool)
- `isUpgrade` / `is_upgrade` (bool)
- `newPlanName` / `new_plan_name` (string)
- `effectiveDate` / `effective_date` (string -- RFC3339 timestamp)
- `message` (string)
- `scheduleId` / `schedule_id` (string, nullable -- set for downgrades)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| No Stripe subscription | N/A | 400 (ErrNoStripeSubscription) |
| Invalid price | 400 | 400 (ErrInvalidPrice) |
| Generic exception | 500 | 500 |
| Invalid body | 422 | 422 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Get plan | `get_active_plan_by_id(plan_id, db)` | `planRepo.GetByID(req.PlanID)` |
| 3. Get Stripe price | Implicit in checkout service | `planPriceRepo.GetByPlanAndProvider(planID, "STRIPE")` |
| 4. Determine direction | Compares billing period durations | Compares `BillingPeriod.DurationDays` (target < current = downgrade) |
| 5. Upgrade (immediate) | Modifies Stripe subscription with proration | `provider.UpdateSubscription(subID, proration_behavior=create_prorations)` + updates local DB |
| 6. Downgrade (scheduled) | Creates Stripe Schedule for deferred change | Creates Stripe Subscription Schedule with two phases (current until period end, then target) + stores schedule ID |
| 7. Error handling | SubscriptionChangeError(400), InvalidPriceError(400), Exception(500) | Mapped via `mapServiceError()` |

## Notes
- **Plan change with Stripe is now fully implemented in Go.** The handler delegates to `PaymentService.ChangePlan()` which handles both upgrades and downgrades.
- **Upgrades** (target billing period >= current): Applied immediately with Stripe prorations via `UpdateSubscription()`. Local DB record is updated with the new plan.
- **Downgrades** (target billing period < current): Scheduled via Stripe Subscription Schedule. Two phases: (1) current plan until `current_period_end`, (2) target plan after. Schedule ID is stored on the provider subscription record. Pending plan ID is stored on the subscription.
- Go requires an active Stripe subscription (`ExternalSubscriptionID` must exist); returns 400 if not present.
- `isUpgrade` now correctly reflects the direction of the plan change (true for upgrades, false for scheduled downgrades).
- `effectiveDate` is an RFC3339 timestamp: `time.Now()` for upgrades, `current_period_end` for downgrades.
- `scheduleId` is populated for downgrades (Stripe schedule ID).

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 7 (TestChangePlanEndpoint class)
- Tests: upgrade_success, same_plan_error, unauthorized, not_found, invalid_price_error, internal_error, success

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 5
- Tests: TestChangePlanInvalidPlan, TestChangePlanUnauthorized, TestChangePlanMissingPlanID, TestChangePlanInvalidBody, TestChangePlanWithValidPlanNoSubscription, TestChangePlanWithExistingSubscription

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 6
- Tests: TestChangePlanBindError, TestChangePlanValidateError, TestChangePlanNotFound, TestChangePlanNoSubscription, TestChangePlanUpdateError, TestChangePlanSuccess
