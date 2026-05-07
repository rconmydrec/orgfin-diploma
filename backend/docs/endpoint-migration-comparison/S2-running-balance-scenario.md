# Scenario S2: Running Balance

## Description
Tests that account balances are correctly maintained across a sequence of income and expense transactions. Verifies the running balance after multiple operations: create account with 0 balance, add incomes, subtract expense, and verify the final balance matches the expected calculation.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_transactions_running_balance_scenario.py`
- **Test count**: 1 (in `TestTransactionsRunningBalanceScenario` class)
- **Key assertions**:
  1. `test_incomes_and_expense_update_account_balance`:
     - Creates account with 0 balance via API
     - +100 income -> +200 income -> -50 expense
     - Verifies final DB balance == 250.00 (via `db.refresh(account)`)
     - Uses helper `_register_activate_and_login` to create fresh user
     - Fetches currency/account type/categories via API endpoints

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/running_balance_test.go`
- **Test count**: 1
- **Key assertions**:
  1. `TestRunningBalanceAfterIncomesAndExpenses`:
     - Creates account with 0 balance via `POST /accounts/`
     - +100 income -> +200 income -> -50 expense (via `createTx` helper)
     - Verifies final balance via `GET /accounts/:id` API response == 250
     - Also verifies via direct DB query `SELECT balance FROM accounts` == 250.0

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| Create account with 0 balance | Yes (API) | Yes (API) |
| Multiple income transactions | Yes (+100, +200) | Yes (+100, +200) |
| Expense transaction | Yes (-50) | Yes (-50) |
| Final balance via DB | Yes (ORM) | Yes (raw SQL) |
| Final balance via API | No | Yes |

### Differences in Assertions
- Go verifies balance both via API response and direct DB query; Python only verifies via DB (ORM refresh).
- Python creates a fresh user within the test using `_register_activate_and_login`; Go uses `testutil.CreateTestUser`.
- Python fetches category IDs via `/categories/grouped/` API; Go uses `testutil.CreateTestCategory` to create dedicated test categories.
- Python includes `dateTime` and `notes` in transaction creation; Go uses a minimal transaction body.

### Missing Scenarios
- Both implementations have identical scenario coverage (1 test each, same logic).
- Neither tests edge cases like negative balances or very large amounts.

## Notes
- The scenario is identical in both implementations -- both cover the same sequence of +100, +200, -50 on a zero-balance account and verify 250.
- Go's approach of verifying via both API and DB is more thorough.
- Python uses Decimal quantization for comparison (`Decimal("250.00")`); Go compares float64 directly.
