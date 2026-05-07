package currency

import (
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// Service implements the currency service using the local ExchangeRateRepository
// interface defined in contract.go.
type Service struct {
	exchangeRateRepo ExchangeRateRepository
	currencyRepo     CurrencyRepository
	logger           *slog.Logger
}

func New(exchangeRateRepo ExchangeRateRepository, logger *slog.Logger) *Service {
	return &Service{
		exchangeRateRepo: exchangeRateRepo,
		logger:           logger,
	}
}

// NewWithCurrencyRepo creates a new Service with both exchange rate and currency repositories.
func NewWithCurrencyRepo(exchangeRateRepo ExchangeRateRepository, currencyRepo CurrencyRepository, logger *slog.Logger) *Service {
	return &Service{
		exchangeRateRepo: exchangeRateRepo,
		currencyRepo:     currencyRepo,
		logger:           logger,
	}
}

// GetAll returns all currencies.
func (s *Service) GetAll() ([]*Currency, error) {
	currencies, err := s.currencyRepo.GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]*Currency, len(currencies))
	for i, c := range currencies {
		result[i] = toCurrency(c)
	}
	return result, nil
}

// toCurrency converts a models.Currency to the service domain type.
func toCurrency(m *models.Currency) *Currency {
	if m == nil {
		return nil
	}
	return &Currency{
		ID:   m.ID,
		Code: m.Code,
		Name: m.Name,
	}
}

// ConvertAmount converts an amount from one currency to another using exchange rates.
// The exchange_rates table stores all rates relative to USD base currency as JSONB.
// Conversion formula: amount / sourceRate * targetRate.
// If currencies are the same or rate not found, returns the original amount.
func (s *Service) ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal {
	if fromCurrency == toCurrency {
		return amount
	}

	dateStr := date.Format("2006-01-02")
	rates, err := s.exchangeRateRepo.GetAllRatesForDate(dateStr)
	if err != nil {
		s.logger.Debug("exchange rates not found, using original amount",
			"from", fromCurrency,
			"to", toCurrency,
			"date", dateStr,
			"error", err)
		return amount
	}

	sourceRate, sourceOK := rates.Rates[fromCurrency]
	targetRate, targetOK := rates.Rates[toCurrency]

	if !sourceOK || !targetOK || sourceRate == 0 {
		s.logger.Debug("currency not found in rates, using original amount",
			"from", fromCurrency,
			"to", toCurrency,
			"date", dateStr,
			"source_found", sourceOK,
			"target_found", targetOK)
		return amount
	}

	// converted = amount / sourceRate * targetRate
	sourceRateDec := decimal.NewFromFloat(sourceRate)
	targetRateDec := decimal.NewFromFloat(targetRate)
	return amount.Div(sourceRateDec).Mul(targetRateDec)
}

// GetExchangeRate gets the exchange rate between two currencies.
// Returns the conversion factor: to convert from fromCurrency to toCurrency,
// multiply by this factor.
func (s *Service) GetExchangeRate(fromCurrency, toCurrency string, date time.Time) decimal.Decimal {
	if fromCurrency == toCurrency {
		return decimal.NewFromInt(1)
	}

	dateStr := date.Format("2006-01-02")
	rates, err := s.exchangeRateRepo.GetAllRatesForDate(dateStr)
	if err != nil {
		return decimal.NewFromInt(1)
	}

	sourceRate, sourceOK := rates.Rates[fromCurrency]
	targetRate, targetOK := rates.Rates[toCurrency]

	if !sourceOK || !targetOK || sourceRate == 0 {
		return decimal.NewFromInt(1)
	}

	sourceRateDec := decimal.NewFromFloat(sourceRate)
	targetRateDec := decimal.NewFromFloat(targetRate)
	return targetRateDec.Div(sourceRateDec)
}
