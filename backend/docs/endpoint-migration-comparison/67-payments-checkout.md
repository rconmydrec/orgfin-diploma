# Endpoint #67: POST /payments/checkout

## Route Definition
- **Python**: `@router.post("/checkout", response_model=CheckoutResponse, dependencies=[Depends(check_token)])`
- **Go**: `protected.POST("/checkout", h.CreateCheckout)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**:
  - Python: `CheckoutRequest` -- `plan_id` (int)
  - Go: `CheckoutRequest` -- `planId` (int, `validate:"required"`)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 500 (dependency chain issue) | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| Invalid price | 400 | 400 |
| Stripe error | 500 | 400 (ErrCheckoutSessionFailed) |
| Validation error | 422 | 422 |

### Response Body Structure
Both return `CheckoutResponse`:
- `checkoutUrl` / `checkout_url` (string)
- `sessionId` / `session_id` (string)
- `creditAppliedCents` / `credit_applied_cents` (int)
- `creditAppliedFormatted` / `credit_applied_formatted` (string)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 (internal) | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| InvalidPriceError | 400 | 400 |
| CheckoutSessionError | 500 | 400 |
| Generic exception | 500 | 500 |
| Invalid body | 422 | 422 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Get plan | `get_active_plan_by_id(plan_id, db)` | `planRepo.GetByID(req.PlanID)` |
| 3. Look up Stripe price | Implicit in checkout service | `planPriceRepo.GetByPlanAndProvider(planID, "STRIPE")` via `plan_prices` table |
| 4. Get/create customer | `checkout_service.get_or_create_customer()` | `paymentSvc.GetOrCreateCustomer(userID, email, name)` |
| 5. Calculate credit | `checkout_service.calculate_credit()` | `paymentSvc.calculateCurrentCredit(userID)` via `credit.CreditCalculator` |
| 6. Apply credit | Sets Stripe customer balance | `provider.UpdateCustomerBalance(customerID, creditCents)` |
| 7. Create session | `checkout_service.create_checkout_session()` | `provider.CreateCheckoutSession(params)` |
| 8. Error handling | Three error types: InvalidPriceError(400), CheckoutSessionError(500), Exception(500) | Service errors mapped via `mapServiceError()` |
| 9. Response | Real Stripe checkout URL, session ID, credit info | Real Stripe checkout URL, session ID, credit info |

## Notes
- **Stripe integration is now complete in Go.** The handler delegates to `PaymentService` which uses the `PaymentProvider` interface (implemented by `StripeClient`).
- Go uses a separate `plan_prices` table to map plans to Stripe price IDs, while Python stored price IDs differently.
- Go uses `credit.CreditCalculator` for prorated credit calculations (daily rate * days remaining).
- Python's unauthorized response is 500 due to dependency chain issue (get_current_user fails before check_token validates); Go returns clean 401.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 7 (TestCreateCheckoutEndpoint class)
- Tests: success_new_customer, success_existing_customer, plan_not_found, stripe_error, invalid_price_error, checkout_session_error, generic_exception, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 4
- Tests: TestCreateCheckoutInvalidPlan, TestCreateCheckoutMissingPlanID, TestCreateCheckoutUnauthorized, TestCreateCheckoutInvalidBody, TestCreateCheckoutWithValidPlan

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 6
- Tests: TestCreateCheckoutUserNotActive, TestCreateCheckoutUserRepoError, TestCreateCheckoutUserDeleted, TestCreateCheckoutBindError, TestCreateCheckoutValidateError, TestCreateCheckoutPlanNotFound, TestCreateCheckoutSuccess
