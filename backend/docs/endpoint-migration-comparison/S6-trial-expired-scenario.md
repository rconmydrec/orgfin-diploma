# Scenario S6: Trial Expired

## Description
Tests the behavior when a user's trial period expires and they need to either upgrade or comply with free-tier limits. Validates that limits are enforced, listing endpoints remain accessible, and the downgrade/upgrade flow works correctly for expired trial users.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_trial_expired_free_limits_enforced_scenario.py`
- **Test count**: 1 (large multi-step test in `TestTrialExpiredFreeLimitsEnforcedScenario` class)
- **Key assertions**:
  1. `test_trial_expired_over_limits_blocks_most_endpoints_until_selection`:
     - Creates 3 accounts while trial is active
     - Expires trial via DB modification
     - Verifies: `GET /accounts/` still returns 200 (listing works)
     - Verifies: `POST /transactions/` returns 402 (PAYMENT_REQUIRED) -- blocked due to over-limit
     - Applies downgrade selection (keep 2 accounts) via `POST /subscriptions/downgrade`
     - Verifies: `POST /transactions/` now returns 200 (unblocked after selection)

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/trial_expired_test.go`
- **Test count**: 3
- **Key assertions**:
  1. `TestTrialExpiredFreeLimitsEnforced`:
     - Creates 3 accounts while trial is active
     - Expires trial via DB modification (creates subscription if needed)
     - Verifies: subscription status shows trial/free with trialDaysRemaining == 0
     - Verifies: `GET /accounts/` returns 200 (listing still works)
     - Applies downgrade selection (keep 2 accounts) via `POST /subscriptions/downgrade`
     - Verifies: pendingDowngrade == true
     - Verifies: subscription status shows pendingPlanId
  2. `TestFreeUserSubscriptionLimitsDisplayed`:
     - Creates new user (no subscription)
     - Verifies: status includes `limits` with `accounts` and `budgets` keys
     - Verifies: `isActive == true`
  3. `TestTrialExpiredThenUpgrade`:
     - Creates expired trial subscription via DB
     - Upgrades to premium via `POST /subscriptions/upgrade`
     - Verifies: planType == "premium"
     - Verifies: limits are nil for premium users

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| Create accounts during trial | Yes (3) | Yes (3) |
| Expire trial via DB | Yes | Yes |
| Listing endpoints still work | Yes | Yes |
| Transaction creation blocked (402) | Yes | No |
| Downgrade with account selection | Yes | Yes |
| Transaction creation unblocked after selection | Yes | No |
| Trial days remaining = 0 | No | Yes |
| Free user limits displayed | No | Yes |
| Trial expired -> upgrade to premium | No | Yes |
| Premium user has no limits | No | Yes |

### Differences in Assertions
- **Python tests enforcement**: The key test verifies that endpoints are actually BLOCKED (402) when over free-tier limits, and UNBLOCKED after downgrade selection. This tests the `enforce_free_plan_compliance` dependency.
- **Go does not test enforcement**: Go verifies the subscription state transitions (trial -> pending downgrade) but does not verify that endpoints are blocked. Go does not have a 402 PAYMENT_REQUIRED test.
- Go has additional tests for limits display and trial-to-premium upgrade that Python does not have in this scenario file.
- Go creates the subscription manually if it does not exist (handles edge case); Python assumes subscription exists from registration.

### Missing Scenarios
- Go is missing the critical enforcement test: verifying that transaction creation returns 402 when over limits and 200 after applying selection.
- Python is missing free-user limits display verification and trial-to-upgrade recovery test.
- Neither tests what happens when trial expires but user is within free-tier limits (no selection needed).

## Notes
- The Python enforcement test is the most important test in this scenario -- it validates the end-to-end behavior that free-plan compliance middleware works correctly. Go does not have this middleware/dependency equivalent, or at least does not test it in this scenario.
- Go's `TestTrialExpiredThenUpgrade` is a valuable recovery test that verifies users can escape an expired trial by upgrading.
- Both implementations manipulate the DB directly to expire trials (no time-freezing), which is practical for integration tests.
- Go handles the case where the user may not have a subscription row at all (edge case in some configurations), which Python does not need to handle due to its registration flow always creating a subscription.
