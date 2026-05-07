package currencies

import (
	currencyservice "github.com/go-budget/backend/internal/services/currency"
)

// CurrencyService defines the interface for currency service operations
// needed by this handler.
type CurrencyService interface {
	GetAll() ([]*currencyservice.Currency, error)
}
