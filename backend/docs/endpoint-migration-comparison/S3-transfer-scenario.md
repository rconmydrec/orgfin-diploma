# Scenario S3: Transfer

## Description
Tests that transfers between accounts correctly update both source and destination balances. Covers same-currency and cross-currency transfer scenarios, verifying that linked transactions are created.

## Python Implementation
- **File**: `/Users/Projects/budget-tracker/back-fastapi/app/tests/endpoints/scenarios/test_transactions_transfer_scenarios.py`
- **Test count**: 2 (in `TestTransactionTransferScenarios` class)
- **Key assertions**:
  1. `test_transfer_same_currency_updates_both_balances`:
     - Source: USD 1000, Destination: USD 500
     - Transfer 200 USD
     - Source balance == 800.00, Destination balance == 700.00 (via DB)
     - 2 transactions exist, both linked to each other (`linked_transaction_id in ids`)
  2. `test_transfer_different_currencies_uses_target_amount_and_updates_balances`:
     - Source: USD 1000, Destination: EUR 0
     - Transfer 100 USD -> 92 EUR
     - Source balance == 900.00, Destination balance == 92.00 (via DB)

## Go Implementation
- **File**: `/Users/Projects/go-budget/backend/internal/handlers/scenarios/transfer_test.go`
- **Test count**: 2
- **Key assertions**:
  1. `TestTransferSameCurrencyUpdatesBothBalances`:
     - Source: USD 1000, Destination: USD 500
     - Transfer 200 USD via `POST /transactions/`
     - Verifies `isTransfer: true` and `linkedTransactionId` not nil in API response
     - Source balance == 800.0, Destination balance == 700.0 (via DB)
     - Transfer transaction count == 2 (via DB query with `is_transfer = true`)
  2. `TestTransferDifferentCurrenciesUsesTargetAmount`:
     - Source: USD 1000, Destination: EUR 0
     - Transfer 100 USD -> 92 EUR
     - Source balance == 900.0, Destination balance == 92.0 (via DB)

## Comparison

### Test Coverage
| Test | Python | Go |
|---|---|---|
| Same-currency transfer (balances) | Yes | Yes |
| Same-currency linked transactions | Yes (IDs cross-linked) | Yes (count + linkedTransactionId) |
| Cross-currency transfer | Yes | Yes |
| API response validation | No | Yes (isTransfer, linkedTransactionId) |
| DB balance verification | Yes | Yes |

### Differences in Assertions
- Python verifies that each transaction's `linked_transaction_id` is in the set of transaction IDs (bidirectional linking). Go verifies the count of transfer transactions and checks `linkedTransactionId` in the API response.
- Go includes API response assertions (`isTransfer`, `linkedTransactionId`); Python only checks DB state.
- Python includes a `categoryId` in the transfer request; Go does not (different transfer API design).
- Both use the same amounts (200 same-currency, 100/92 cross-currency).

### Missing Scenarios
- Neither tests transfer deletion and balance rollback.
- Neither tests transfer between accounts of different users (ownership validation).
- Neither tests transfer with zero amount.

## Notes
- The transfer scenarios are functionally identical between Python and Go.
- Go uses `testutil.CreateTestAccount` for direct DB setup; Python uses `_create_account` helper that goes through the API.
- Python requires a `categoryId` for transfers; Go's transfer API does not require it.
