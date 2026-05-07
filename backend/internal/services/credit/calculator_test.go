package credit

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreditCalculator(t *testing.T) {
	calc := NewCreditCalculator()
	require.NotNil(t, calc)
}

// --- CalculateDailyRate ---

func TestCalculateDailyRate_Normal(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("30")
	rate := calc.CalculateDailyRate(price, 30)
	assert.True(t, rate.Equal(decimal.RequireFromString("1")),
		"expected 1, got %s", rate.String())
}

func TestCalculateDailyRate_MonthlyPlan(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("5")
	rate := calc.CalculateDailyRate(price, 30)
	// 5 / 30 = 0.1666666...
	expected := decimal.RequireFromString("5").Div(decimal.NewFromInt(30))
	assert.True(t, rate.Equal(expected),
		"expected %s, got %s", expected.String(), rate.String())
}

func TestCalculateDailyRate_YearlyPlan(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("50")
	rate := calc.CalculateDailyRate(price, 365)
	expected := decimal.RequireFromString("50").Div(decimal.NewFromInt(365))
	assert.True(t, rate.Equal(expected),
		"expected %s, got %s", expected.String(), rate.String())
}

func TestCalculateDailyRate_ZeroPeriod(t *testing.T) {
	calc := NewCreditCalculator()
	rate := calc.CalculateDailyRate(decimal.RequireFromString("10"), 0)
	assert.True(t, rate.Equal(decimal.Zero),
		"expected 0, got %s", rate.String())
}

func TestCalculateDailyRate_NegativePeriod(t *testing.T) {
	calc := NewCreditCalculator()
	rate := calc.CalculateDailyRate(decimal.RequireFromString("10"), -5)
	assert.True(t, rate.Equal(decimal.Zero),
		"expected 0, got %s", rate.String())
}

func TestCalculateDailyRate_ZeroPrice(t *testing.T) {
	calc := NewCreditCalculator()
	rate := calc.CalculateDailyRate(decimal.Zero, 30)
	assert.True(t, rate.Equal(decimal.Zero),
		"expected 0, got %s", rate.String())
}

// --- GetDaysRemaining ---

func TestGetDaysRemaining_FutureDateFullDays(t *testing.T) {
	calc := NewCreditCalculator()
	// Exactly 10 days from now.
	expiresAt := time.Now().Add(10 * 24 * time.Hour)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 10, days)
}

func TestGetDaysRemaining_PartialDayRoundsUp(t *testing.T) {
	calc := NewCreditCalculator()
	// 5 days and 1 hour from now -- should round up to 6.
	expiresAt := time.Now().Add(5*24*time.Hour + 1*time.Hour)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 6, days)
}

func TestGetDaysRemaining_PartialDaySmallFraction(t *testing.T) {
	calc := NewCreditCalculator()
	// 3 days and 1 minute from now -- should round up to 4.
	expiresAt := time.Now().Add(3*24*time.Hour + 1*time.Minute)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 4, days)
}

func TestGetDaysRemaining_LessThanOneDay(t *testing.T) {
	calc := NewCreditCalculator()
	// 6 hours from now -- should be 1 day (partial day rounds up).
	expiresAt := time.Now().Add(6 * time.Hour)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 1, days)
}

func TestGetDaysRemaining_ExpiredInPast(t *testing.T) {
	calc := NewCreditCalculator()
	expiresAt := time.Now().Add(-24 * time.Hour)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 0, days)
}

func TestGetDaysRemaining_JustExpired(t *testing.T) {
	calc := NewCreditCalculator()
	expiresAt := time.Now().Add(-1 * time.Second)
	days := calc.GetDaysRemaining(expiresAt)
	assert.Equal(t, 0, days)
}

// --- CalculateRemainingCredit ---

func TestCalculateRemainingCredit_MidBillingPeriod(t *testing.T) {
	calc := NewCreditCalculator()
	// $5.00 plan, 30-day billing period, 15 days remaining.
	price := decimal.RequireFromString("5")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 30, expiresAt)

	// dailyRate = 5 / 30 = 0.16666...
	// credit = dailyRate * 15
	dailyRate := decimal.RequireFromString("5").Div(decimal.NewFromInt(30))
	expected := dailyRate.Mul(decimal.NewFromInt(15))
	assert.True(t, credit.Equal(expected),
		"expected %s, got %s", expected.String(), credit.String())
}

func TestCalculateRemainingCredit_FullPeriodRemaining(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("10")
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 30, expiresAt)

	// dailyRate = 10 / 30 = 0.33333..., credit = dailyRate * 30
	dailyRate := decimal.RequireFromString("10").Div(decimal.NewFromInt(30))
	expected := dailyRate.Mul(decimal.NewFromInt(30))
	assert.True(t, credit.Equal(expected),
		"expected %s, got %s", expected.String(), credit.String())
}

func TestCalculateRemainingCredit_ExactDivision(t *testing.T) {
	calc := NewCreditCalculator()
	// $30.00 plan, 30-day period, 1 day remaining = $1.00 credit exactly.
	price := decimal.RequireFromString("30")
	expiresAt := time.Now().Add(24 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 30, expiresAt)

	expected := decimal.RequireFromString("1")
	assert.True(t, credit.Equal(expected),
		"expected %s, got %s", expected.String(), credit.String())
}

func TestCalculateRemainingCredit_Expired(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("5")
	expiresAt := time.Now().Add(-24 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 30, expiresAt)

	assert.True(t, credit.Equal(decimal.Zero),
		"expired subscription should have 0 credit, got %s", credit.String())
}

func TestCalculateRemainingCredit_ZeroTime(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("5")
	credit := calc.CalculateRemainingCredit(price, 30, time.Time{})

	assert.True(t, credit.Equal(decimal.Zero),
		"zero time (no expiry) should return 0 credit, got %s", credit.String())
}

func TestCalculateRemainingCredit_ZeroPricePlan(t *testing.T) {
	calc := NewCreditCalculator()
	expiresAt := time.Now().Add(15 * 24 * time.Hour)
	credit := calc.CalculateRemainingCredit(decimal.Zero, 30, expiresAt)

	assert.True(t, credit.Equal(decimal.Zero),
		"zero-price plan should have 0 credit, got %s", credit.String())
}

func TestCalculateRemainingCredit_ZeroBillingPeriod(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("5")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 0, expiresAt)

	assert.True(t, credit.Equal(decimal.Zero),
		"zero billing period should return 0 credit, got %s", credit.String())
}

func TestCalculateRemainingCredit_SameDayExpiry(t *testing.T) {
	calc := NewCreditCalculator()
	price := decimal.RequireFromString("5")
	// Expires in 1 hour -- partial day rounds up to 1 day.
	expiresAt := time.Now().Add(1 * time.Hour)
	credit := calc.CalculateRemainingCredit(price, 30, expiresAt)

	// dailyRate = 5 / 30, 1 day remaining
	dailyRate := decimal.RequireFromString("5").Div(decimal.NewFromInt(30))
	expected := dailyRate.Mul(decimal.NewFromInt(1))
	assert.True(t, credit.Equal(expected),
		"expected %s, got %s", expected.String(), credit.String())
}

// --- CalculateProratedPrice ---

func TestCalculateProratedPrice_NormalCase(t *testing.T) {
	calc := NewCreditCalculator()
	newPrice := decimal.RequireFromString("10")
	credit := decimal.RequireFromString("2.50")
	result := calc.CalculateProratedPrice(newPrice, credit)

	expected := decimal.RequireFromString("7.50")
	assert.True(t, result.Equal(expected),
		"expected %s, got %s", expected.String(), result.String())
}

func TestCalculateProratedPrice_CreditExceedsPrice(t *testing.T) {
	calc := NewCreditCalculator()
	newPrice := decimal.RequireFromString("5")
	credit := decimal.RequireFromString("10")
	result := calc.CalculateProratedPrice(newPrice, credit)

	assert.True(t, result.Equal(decimal.Zero),
		"prorated price should clamp to 0 when credit > price, got %s", result.String())
}

func TestCalculateProratedPrice_CreditEqualsPrice(t *testing.T) {
	calc := NewCreditCalculator()
	newPrice := decimal.RequireFromString("5")
	credit := decimal.RequireFromString("5")
	result := calc.CalculateProratedPrice(newPrice, credit)

	assert.True(t, result.Equal(decimal.Zero),
		"prorated price should be 0 when credit == price, got %s", result.String())
}

func TestCalculateProratedPrice_ZeroCredit(t *testing.T) {
	calc := NewCreditCalculator()
	newPrice := decimal.RequireFromString("10")
	result := calc.CalculateProratedPrice(newPrice, decimal.Zero)

	assert.True(t, result.Equal(newPrice),
		"expected %s, got %s", newPrice.String(), result.String())
}

func TestCalculateProratedPrice_ZeroPrice(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("5")
	result := calc.CalculateProratedPrice(decimal.Zero, credit)

	assert.True(t, result.Equal(decimal.Zero),
		"prorated price should clamp to 0, got %s", result.String())
}

// --- FormatCreditForDisplay ---

func TestFormatCreditForDisplay_USD(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("5.25")
	result := calc.FormatCreditForDisplay(credit, "USD")
	assert.Equal(t, "$5.25", result)
}

func TestFormatCreditForDisplay_EUR(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("3.50")
	result := calc.FormatCreditForDisplay(credit, "EUR")
	assert.Equal(t, "\u20ac3.50", result)
}

func TestFormatCreditForDisplay_GBP(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("3.50")
	result := calc.FormatCreditForDisplay(credit, "GBP")
	assert.Equal(t, "\u00a33.50", result)
}

func TestFormatCreditForDisplay_OtherCurrency(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("12.50")
	result := calc.FormatCreditForDisplay(credit, "UAH")
	assert.Equal(t, "UAH12.50", result)
}

func TestFormatCreditForDisplay_ZeroAmount(t *testing.T) {
	calc := NewCreditCalculator()
	result := calc.FormatCreditForDisplay(decimal.Zero, "USD")
	assert.Equal(t, "$0.00", result)
}

func TestFormatCreditForDisplay_LargeAmount(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("1234.56")
	result := calc.FormatCreditForDisplay(credit, "USD")
	assert.Equal(t, "$1234.56", result)
}

func TestFormatCreditForDisplay_HighPrecisionRounded(t *testing.T) {
	calc := NewCreditCalculator()
	// 1/3 = 0.333333... should display as 0.33
	credit := decimal.NewFromInt(1).Div(decimal.NewFromInt(3))
	result := calc.FormatCreditForDisplay(credit, "USD")
	assert.Equal(t, "$0.33", result)
}

func TestFormatCreditForDisplay_JPY(t *testing.T) {
	calc := NewCreditCalculator()
	credit := decimal.RequireFromString("500")
	result := calc.FormatCreditForDisplay(credit, "JPY")
	assert.Equal(t, "JPY500.00", result)
}

// --- Precision / Decimal Tests ---

func TestDecimalPrecision_NoDrift(t *testing.T) {
	calc := NewCreditCalculator()

	// This is a classic floating-point trap: 0.1 + 0.2 != 0.3 in float64.
	// With shopspring/decimal from string, there is no drift.
	price := decimal.RequireFromString("0.30")
	rate := calc.CalculateDailyRate(price, 3)
	// 0.30 / 3 = 0.10 exactly
	expected := decimal.RequireFromString("0.10")
	assert.True(t, rate.Equal(expected),
		"expected %s, got %s (precision drift)", expected.String(), rate.String())
}

func TestDecimalPrecision_RepeatingDecimal(t *testing.T) {
	calc := NewCreditCalculator()
	// $10 / 3 days = 3.333333... per day -- inherently repeating.
	// Verify that the result rounded to 2 decimal places is consistent.
	price := decimal.RequireFromString("10")
	rate := calc.CalculateDailyRate(price, 3)
	total := rate.Mul(decimal.NewFromInt(3))
	// With finite-precision division, total may not equal price exactly,
	// but it must be equal when rounded to 2 decimal places (monetary precision).
	assert.Equal(t, price.StringFixed(2), total.StringFixed(2),
		"expected %s, got %s at 2 decimal places", price.StringFixed(2), total.StringFixed(2))
}

func TestDecimalPrecision_ExactDivision(t *testing.T) {
	calc := NewCreditCalculator()
	// 12 / 4 = 3 exactly, no repeating decimals.
	price := decimal.RequireFromString("12")
	rate := calc.CalculateDailyRate(price, 4)
	total := rate.Mul(decimal.NewFromInt(4))
	assert.True(t, total.Equal(price),
		"expected %s, got %s", price.String(), total.String())
}

func TestDecimalPrecision_LargePlanSmallDays(t *testing.T) {
	calc := NewCreditCalculator()
	// $999.99 / 365 days -- verify monetary precision (2 decimal places).
	price := decimal.RequireFromString("999.99")
	rate := calc.CalculateDailyRate(price, 365)
	reconstructed := rate.Mul(decimal.NewFromInt(365))
	assert.Equal(t, price.StringFixed(2), reconstructed.StringFixed(2),
		"expected %s, got %s at 2 decimal places", price.StringFixed(2), reconstructed.StringFixed(2))
}

func TestDecimalPrecision_AvoidFloat64Binary(t *testing.T) {
	calc := NewCreditCalculator()
	// 0.1 + 0.2 should equal 0.3 in decimal (fails in float64).
	a := decimal.RequireFromString("0.1")
	b := decimal.RequireFromString("0.2")
	sum := a.Add(b)
	expected := decimal.RequireFromString("0.3")
	assert.True(t, sum.Equal(expected),
		"0.1 + 0.2 should equal 0.3, got %s", sum.String())

	// Verify this in context of daily rate calculation.
	price := decimal.RequireFromString("0.3")
	rate := calc.CalculateDailyRate(price, 1)
	assert.True(t, rate.Equal(expected),
		"daily rate of $0.3 for 1 day should be $0.3, got %s", rate.String())
}

// --- Integration-style: full credit calculation flow ---

func TestFullCreditFlow_UpgradeScenario(t *testing.T) {
	calc := NewCreditCalculator()

	// User is on $5.00/month (30 days), 15 days remaining.
	oldPrice := decimal.RequireFromString("5")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	credit := calc.CalculateRemainingCredit(oldPrice, 30, expiresAt)

	// Credit = (5/30) * 15 -- compute with same decimal operations.
	dailyRate := decimal.RequireFromString("5").Div(decimal.NewFromInt(30))
	expectedCredit := dailyRate.Mul(decimal.NewFromInt(15))
	assert.True(t, credit.Equal(expectedCredit),
		"credit: expected %s, got %s", expectedCredit.String(), credit.String())

	// Verify it rounds to $2.50 for display purposes.
	assert.Equal(t, "2.50", credit.StringFixed(2))

	// Upgrading to $10.00/month. Prorated price = 10.00 - credit.
	newPrice := decimal.RequireFromString("10")
	prorated := calc.CalculateProratedPrice(newPrice, credit)
	expectedProrated := newPrice.Sub(expectedCredit)
	assert.True(t, prorated.Equal(expectedProrated),
		"prorated: expected %s, got %s", expectedProrated.String(), prorated.String())

	// Verify prorated rounds to $7.50 for display.
	assert.Equal(t, "7.50", prorated.StringFixed(2))

	// Display the credit.
	display := calc.FormatCreditForDisplay(credit, "USD")
	assert.Equal(t, "$2.50", display)
}

func TestFullCreditFlow_ExpiredSubscription(t *testing.T) {
	calc := NewCreditCalculator()

	// Expired subscription -- should get 0 credit and pay full price.
	oldPrice := decimal.RequireFromString("5")
	expiresAt := time.Now().Add(-48 * time.Hour)

	credit := calc.CalculateRemainingCredit(oldPrice, 30, expiresAt)
	assert.True(t, credit.Equal(decimal.Zero))

	newPrice := decimal.RequireFromString("10")
	prorated := calc.CalculateProratedPrice(newPrice, credit)
	assert.True(t, prorated.Equal(newPrice),
		"expired subscription should pay full price")
}

func TestFullCreditFlow_CreditExceedsNewPrice(t *testing.T) {
	calc := NewCreditCalculator()

	// User is on $50/year (365 days), 300 days remaining.
	oldPrice := decimal.RequireFromString("50")
	expiresAt := time.Now().Add(300 * 24 * time.Hour)

	credit := calc.CalculateRemainingCredit(oldPrice, 365, expiresAt)

	// Credit is roughly (50/365)*300 = ~41.10 -- more than $5.
	newPrice := decimal.RequireFromString("5")
	prorated := calc.CalculateProratedPrice(newPrice, credit)
	assert.True(t, prorated.Equal(decimal.Zero),
		"prorated price should clamp to 0 when credit > new price, got %s", prorated.String())
}
