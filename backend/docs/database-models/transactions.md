# Transactions

**Table:** `transactions`
**Model:** `models.Transaction` (`backend/internal/models/transaction.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `account_id` | `int` | `INT` | No | FK to `accounts.id` |
| `amount` | `decimal.Decimal` | `NUMERIC` | No | Transaction amount |
| `new_balance` | `*decimal.Decimal` | `NUMERIC` | Yes | Account balance after this transaction |
| `category_id` | `*int` | `INT` | Yes | FK to `user_categories.id` |
| `label` | `*string` | `VARCHAR(255)` | Yes | Transaction label. Was `VARCHAR(50)` before migration 00006. DTO-layer cap is 255 characters. |
| `is_income` | `bool` | `BOOLEAN` | No | `true` = income, `false` = expense |
| `is_transfer` | `bool` | `BOOLEAN` | No | Whether this is a transfer between accounts |
| `linked_transaction_id` | `*int` | `INT` | Yes | FK to `transactions.id` (paired transfer transaction) |
| `notes` | `*string` | `VARCHAR` | Yes | Additional notes. Unlimited at the DB level; DTO-layer soft cap is 4000 characters. |
| `date_time` | `*time.Time` | `TIMESTAMPTZ` | Yes | Transaction date/time |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag (hidden from JSON) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |
| `is_adjustment` | `bool` | `BOOLEAN` | No | Balance adjustment transaction |
| `exclude_from_reports` | `bool` | `BOOLEAN` | No | Exclude from financial reports |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [accounts](accounts.md) | `account_id` | Many-to-One |
| Belongs to | [user_categories](user-categories.md) | `category_id` | Many-to-One (nullable) |
| Self-reference | [transactions](transactions.md) | `linked_transaction_id` | One-to-One (nullable) |

## Notes

- **Transfers** between accounts are implemented as a pair of transactions linked via `linked_transaction_id`. One transaction is the debit (expense), the other is the credit (income).
- **Adjustments** (`is_adjustment = true`) are special transactions that correct the account balance without representing a real income/expense.

### Adjustment row invariants

For rows with `is_adjustment = true` the two numeric columns have **different** meanings than for regular transactions:

- `amount` stores the **absolute delta** between the previous balance and the target — i.e. `|target - previous|`.
- `new_balance` stores the **target** balance the user typed in (the trusted post-adjustment balance).
- `is_income` is `true` when the adjustment increases the balance (`target > previous`) and `false` otherwise. It identifies the *direction* of the delta but is not used by recalc.

Balance recalculation (`services/transactions.recalculateBalances`, and the pure helper `services/transactions.RecalculateFromTransactions`) **keys off `new_balance`, never `amount`** for adjustment rows. Concretely, when the recalc loop reaches an adjustment row it sets the running balance to `*tx.NewBalance` and ignores `tx.Amount`. The historical bug fixed in 2026-05 used `tx.Amount` (the delta) instead of `*tx.NewBalance` (the target) and silently corrupted account balances any time recalc ran across an adjustment.

If an adjustment row is encountered with `new_balance IS NULL` the recalc helper returns an error mentioning the offending transaction ID. Every adjustment created by the codebase sets `new_balance`; a `NULL` here indicates either data corruption or a row written by the pre-fix buggy path.

### Reconciliation rows

Rows inserted by the `reconcile-balances` CLI (see [`docs/cli-tools.md`](../cli-tools.md)) follow the same adjustment contract: `is_adjustment = true`, `new_balance` = the trusted stored `accounts.balance` at the time of the run, `amount` = `|stored - recalc|`, `is_income` = `(stored > recalc)`, `exclude_from_reports = true`, `label = "Reconciliation adjustment"`. They are indistinguishable from a normal `is_adjustment` row to recalc.
