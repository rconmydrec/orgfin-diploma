# Payments Endpoints

Endpoints for Stripe-based payment flows: checkout sessions, customer portal, payment status, plan changes, and webhook processing.

All payment-related business logic is handled by the `PaymentService` (`internal/services/payment/service.go`), which coordinates between the `PaymentProvider` interface (Stripe SDK client), subscription repositories, the credit calculator, and the plan price repository. The HTTP handler is a thin adapter that delegates to the service layer.

## Table of Contents

- [POST /payments/checkout](#post-paymentscheckout)
- [POST /payments/portal](#post-paymentsportal)
- [GET /payments/status](#get-paymentsstatus)
- [GET /payments/session/:session_id](#get-paymentssessionsession_id)
- [GET /payments/upgrade-price/:plan_id](#get-paymentsupgrade-priceplan_id)
- [POST /payments/change-plan](#post-paymentschange-plan)
- [POST /payments/cancel-scheduled-change](#post-paymentscancel-scheduled-change)
- [POST /payments/webhook](#post-paymentswebhook)

---

## POST /payments/checkout

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`CreateCheckout`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| planId | int | Yes | validate:"required" |

### Response

**Success**: HTTP 200

```json
{
  "checkoutUrl": "https://checkout.stripe.com/c/pay/cs_test_...",
  "sessionId": "cs_test_...",
  "creditAppliedCents": 450,
  "creditAppliedFormatted": "$4.50"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| Validation failed | 422 | "Invalid request data" |
| Plan not found | 404 | "Plan not found" |
| No Stripe price for plan | 400 | "Invalid price configuration" |
| Stripe checkout session fails | 400 | "Checkout session creation failed" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Validates body with `c.Bind` and `c.Validate`.
3. Fetches the user via `userRepo.GetByID` to obtain email and name for Stripe customer creation.
4. Delegates to `paymentSvc.CreateCheckout(userID, planID, email, name)`:
   a. Looks up the target plan via `planRepo.GetByID`. Returns `ErrPlanNotFound` if not found.
   b. Looks up the Stripe price ID via `planPriceRepo.GetByPlanAndProvider(planID, "STRIPE")`. Returns `ErrInvalidPrice` if not found.
   c. Gets or creates a Stripe customer for the user (`GetOrCreateCustomer`).
   d. Calculates prorated credit from the current subscription if applicable (daily rate * days remaining on current plan).
   e. Applies credit to Stripe customer balance if credit > 0.
   f. Builds success/cancel URLs from `FRONTEND_URL + STRIPE_SUCCESS_URL` and `FRONTEND_URL + STRIPE_CANCEL_URL`.
   g. Creates a Stripe Checkout session with the customer ID, price ID, URLs, and metadata (`user_id`, `plan_id`).
5. Returns the real Stripe checkout URL, session ID, and credit information.

---

## POST /payments/portal

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`CreatePortal`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| returnUrl | *string | No | Optional URL to redirect to after the portal session (must start with FRONTEND_URL) |

### Response

**Success**: HTTP 200

```json
{ "portalUrl": "https://billing.stripe.com/p/session/..." }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| No Stripe customer | 400 | "No Stripe customer found" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Binds the optional `PortalRequest` body.
3. Delegates to `paymentSvc.CreatePortal(userID, returnURL)`:
   a. Fetches the user's subscription and provider subscription.
   b. Returns `ErrNoStripeCustomer` if no external customer ID exists.
   c. Validates the `returnUrl` starts with `FRONTEND_URL` (prevents open redirect); falls back to `FRONTEND_URL` if invalid.
   d. Creates a Stripe Billing Portal session with the customer ID and return URL.
4. Returns the real Stripe portal URL.

---

## GET /payments/status

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`GetPaymentStatus`)

### Request

No request body or query parameters.

### Response

**Success**: HTTP 200

```json
{
  "hasStripeCustomer": true,
  "stripeSubscriptionActive": true,
  "stripeSubscriptionId": "sub_...",
  "currentPlanType": "premium",
  "lastPaymentFailed": false
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Delegates to `paymentSvc.GetPaymentStatus(userID)`:
   a. Fetches the user's subscription. If none exists, returns free-plan defaults.
   b. Fetches the provider subscription. If none exists, returns the plan type from the subscription only.
   c. `hasStripeCustomer`: true if `ExternalCustomerID` is non-nil and non-empty.
   d. `stripeSubscriptionActive`: true if `ExternalSubscriptionID` is non-nil and non-empty.
   e. `stripeSubscriptionId`: the external subscription ID from the provider subscription record.
   f. `lastPaymentFailed`: read from `providerSub.LastPaymentFailed` (set/cleared by `invoice.payment_failed` / `invoice.paid` webhooks).

---

## GET /payments/session/:session_id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`GetSessionStatus`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| session_id | string (path) | Yes | Stripe checkout session ID |

### Response

**Success**: HTTP 200

```json
{
  "sessionId": "cs_test_...",
  "status": "complete",
  "paymentStatus": "paid",
  "customerId": "cus_...",
  "subscriptionId": "sub_...",
  "amountTotal": 999,
  "currency": "usd"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Session not found or ownership mismatch | 403 | "Session not found" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Delegates to `paymentSvc.GetSessionStatus(userID, sessionID)`:
   a. Retrieves the real Stripe Checkout session via the `PaymentProvider`.
   b. Verifies the session belongs to the authenticated user by checking the `user_id` metadata field.
   c. Returns `ErrSessionNotFound` (HTTP 403) if the session does not belong to the user.
   d. Returns real session data: status, payment status, customer ID, subscription ID, amount, currency.

---

## GET /payments/upgrade-price/:plan_id

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`GetUpgradePrice`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| plan_id | int (path) | Yes | ID of the target plan |

### Response

**Success**: HTTP 200

```json
{
  "planId": 2,
  "planName": "Premium Monthly",
  "planType": "premium",
  "originalPriceCents": 999,
  "creditCents": 450,
  "finalPriceCents": 549,
  "originalPriceFormatted": "$9.99",
  "creditFormatted": "$4.50",
  "finalPriceFormatted": "$5.49",
  "currencyCode": "USD"
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid plan_id (non-integer) | 400 | "Invalid plan ID" |
| Plan not found | 404 | "Plan not found" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Parses `plan_id` from the path. Returns 400 if not a valid integer.
3. Delegates to `paymentSvc.GetUpgradePrice(userID, planID)`:
   a. Fetches the target plan. Returns `ErrPlanNotFound` if not found.
   b. Calculates prorated credit from the current subscription (daily rate * days remaining on current plan).
   c. `originalPriceCents`: target plan price * 100.
   d. `finalPriceCents`: original - credit, clamped to 0.
   e. Formatted prices use the `CreditCalculator.FormatCreditForDisplay()` (currency symbol + amount to 2 decimal places).

---

## POST /payments/change-plan

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`ChangePlan`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| planId | int | Yes | validate:"required" |

### Response

**Success (immediate upgrade)**: HTTP 200

```json
{
  "success": true,
  "isUpgrade": true,
  "newPlanName": "Premium Yearly",
  "effectiveDate": "2026-02-25T10:30:00Z",
  "message": "Plan upgraded successfully",
  "scheduleId": null
}
```

**Success (scheduled downgrade)**: HTTP 200

```json
{
  "success": true,
  "isUpgrade": false,
  "newPlanName": "Premium Monthly",
  "effectiveDate": "2026-03-25T10:30:00Z",
  "message": "Plan downgrade scheduled",
  "scheduleId": "sub_sched_..."
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| Validation failed | 422 | "Invalid request data" |
| Plan not found | 404 | "Plan not found" |
| No active Stripe subscription | 400 | "No active Stripe subscription" |
| Invalid price configuration | 400 | "Invalid price configuration" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Validates body with `c.Bind` and `c.Validate`.
3. Delegates to `paymentSvc.ChangePlan(userID, targetPlanID)`:
   a. Fetches the user's subscription and provider subscription. Requires an active Stripe subscription.
   b. Fetches the target plan and its Stripe price ID.
   c. Fetches the current plan for comparison.
   d. **Upgrade** (target billing period >= current): Updates the Stripe subscription immediately with `proration_behavior=create_prorations`. Updates the local DB record with the new plan.
   e. **Downgrade** (target billing period < current): Gets the current Stripe subscription's `current_period_end`. Creates a Stripe Subscription Schedule with two phases: (1) current plan until period end, (2) target plan after. Stores the schedule ID on the provider subscription record and the pending plan ID on the subscription.

---

## POST /payments/cancel-scheduled-change

**Auth**: Required (JWT)
**Handler**: `internal/handlers/payments/handler.go`
**Service**: `internal/services/payment/service.go` (`CancelScheduledChange`)

### Request

No request body.

### Response

**Success**: HTTP 200

```json
{ "success": true, "message": "Scheduled change canceled" }
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| No active subscription schedule | 400 | "No active subscription schedule" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Delegates to `paymentSvc.CancelScheduledChange(userID)`:
   a. Fetches the user's subscription and provider subscription.
   b. Returns `ErrScheduleNotFound` if no `ExternalScheduleID` exists.
   c. Cancels the schedule in Stripe via `CancelSubscriptionSchedule`.
   d. Clears the schedule ID from the provider subscription.
   e. Clears `PendingPlanID`, `PendingDowngradeAccountIDs`, and `PendingDowngradeBudgetID` on the subscription.
   f. If the subscription was being canceled (has `CanceledAt` set), re-enables it in Stripe and clears `CanceledAt`, setting `AutoRenew = true`.

---

## POST /payments/webhook

**Auth**: Public (no JWT required). Uses Stripe signature verification.
**Handler**: `internal/handlers/payments/handler.go`
**Webhook processor**: `internal/services/payment/webhook_handler.go`

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| Stripe-Signature | header | Yes | Stripe webhook signature for cryptographic verification |
| body | raw bytes | Yes | Raw Stripe event JSON payload (limited to 64KB) |

### Response

**Success**: HTTP 200

```json
{
  "received": true,
  "success": true,
  "eventType": "checkout.session.completed",
  "error": null
}
```

**Signature verification failure**: HTTP 200

```json
{
  "received": true,
  "success": false,
  "eventType": "",
  "error": null
}
```

### Error Responses

All responses return HTTP 200 (to prevent Stripe from retrying). Failures are indicated by `success: false` in the response body.

### Business Logic

1. Public endpoint -- no auth middleware.
2. Reads the raw request body with `io.ReadAll` (limited to 64KB to prevent memory exhaustion).
3. Passes the body and `Stripe-Signature` header to `webhookHandler.HandleWebhook()`.
4. **Signature verification**: Uses `stripe.webhook.ConstructEvent()` to verify the cryptographic signature against `STRIPE_WEBHOOK_SECRET`.
5. **Event routing**: The webhook handler parses the event JSON and routes to the appropriate handler based on event type.

### Handled Event Types (7)

| Event Type | Handler | Description |
|------------|---------|-------------|
| `checkout.session.completed` | `handleCheckoutSessionCompleted` | Links Stripe customer/subscription to internal subscription, activates the target plan, clears trial and pending downgrade state, sets expiration from Stripe period end |
| `customer.subscription.updated` | `handleSubscriptionUpdated` | Synchronizes plan changes (via price ID lookup), cancellation state, expiration, and active status from Stripe |
| `customer.subscription.deleted` | `handleSubscriptionDeleted` | Downgrades user to free plan, clears all Stripe data from provider subscription, clears pending fields |
| `invoice.paid` | `handleInvoicePaid` | Clears `LastPaymentFailed` flag on the provider subscription |
| `invoice.payment_failed` | `handleInvoicePaymentFailed` | Sets `LastPaymentFailed` flag on the provider subscription |
| `subscription_schedule.completed` | `handleScheduleCompleted` | Applies the pending plan transition (moves subscription to pending plan), clears schedule ID |
| `subscription_schedule.released` | `handleScheduleReleased` | Clears the schedule ID from the provider subscription (schedule was canceled before completion) |

### Webhook Event Details

**checkout.session.completed**:
- Extracts `customer`, `subscription`, `user_id`, and `plan_id` from the event.
- Gets the Stripe subscription details for `current_period_end`.
- Creates or updates the provider subscription with customer and subscription IDs.
- Updates the internal subscription: sets plan, plan type, subscribedAt, expiresAt, isActive, autoRenew, hasStripeSubscription, billing period. Clears trial fields and pending downgrade fields.

**customer.subscription.updated**:
- Finds the internal subscription via `GetProviderSubscriptionByExternalID`.
- If the price changed: looks up the plan via `planPriceRepo.GetByExternalPriceID` and updates plan ID, plan type, and billing period.
- Updates `expiresAt` from `current_period_end`.
- Maps `cancel_at_period_end` to `CanceledAt`/`AutoRenew`.
- Maps Stripe status (`active`/`past_due` -> active, `canceled`/`unpaid` -> inactive).

**customer.subscription.deleted**:
- Downgrades to free plan. Sets `IsActive=true`, `AutoRenew=false`, `CanceledAt=now`, clears `ExpiresAt`, `CurrentBillingPeriodID`, and all pending fields.
- Clears `ExternalSubscriptionID` and `ExternalScheduleID` from provider subscription.

**invoice.paid / invoice.payment_failed**:
- Skips invoices without a subscription (e.g., one-off payments).
- Finds provider subscription by external subscription ID.
- Sets/clears `LastPaymentFailed` flag.

**subscription_schedule.completed / subscription_schedule.released**:
- Finds provider subscription by `ExternalScheduleID` via `GetProviderSubscriptionByScheduleID`.
- `completed`: Applies pending plan transition if `PendingPlanID` is set. Clears schedule ID.
- `released`: Only clears schedule ID (schedule was canceled).
