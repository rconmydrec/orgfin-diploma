package types

// ExchangeRateSnapshot represents all exchange rates for a specific date.
type ExchangeRateSnapshot struct {
	ID               int
	Rates            map[string]float64
	ActualDate       string
	BaseCurrencyCode string
}
