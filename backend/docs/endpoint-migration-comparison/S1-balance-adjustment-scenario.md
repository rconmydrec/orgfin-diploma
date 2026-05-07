# Scenario S1: Balance Adjustment

## Description
Tests that balance adjustment transactions work correctly: they update the account balance to a specific target value, appear in transaction history, are excluded from cash flow reports and budget calculations, and work alongside regular income/expense transactions.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_balance_adjustment_scenario.py`
- **Test count**: 6 (in `TestBalanceAdjustmentScenario` class)
- **Key assertions**:
  1. `test_adjustment_excluded_from_cashflow_report` -- cash flow `total_income` and `total_expenses` do not change after adjustment
  2. `test_adjustment_appears_in_transaction_history` -- adjustment found by ID in transaction list
  3. `test_adjustment_updates_account_balance` -- account balance matches target (2500.50)
  4. `test_multiple_adjustments_in_sequence` -- 3 sequential adjustments, final balance 3000, all 3 visible in history
  5. `test_adjustment_with_regular_transactions` -- expense(-100) + adjustment(to 1500) + income(+200) = 1700
  6. `test_adjustment_excluded_from_budget_calculation` -- budget `collected_amount` only counts regular expense (100), not adjustment

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/balance_adjustment_test.go`
- **Test count**: 6
- **Key assertions**:
  1. `TestAdjustmentUpdatesAccountBalance` -- balance equals 2500.5 via GET /accounts/:id
  2. `TestAdjustmentExcludedFromCashFlowReport` -- totalIncome and totalExpenses unchanged after adjustment
  3. `TestAdjustmentAppearsInTransactionHistory` -- adjustment ID found in transaction list, `isAdjustment: true`
  4. `TestMultipleAdjustmentsInSequence` -- 3 adjustments, final balance 3000, adjustmentCount == 3
  5. `TestAdjustmentWithRegularTransactions` -- expense(-100) + adjustment(to 1500) + income(+200) = 1700
  6. `TestAdjustmentDownward` -- downward adjustment (5000 -> 2000), verifies `isIncome: false`, `isAdjustment: true`, `excludeFromReports: true`, DB balance == 2000

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| Adjustment updates balance | Yes | Yes |
| Excluded from cash flow report | Yes | Yes |
| Appears in transaction history | Yes | Yes |
| Multiple adjustments in sequence | Yes | Yes |
| Mixed with regular transactions | Yes | Yes |
| Excluded from budget calculation | Yes | No |
| Downward adjustment flags | No | Yes |

### Differences in Assertions
- Python verifies budget calculation exclusion (`fill_budget_with_existing_transactions`); Go does not test this.
- Go tests downward adjustment explicitly and verifies `isIncome`, `isAdjustment`, and `excludeFromReports` flags on the adjustment transaction; Python does not.
- Both test the same cash flow exclusion scenario with matching logic.
- Python uses `TestClient` (FastAPI); Go uses `httptest.NewRecorder` + `Echo.ServeHTTP`.

### Missing Scenarios
- Go is missing a budget calculation exclusion test.
- Python is missing a downward adjustment flag verification test.

## Notes
- Both implementations cover the core adjustment scenarios thoroughly.
- Go uses direct DB queries for balance verification alongside API checks; Python primarily uses API responses with some ORM refreshes.
- Test data setup differs: Python uses factory classes (`AccountFactory`), Go uses `testutil` helpers.
