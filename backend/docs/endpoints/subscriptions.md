# Subscriptions Endpoints

Endpoints for retrieving and managing user subscription plans, including status, plan listing, upgrades, and downgrades.

All subscription business logic is handled by the `SubscriptionService` (`internal/services/subscription/service.go`). The HTTP handler delegates to this service and maps results/errors to HTTP responses.

## Table of Contents

- [GET /subscriptions/status](#get-subscriptionsstatus)
- [GET /subscriptions/plans](#get-subscriptionsplans)
- [POST /subscriptions/upgrade](#post-subscriptionsupgrade)
- [POST /subscriptions/downgrade](#post-subscriptionsdowngrade)

---

## GET /subscriptions/status

**Auth**: Required (JWT)
**Handler**: `internal/handlers/subscriptions/handler.go`
**Service**: `internal/services/subscription/service.go` (`GetStatus`)

### Request

No request body or query parameters.

### Response

**Success**: HTTP 200

```json
{
  "planType": "premium",
  "isActive": true,
  "planId": 3,
  "trialDaysRemaining": null,
  "requiresDowngrade": false,
  "limits": null,
  "subscribedAt": "2026-02-20T10:00:00Z",
  "expiresAt": "2026-03-20T10:00:00Z",
  "pendingPlanId": null,
  "pendingPlanName": null,
  "hasStripeSubscription": true
}
```

**Free plan (no subscription)**: HTTP 200

```json
{
  "planType": "free",
  "isActive": true,
  "planId": null,
  "trialDaysRemaining": null,
  "requiresDowngrade": false,
  "limits": { "accounts": 2, "budgets": 1, "planningDays": 14 },
  "subscribedAt": null,
  "expiresAt": null,
  "pendingPlanId": null,
  "pendingPlanName": null,
  "hasStripeSubscription": false
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active or deleted | 401 | "User not activated" |
| DB error on user lookup | 401 | (propagated from RequireActiveUser middleware) |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Delegates to `subscriptionSvc.GetStatus(userID)`:
   a. Fetches subscription via `subscriptionRepo.GetByUserID`. If none found, returns hardcoded free-plan status.
   b. **Auto-downgrades expired trials**: if plan type is "trial" and `TrialEndsAt` is past, automatically downgrades to free plan in DB.
   c. Computes "effective plan type" -- accounts for expired trials and expired premium subscriptions showing as "free".
   d. `limits` is populated from `PlanLimits` map: free = `{accounts: 2, budgets: 1, planningDays: 14}`, trial/premium = nil (unlimited).
   e. `requiresDowngrade`: for free plan users, checks if active account count > 2 or active budget count > 1 (via `AccountCounter.CountActiveByUserID` and `BudgetCounter.CountActiveByUserID`). Non-fatal on error (defaults to false).
   f. Trial days remaining: `math.Ceil(time.Until(trialEndsAt).Hours() / 24)`, clamped to 0.
   g. Pending plan name: looked up via `subscriptionPlanRepo.GetByID(*PendingPlanID)` if set.
3. Handler maps `PlanID == 0` to `nil` in the JSON response (for backward compatibility with users who have no subscription record).
4. Handler maps `Limits` to `map[string]int{"accounts": N, "budgets": N, "planningDays": N}` when non-nil.

---

## GET /subscriptions/plans

**Auth**: Public (no JWT required)
**Handler**: `internal/handlers/subscriptions/handler.go`
**Service**: `internal/services/subscription/service.go` (`GetPlans`)

### Request

No request body or query parameters.

### Response

**Success**: HTTP 200

```json
[
  {
    "id": 1,
    "name": "Free",
    "translationKey": "plan.free",
    "planType": "free",
    "billingPeriod": null,
    "price": "0.00",
    "currencyCode": "USD",
    "isFeatured": false,
    "description": null,
    "sortOrder": 0
  }
]
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| DB error | 500 | "Failed to get subscription plans" |

### Business Logic

- Public route -- no authentication required.
- Delegates to `subscriptionSvc.GetPlans()` which calls `subscriptionPlanRepo.GetActivePlans()`.
- Handler constructs response DTOs with plan type mapping via `mapPlanType` (switch on `strings.ToLower`).
- Unknown plan types fall back to `"free"`.
- If the plan has a billing period, constructs a `BillingPeriodResponse` with `id`, `code`, `name`, and `durationDays`.

---

## POST /subscriptions/upgrade

**Auth**: Required (JWT)
**Handler**: `internal/handlers/subscriptions/handler.go`
**Service**: `internal/services/subscription/service.go` (`Upgrade`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| planId | int | Yes | validate:"required" |
| adjustedPrice | decimal | No | Accepted in the DTO but not used in business logic |

### Response

**Success**: HTTP 200

```json
{
  "message": "Subscribed to Premium Monthly plan",
  "subscription": { ... },
  "expiresAt": "2026-03-25T10:30:00Z"
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
| Already on this plan | 400 | "Already on this plan" |
| Invalid plan transition (e.g., to trial) | 400 | "Invalid plan transition" |
| Downgrade via this endpoint | 400 | "Downgrade not allowed via upgrade endpoint" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Validates the body with `c.Bind` and `c.Validate`.
3. Delegates to `subscriptionSvc.Upgrade(userID, planID)`:
   a. If no existing subscription: creates a new one with the target plan, billing period, and expiration date. Returns `ChangeType: "upgrade"`.
   b. If subscription exists: validates the plan transition:
      - Same plan -> `ErrSamePlan`
      - Target is "trial" -> `ErrInvalidPlanTransition`
      - Current is "trial", target is "free" -> `ErrInvalidPlanTransition`
      - Current is "premium", target is "free" -> `ErrDowngradeNotAllowed`
   c. Uses "effective plan type" for validation (accounts for expired trials/premium).
   d. Determines change type by comparing billing period durations ("upgrade" or "plan_change").
   e. Updates plan, plan type, billing period, subscribedAt, and expiresAt on the subscription.
4. `adjustedPrice` is accepted but not applied to any calculation.
5. Returns the updated subscription object and expiry date.

---

## POST /subscriptions/downgrade

**Auth**: Required (JWT)
**Handler**: `internal/handlers/subscriptions/handler.go`
**Service**: `internal/services/subscription/service.go` (`ScheduleDowngrade`)

### Request

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| accountIds | []int | No | Account IDs the user wants to keep on the free plan |
| budgetId | *int | No | Budget ID the user wants to keep |

### Response

**Success (immediate, no Stripe)**: HTTP 200

```json
{
  "message": "Downgraded to free plan immediately",
  "expiresAt": null,
  "pendingDowngrade": false,
  "pendingPlanId": null
}
```

**Success (scheduled, with Stripe)**: HTTP 200

```json
{
  "message": "Downgrade scheduled for end of billing period",
  "expiresAt": "2026-03-25T10:30:00Z",
  "pendingDowngrade": true,
  "pendingPlanId": null
}
```

### Error Responses

| Condition | Status | Message |
|-----------|--------|---------|
| Unauthorized | 401 | (auth middleware) |
| User not active | 401 | "User not activated" |
| Invalid request body | 422 | "Invalid request data" |
| Already on free plan | 400 | "Already on free plan" |
| Entity selection exceeds limits | 400 | "Entity selection violates plan limits" |
| Internal error | 500 | "Internal server error" |

### Business Logic

1. `RequireActiveUser` middleware ensures the user is active and not deleted.
2. Binds the request body (no `c.Validate` call for this endpoint).
3. Delegates to `subscriptionSvc.ScheduleDowngrade(userID, accountIDs, budgetID)`:
   a. Fetches subscription. Returns `ErrAlreadyOnFreePlan` if no subscription or already on free plan.
   b. Validates entity selection against free plan limits (`accountIDs` count must not exceed `PlanLimits["free"].Accounts`).
   c. Fetches the free plan.
   d. **Without Stripe subscription** (`HasStripeSubscription = false`): applies the downgrade immediately -- sets plan to free, clears pending fields, saves. Then calls `ApplyDowngradeEntitySelection` to archive excess accounts and budgets (non-fatal on failure).
   e. **With Stripe subscription**: stores `PendingPlanID`, `PendingDowngradeAccountIDs`, and `PendingDowngradeBudgetID` on the subscription. The actual downgrade will be triggered by the scheduled `subscription:downgrade` worker task or by a Stripe webhook.
