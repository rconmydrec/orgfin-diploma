# Scenario S4: Subscription Plan Transitions

## Description
Tests the full subscription plan lifecycle: trial/free to premium monthly, monthly to yearly upgrade, downgrade scheduling, and pending plan verification. Validates that subscription status, plan IDs, and pending state are correctly reflected across transitions.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_subscription_plan_transitions_scenario.py`
- **Test count**: 1 (large multi-step test in `TestSubscriptionPlanTransitionScenario` class)
- **Key assertions**:
  1. `test_trial_to_monthly_to_yearly_to_monthly_scheduled_and_applied`:
     - Registers user, verifies trial plan type
     - Expires trial via DB
     - **Step 1**: Checkout for monthly plan -> webhook completes checkout -> verifies plan_id, billing_period_id, expires_at, provider data (customer_id, subscription_id)
     - **Step 2**: Change plan monthly -> yearly (immediate) -> verifies plan_id updated, pending_plan_id is None, expires_at updated
     - **Step 3**: Change plan yearly -> monthly (scheduled) -> verifies current plan stays yearly, pending_plan_id = monthly, schedule_id stored
     - **Step 4**: Webhook `subscription_schedule.completed` -> verifies plan_id changed to monthly, pending_plan_id cleared, schedule_id cleared
     - Uses full Stripe mocking throughout

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/subscription_plan_transitions_test.go`
- **Test count**: 2
- **Key assertions**:
  1. `TestTrialToMonthlyToYearlyTransition`:
     - Gets initial status (free or trial)
     - Finds monthly and yearly premium plans from `GET /subscriptions/plans`
     - Upgrades to monthly via `POST /subscriptions/upgrade` -> verifies "Successfully" in message
     - Verifies status shows premium with monthlyPlanID
     - Changes to yearly via `POST /payments/change-plan` -> verifies success=true
     - Verifies status shows premium with yearlyPlanID
     - Skips if no monthly/yearly plans found
  2. `TestDowngradeSchedulesPendingPlan`:
     - Upgrades to premium
     - Creates test accounts
     - Downgrades with account selection
     - Verifies `pendingDowngrade: true`, `pendingPlanId` is not nil
     - Verifies subscription status shows `pendingPlanId`

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| Trial to monthly upgrade | Yes (via checkout + webhook) | Yes (via /subscriptions/upgrade) |
| Monthly to yearly upgrade | Yes (via /payments/change-plan) | Yes (via /payments/change-plan) |
| Yearly to monthly downgrade (scheduled) | Yes | No |
| Schedule completion webhook | Yes | No |
| Downgrade with account selection | No (different scenario) | Yes |
| Pending plan verification | Yes | Yes |
| Stripe provider data verification | Yes (customer_id, subscription_id, schedule_id) | No |
| Billing period verification | Yes | No |
| Expires_at verification | Yes | No |

### Differences in Assertions
- **Python is more comprehensive in Stripe-level verification**: It tests the full 4-step flow including scheduled downgrades and webhook-driven plan application, with Stripe mock verification at each step.
- **Go tests use the DB-level subscription management**: The upgrade goes directly through `/subscriptions/upgrade` rather than checkout + webhook, though the Go backend now has full Stripe integration available for future scenario expansion.
- Python verifies `billing_period_id`, `expires_at`, `provider_data.external_customer_id`, `provider_data.external_subscription_id`, and `provider_data.external_schedule_id`. Go verifies none of these.
- Go includes a separate downgrade test with account selection; Python's downgrade is part of the transition flow.

### Missing Scenarios
- Go does not test scheduled downgrades or webhook-driven plan application in integration scenarios (though the service-level code supports this).
- Go does not verify Stripe-level state (customer, subscription, schedule IDs) in integration scenarios.
- Python does not test the downgrade with account selection (that is in the trial-expired scenario instead).

## Notes
- The Go backend now has full Stripe integration (`services/payment/`), but the integration test scenarios have not yet been expanded to test the full Stripe-driven flow with mocked Stripe responses.
- Go's scenario tests currently exercise the DB-level subscription management path. Expanding them to cover the Stripe checkout -> webhook -> plan change -> schedule -> webhook flow would close the remaining gap with Python.
- Both tests skip gracefully if required plans are not found in the database.
