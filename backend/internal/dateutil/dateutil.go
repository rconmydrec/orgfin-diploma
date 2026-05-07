// Package dateutil provides shared date parsing and manipulation utilities
// used by both the handler and service layers.
package dateutil

import (
	"fmt"
	"time"
)

// dateFormats lists the date formats accepted by ParseDate, ordered from most
// common (bare date) to most flexible (RFC3339).
var dateFormats = []string{
	"2006-01-02",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	time.RFC3339,
}

// ParseDate parses a date string trying multiple formats in order:
// bare date, datetime without timezone, datetime without seconds, and RFC3339.
// Returns an error if none of the formats match.
func ParseDate(s string) (time.Time, error) {
	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

// MakeEndDateInclusive adds 1 day to a date string so that the end date itself
// is included in range queries. On parse failure, returns the original string
// unchanged.
func MakeEndDateInclusive(dateStr string) string {
	t, err := ParseDate(dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}
