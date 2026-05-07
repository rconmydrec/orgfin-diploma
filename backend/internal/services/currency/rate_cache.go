package currency

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// RateCache caches exchange rate lookups per date within a single request.
type RateCache struct {
	cache map[string]*types.ExchangeRateSnapshot
	repo  ExchangeRateRepository
}

// NewRateCache creates a new RateCache using the given repository.
func NewRateCache(repo ExchangeRateRepository) *RateCache {
	return &RateCache{
		cache: make(map[string]*types.ExchangeRateSnapshot),
		repo:  repo,
	}
}

// GetRatesForDate returns exchange rates for the given date, using a cached
// result if available. On cache miss it fetches from the repository and caches
// the result for subsequent calls with the same date.
func (rc *RateCache) GetRatesForDate(dateStr string) (*types.ExchangeRateSnapshot, error) {
	if cached, ok := rc.cache[dateStr]; ok {
		return cached, nil
	}
	rates, err := rc.repo.GetAllRatesForDate(dateStr)
	if err != nil {
		return nil, err
	}
	rc.cache[dateStr] = rates
	return rates, nil
}

// NewRateCacheFromService creates a new RateCache backed by the service's exchange rate
// repository. This is a convenience factory that avoids exposing the repo to
// callers.
func (s *Service) NewRateCache() *RateCache {
	return NewRateCache(s.exchangeRateRepo)
}

// ConvertToBaseCurrency converts an amount from sourceCurrencyCode to baseCurrencyCode
// using exchange rates for the given date.
//
// Error semantics:
//   - Returns ErrNoExchangeRates when the exchange_rates table is completely empty (sql.ErrNoRows).
//     This is a fatal error that should cause the handler to return HTTP 500.
//   - Returns a non-nil error when a specific currency code is missing from the rates map.
//     This is a non-fatal error: the handler should log a warning and skip the item.
//   - Returns nil error on successful conversion.
func (s *Service) ConvertToBaseCurrency(
	amount decimal.Decimal,
	sourceCurrencyCode, baseCurrencyCode string,
	dateStr string,
	rc *RateCache,
) (decimal.Decimal, error) {
	if sourceCurrencyCode == baseCurrencyCode {
		return amount, nil
	}

	rates, err := rc.GetRatesForDate(dateStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, ErrNoExchangeRates
		}
		s.logger.Warn("exchange rates unavailable",
			"date", dateStr,
			"source", sourceCurrencyCode,
			"target", baseCurrencyCode,
			"error", err,
		)
		return decimal.Zero, fmt.Errorf("exchange rates unavailable for %s: %w", dateStr, err)
	}

	if rates == nil || len(rates.Rates) == 0 {
		return decimal.Zero, ErrNoExchangeRates
	}

	sourceRate, sourceOK := rates.Rates[sourceCurrencyCode]
	targetRate, targetOK := rates.Rates[baseCurrencyCode]

	if !sourceOK || !targetOK || sourceRate == 0 || targetRate == 0 {
		s.logger.Warn("exchange rate not found for currency pair",
			"date", dateStr,
			"source", sourceCurrencyCode,
			"target", baseCurrencyCode,
			"source_rate_found", sourceOK,
			"target_rate_found", targetOK,
		)
		return decimal.Zero, fmt.Errorf("currency rate missing: source=%s(%v) target=%s(%v)",
			sourceCurrencyCode, sourceOK, baseCurrencyCode, targetOK)
	}

	// converted = amount / sourceRate * targetRate
	sourceRateDec := decimal.NewFromFloat(sourceRate)
	targetRateDec := decimal.NewFromFloat(targetRate)
	return amount.Div(sourceRateDec).Mul(targetRateDec), nil
}
