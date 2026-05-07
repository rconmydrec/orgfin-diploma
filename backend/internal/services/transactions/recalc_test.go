package transactions

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/models"
)

// dec is a small helper to build decimals from int64 in tests.
func dec(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

// decFromStr parses a decimal value from a string. Test failures here would
// indicate a malformed test input rather than a bug, so we panic.
func decFromStr(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// ptrDec returns a pointer to the supplied decimal value.
func ptrDec(d decimal.Decimal) *decimal.Decimal { return &d }

// TestRecalculateFromTransactions exercises the pure recalc helper that is
// shared by the service and the reconcile-balances CLI. Each case represents
// a chain of transactions on a single account starting from a known initial
// balance and asserts the final running balance the helper produces.
//
// The "adjustment in the middle" case is the regression guard for the bug
// fixed in this change set: prior to the fix the loop overwrote the running
// balance with tx.Amount (the *delta* of the adjustment) instead of
// *tx.NewBalance (the *target*), which produced wildly wrong totals whenever
// recalc ran across an adjustment.
func TestRecalculateFromTransactions(t *testing.T) {
	type tx struct {
		amount       decimal.Decimal
		isIncome     bool
		isAdjustment bool
		newBalance   *decimal.Decimal
		id           int
	}

	cases := []struct {
		name     string
		initial  decimal.Decimal
		txs      []tx
		expected decimal.Decimal
	}{
		{
			name:    "no transactions",
			initial: dec(1000),
			txs:     nil,
			// initial balance is returned unchanged.
			expected: dec(1000),
		},
		{
			name:    "regular income only",
			initial: dec(0),
			txs: []tx{
				{amount: dec(100), isIncome: true},
				{amount: dec(200), isIncome: true},
			},
			expected: dec(300),
		},
		{
			name:    "regular income and expense",
			initial: dec(1000),
			txs: []tx{
				{amount: dec(50), isIncome: false},
				{amount: dec(200), isIncome: true},
				{amount: dec(75), isIncome: false},
			},
			// 1000 - 50 + 200 - 75 = 1075
			expected: dec(1075),
		},
		{
			name:    "adjustment in the middle (the regression case)",
			initial: dec(1000),
			txs: []tx{
				// expense 100 -> 900
				{amount: dec(100), isIncome: false},
				// adjustment from 900 -> 1500. Delta is 600 (income),
				// the fixed helper MUST use *tx.NewBalance = 1500, not 600.
				{amount: dec(600), isIncome: true, isAdjustment: true, newBalance: ptrDec(dec(1500))},
				// income 200 -> 1700
				{amount: dec(200), isIncome: true},
			},
			// 1500 + 200 = 1700
			expected: dec(1700),
		},
		{
			name:    "downward adjustment",
			initial: dec(5000),
			txs: []tx{
				{amount: dec(200), isIncome: false},
				// 4800 -> 2000. Delta 2800 (expense), target 2000.
				{amount: dec(2800), isIncome: false, isAdjustment: true, newBalance: ptrDec(dec(2000))},
				{amount: dec(100), isIncome: true},
				{amount: dec(50), isIncome: false},
			},
			// 2000 + 100 - 50 = 2050
			expected: dec(2050),
		},
		{
			name:    "adjustment as last transaction",
			initial: dec(1000),
			txs: []tx{
				{amount: dec(100), isIncome: true},
				{amount: dec(200), isIncome: false},
				// 900 -> 1234.56
				{amount: decFromStr("334.56"), isIncome: true, isAdjustment: true, newBalance: ptrDec(decFromStr("1234.56"))},
			},
			expected: decFromStr("1234.56"),
		},
		{
			name:    "adjustment as first transaction",
			initial: dec(0),
			txs: []tx{
				// 0 -> 500
				{amount: dec(500), isIncome: true, isAdjustment: true, newBalance: ptrDec(dec(500))},
				{amount: dec(100), isIncome: true},
				{amount: dec(50), isIncome: false},
			},
			// 500 + 100 - 50 = 550
			expected: dec(550),
		},
		{
			name:    "multiple adjustments",
			initial: dec(1000),
			txs: []tx{
				{amount: dec(100), isIncome: true},
				// 1100 -> 2000
				{amount: dec(900), isIncome: true, isAdjustment: true, newBalance: ptrDec(dec(2000))},
				{amount: dec(300), isIncome: false},
				// 1700 -> 750
				{amount: dec(950), isIncome: false, isAdjustment: true, newBalance: ptrDec(dec(750))},
				{amount: dec(50), isIncome: true},
			},
			// 750 + 50 = 800
			expected: dec(800),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			models := make([]*models.Transaction, len(tc.txs))
			for i, raw := range tc.txs {
				m := newTxModel(raw.id, raw.amount, raw.isIncome, raw.isAdjustment, raw.newBalance)
				models[i] = m
			}
			got, err := RecalculateFromTransactions(tc.initial, models)
			require.NoError(t, err)
			assert.True(t, tc.expected.Equal(got),
				"expected %s, got %s", tc.expected.String(), got.String())
		})
	}
}

// TestRecalculateFromTransactions_AdjustmentNilNewBalance verifies the
// invariant violation: an adjustment row with nil NewBalance MUST cause the
// helper to return an error mentioning the offending transaction ID.
func TestRecalculateFromTransactions_AdjustmentNilNewBalance(t *testing.T) {
	txs := []*models.Transaction{
		newTxModel(1, dec(100), true, false, nil),
		newTxModel(42, dec(500), true, true, nil), // bogus adjustment, no new_balance
	}

	_, err := RecalculateFromTransactions(dec(0), txs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "42", "error should mention the offending transaction ID")
	assert.Contains(t, err.Error(), "nil new_balance")
}

// newTxModel builds a *models.Transaction populated with the fields the
// recalc helper actually inspects. id is optional (zero is fine for cases
// that don't care).
func newTxModel(id int, amount decimal.Decimal, isIncome, isAdjustment bool, newBalance *decimal.Decimal) *models.Transaction {
	return &models.Transaction{
		ID:           id,
		Amount:       amount,
		IsIncome:     isIncome,
		IsAdjustment: isAdjustment,
		NewBalance:   newBalance,
	}
}
