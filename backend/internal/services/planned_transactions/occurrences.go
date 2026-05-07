package planned_transactions

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// GenerateOccurrencesForTxs expands a list of planned transactions into
// individual occurrences within the [startDate, endDate] range, sorted by date.
// This is the main entry point for other packages that need occurrence generation.
func GenerateOccurrencesForTxs(txs []*models.PlannedTransaction, startDate, endDate time.Time) []Occurrence {
	var allOccurrences []Occurrence
	for _, tx := range txs {
		occs := generateOccurrences(
			tx.ID, tx.CurrencyID, tx.PlannedDate, tx.Amount,
			tx.IsIncome, tx.Label, tx.IsActive, tx.IsRecurring,
			tx.RecurrenceRule, startDate, endDate,
		)
		allOccurrences = append(allOccurrences, occs...)
	}

	sort.Slice(allOccurrences, func(i, j int) bool {
		return allOccurrences[i].OccurrenceDate.Before(allOccurrences[j].OccurrenceDate)
	})

	return allOccurrences
}

// generateOccurrences expands a planned transaction into individual occurrences
// within the [startDate, endDate] range. For non-recurring transactions, it
// returns a single occurrence if the planned date falls within the range.
// For recurring transactions, it generates occurrences based on the recurrence rule.
func generateOccurrences(
	id, currencyID int,
	plannedDate time.Time,
	amount decimal.Decimal,
	isIncome bool,
	label string,
	isActive bool,
	isRecurring bool,
	recurrenceRuleJSON *string,
	startDate, endDate time.Time,
) []Occurrence {
	makeOcc := func(date time.Time) Occurrence {
		return Occurrence{
			PlannedTransactionID: id,
			CurrencyID:           currencyID,
			OccurrenceDate:       date,
			Amount:               amount,
			IsIncome:             isIncome,
			Label:                label,
			IsActive:             isActive,
			IsRecurring:          isRecurring,
		}
	}

	if !isRecurring || recurrenceRuleJSON == nil {
		// Non-recurring: single occurrence if within range
		pDate := normalizeDate(plannedDate)
		if !pDate.Before(startDate) && !pDate.After(endDate) {
			return []Occurrence{makeOcc(pDate)}
		}
		return nil
	}

	var rule models.RecurrenceRule
	if err := json.Unmarshal([]byte(*recurrenceRuleJSON), &rule); err != nil {
		return nil
	}

	interval := rule.Interval
	if interval < 1 {
		interval = 1
	}

	// Parse the recurrence rule's end date (handles formats with and without timezone)
	var ruleEndDate *time.Time
	if rule.EndDate != nil && *rule.EndDate != "" {
		s := *rule.EndDate
		// Try common formats
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				normalized := normalizeDate(t)
				ruleEndDate = &normalized
				break
			}
		}
	}

	// Use the earlier of endDate and ruleEndDate as the effective end
	effectiveEndDate := endDate
	if ruleEndDate != nil && ruleEndDate.Before(endDate) {
		effectiveEndDate = *ruleEndDate
	}

	var occurrences []Occurrence
	current := normalizeDate(plannedDate)
	totalCount := 0
	const maxIterations = 1000

	for i := 0; i < maxIterations; i++ {
		occDate := current

		// For monthly with dayOfMonth override, adjust the day
		if rule.Frequency == "monthly" && rule.DayOfMonth != nil {
			occDate = adjustDayOfMonth(occDate, *rule.DayOfMonth)
		}

		// Stop if occurrence exceeds effective end date
		if occDate.After(effectiveEndDate) {
			break
		}

		// Stop if count limit reached
		if rule.Count != nil && totalCount >= *rule.Count {
			break
		}

		totalCount++

		// Only include occurrences within [startDate, endDate]
		if !occDate.Before(startDate) {
			occurrences = append(occurrences, makeOcc(occDate))
		}

		// Advance to next occurrence
		switch rule.Frequency {
		case "daily":
			current = current.AddDate(0, 0, interval)
		case "weekly":
			current = current.AddDate(0, 0, interval*7)
		case "monthly":
			current = current.AddDate(0, interval, 0)
		case "yearly":
			current = current.AddDate(interval, 0, 0)
		default:
			// Unknown frequency: stop generating
			return occurrences
		}
	}

	return occurrences
}

// normalizeDate strips the time component from a time.Time, keeping only the date.
func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// adjustDayOfMonth returns a date with the day set to dayOfMonth,
// clamping to the last day of the month if dayOfMonth exceeds the number
// of days in the month.
func adjustDayOfMonth(t time.Time, dayOfMonth int) time.Time {
	year, month := t.Year(), t.Month()
	// Last day of the month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, t.Location()).Day()
	day := dayOfMonth
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
