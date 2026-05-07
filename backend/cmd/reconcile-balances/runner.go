// Package main implements the reconcile-balances one-shot CLI.
//
// This file (runner.go) holds the testable pieces: pure helpers and the Run
// function, which is parameterised on a *sql.DB and the parsed flags. The
// thin main() function (main.go) wires environment loading, flag parsing,
// and DB connection, then calls Run.
package main

import (
	"database/sql"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/shopspring/decimal"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/transactions"
)

// Options carries the CLI inputs that drive a single Run invocation.
type Options struct {
	DryRun    bool
	UserID    int // 0 = all users
	AccountID int // 0 = all accounts in scope
	Verbose   bool
}

// Divergence describes a single divergent account: its stored vs. recalc
// balance, the resulting compensating-row direction, and the absolute
// amount the would-be insert carries.
type Divergence struct {
	AccountID    int
	UserID       int
	Name         string
	CurrencyCode string
	Stored       decimal.Decimal
	Recalc       decimal.Decimal
}

// Diff returns Stored - Recalc.
func (d Divergence) Diff() decimal.Decimal { return d.Stored.Sub(d.Recalc) }

// Direction reports whether recalc must be brought up or down to match
// stored. "up" if stored > recalc (we need an income compensating row),
// "down" if stored < recalc.
func (d Divergence) Direction() string {
	if d.Diff().GreaterThan(decimal.Zero) {
		return "up"
	}
	return "down"
}

// Amount is the unsigned magnitude of the divergence.
func (d Divergence) Amount() decimal.Decimal { return d.Diff().Abs() }

// IsIncome is the value to write into the compensating transaction's
// is_income column. Income when stored > recalc.
func (d Divergence) IsIncome() bool { return d.Diff().GreaterThan(decimal.Zero) }

// accountRow is the minimal account info the runner needs.
type accountRow struct {
	ID             int
	UserID         int
	Name           string
	CurrencyID     int
	CurrencyCode   string
	InitialBalance decimal.Decimal
	Balance        decimal.Decimal
}

// loadAccounts fetches accounts in scope according to the options.
//
// Default (no flags): all non-deleted, non-archived accounts of all users.
// With --user-id: scope to that user.
// With --account-id: scope to that single account.
// With both: validates that the account belongs to the supplied user and
// returns a clear error if it does not.
func loadAccounts(db *sql.DB, opts Options) ([]accountRow, error) {
	if opts.AccountID > 0 {
		acc, err := loadOneAccount(db, opts.AccountID)
		if err != nil {
			return nil, err
		}
		if opts.UserID > 0 && acc.UserID != opts.UserID {
			return nil, fmt.Errorf("account %d does not belong to user %d (owner is user %d)", opts.AccountID, opts.UserID, acc.UserID)
		}
		return []accountRow{acc}, nil
	}

	query := `
		SELECT a.id, a.user_id, a.name, a.currency_id, c.code,
		       a.initial_balance, a.balance
		FROM accounts a
		JOIN currencies c ON a.currency_id = c.id
		WHERE a.is_deleted = false AND a.is_archived = false`
	args := []interface{}{}
	if opts.UserID > 0 {
		query += " AND a.user_id = $1"
		args = append(args, opts.UserID)
	}
	query += " ORDER BY a.id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var out []accountRow
	for rows.Next() {
		var a accountRow
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.CurrencyID, &a.CurrencyCode, &a.InitialBalance, &a.Balance); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return out, nil
}

// loadOneAccount fetches a single account by ID. Returns sql.ErrNoRows if
// the account is missing, deleted, or archived.
func loadOneAccount(db *sql.DB, id int) (accountRow, error) {
	query := `
		SELECT a.id, a.user_id, a.name, a.currency_id, c.code,
		       a.initial_balance, a.balance
		FROM accounts a
		JOIN currencies c ON a.currency_id = c.id
		WHERE a.id = $1 AND a.is_deleted = false AND a.is_archived = false`
	var a accountRow
	err := db.QueryRow(query, id).Scan(&a.ID, &a.UserID, &a.Name, &a.CurrencyID, &a.CurrencyCode, &a.InitialBalance, &a.Balance)
	if err != nil {
		return accountRow{}, fmt.Errorf("load account %d: %w", id, err)
	}
	return a, nil
}

// loadTransactionsForRecalc fetches the rows the recalc helper consumes for
// a single account in chronological order, mirroring
// TransactionRepository.GetByAccountIDForRecalc.
func loadTransactionsForRecalc(db *sql.DB, accountID int) ([]*models.Transaction, error) {
	query := `
		SELECT id, amount, new_balance, is_income, is_adjustment
		FROM transactions
		WHERE account_id = $1 AND is_deleted = false
		ORDER BY date_time ASC, id ASC`
	rows, err := db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query transactions for account %d: %w", accountID, err)
	}
	defer rows.Close()

	var txs []*models.Transaction
	for rows.Next() {
		tx := &models.Transaction{}
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.NewBalance, &tx.IsIncome, &tx.IsAdjustment); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

// detectDivergences computes recalc-from-scratch for every account, returns
// only the divergent ones (in input order), and reports the inspection
// count. Uses the canonical helper in services/transactions so the CLI
// cannot drift from the live recalc algorithm.
func detectDivergences(db *sql.DB, accounts []accountRow) ([]Divergence, error) {
	var diverged []Divergence
	for _, a := range accounts {
		txs, err := loadTransactionsForRecalc(db, a.ID)
		if err != nil {
			return nil, err
		}
		recalc, err := transactions.RecalculateFromTransactions(a.InitialBalance, txs)
		if err != nil {
			return nil, fmt.Errorf("recalc account %d: %w", a.ID, err)
		}
		if a.Balance.Equal(recalc) {
			continue
		}
		diverged = append(diverged, Divergence{
			AccountID:    a.ID,
			UserID:       a.UserID,
			Name:         a.Name,
			CurrencyCode: a.CurrencyCode,
			Stored:       a.Balance,
			Recalc:       recalc,
		})
	}
	return diverged, nil
}

// reconciliationLabel is the label set on every compensating row inserted
// by this CLI. The constant is exported so tests can assert against it.
const reconciliationLabel = "Reconciliation adjustment"

// insertCompensatingRow inserts a single is_adjustment=true transaction
// that brings the recalc result up to (or down to) the stored balance.
//
// The function does NOT touch accounts.balance because, by the time we run,
// stored == accounts.balance is the trusted target.
func insertCompensatingRow(db *sql.DB, d Divergence, runID string, now time.Time) (int, error) {
	notes := fmt.Sprintf("Inserted by reconcile-balances CLI on %s", runID)
	label := reconciliationLabel

	const query = `
		INSERT INTO transactions (
			user_id, account_id, amount, new_balance, category_id, label,
			is_income, is_transfer, linked_transaction_id,
			notes, date_time, is_deleted, is_adjustment, exclude_from_reports
		)
		VALUES ($1, $2, $3, $4, NULL, $5, $6, false, NULL, $7, $8, false, true, true)
		RETURNING id`

	var id int
	err := db.QueryRow(query,
		d.UserID,
		d.AccountID,
		d.Amount(),
		d.Stored, // new_balance is the target
		label,
		d.IsIncome(),
		notes,
		now.UTC(),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert compensating row for account %d: %w", d.AccountID, err)
	}
	return id, nil
}

// Run executes one reconciliation pass. It reads accounts in scope, detects
// divergences, and either prints the dry-run table or inserts compensating
// rows. Output is written to out.
//
// Run is idempotent: a second invocation immediately after a successful
// first one inserts zero rows, because the first run made the recalc
// result equal the stored balance for every divergent account.
func Run(db *sql.DB, opts Options, out io.Writer, now time.Time) error {
	accounts, err := loadAccounts(db, opts)
	if err != nil {
		return err
	}

	diverged, err := detectDivergences(db, accounts)
	if err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Fprintf(out, "Inspected %d account(s); %d divergent.\n", len(accounts), len(diverged))
	}

	if opts.DryRun {
		printDryRunTable(out, diverged)
		fmt.Fprintf(out, "---\n%d divergent account(s) out of %d inspected. Dry run — no changes written.\n",
			len(diverged), len(accounts))
		return nil
	}

	runID := now.UTC().Format("2006-01-02")
	inserted := 0
	for _, d := range diverged {
		id, err := insertCompensatingRow(db, d, runID, now)
		if err != nil {
			return err
		}
		inserted++
		fmt.Fprintf(out, "INSERT account_id=%d user_id=%d direction=%s amount=%s new_balance=%s transaction_id=%d\n",
			d.AccountID, d.UserID, d.Direction(), d.Amount().String(), d.Stored.String(), id)
	}
	fmt.Fprintf(out, "Done. Inserted %d reconciliation adjustments across %d divergent account(s) (out of %d inspected).\n",
		inserted, len(diverged), len(accounts))
	return nil
}

// printDryRunTable writes the divergent-accounts table to out. The header
// columns match the contract in requirements.md exactly. Decimal columns
// keep full decimal.Decimal precision (no premature rounding).
func printDryRunTable(out io.Writer, diverged []Divergence) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	header := "account_id\tuser_id\tname\tcurrency\tstored\trecalc\tdiff\tdirection\twould_insert_amount\n"
	fmt.Fprint(tw, header)
	for _, d := range diverged {
		fmt.Fprintf(tw, "%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			d.AccountID,
			d.UserID,
			d.Name,
			d.CurrencyCode,
			d.Stored.String(),
			d.Recalc.String(),
			d.Diff().String(),
			d.Direction(),
			d.Amount().String(),
		)
	}
	_ = tw.Flush()
}
