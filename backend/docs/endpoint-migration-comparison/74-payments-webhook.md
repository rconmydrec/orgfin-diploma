# Endpoint #74: POST /payments/webhook

## Route Definition
- **Python**: `@router.post("/webhook")` (no auth dependency -- public)
- **Go**: `g.POST("/webhook", h.Webhook)` (registered outside protected group -- public)

## Request
- **Auth**: Not required (both). Uses Stripe signature verification instead.
- **Headers**: `Stripe-Signature` / `stripe-signature`
- **Body**: Raw payload (Stripe event JSON)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Missing signature | 400 | 200 (with success=false) |
| Invalid signature | 400 | 200 (with success=false) |
| Processing error | 200 (to prevent retry) | 200 (always) |
| Read body error | N/A | 200 (with success=false) |

### Response Body Structure
- Python: `{"received": True, "success": True, "event_type": "...", ...}` (on success) or `{"received": True, "error": "..."}` (on internal error)
- Go: `WebhookResponse` -- `received` (bool), `success` (bool), `eventType` (string), `error` (string pointer)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing signature | 400 ("Missing stripe-signature header") | 200 with success=false |
| Invalid signature | 400 ("Invalid webhook signature") | 200 with success=false |
| WebhookValidationError | 400 | 200 with success=false |
| WebhookSignatureError | 400 | 200 with success=false |
| Processing error | 200 with error field | 200 (errors logged server-side) |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Read body | `await request.body()` (async) | `io.ReadAll(io.LimitReader(body, 64KB))` |
| 2. Check signature | Required: 400 if missing | Part of VerifyWebhookSignature |
| 3. Validate signature | `webhook_handler.validate_webhook_signature(payload, signature)` | `provider.VerifyWebhookSignature(payload, signature)` via Stripe SDK `webhook.ConstructEvent()` |
| 4. Process event | `webhook_handler.handle_event(event, db)` | `webhookHandler.HandleWebhook(payload, signature)` -- routes to 7 event handlers |
| 5. Error handling | Validation errors -> 400; Processing errors -> 200 | Always 200; failures logged internally |

## Handled Event Types

| Event Type | Python | Go |
|---|---|---|
| `checkout.session.completed` | Yes | Yes |
| `customer.subscription.updated` | Yes | Yes |
| `customer.subscription.deleted` | N/A | Yes |
| `invoice.paid` | Yes | Yes |
| `invoice.payment_failed` | Yes | Yes |
| `subscription_schedule.completed` | Yes | Yes |
| `subscription_schedule.released` | N/A | Yes |

## Notes
- **Stripe webhook handling is now fully implemented in Go.** The `WebhookHandler` processes 7 event types with full state management.
- Go handles body size limits (64KB) to prevent memory exhaustion; Python does not.
- Go always returns HTTP 200 to prevent Stripe from retrying, with `success: false` indicating signature verification failure. Python returns 400 for signature issues.
- Go handles two additional event types not in Python: `customer.subscription.deleted` (downgrades to free) and `subscription_schedule.released` (clears schedule on cancellation).
- Both implementations log processing errors but return 200 to prevent Stripe from retrying.

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 6 (TestWebhookEndpoint class)
- Tests: checkout_completed, subscription_updated, invalid_signature, missing_signature, unknown_event, internal_error

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 2
- Tests: TestWebhookSuccess, TestWebhookEmptyBody

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 2
- Tests: TestWebhookSuccess, TestWebhookMissingSignature
