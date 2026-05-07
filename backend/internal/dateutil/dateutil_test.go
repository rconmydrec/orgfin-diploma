package dateutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== ParseDate Tests ====================

func TestParseDateBareDate(t *testing.T) {
	result, err := ParseDate("2024-06-15")
	assert.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDateDatetimeWithoutTimezone(t *testing.T) {
	result, err := ParseDate("2024-06-15T10:30:00")
	assert.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseDateDatetimeWithoutSeconds(t *testing.T) {
	result, err := ParseDate("2024-06-15T10:30")
	assert.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 10, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseDateRFC3339(t *testing.T) {
	result, err := ParseDate("2024-06-15T10:30:00Z")
	assert.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDateRFC3339WithOffset(t *testing.T) {
	result, err := ParseDate("2024-06-15T10:30:00+02:00")
	assert.NoError(t, err)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDateInvalid(t *testing.T) {
	_, err := ParseDate("not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse date")
}

func TestParseDateEmptyString(t *testing.T) {
	_, err := ParseDate("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse date")
}

// ==================== MakeEndDateInclusive Tests ====================

func TestMakeEndDateInclusiveNormalDate(t *testing.T) {
	result := MakeEndDateInclusive("2024-06-15")
	assert.Equal(t, "2024-06-16", result)
}

func TestMakeEndDateInclusiveYearBoundary(t *testing.T) {
	result := MakeEndDateInclusive("2024-12-31")
	assert.Equal(t, "2025-01-01", result)
}

func TestMakeEndDateInclusiveInvalidFallback(t *testing.T) {
	result := MakeEndDateInclusive("invalid")
	assert.Equal(t, "invalid", result)
}
