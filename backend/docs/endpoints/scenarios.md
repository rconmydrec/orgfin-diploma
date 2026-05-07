# Cross-Domain Test Scenarios

Integration test scenarios that exercise multiple endpoints together to verify end-to-end behavior. Each scenario creates real DB state, calls real HTTP handlers, and asserts outcomes across the full stack.

## Table of Contents

- [S1: Balance Adjustment](#s1-balance-adjustment)
- [S2: Running Balance](#s2-running-balance)
- [S3: Transfer](#s3-transfer)
- [S4: Subscription Plan Transitions](#s4-subscription-plan-transitions)
- [S5: Subscription Webhooks](#s5-subscription-webhooks)
- [S6: Trial Expired](#s6-trial-expired)

---

## S1: Balance Adjustment

**Test file**: `internal/handlers/scenarios/balance_adjustment_test.go`
**Test count**: 6

### What it tests

Verifies that balance adjustment transactions correctly set the account balance to a target value, appear in transaction history with the correct flags (`isAdjustment: true`, `excludeFromReports: true`), and are excluded from cash flow report totals.

### Test cases

- `TestAdjustmentUpdatesAccountBalance` -- creates an adjustment transaction and verifies the account balance equals the target value via `GET /accounts/:id`
- `TestAdjustmentExcludedFromCashFlowReport` -- verifies that `totalIncome` and `totalExpenses` in the cash flow report do not change after an adjustment
- `TestAdjustmentAppearsInTransactionHistory` -- verifies the adjustment appears in the transaction list and has `isAdjustment: true`
- `TestMultipleAdjustmentsInSequence` -- applies 3 sequential adjustments and verifies final balance equals 3000 and all 3 appear in history
- `TestAdjustmentWithRegularTransactions` -- mixes a regular expense (-100), an adjustment (to 1500), and a regular income (+200) and verifies the final balance is 1700
- `TestAdjustmentDownward` -- applies a downward adjustment (5000 to 2000) and verifies `isIncome: false`, `isAdjustment: true`, `excludeFromReports: true`, and DB balance equals 2000

---

## S2: Running Balance

**Test file**: `internal/handlers/scenarios/running_balance_test.go`
**Test count**: 1

### What it tests

Verifies that account balances are correctly maintained across a sequence of income and expense transactions. Creates a zero-balance account, applies two income transactions and one expense, then asserts the final balance is correct via both the API and a direct DB query.

### Test cases

- `TestRunningBalanceAfterIncomesAndExpenses` -- creates a zero-balance account, applies +100 income, +200 income, and -50 expense, then verifies the final balance is 250 via `GET /accounts/:id` and via a direct SQL `SELECT balance FROM accounts` query

---

## S3: Transfer

**Test file**: `internal/handlers/scenarios/transfer_test.go`
**Test count**: 2

### What it tests

Verifies that transfers between accounts correctly update both the source and destination balances and create linked transaction pairs. Covers same-currency and cross-currency transfer cases.

### Test cases

- `TestTransferSameCurrencyUpdatesBothBalances` -- transfers 200 USD between two USD accounts (initial balances 1000 and 500), verifies source balance is 800, destination balance is 700, API response has `isTransfer: true` and `linkedTransactionId` set, and DB shows 2 transfer transactions
- `TestTransferDifferentCurrenciesUsesTargetAmount` -- transfers 100 USD to a EUR account at a rate yielding 92 EUR, verifies source balance is 900 USD and destination balance is 92 EUR in the DB

---

## S4: Subscription Plan Transitions

**Test file**: `internal/handlers/scenarios/subscription_plan_transitions_test.go`
**Test count**: 2

### What it tests

Verifies the subscription plan lifecycle through DB-level plan management: initial status, upgrade from free/trial to monthly premium, change from monthly to yearly premium, and scheduling a downgrade with account selection. Skips gracefully if required plans are not seeded in the database.

### Test cases

- `TestTrialToMonthlyToYearlyTransition` -- gets initial subscription status, finds monthly and yearly premium plans via `GET /subscriptions/plans`, upgrades to monthly via `POST /subscriptions/upgrade`, verifies status shows the monthly plan, then changes to yearly via `POST /payments/change-plan` and verifies status shows the yearly plan
- `TestDowngradeSchedulesPendingPlan` -- upgrades to premium, creates test accounts, submits a downgrade with account selection via `POST /subscriptions/downgrade`, and verifies that `pendingDowngrade: true` and `pendingPlanId` are set in the subscription status response

---

## S5: Subscription Webhooks

**Test file**: `internal/handlers/scenarios/subscription_webhooks_test.go`
**Test count**: 5

### What it tests

Verifies that the webhook endpoint receives Stripe event payloads and returns the expected acknowledgement response. The webhook handler (`internal/services/payment/webhook_handler.go`) is fully functional and processes 7 event types with real state management, but integration scenario tests currently verify reachability and response structure. Webhook signature verification requires valid Stripe signatures which are not available in the test environment.

### Test cases

- `TestWebhookReceivesAndAcknowledges` -- sends a `checkout.session.completed` payload to `POST /payments/webhook` and verifies the response has `received: true` and `success: true`
- `TestWebhookWithStripeSignatureHeader` -- sends a payload with a `Stripe-Signature` header present and verifies `received: true`
- `TestWebhookMultipleEventTypes` -- sends 5 event types (`checkout.session.completed`, `customer.subscription.updated`, `invoice.payment_failed`, `invoice.paid`, `subscription_schedule.completed`) as sub-tests and verifies each returns 200 with `received: true`
- `TestSubscriptionStateUpdateViaUpgradeAndWebhook` -- upgrades to premium via `POST /subscriptions/upgrade`, verifies status is premium, sends a `checkout.session.completed` webhook, and verifies the subscription state is still premium (webhook did not corrupt it)
- `TestPaymentStatusAfterUpgrade` -- verifies initial payment status is free, upgrades to premium, and verifies `GET /payments/status` reflects the premium plan type

---

## S6: Trial Expired

**Test file**: `internal/handlers/scenarios/trial_expired_test.go`
**Test count**: 3

### What it tests

Verifies behavior when a user's trial period expires: subscription status reflects zero trial days remaining, listing endpoints remain accessible, a downgrade selection can be submitted, and an upgrade to premium can recover the account from an expired trial state.

### Test cases

- `TestTrialExpiredFreeLimitsEnforced` -- creates 3 accounts while trial is active, expires the trial via a direct DB update, verifies subscription status shows `trialDaysRemaining: 0`, verifies `GET /accounts/` still returns 200, submits a downgrade via `POST /subscriptions/downgrade` keeping 2 accounts, and verifies `pendingDowngrade: true` and `pendingPlanId` are set
- `TestFreeUserSubscriptionLimitsDisplayed` -- creates a new user with no subscription, verifies that the status response includes a `limits` field containing `accounts` and `budgets` keys, and that `isActive: true`
- `TestTrialExpiredThenUpgrade` -- creates an expired trial subscription via direct DB insert, upgrades to premium via `POST /subscriptions/upgrade`, verifies `planType` is `"premium"`, and verifies the `limits` field is nil for premium users
