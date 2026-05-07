# CLI Tools

One-off command-line utilities that ship as separate binaries under `backend/cmd/`. These are not wired into the HTTP server, the Asynq worker, or any scheduled job — they are manual operator tools.

## `reconcile-balances`

**Path:** `backend/cmd/reconcile-balances/`
**Binary:** `bin/reconcile-balances`
**Status:** Manual one-shot. Not scheduled, not wired into any service.

### Purpose

Some accounts in legacy data have `accounts.balance` that diverges from a recalc-from-scratch result, because long ago balances were edited directly without creating an adjustment-type transaction. The user does not want to rewrite history; they want a single compensating row inserted per divergent account so future calls to `recalculateBalances` land on the currently trusted stored balance.

The tool also serves as a one-time cleanup after the recalc bug fixed in 2026-05 (the loop used `tx.Amount` instead of `*tx.NewBalance` for adjustment rows). On accounts that were silently re-written by the buggy path before the fix, this CLI re-aligns recalc and stored.

### What it does

For every in-scope account:

1. Reads `accounts.balance` (`stored_balance`) and `accounts.initial_balance`.
2. Computes `recalc_balance` from `initial_balance` plus the chronological transaction chain, using the **same algorithm** as the live service via the shared helper `services/transactions.RecalculateFromTransactions`. The CLI cannot drift from the live recalc.
3. Compares stored vs recalc with **exact `decimal.Decimal` equality** (no epsilon).
4. If equal, the account is skipped silently (printed only when `--verbose`).
5. If not equal, the CLI computes `diff = stored - recalc`, `direction` (`up` if `diff > 0` else `down`), `amount = |diff|`, `is_income = (diff > 0)`.

In **dry-run** mode the tool prints a table and writes nothing.

In **write** mode the tool inserts exactly one row per divergent account into `transactions`:

| Column | Value |
|--------|-------|
| `user_id` | the account's owner |
| `account_id` | the divergent account |
| `amount` | `|diff|` (full `decimal.Decimal` precision) |
| `new_balance` | `stored_balance` (the trusted target) |
| `is_income` | `(diff > 0)` |
| `is_adjustment` | `true` |
| `is_transfer` | `false` |
| `linked_transaction_id` | `NULL` |
| `category_id` | `NULL` |
| `exclude_from_reports` | `true` |
| `date_time` | `time.Now().UTC()` (after all existing transactions) |
| `label` | `"Reconciliation adjustment"` |
| `notes` | `"Inserted by reconcile-balances CLI on YYYY-MM-DD"` |
| `is_deleted` | `false` |

`accounts.balance` is **not** modified — it already equals the trusted target.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `false` | Print the divergence table and do not write any rows. |
| `--user-id` | int | `0` (= all users) | Limit the run to one user's accounts. |
| `--account-id` | int | `0` (= all accounts in scope) | Limit the run to a single account. The tool resolves the user from the account. |
| `--verbose` | bool | `false` | Print a short summary line ("Inspected N account(s); M divergent.") in addition to the per-row output. |

If both `--user-id` and `--account-id` are set, the tool errors out with a clear message when the account does not belong to the supplied user.

### Build

From `backend/`:

```bash
go build -o bin/reconcile-balances ./cmd/reconcile-balances
```

There is no Makefile target — the CLI is a manual one-shot.

### Run

The tool uses the same database environment variables as the main app (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`). `DATABASE_URL` is still honored if set and takes precedence. A `.env` file is loaded via godotenv if present.

```bash
# Dry run across all users (recommended first step).
./bin/reconcile-balances --dry-run

# Dry run for one user.
./bin/reconcile-balances --dry-run --user-id=42

# Surgical insert for a single account.
./bin/reconcile-balances --account-id=109

# Full write across the whole database.
./bin/reconcile-balances
```

### Dry-run output format

Plain text, fixed-width columns, written to stdout. Header row, then one row per **divergent** account. Trailing summary line.

```
account_id  user_id  name                 currency  stored        recalc        diff          direction  would_insert_amount
       42        7  Main checking        EUR       10596.560000  -1310.830000   11907.390000  up         11907.390000
      109       14  Old savings          USD        2000.000000   2350.000000    -350.000000  down       350.000000
---
2 divergent account(s) out of 87 inspected. Dry run — no changes written.
```

Numbers are formatted with full `decimal.Decimal` precision (no premature rounding).

### Write-mode output format

One log line per insert:

```
INSERT account_id=42 user_id=7 direction=up amount=11907.390000 new_balance=10596.560000 transaction_id=99812
```

Final summary line:

```
Done. Inserted N reconciliation adjustments across M divergent account(s) (out of K inspected).
```

### Idempotency

A second invocation immediately after a successful first one inserts **zero** new rows: the first run's compensating row makes recalc-from-scratch land on the stored balance, so no divergence remains.

The integration test `TestRun_IntegrationIdempotency` in `cmd/reconcile-balances/runner_test.go` enforces this guarantee.

### Exit codes

- `0` — success (including a no-divergence run).
- `1` — DB connection or operation failure, or a flag-validation error from `Run`.
- `2` — database connection settings are missing (neither `DATABASE_URL` nor the `DB_*` parts produce a usable DSN).
