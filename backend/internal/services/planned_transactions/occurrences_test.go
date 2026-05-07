package planned_transactions

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRuleJSON(rule map[string]interface{}) *string {
	b, _ := json.Marshal(rule)
	s := string(b)
	return &s
}

func TestGenerateOccurrences_NonRecurringInRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Test", true, false, nil, start, end)

	require.Len(t, occs, 1)
	assert.Equal(t, 1, occs[0].PlannedTransactionID)
	assert.Equal(t, 1, occs[0].CurrencyID)
	assert.Equal(t, "100", occs[0].Amount.String())
	assert.Equal(t, "Test", occs[0].Label)
	assert.False(t, occs[0].IsRecurring)
	// Time component should be stripped
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
}

func TestGenerateOccurrences_NonRecurringOutOfRange(t *testing.T) {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) // Before the range

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Test", true, false, nil, start, end)

	assert.Empty(t, occs)
}

func TestGenerateOccurrences_NonRecurringAfterRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC) // After the range

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Test", true, false, nil, start, end)

	assert.Empty(t, occs)
}

func TestGenerateOccurrences_NonRecurringOnBoundary(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	// On start date
	occs := generateOccurrences(1, 1, start, decimal.NewFromInt(100), false, "Test", true, false, nil, start, end)
	require.Len(t, occs, 1)

	// On end date
	occs = generateOccurrences(1, 1, end, decimal.NewFromInt(100), false, "Test", true, false, nil, start, end)
	require.Len(t, occs, 1)
}

func TestGenerateOccurrences_DailyRecurrence(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "daily",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(50), true, "Daily", true, true, rule, start, end)

	// Jan 1, 2, 3, 4, 5 = 5 occurrences
	require.Len(t, occs, 5)
	for i, occ := range occs {
		expected := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, occ.OccurrenceDate)
		assert.Equal(t, "50", occ.Amount.String())
		assert.True(t, occ.IsIncome)
	}
}

func TestGenerateOccurrences_DailyWithInterval(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "daily",
		"interval":  3,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(50), false, "Every 3 days", true, true, rule, start, end)

	// Jan 1, 4, 7, 10 = 4 occurrences
	require.Len(t, occs, 4)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), occs[3].OccurrenceDate)
}

func TestGenerateOccurrences_WeeklyRecurrence(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Thursday

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "weekly",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(75), false, "Weekly", true, true, rule, start, end)

	// Jan 1, 8, 15, 22, 29 = 5 occurrences (Feb 5 > end)
	require.Len(t, occs, 5)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC), occs[4].OccurrenceDate)
}

func TestGenerateOccurrences_WeeklyWithInterval(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "weekly",
		"interval":  2,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(75), false, "Bi-weekly", true, true, rule, start, end)

	// Every 2 weeks from Jan 1: Jan 1, 15, 29, Feb 12, 26 = 5 occurrences
	require.Len(t, occs, 5)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), occs[3].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC), occs[4].OccurrenceDate)
}

func TestGenerateOccurrences_MonthlyRecurrence(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "monthly",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(200), false, "Monthly", true, true, rule, start, end)

	// Jan 15, Feb 15, Mar 15, Apr 15, May 15, Jun 15 = 6 occurrences
	require.Len(t, occs, 6)
	for i, occ := range occs {
		expected := time.Date(2026, time.Month(1+i), 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, occ.OccurrenceDate)
	}
}

func TestGenerateOccurrences_YearlyRecurrence(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "yearly",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(500), true, "Yearly", true, true, rule, start, end)

	// Jun 15 of 2026, 2027, 2028, 2029, 2030 = 5 occurrences
	require.Len(t, occs, 5)
	for i, occ := range occs {
		expected := time.Date(2026+i, 6, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, occ.OccurrenceDate)
	}
}

func TestGenerateOccurrences_RecurrenceWithEndDate(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ruleEndDate := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "monthly",
		"interval":  1,
		"end_date":  ruleEndDate.Format(time.RFC3339),
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Limited", true, true, rule, start, end)

	// Jan 1, Feb 1, Mar 1 = 3 occurrences (Apr 1 > endDate)
	require.Len(t, occs, 3)
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
}

func TestGenerateOccurrences_RecurrenceWithCountLimit(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "monthly",
		"interval":  1,
		"count":     4,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Counted", true, true, rule, start, end)

	// Jan 1, Feb 1, Mar 1, Apr 1 = 4 occurrences (count limit)
	require.Len(t, occs, 4)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), occs[3].OccurrenceDate)
}

func TestGenerateOccurrences_CountFromPastStart(t *testing.T) {
	// Recurring started in the past. Count is total from original start, not just in the window.
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Started 3 months ago

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "monthly",
		"interval":  1,
		"count":     5, // Total of 5 from the start
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Past start", true, true, rule, start, end)

	// Occurrences generated from Jan 1: Jan 1, Feb 1, Mar 1, Apr 1, May 1 (5 total)
	// But only Apr 1 and May 1 are in the [Apr 1, Dec 31] window
	require.Len(t, occs, 2)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
}

func TestGenerateOccurrences_MonthlyDayOfMonth(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency":    "monthly",
		"interval":     1,
		"day_of_month": 20,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "DayOfMonth", true, true, rule, start, end)

	// Jan 20, Feb 20, Mar 20, Apr 20 = 4 occurrences
	require.Len(t, occs, 4)
	assert.Equal(t, time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), occs[3].OccurrenceDate)
}

func TestGenerateOccurrences_MonthlyDayOfMonthClamp(t *testing.T) {
	// Test that dayOfMonth > 28 is clamped to end-of-month for February
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency":    "monthly",
		"interval":     1,
		"day_of_month": 31,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "LastDay", true, true, rule, start, end)

	require.Len(t, occs, 4)
	// Jan 31
	assert.Equal(t, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	// Feb 28 (non-leap year 2026)
	assert.Equal(t, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	// Mar 31
	assert.Equal(t, time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
	// Apr 30
	assert.Equal(t, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), occs[3].OccurrenceDate)
}

func TestGenerateOccurrences_PastStartFutureOccurrences(t *testing.T) {
	// Recurring transaction started in the past but generates future occurrences
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC) // Way in the past

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "monthly",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Past", true, true, rule, start, end)

	// Should generate Jun 15, Jul 15, Aug 15 = 3 occurrences
	require.Len(t, occs, 3)
	assert.Equal(t, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), occs[2].OccurrenceDate)
}

func TestGenerateOccurrences_SafetyCap(t *testing.T) {
	// A recurring transaction that would generate infinite occurrences is capped at 1000
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC) // Very far future
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "daily",
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(1), false, "Lots", true, true, rule, start, end)

	// Should be capped at 1000
	assert.Len(t, occs, 1000)
}

func TestGenerateOccurrences_InvalidRuleJSON(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	invalidJSON := "this is not json"

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Invalid", true, true, &invalidJSON, start, end)

	assert.Empty(t, occs)
}

func TestGenerateOccurrences_UnknownFrequency(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "biweekly", // Not a valid frequency
		"interval":  1,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "Unknown", true, true, rule, start, end)

	// First occurrence is still generated (before the switch), then stops
	require.Len(t, occs, 1)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
}

func TestGenerateOccurrences_IntervalZeroDefaultsToOne(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "daily",
		"interval":  0, // Invalid, should default to 1
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(10), false, "ZeroInterval", true, true, rule, start, end)

	// Should work like interval=1: Jan 1, 2, 3, 4, 5 = 5 occurrences
	require.Len(t, occs, 5)
}

func TestGenerateOccurrences_RecurringWithNilRule(t *testing.T) {
	// isRecurring=true but recurrenceRule is nil: treated as non-recurring
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(100), false, "NoRule", true, true, nil, start, end)

	require.Len(t, occs, 1)
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
}

func TestGenerateOccurrences_YearlyWithInterval(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2036, 12, 31, 0, 0, 0, 0, time.UTC)
	planned := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	rule := makeRuleJSON(map[string]interface{}{
		"frequency": "yearly",
		"interval":  2,
	})

	occs := generateOccurrences(1, 1, planned, decimal.NewFromInt(1000), false, "Biannual", true, true, rule, start, end)

	// Every 2 years from Jun 1, 2026: 2026, 2028, 2030, 2032, 2034, 2036 = 6 occurrences
	require.Len(t, occs, 6)
	assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), occs[0].OccurrenceDate)
	assert.Equal(t, time.Date(2028, 6, 1, 0, 0, 0, 0, time.UTC), occs[1].OccurrenceDate)
	assert.Equal(t, time.Date(2036, 6, 1, 0, 0, 0, 0, time.UTC), occs[5].OccurrenceDate)
}

func TestNormalizeDate(t *testing.T) {
	input := time.Date(2026, 3, 15, 14, 30, 45, 123456789, time.UTC)
	result := normalizeDate(input)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), result)
}

func TestAdjustDayOfMonth(t *testing.T) {
	// Normal case: day exists in month
	result := adjustDayOfMonth(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 15)
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), result)

	// February non-leap: clamp 31 -> 28
	result = adjustDayOfMonth(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), 31)
	assert.Equal(t, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), result)

	// February leap year (2028): clamp 31 -> 29
	result = adjustDayOfMonth(time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC), 31)
	assert.Equal(t, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), result)

	// April: clamp 31 -> 30
	result = adjustDayOfMonth(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 31)
	assert.Equal(t, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), result)
}

// TestGenerateOccurrencesForTxs tests the exported function that processes
// a slice of PlannedTransaction models and produces sorted occurrences.
func TestGenerateOccurrencesForTxs_MultipleTxs(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	dailyRule := makeRuleJSON(map[string]interface{}{
		"frequency": "daily",
		"interval":  7,
	})

	txs := []*models.PlannedTransaction{
		{
			ID:          1,
			CurrencyID:  1,
			PlannedDate: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.NewFromInt(100),
			IsIncome:    false,
			Label:       "One-time",
			IsActive:    true,
			IsRecurring: false,
		},
		{
			ID:             2,
			CurrencyID:     1,
			PlannedDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:         decimal.NewFromInt(50),
			IsIncome:       true,
			Label:          "Weekly",
			IsActive:       true,
			IsRecurring:    true,
			RecurrenceRule: dailyRule,
		},
	}

	occs := GenerateOccurrencesForTxs(txs, start, end)

	// Weekly tx: Jan 1, 8, 15, 22, 29 = 5 occurrences
	// One-time tx: Jan 10 = 1 occurrence
	// Total = 6, sorted by date
	require.Len(t, occs, 6)

	// Verify sorted order
	for i := 1; i < len(occs); i++ {
		assert.False(t, occs[i].OccurrenceDate.Before(occs[i-1].OccurrenceDate),
			"occurrences should be sorted by date")
	}

	// Verify the one-time tx appears at the right position
	assert.Equal(t, 2, occs[0].PlannedTransactionID) // Jan 1 weekly
	assert.Equal(t, 2, occs[1].PlannedTransactionID) // Jan 8 weekly
	assert.Equal(t, 1, occs[2].PlannedTransactionID) // Jan 10 one-time
}

func TestGenerateOccurrencesForTxs_Empty(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	occs := GenerateOccurrencesForTxs(nil, start, end)
	assert.Empty(t, occs)
}
