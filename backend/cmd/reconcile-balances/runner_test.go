package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/testutil"
)

// dec is a small decimal builder used throughout the tests.
func dec(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

// decFromStr parses a decimal string. Test failures here would indicate a
// bad test input rather than a runtime bug, so we panic.
func decFromStr(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// TestDivergenceMath exercises the pure Divergence helpers without touching
// the DB. The cases cover: equal balances (no row produced), stored above
// recalc (income compensating row), stored below recalc (expense
// compensating row), sub-cent diffs (must still be reported as divergent
// because comparison is exact), and large diffs.
func TestDivergenceMath(t *testing.T) {
	cases := []struct {
		name             string
		stored           decimal.Decimal
		recalc           decimal.Decimal
		wantDirection    string
		wantAmount       decimal.Decimal
		wantIsIncome     bool
		wantDiverges     bool // true if Stored != Recalc
		wantAbsDiffEqual decimal.Decimal
	}{
		{
			name:         "equal balances",
			stored:       dec(1000),
			recalc:       dec(1000),
			wantDiverges: false,
		},
		{
			name:             "stored greater (recalc missed an income)",
			stored:           decFromStr("10596.560000"),
			recalc:           decFromStr("-1310.830000"),
			wantDirection:    "up",
			wantAmount:       decFromStr("11907.390000"),
			wantIsIncome:     true,
			wantDiverges:     true,
			wantAbsDiffEqual: decFromStr("11907.390000"),
		},
		{
			name:             "stored less (recalc overshot)",
			stored:           dec(2000),
			recalc:           dec(2350),
			wantDirection:    "down",
			wantAmount:       dec(350),
			wantIsIncome:     false,
			wantDiverges:     true,
			wantAbsDiffEqual: dec(350),
		},
		{
			name:             "sub-cent positive diff",
			stored:           decFromStr("100.001"),
			recalc:           dec(100),
			wantDirection:    "up",
			wantAmount:       decFromStr("0.001"),
			wantIsIncome:     true,
			wantDiverges:     true,
			wantAbsDiffEqual: decFromStr("0.001"),
		},
		{
			name:             "sub-cent negative diff",
			stored:           dec(100),
			recalc:           decFromStr("100.001"),
			wantDirection:    "down",
			wantAmount:       decFromStr("0.001"),
			wantIsIncome:     false,
			wantDiverges:     true,
			wantAbsDiffEqual: decFromStr("0.001"),
		},
		{
			name:             "very large positive diff",
			stored:           decFromStr("999999999.99"),
			recalc:           decFromStr("0.01"),
			wantDirection:    "up",
			wantAmount:       decFromStr("999999999.98"),
			wantIsIncome:     true,
			wantDiverges:     true,
			wantAbsDiffEqual: decFromStr("999999999.98"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Divergence{Stored: tc.stored, Recalc: tc.recalc}
			diverges := !d.Diff().IsZero()
			assert.Equal(t, tc.wantDiverges, diverges)
			if !tc.wantDiverges {
				return
			}
			assert.Equal(t, tc.wantDirection, d.Direction())
			assert.True(t, tc.wantAmount.Equal(d.Amount()),
				"want amount %s, got %s", tc.wantAmount, d.Amount())
			assert.Equal(t, tc.wantIsIncome, d.IsIncome())
			assert.True(t, tc.wantAbsDiffEqual.Equal(d.Amount()))
		})
	}
}

// TestRun_IntegrationIdempotency seeds a divergent account in the test
// database, runs the reconcile logic once and asserts the inserted row
// matches the requirements, then runs it again and asserts no further row
// was inserted (idempotency).
func TestRun_IntegrationIdempotency(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Seed a fresh user and account directly via the testutil helpers. We
	// don't need the full TestApp here because the CLI only reads accounts
	// and transactions and writes one transactions row.
	app := testutil.NewTestApp(t)

	email := fmt.Sprintf("reconcile-cli-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"
	testutil.CleanupUserByEmail(t, db, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, db, userID)

	usdID := testutil.GetCurrencyID(t, db, "USD")
	accTypeID := testutil.GetAccountTypeID(t, db, false)

	// Account A: legitimately divergent. We seed it with a stored balance
	// of 5000 but no transactions and initial_balance of 1000 — recalc
	// will produce 1000, stored is 5000, diff = +4000, direction = up.
	var divergentAccID int
	err := db.QueryRow(`
		INSERT INTO accounts (user_id, name, currency_id, account_type_id, initial_balance, balance,
		                     is_hidden, show_in_reports, is_deleted, is_archived, opening_date)
		VALUES ($1, $2, $3, $4, 1000, 5000, false, true, false, false, NOW())
		RETURNING id
	`, userID, "Divergent Reconcile Test", usdID, accTypeID).Scan(&divergentAccID)
	require.NoError(t, err)
	defer testutil.DeleteTestAccount(t, db, divergentAccID)

	// Account B: clean (no divergence). Seed with stored=2000 and no
	// transactions; initial_balance=2000 so recalc=stored.
	var cleanAccID int
	err = db.QueryRow(`
		INSERT INTO accounts (user_id, name, currency_id, account_type_id, initial_balance, balance,
		                     is_hidden, show_in_reports, is_deleted, is_archived, opening_date)
		VALUES ($1, $2, $3, $4, 2000, 2000, false, true, false, false, NOW())
		RETURNING id
	`, userID, "Clean Reconcile Test", usdID, accTypeID).Scan(&cleanAccID)
	require.NoError(t, err)
	defer testutil.DeleteTestAccount(t, db, cleanAccID)

	opts := Options{
		UserID:  userID, // scope to this user so other accounts in the test DB don't interfere
		Verbose: true,
	}

	// First run: must insert exactly one compensating row on the divergent
	// account. The clean account must be left alone.
	var buf bytes.Buffer
	now := time.Now().UTC()
	require.NoError(t, Run(db, opts, &buf, now))

	out := buf.String()
	t.Logf("first run output:\n%s", out)
	assert.Contains(t, out, fmt.Sprintf("account_id=%d", divergentAccID))
	assert.Contains(t, out, "direction=up")
	assert.Contains(t, out, "Done. Inserted 1 reconciliation adjustments")

	// Verify the inserted row's fields match the contract.
	insertedTxs := loadAdjustmentRows(t, db, divergentAccID)
	require.Len(t, insertedTxs, 1, "exactly one reconciliation row must be inserted")
	row := insertedTxs[0]
	assert.Equal(t, userID, row.UserID)
	assert.Equal(t, divergentAccID, row.AccountID)
	assert.True(t, dec(4000).Equal(row.Amount), "amount must be |stored-recalc|=4000, got %s", row.Amount)
	require.NotNil(t, row.NewBalance, "new_balance must be set on adjustment rows")
	assert.True(t, dec(5000).Equal(*row.NewBalance), "new_balance must equal stored balance, got %s", row.NewBalance)
	assert.True(t, row.IsIncome, "is_income must be true when stored > recalc")
	assert.True(t, row.IsAdjustment)
	assert.False(t, row.IsTransfer)
	assert.True(t, row.ExcludeFromReports)
	require.NotNil(t, row.Label)
	assert.Equal(t, reconciliationLabel, *row.Label)
	require.NotNil(t, row.Notes)
	assert.Contains(t, *row.Notes, "Inserted by reconcile-balances CLI")

	// Verify accounts.balance was NOT modified.
	var afterStored decimal.Decimal
	err = db.QueryRow("SELECT balance FROM accounts WHERE id = $1", divergentAccID).Scan(&afterStored)
	require.NoError(t, err)
	assert.True(t, dec(5000).Equal(afterStored), "accounts.balance must not be modified by the CLI")

	// Verify no row was inserted on the clean account.
	cleanRows := loadAdjustmentRows(t, db, cleanAccID)
	assert.Len(t, cleanRows, 0, "clean account must not get any reconciliation row")

	// Second run: must insert zero rows. After the first insert,
	// recalc-from-scratch lands on the stored balance because the row's
	// new_balance == stored, so no divergence remains.
	buf.Reset()
	require.NoError(t, Run(db, opts, &buf, now.Add(time.Second)))

	out2 := buf.String()
	t.Logf("second run output:\n%s", out2)
	assert.Contains(t, out2, "Done. Inserted 0 reconciliation adjustments across 0 divergent account(s)")

	// Still exactly one reconciliation row exists for the divergent account.
	finalRows := loadAdjustmentRows(t, db, divergentAccID)
	assert.Len(t, finalRows, 1, "second run must not insert any new rows")
}

// TestRun_DryRunDoesNotWrite confirms that --dry-run never inserts a row.
func TestRun_DryRunDoesNotWrite(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := testutil.NewTestApp(t)

	email := fmt.Sprintf("reconcile-cli-dry-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"
	testutil.CleanupUserByEmail(t, db, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, db, userID)

	usdID := testutil.GetCurrencyID(t, db, "USD")
	accTypeID := testutil.GetAccountTypeID(t, db, false)

	var accID int
	err := db.QueryRow(`
		INSERT INTO accounts (user_id, name, currency_id, account_type_id, initial_balance, balance,
		                     is_hidden, show_in_reports, is_deleted, is_archived, opening_date)
		VALUES ($1, 'Dry Run Test', $2, $3, 100, 999, false, true, false, false, NOW())
		RETURNING id
	`, userID, usdID, accTypeID).Scan(&accID)
	require.NoError(t, err)
	defer testutil.DeleteTestAccount(t, db, accID)

	var buf bytes.Buffer
	require.NoError(t, Run(db, Options{UserID: userID, DryRun: true}, &buf, time.Now().UTC()))

	out := buf.String()
	t.Logf("dry-run output:\n%s", out)
	assert.Contains(t, out, "account_id")
	assert.Contains(t, out, "user_id")
	assert.Contains(t, out, "currency")
	assert.Contains(t, out, "would_insert_amount")
	assert.Contains(t, out, "Dry run — no changes written.")
	assert.Contains(t, out, "1 divergent account(s)")

	// Assert no rows were written.
	rows := loadAdjustmentRows(t, db, accID)
	assert.Len(t, rows, 0, "dry-run must not insert any rows")
}

// TestRun_AccountIDUserIDMismatch verifies that combining --user-id with
// --account-id where the account belongs to a different user errors out
// with a clear message.
func TestRun_AccountIDUserIDMismatch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	app := testutil.NewTestApp(t)

	email := fmt.Sprintf("reconcile-cli-mismatch-%d@example.com", time.Now().UnixNano())
	password := "TestPass123!"
	testutil.CleanupUserByEmail(t, db, email)
	userID := testutil.CreateTestUser(t, app, email, password)
	defer testutil.CleanupUser(t, db, userID)

	usdID := testutil.GetCurrencyID(t, db, "USD")
	accTypeID := testutil.GetAccountTypeID(t, db, false)
	accID := testutil.CreateTestAccount(t, db, userID, "Owned by user", usdID, accTypeID, 100)
	defer testutil.DeleteTestAccount(t, db, accID)

	// Use a deliberately mismatched user-id. Pick a value that almost
	// certainly does not match the real owner.
	var buf bytes.Buffer
	err := Run(db, Options{
		UserID:    userID + 999_999,
		AccountID: accID,
	}, &buf, time.Now().UTC())
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "does not belong")
}

// adjRow is a minimal projection of the columns we care about when
// asserting on inserted reconciliation rows.
type adjRow struct {
	UserID             int
	AccountID          int
	Amount             decimal.Decimal
	NewBalance         *decimal.Decimal
	IsIncome           bool
	IsAdjustment       bool
	IsTransfer         bool
	ExcludeFromReports bool
	Label              *string
	Notes              *string
}

// loadAdjustmentRows returns every reconciliation-labelled row for the
// account, in insert order.
func loadAdjustmentRows(t *testing.T, db *sql.DB, accountID int) []adjRow {
	t.Helper()
	rows, err := db.Query(`
		SELECT user_id, account_id, amount, new_balance, is_income, is_adjustment,
		       is_transfer, exclude_from_reports, label, notes
		FROM transactions
		WHERE account_id = $1 AND is_deleted = false AND label = $2
		ORDER BY id ASC
	`, accountID, reconciliationLabel)
	require.NoError(t, err)
	defer rows.Close()

	var out []adjRow
	for rows.Next() {
		var r adjRow
		require.NoError(t, rows.Scan(&r.UserID, &r.AccountID, &r.Amount, &r.NewBalance,
			&r.IsIncome, &r.IsAdjustment, &r.IsTransfer, &r.ExcludeFromReports, &r.Label, &r.Notes))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}
