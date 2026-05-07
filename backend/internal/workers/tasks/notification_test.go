package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildExchangeRateNotificationSubject(t *testing.T) {
	subject := buildExchangeRateNotificationSubject("production")
	assert.Equal(t, "[production] Currency rates updated", subject)
}

func TestBuildExchangeRateNotificationBody(t *testing.T) {
	body := buildExchangeRateNotificationBody("staging", "2026-02-23T14:00:00Z", 157, "USD")

	assert.True(t, len(body) > 0)
	assert.Contains(t, body, "staging")
	assert.Contains(t, body, "2026-02-23T14:00:00Z")
	assert.Contains(t, body, "157")
	assert.Contains(t, body, "USD")
	assert.Contains(t, body, "Currency Rate Update Notification")
	assert.Contains(t, body, "Dear Team")
	assert.Contains(t, body, "Orgfin")
}

func TestBuildExchangeRateNotificationBody_ZeroRates(t *testing.T) {
	body := buildExchangeRateNotificationBody("dev", "2026-02-23T14:00:00Z", 0, "EUR")

	assert.Contains(t, body, "0")
	assert.Contains(t, body, "EUR")
}
