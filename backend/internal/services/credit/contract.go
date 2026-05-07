// Package credit provides pure-logic methods for prorated credit calculations
// with no external dependencies.
//
// Invariants:
//   - All monetary values use shopspring/decimal to avoid floating-point precision issues.
//   - Credit cannot be negative (clamped to zero).
//   - Daily rate is zero if billing period is zero or negative.
//   - Days remaining uses ceiling (partial day counts as full day).
package credit

import (
	"time"

	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the credit calculator.
type ServiceInterface interface {
	CalculateRemainingCredit(currentPlanPrice decimal.Decimal, billingPeriodDays int, expiresAt time.Time) decimal.Decimal
	CalculateDailyRate(planPrice decimal.Decimal, billingPeriodDays int) decimal.Decimal
	CalculateProratedPrice(newPlanPrice decimal.Decimal, credit decimal.Decimal) decimal.Decimal
	GetDaysRemaining(expiresAt time.Time) int
	FormatCreditForDisplay(credit decimal.Decimal, currencyCode string) string
}
