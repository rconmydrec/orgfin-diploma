// Package currency provides currency listing and conversion services.
//
// Invariants:
//   - ConvertAmount returns the original amount if currencies are the same or rates are unavailable.
//   - GetExchangeRate returns 1 if currencies are the same or rates are unavailable.
//   - ConvertToBaseCurrency distinguishes between fatal errors (empty rates table)
//     and non-fatal errors (specific currency missing from rates).
package currency

import (
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

// ServiceInterface defines the public API of the currency service.
type ServiceInterface interface {
	GetAll() ([]*Currency, error)
}

// Currency represents a currency in the currency service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// CurrencyRepository defines currency data access methods needed by this service.
type CurrencyRepository interface {
	GetAll() ([]*models.Currency, error)
}

// ExchangeRateRepository defines exchange rate data access methods needed by this service.
type ExchangeRateRepository interface {
	GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error)
}
