// Contract tests for the credit service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box).
package credit_test

import (
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/credit"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupService creates a credit calculator for contract testing.
func setupService(t *testing.T) credit.ServiceInterface {
	t.Helper()
	return credit.NewCreditCalculator()
}

// --- CalculateRemainingCredit ---

func TestCalculateRemainingCredit_ReturnsPositiveCredit(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	result := svc.CalculateRemainingCredit(price, 30, expiresAt)

	assert.True(t, result.GreaterThan(decimal.Zero),
		"credit should be positive for active subscription, got %s", result.String())
}

func TestCalculateRemainingCredit_ZeroForExpiredSubscription(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")
	expiresAt := time.Now().Add(-24 * time.Hour)

	result := svc.CalculateRemainingCredit(price, 30, expiresAt)

	assert.True(t, result.Equal(decimal.Zero),
		"credit should be zero for expired subscription, got %s", result.String())
}

func TestCalculateRemainingCredit_ZeroForZeroExpiry(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")

	result := svc.CalculateRemainingCredit(price, 30, time.Time{})

	assert.True(t, result.Equal(decimal.Zero),
		"credit should be zero for zero expiry time, got %s", result.String())
}

func TestCalculateRemainingCredit_ZeroForZeroBillingPeriod(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	result := svc.CalculateRemainingCredit(price, 0, expiresAt)

	assert.True(t, result.Equal(decimal.Zero),
		"credit should be zero for zero billing period, got %s", result.String())
}

func TestCalculateRemainingCredit_NeverNegative(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	result := svc.CalculateRemainingCredit(price, -5, expiresAt)

	assert.True(t, result.GreaterThanOrEqual(decimal.Zero),
		"credit must never be negative (invariant), got %s", result.String())
}

func TestCalculateRemainingCredit_ZeroPricePlan(t *testing.T) {
	svc := setupService(t)
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	result := svc.CalculateRemainingCredit(decimal.Zero, 30, expiresAt)

	assert.True(t, result.Equal(decimal.Zero),
		"credit should be zero for zero-price plan, got %s", result.String())
}

// --- CalculateDailyRate ---

func TestCalculateDailyRate_NormalDivision(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("30.00")

	rate := svc.CalculateDailyRate(price, 30)

	expected := decimal.RequireFromString("1.00")
	assert.True(t, rate.Equal(expected),
		"expected daily rate 1.00, got %s", rate.String())
}

func TestCalculateDailyRate_ZeroPeriodReturnsZero(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")

	rate := svc.CalculateDailyRate(price, 0)

	assert.True(t, rate.Equal(decimal.Zero),
		"daily rate should be zero for zero period, got %s", rate.String())
}

func TestCalculateDailyRate_NegativePeriodReturnsZero(t *testing.T) {
	svc := setupService(t)
	price := decimal.RequireFromString("10.00")

	rate := svc.CalculateDailyRate(price, -5)

	assert.True(t, rate.Equal(decimal.Zero),
		"daily rate should be zero for negative period, got %s", rate.String())
}

func TestCalculateDailyRate_ZeroPriceReturnsZero(t *testing.T) {
	svc := setupService(t)

	rate := svc.CalculateDailyRate(decimal.Zero, 30)

	assert.True(t, rate.Equal(decimal.Zero),
		"daily rate should be zero for zero price, got %s", rate.String())
}

// --- GetDaysRemaining ---

func TestGetDaysRemaining_FutureDate(t *testing.T) {
	svc := setupService(t)
	expiresAt := time.Now().Add(10 * 24 * time.Hour)

	days := svc.GetDaysRemaining(expiresAt)

	assert.Equal(t, 10, days)
}

func TestGetDaysRemaining_PartialDayCeiling(t *testing.T) {
	svc := setupService(t)
	expiresAt := time.Now().Add(5*24*time.Hour + 1*time.Hour)

	days := svc.GetDaysRemaining(expiresAt)

	assert.Equal(t, 6, days, "partial day should round up (ceiling)")
}

func TestGetDaysRemaining_ExpiredReturnsZero(t *testing.T) {
	svc := setupService(t)
	expiresAt := time.Now().Add(-24 * time.Hour)

	days := svc.GetDaysRemaining(expiresAt)

	assert.Equal(t, 0, days, "expired subscription should return 0 days")
}

func TestGetDaysRemaining_LessThanOneDay(t *testing.T) {
	svc := setupService(t)
	expiresAt := time.Now().Add(6 * time.Hour)

	days := svc.GetDaysRemaining(expiresAt)

	assert.Equal(t, 1, days, "less than one day should round up to 1")
}

// --- CalculateProratedPrice ---

func TestCalculateProratedPrice_SubtractsCredit(t *testing.T) {
	svc := setupService(t)
	newPrice := decimal.RequireFromString("10.00")
	credit := decimal.RequireFromString("3.00")

	result := svc.CalculateProratedPrice(newPrice, credit)

	expected := decimal.RequireFromString("7.00")
	assert.True(t, result.Equal(expected),
		"expected 7.00, got %s", result.String())
}

func TestCalculateProratedPrice_ClampsToZero(t *testing.T) {
	svc := setupService(t)
	newPrice := decimal.RequireFromString("5.00")
	credit := decimal.RequireFromString("10.00")

	result := svc.CalculateProratedPrice(newPrice, credit)

	assert.True(t, result.Equal(decimal.Zero),
		"prorated price should clamp to 0 when credit exceeds price, got %s", result.String())
}

func TestCalculateProratedPrice_CreditEqualsPrice(t *testing.T) {
	svc := setupService(t)
	newPrice := decimal.RequireFromString("5.00")
	credit := decimal.RequireFromString("5.00")

	result := svc.CalculateProratedPrice(newPrice, credit)

	assert.True(t, result.Equal(decimal.Zero),
		"prorated price should be 0 when credit equals price, got %s", result.String())
}

func TestCalculateProratedPrice_ZeroCredit(t *testing.T) {
	svc := setupService(t)
	newPrice := decimal.RequireFromString("10.00")

	result := svc.CalculateProratedPrice(newPrice, decimal.Zero)

	assert.True(t, result.Equal(newPrice),
		"prorated price should equal new price when no credit, got %s", result.String())
}

// --- FormatCreditForDisplay ---

func TestFormatCreditForDisplay_USD(t *testing.T) {
	svc := setupService(t)
	credit := decimal.RequireFromString("5.25")

	result := svc.FormatCreditForDisplay(credit, "USD")

	assert.Equal(t, "$5.25", result)
}

func TestFormatCreditForDisplay_EUR(t *testing.T) {
	svc := setupService(t)
	credit := decimal.RequireFromString("3.50")

	result := svc.FormatCreditForDisplay(credit, "EUR")

	assert.Equal(t, "\u20ac3.50", result)
}

func TestFormatCreditForDisplay_GBP(t *testing.T) {
	svc := setupService(t)
	credit := decimal.RequireFromString("3.50")

	result := svc.FormatCreditForDisplay(credit, "GBP")

	assert.Equal(t, "\u00a33.50", result)
}

func TestFormatCreditForDisplay_UnknownCurrencyUsesCode(t *testing.T) {
	svc := setupService(t)
	credit := decimal.RequireFromString("12.50")

	result := svc.FormatCreditForDisplay(credit, "UAH")

	assert.Equal(t, "UAH12.50", result)
}

func TestFormatCreditForDisplay_ZeroAmount(t *testing.T) {
	svc := setupService(t)

	result := svc.FormatCreditForDisplay(decimal.Zero, "USD")

	assert.Equal(t, "$0.00", result)
}

func TestFormatCreditForDisplay_TwoDecimalPrecision(t *testing.T) {
	svc := setupService(t)
	// 1/3 = 0.333... should display as 0.33
	credit := decimal.NewFromInt(1).Div(decimal.NewFromInt(3))

	result := svc.FormatCreditForDisplay(credit, "USD")

	assert.Equal(t, "$0.33", result)
}

// --- Integration: full upgrade flow ---

func TestFullFlow_UpgradeScenario(t *testing.T) {
	svc := setupService(t)

	// User on $5/month (30-day billing), 15 days left
	oldPrice := decimal.RequireFromString("5.00")
	expiresAt := time.Now().Add(15 * 24 * time.Hour)

	credit := svc.CalculateRemainingCredit(oldPrice, 30, expiresAt)
	require.True(t, credit.GreaterThan(decimal.Zero))

	// Prorated price for upgrade to $10/month
	newPrice := decimal.RequireFromString("10.00")
	prorated := svc.CalculateProratedPrice(newPrice, credit)
	assert.True(t, prorated.GreaterThan(decimal.Zero))
	assert.True(t, prorated.LessThan(newPrice))

	// Display
	display := svc.FormatCreditForDisplay(credit, "USD")
	assert.Contains(t, display, "$")
}

func TestFullFlow_ExpiredSubscriptionPaysFullPrice(t *testing.T) {
	svc := setupService(t)

	oldPrice := decimal.RequireFromString("5.00")
	expiresAt := time.Now().Add(-48 * time.Hour)

	credit := svc.CalculateRemainingCredit(oldPrice, 30, expiresAt)
	assert.True(t, credit.Equal(decimal.Zero))

	newPrice := decimal.RequireFromString("10.00")
	prorated := svc.CalculateProratedPrice(newPrice, credit)
	assert.True(t, prorated.Equal(newPrice),
		"expired subscription should pay full price")
}
