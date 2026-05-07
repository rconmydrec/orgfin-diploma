# Endpoint #71: GET /payments/upgrade-price/:plan_id

## Route Definition
- **Python**: `@router.get("/upgrade-price/{plan_id}", response_model=UpgradePriceInfo, dependencies=[Depends(check_token)])`
- **Go**: `protected.GET("/upgrade-price/:plan_id", h.GetUpgradePrice)` (inside `authMiddleware` group)

## Request
- **Auth**: Required (both)
- **Path params**: `plan_id` (int)
- **Body**: None (both)

## Response

### Status Codes
| Scenario | Python | Go |
|---|---|---|
| Success | 200 | 200 |
| Unauthorized | 500 (dependency chain) | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| Invalid plan ID | 422 (FastAPI path validation) | 400 ("Invalid plan ID") |

### Response Body Structure
Both return `UpgradePriceInfo`:
- `planId` / `plan_id` (int)
- `planName` / `plan_name` (string)
- `planType` / `plan_type` (string)
- `originalPriceCents` / `original_price_cents` (int)
- `creditCents` / `credit_cents` (int)
- `finalPriceCents` / `final_price_cents` (int)
- `originalPriceFormatted` / `original_price_formatted` (string)
- `creditFormatted` / `credit_formatted` (string, nullable)
- `finalPriceFormatted` / `final_price_formatted` (string)
- `currencyCode` / `currency_code` (string)

## Error Responses
| Error | Python | Go |
|---|---|---|
| Missing auth | 500 | 401 |
| User not active | N/A | 401 |
| Plan not found | 404 | 404 |
| Invalid plan ID | 422 | 400 |

## Business Logic Comparison

| Step | Python | Go |
|---|---|---|
| 1. Auth | check_token + get_current_user | authMiddleware + RequireActiveUser middleware |
| 2. Parse plan_id | FastAPI path parameter (auto int conversion) | `fmt.Sscanf(c.Param("plan_id"), "%d", &planID)` |
| 3. Get plan | `get_active_plan_by_id(plan_id, db)` | `paymentSvc.GetUpgradePrice(userID, planID)` via PaymentService |
| 4. Calculate price | `checkout_service.get_upgrade_price_info(user, plan)` -- factors in credit from current subscription | Service calculates via `calculateCurrentCredit` using CreditCalculator |
| 5. Credit | Calculated from current subscription remaining time | Calculated from current subscription via `CreditCalculator.CalculateRemainingCredit()` |
| 6. Formatted price | From service | From `CreditCalculator.FormatCreditForDisplay()` |

## Notes
- Both Python and Go now calculate prorated credit from the user's current subscription and apply it to the upgrade price.
- **Invalid plan_id handling differs**: Python returns 422 (FastAPI auto-validates path param as int). Go manually parses with `fmt.Sscanf` and returns 400.
- Go uses `shopspring/decimal` for all monetary calculations (never float64).

## Tests

### Python
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/test_payments_endpoints.py`
- **Test count**: 3 (TestGetUpgradePriceEndpoint class)
- Tests: success, plan_not_found, unauthorized

### Go Integration Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_test.go`
- **Test count**: 3
- Tests: TestGetUpgradePriceInvalidPlan, TestGetUpgradePriceUnauthorized, TestGetUpgradePriceInvalidID, TestGetUpgradePriceWithValidPlan

### Go Unit Tests
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/payments/handler_unit_test.go`
- **Test count**: 3
- Tests: TestGetUpgradePriceInvalidID, TestGetUpgradePricePlanNotFound, TestGetUpgradePriceSuccess
