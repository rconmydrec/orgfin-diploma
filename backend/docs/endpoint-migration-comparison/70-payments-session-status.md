# Endpoint #70: GET /payments/session/:session_id

## Route Definition
- **Python**: `@router.get("/session/{session_id}", response_model=CheckoutSessionStatus, dependencies=[Depends(check_token)])`
- **Go**: `protected.GET("/session/:session_id", h.GetSessionStatus)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Path params**: `session_id` (string)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 422 | 401 |
| User not active | N/A | 401 |
| Session not found | 404 (CheckoutSessionError) | 400 ("Checkout session creation failed") |
| Session not owned by user | N/A | 403 ("Session not found") |
| Internal error | 500 | 500 |

### Response Body Structure
Both return `CheckoutSessionStatus`:
- `sessionId` / `session_id` (string)
- `status` (string)
- `paymentStatus` / `payment_status` (string)
- `customerId` / `customer_id` (string)
- `subscriptionId` / `subscription_id` (string)
- `amountTotal` / `amount_total` (int)
- `currency` (string)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 422 | 401 |
| User not active | N/A | 401 |
| Session not found | 404 | 400 |
| Session belongs to different user | N/A | 403 |
| Internal error | 500 | 500 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token dependency | authMiddleware + RequireActiveUser middleware |
| 2. Get session | `checkout_service.get_checkout_session_status(session_id)` -- retrieves from Stripe | `paymentSvc.GetSessionStatus(userID, sessionID)` -- retrieves from Stripe via PaymentProvider |
| 3. Ownership check | N/A | Verifies session metadata `user_id` matches authenticated user |
| 4. Error handling | CheckoutSessionError(404), Exception(500) | ErrSessionNotFound(403), ErrCheckoutSessionFailed(400) |

## Notes
- Go adds an ownership check that Python lacks: session metadata `user_id` is verified against the authenticated user, preventing IDOR.
- Both now retrieve real session data from Stripe.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 4 (TestGetCheckoutSessionStatusEndpoint class)
- Tests: success, not_found, unauthorized, internal_error

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 3
- Tests: TestGetSessionStatusSuccess, TestGetSessionStatusUnauthorized, TestGetSessionStatusDifferentSession

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 1
- Tests: TestGetSessionStatus
