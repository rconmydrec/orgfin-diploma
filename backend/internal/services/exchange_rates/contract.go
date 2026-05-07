// Package exchange_rates provides exchange rate fetching from the CurrencyBeacon
// API, persistence to the database, and querying stored rates.
//
// Invariants:
//   - FetchAndSaveRatesRange validates date format, range order, and max range length.
//   - Range processing stops on the first error.
//   - API key must be configured for fetch operations.
package exchange_rates

import (
	"net/http"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

// ServiceInterface defines the public API of the exchange rates service.
type ServiceInterface interface {
	GetRatesForDate(date string) (*types.ExchangeRateSnapshot, error)
	FetchAndSaveRates(date string) (*FetchResult, error)
	FetchAndSaveRatesRange(startDateStr, endDateStr string) (*FetchAndSaveRatesRangeResult, error)
	SetHTTPClient(client HTTPClient)
}

// ExchangeRateRepository defines exchange rate data access methods needed by the service.
type ExchangeRateRepository interface {
	GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error)
	SaveRate(rate *models.ExchangeRateHistory) error
}

// HTTPClient is an interface for making HTTP requests. It allows injecting a
// mock client in tests without hitting the real CurrencyBeacon API.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MaxDateRangeDays is the maximum number of days allowed for a range rate update.
const MaxDateRangeDays = 365

// FetchResult holds metadata about a single fetch-and-save operation.
// Used by consumers (e.g. worker tasks) that need details from the API response.
type FetchResult struct {
	BaseCurrency string
	RateCount    int
}

// FetchAndSaveRatesRangeResult holds the result of a range rate update.
type FetchAndSaveRatesRangeResult struct {
	DatesProcessed int
	StartDate      string
	EndDate        string
}
