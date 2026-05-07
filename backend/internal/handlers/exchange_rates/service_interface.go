package exchange_rates

import (
	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/types"
)

// ExchangeRatesService defines the interface for exchange rate service operations
// needed by this handler.
type ExchangeRatesService interface {
	GetRatesForDate(date string) (*types.ExchangeRateSnapshot, error)
	FetchAndSaveRatesRange(startDateStr, endDateStr string) (*exchangerates.FetchAndSaveRatesRangeResult, error)
}

// ExchangeRateListResponse is the HTTP response DTO for exchange rate data.
type ExchangeRateListResponse struct {
	ID               int                `json:"id"`
	Rates            map[string]float64 `json:"rates"`
	ActualDate       string             `json:"actualDate"`
	BaseCurrencyCode string             `json:"baseCurrencyCode"`
}

// toExchangeRateListResponse converts a domain snapshot to the HTTP response DTO.
func toExchangeRateListResponse(s *types.ExchangeRateSnapshot) *ExchangeRateListResponse {
	return &ExchangeRateListResponse{
		ID:               s.ID,
		Rates:            s.Rates,
		ActualDate:       s.ActualDate,
		BaseCurrencyCode: s.BaseCurrencyCode,
	}
}
