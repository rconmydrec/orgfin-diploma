# Scenario S5: Subscription Webhooks

## Description
Tests that Stripe webhook events are correctly received and processed. Validates the full lifecycle of webhook-driven subscription state changes: checkout completion, subscription updates, invoice payment failures, and successful invoice payments.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_subscription_webhooks_scenario.py`
- **Test count**: 1 (large multi-step test in `TestSubscriptionWebhooksScenario` class)
- **Key assertions**:
  1. `test_subscription_webhooks_update_state`:
     - Registers user, expires trial
     - Creates checkout session -> sends `checkout.session.completed` webhook
     - Verifies: plan_id updated, expires_at set, provider_data has customer_id and subscription_id
     - Sends `customer.subscription.updated` webhook -> verifies is_active=True, expires_at updated
     - Sends `invoice.payment_failed` webhook -> verifies `provider_data.last_payment_failed = True`
     - Sends `invoice.paid` webhook -> verifies `provider_data.last_payment_failed = False`
     - All webhooks return 200 with `success: True` and correct `event_type`

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/subscription_webhooks_test.go`
- **Test count**: 5
- **Key assertions**:
  1. `TestWebhookReceivesAndAcknowledges`:
     - Sends checkout.session.completed payload -> verifies `received: true`, `success: true`
     - No Stripe signature required (non-production)
  2. `TestWebhookWithStripeSignatureHeader`:
     - Sends payload with Stripe-Signature header -> verifies `received: true`
  3. `TestWebhookMultipleEventTypes`:
     - Sends 5 event types (checkout.session.completed, customer.subscription.updated, invoice.payment_failed, invoice.paid, subscription_schedule.completed)
     - Each returns 200 with `received: true`
  4. `TestSubscriptionStateUpdateViaUpgradeAndWebhook`:
     - Upgrades to premium via `/subscriptions/upgrade`
     - Verifies status is premium
     - Sends checkout.session.completed webhook
     - Verifies webhook acknowledged (success: true)
     - Verifies subscription state is still premium (webhook did not corrupt it)
  5. `TestPaymentStatusAfterUpgrade`:
     - Verifies initial payment status is free
     - Upgrades to premium
     - Verifies payment status reflects premium (case-insensitive)

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| checkout.session.completed webhook | Yes (full state update) | Yes (acknowledgement + state preservation) |
| customer.subscription.updated webhook | Yes (updates expires_at, is_active) | Yes (acknowledgement) |
| customer.subscription.deleted webhook | No | No (Go handles this event but no scenario test) |
| invoice.payment_failed webhook | Yes (sets last_payment_failed) | Yes (acknowledgement) |
| invoice.paid webhook | Yes (clears last_payment_failed) | Yes (acknowledgement) |
| subscription_schedule.completed | No (in S4 scenario) | Yes (acknowledgement) |
| subscription_schedule.released | No | No (Go handles this event but no scenario test) |
| Webhook signature validation | Yes (via mock) | No (integration tests run in non-production mode) |
| Provider data verification | Yes (customer_id, subscription_id) | No |
| State change via webhook | Yes (plan changes, expires_at) | No (scenario tests verify acknowledgement only) |
| Webhook does not break state | N/A | Yes |
| Payment status after upgrade | N/A | Yes |

### Differences in Assertions
- **Python tests actual state changes**: Each webhook modifies the subscription state and the test verifies the DB changes (plan_id, expires_at, last_payment_failed).
- **Go scenario tests verify acknowledgement and state preservation**: The webhook handler is now fully functional (processes 7 event types with real state changes), but the integration scenario tests have not yet been updated to verify state changes through webhooks with mocked Stripe signature verification.
- Go has a unique test (`TestSubscriptionStateUpdateViaUpgradeAndWebhook`) that verifies the webhook does not corrupt existing subscription state.
- Go tests multiple event types in a sub-test loop; Python tests them sequentially in one large test.

### Missing Scenarios
- Go scenario tests do not yet verify that webhook events cause the expected DB state changes (plan_id, expires_at, last_payment_failed, etc.), despite the webhook handler being fully implemented at the service level.
- Go does not test webhook signature validation/rejection in scenarios (the `WebhookHandler` does verify signatures via `provider.VerifyWebhookSignature`, but scenario tests run without valid Stripe signatures).
- Python does not test the "webhook does not break state" scenario (not needed since webhooks actually work).
- Go handles `customer.subscription.deleted` and `subscription_schedule.released` events but has no scenario tests for them.

## Notes
- **The Go webhook handler is now fully functional** (`internal/services/payment/webhook_handler.go`) and processes 7 Stripe event types with real state management. The gap is in scenario-level test coverage, not in implementation.
- The Go `WebhookHandler` verifies Stripe signatures cryptographically via `stripe.webhook.ConstructEvent()`. Scenario tests would need to mock signature generation to test this flow end-to-end.
- Expanding Go's scenario tests to send properly-structured webhook payloads and verify resulting DB state changes would close the coverage gap with Python.
