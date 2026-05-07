package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== FlexDate Tests ====================

func TestFlexDateUnmarshalRFC3339(t *testing.T) {
	var fd FlexDate
	err := json.Unmarshal([]byte(`"2024-01-15T10:30:00Z"`), &fd)
	assert.NoError(t, err)
	assert.Equal(t, 2024, fd.Time().Year())
	assert.Equal(t, time.January, fd.Time().Month())
	assert.Equal(t, 15, fd.Time().Day())
}

func TestFlexDateUnmarshalDatetime(t *testing.T) {
	var fd FlexDate
	err := json.Unmarshal([]byte(`"2024-01-15T10:30:00"`), &fd)
	assert.NoError(t, err)
	assert.Equal(t, 2024, fd.Time().Year())
	assert.Equal(t, time.January, fd.Time().Month())
	assert.Equal(t, 15, fd.Time().Day())
}

func TestFlexDateUnmarshalBareDate(t *testing.T) {
	var fd FlexDate
	err := json.Unmarshal([]byte(`"2026-03-23"`), &fd)
	assert.NoError(t, err)
	assert.Equal(t, 2026, fd.Time().Year())
	assert.Equal(t, time.March, fd.Time().Month())
	assert.Equal(t, 23, fd.Time().Day())
}

func TestFlexDateUnmarshalInvalid(t *testing.T) {
	var fd FlexDate
	err := json.Unmarshal([]byte(`"not-a-date"`), &fd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse date")
}

func TestFlexDateUnmarshalBadJSON(t *testing.T) {
	var fd FlexDate
	err := json.Unmarshal([]byte(`123`), &fd)
	assert.Error(t, err)
}

func TestFlexDateMarshalJSON(t *testing.T) {
	fd := FlexDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	data, err := json.Marshal(fd)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "2024-06-15")
}

// ==================== DateOnly Tests ====================

func TestDateOnlyUnmarshalBareDate(t *testing.T) {
	var d DateOnly
	err := json.Unmarshal([]byte(`"2024-06-15"`), &d)
	assert.NoError(t, err)
	assert.Equal(t, 2024, d.Time.Year())
	assert.Equal(t, time.June, d.Time.Month())
	assert.Equal(t, 15, d.Time.Day())
}

func TestDateOnlyUnmarshalRFC3339(t *testing.T) {
	var d DateOnly
	err := json.Unmarshal([]byte(`"2024-06-15T10:30:00Z"`), &d)
	assert.NoError(t, err)
	assert.Equal(t, 2024, d.Time.Year())
	assert.Equal(t, time.June, d.Time.Month())
	assert.Equal(t, 15, d.Time.Day())
}

func TestDateOnlyUnmarshalDatetimeWithoutTimezone(t *testing.T) {
	var d DateOnly
	err := json.Unmarshal([]byte(`"2024-06-15T10:30:00"`), &d)
	assert.NoError(t, err)
	assert.Equal(t, 2024, d.Time.Year())
	assert.Equal(t, time.June, d.Time.Month())
	assert.Equal(t, 15, d.Time.Day())
}

func TestDateOnlyUnmarshalEmptyString(t *testing.T) {
	var d DateOnly
	err := json.Unmarshal([]byte(`""`), &d)
	assert.NoError(t, err)
	assert.True(t, d.Time.IsZero())
}

func TestDateOnlyUnmarshalInvalid(t *testing.T) {
	var d DateOnly
	err := json.Unmarshal([]byte(`"not-a-date"`), &d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse date")
}
