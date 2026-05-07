# Endpoint #68: POST /payments/portal

## Route Definition
- **Python**: `@router.post("/portal", response_model=PortalResponse, dependencies=[Depends(check_token)])`
- **Go**: `protected.POST("/portal", h.CreatePortal)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Body**:
  - Python: `PortalRequest | None` -- `return_url` (string, optional)
  - Go: `PortalRequest` -- `returnUrl` (string pointer, optional)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 500 (dependency chain) | 401 |
| User not active | N/A | 401 |
| No Stripe customer | 400 (CheckoutSessionError) | 400 (ErrNoStripeCustomer) |
| Internal error | 500 | 500 |
| Invalid body | 422 | 422 |

### Response Body Structure
Both return `PortalResponse`:
- `portalUrl` / `portal_url` (string)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 | 401 |
| User not active | N/A | 401 |
| No Stripe customer | 400 | 400 |
| CheckoutSessionError | 400 | 400 |
| Generic exception | 500 | 500 |
| Invalid body | 422 | 422 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Parse body | Optional Pydantic model | Bind (with PortalRequest) |
| 3. Get customer | From subscription.payment_provider_data | `subscriptionRepo.GetProviderSubscription()` -> check `ExternalCustomerID` |
| 4. Validate return URL | N/A | Must start with `FRONTEND_URL` (prevents open redirect) |
| 5. Create session | `checkout_service.create_portal_session(user, db, return_url)` | `provider.CreatePortalSession(customerID, returnURL)` via Stripe SDK |
| 6. Error handling | CheckoutSessionError(400), Exception(500) | ErrNoStripeCustomer(400), internal errors(500) |

## Notes
- **Stripe Billing Portal integration is now complete in Go.** The handler delegates to `PaymentService.CreatePortal()` which creates a real Stripe Billing Portal session.
- Go validates the `returnUrl` parameter to prevent open redirects (must start with `FRONTEND_URL`). Python does not have this validation.
- Python allows `None` body (portal data is optional); Go binds into `PortalRequest` but both handle the optional `returnUrl`.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 4 (TestCreatePortalEndpoint class)
- Tests: success, no_customer, unauthorized, internal_error

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 3
- Tests: TestCreatePortalSuccess, TestCreatePortalUnauthorized, TestCreatePortalWithReturnURL

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 2
- Tests: TestCreatePortalBindError, TestCreatePortalSuccess
