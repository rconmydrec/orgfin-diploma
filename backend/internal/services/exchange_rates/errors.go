package exchange_rates

import (
	"fmt"

	"github.com/go-budget/backend/internal/serviceerrors"
)

// Sentinel errors for the exchange rates service.
var (
	ErrAPIKeyNotConfigured = serviceerrors.New(serviceerrors.InvalidInput, "CurrencyBeacon API key is not configured")
	ErrEndBeforeStart      = serviceerrors.New(serviceerrors.InvalidInput, "end date must not be before start date")
	ErrDateRangeExceeded   = serviceerrors.New(serviceerrors.InvalidInput, "date range exceeds maximum allowed days")
	ErrInvalidStartDate    = serviceerrors.New(serviceerrors.InvalidInput, "invalid start_date format")
	ErrInvalidEndDate      = serviceerrors.New(serviceerrors.InvalidInput, "invalid end_date format")
)

// DateError wraps an error with the date that caused it and the operation.
type DateError struct {
	Date      string
	Operation string // "fetch" or "save"
	Err       error
}

func (e *DateError) Error() string {
	return fmt.Sprintf("%s rates for %s: %s", e.Operation, e.Date, e.Err)
}

func (e *DateError) Unwrap() error { return e.Err }
