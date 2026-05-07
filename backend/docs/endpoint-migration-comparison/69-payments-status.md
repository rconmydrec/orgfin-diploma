# Endpoint #69: GET /payments/status

## Route Definition
- **Python**: `@router.get("/status", response_model=PaymentStatusResponse, dependencies=[Depends(check_token)])`
- **Go**: `protected.GET("/status", h.GetStatus)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Params**: None (both)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success with subscription | 200 | 200 |
| Success without subscription | 404 | 200 (returns free defaults) |
| Unauthorized | 500 (dependency chain) | 401 |
| User not active | N/A | 401 |

### Response Body Structure
Both return `PaymentStatusResponse`:
- `hasStripeCustomer` / `has_stripe_customer` (bool)
- `stripeSubscriptionActive` / `stripe_subscription_active` (bool)
- `stripeSubscriptionId` / `stripe_subscription_id` (string, nullable)
- `currentPlanType` / `current_plan_type` (string)
- `lastPaymentFailed` / `last_payment_failed` (bool)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 | 401 |
| User not active | N/A | 401 |
| No subscription | 404 | 200 (free plan defaults) |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Get subscription | `user.subscription` (ORM relationship) | `subscriptionRepo.GetByUserID(userID)` |
| 3. No subscription | Returns 404 | Returns 200 with free defaults (hasStripeCustomer=false, etc.) |
| 4. Provider data | Checks `subscription.payment_provider_data` for customer/subscription IDs | Fetches `PaymentProviderSubscription` via `GetProviderSubscription()` |
| 5. Has customer | From provider_data.external_customer_id | `ExternalCustomerID != nil && != ""` |
| 6. Sub active | From provider_data.external_subscription_id | `ExternalSubscriptionID != nil && != ""` |
| 7. Sub ID | From provider_data.external_subscription_id | `providerSub.ExternalSubscriptionID` |
| 8. Payment failed | From `provider_data.last_payment_failed` | From `providerSub.LastPaymentFailed` (set/cleared by webhook events) |

## Notes
- **Payment status is now fully implemented in Go.** The `PaymentService.GetPaymentStatus()` reads real data from the `payment_provider_subscriptions` table.
- **Behavioral difference preserved**: Python returns 404 when user has no subscription. Go returns 200 with free-plan defaults. This is intentional.
- `lastPaymentFailed` is now functional in Go -- it is set by `invoice.payment_failed` webhooks and cleared by `invoice.paid` webhooks.
- `stripeSubscriptionId` is now populated from the `ExternalSubscriptionID` field of the provider subscription record.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 4 (TestGetPaymentStatusEndpoint class)
- Tests: success_no_subscription, success_with_subscription, unauthorized, no_subscription_404

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 3
- Tests: TestGetPaymentStatusSuccess, TestGetPaymentStatusUnauthorized, TestGetPaymentStatusReturnsCurrentPlan, TestGetPaymentStatusWithSubscription

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 2
- Tests: TestGetStatusNoSubscription, TestGetStatusWithSubscription
