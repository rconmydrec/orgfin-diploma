package common

import (
	"encoding/json"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
)

// FlexDate is a time.Time wrapper that handles multiple date formats during
// JSON unmarshalling. It delegates parsing to dateutil.ParseDate.
type FlexDate time.Time

// UnmarshalJSON parses a JSON string into a FlexDate using dateutil.ParseDate.
func (fd *FlexDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := dateutil.ParseDate(s)
	if err != nil {
		return err
	}
	*fd = FlexDate(t)
	return nil
}

// MarshalJSON marshals the FlexDate as an RFC3339 string (via time.Time default marshaling).
func (fd FlexDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(fd))
}

// Time converts the FlexDate to a time.Time value.
func (fd FlexDate) Time() time.Time {
	return time.Time(fd)
}

// DateOnly handles JSON date strings in multiple formats, with special handling
// for empty strings (returns zero time without error). Used for optional date
// fields in request DTOs.
type DateOnly struct {
	time.Time
}

// UnmarshalJSON parses a JSON string into a DateOnly. Empty strings are treated
// as "not provided" and set the time to zero value. Non-empty strings are
// parsed using dateutil.ParseDate.
func (d *DateOnly) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := dateutil.ParseDate(s)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}
