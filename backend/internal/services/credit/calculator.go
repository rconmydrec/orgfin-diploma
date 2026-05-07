package credit

import (
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// CreditCalculator provides pure-logic methods for prorated credit calculations.
// It has no external dependencies and all monetary values use shopspring/decimal
// to avoid floating-point precision issues.
type CreditCalculator struct{}

// NewCreditCalculator creates a new CreditCalculator instance.
func NewCreditCalculator() *CreditCalculator {
	return &CreditCalculator{}
}

// CalculateRemainingCredit computes the prorated remaining value of a subscription.
// Formula: dailyRate * daysRemaining.
// Returns zero if expiresAt is the zero time (no expiry) or if the subscription is already expired.
func (c *CreditCalculator) CalculateRemainingCredit(
	currentPlanPrice decimal.Decimal,
	billingPeriodDays int,
	expiresAt time.Time,
) decimal.Decimal {
	if expiresAt.IsZero() {
		return decimal.Zero
	}

	daysRemaining := c.GetDaysRemaining(expiresAt)
	if daysRemaining <= 0 {
		return decimal.Zero
	}

	dailyRate := c.CalculateDailyRate(currentPlanPrice, billingPeriodDays)
	credit := dailyRate.Mul(decimal.NewFromInt(int64(daysRemaining)))

	// Credit cannot be negative (clamp to zero).
	if credit.IsNegative() {
		return decimal.Zero
	}

	return credit
}

// GetDaysRemaining returns the number of days between now and the expiration time.
// If there are remaining seconds beyond whole days, an extra day is added (ceiling).
// Returns 0 if the subscription is already expired.
func (c *CreditCalculator) GetDaysRemaining(expiresAt time.Time) int {
	now := time.Now()
	remaining := expiresAt.Sub(now)

	if remaining <= 0 {
		return 0
	}

	days := remaining.Hours() / 24
	return int(math.Ceil(days))
}

// CalculateDailyRate computes the price per day for a given plan price and billing period.
// Returns zero if billingPeriodDays is zero or negative.
func (c *CreditCalculator) CalculateDailyRate(
	planPrice decimal.Decimal,
	billingPeriodDays int,
) decimal.Decimal {
	if billingPeriodDays <= 0 {
		return decimal.Zero
	}

	return planPrice.Div(decimal.NewFromInt(int64(billingPeriodDays)))
}

// CalculateProratedPrice computes the price a user owes after applying credit.
// Formula: newPlanPrice - credit, clamped to a minimum of zero.
func (c *CreditCalculator) CalculateProratedPrice(
	newPlanPrice decimal.Decimal,
	credit decimal.Decimal,
) decimal.Decimal {
	result := newPlanPrice.Sub(credit)
	if result.IsNegative() {
		return decimal.Zero
	}
	return result
}

// currencySymbols maps well-known currency codes to their display symbols.
var currencySymbols = map[string]string{
	"USD": "$",
	"EUR": "\u20ac",
	"GBP": "\u00a3",
}

// FormatCreditForDisplay formats a credit amount for user-facing display.
// Uses currency symbols for USD ($), EUR, and GBP; falls back to the
// currency code for all other currencies (e.g., "UAH12.50").
func (c *CreditCalculator) FormatCreditForDisplay(
	credit decimal.Decimal,
	currencyCode string,
) string {
	symbol, ok := currencySymbols[currencyCode]
	if !ok {
		symbol = currencyCode
	}
	return fmt.Sprintf("%s%s", symbol, credit.StringFixed(2))
}
