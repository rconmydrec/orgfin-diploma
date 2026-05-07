package currency

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExchangeRateRepo implements ExchangeRateRepository for testing.
type MockExchangeRateRepo struct {
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
	callCount              int
}

func (m *MockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *MockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}
func (m *MockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error { return nil }
func (m *MockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	m.callCount++
	if m.GetAllRatesForDateFunc != nil {
		return m.GetAllRatesForDateFunc(date)
	}
	return &types.ExchangeRateSnapshot{
		Rates:            map[string]float64{"USD": 1.0, "EUR": 0.85, "GBP": 0.73},
		BaseCurrencyCode: "USD",
	}, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ==================== RateCache Tests ====================

func TestRateCacheCachesResults(t *testing.T) {
	repo := &MockExchangeRateRepo{}
	rc := NewRateCache(repo)

	// First call: cache miss, calls repo
	rates1, err := rc.GetRatesForDate("2024-01-01")
	require.NoError(t, err)
	require.NotNil(t, rates1)
	assert.Equal(t, 1, repo.callCount)

	// Second call for same date: cache hit, no additional repo call
	rates2, err := rc.GetRatesForDate("2024-01-01")
	require.NoError(t, err)
	assert.Equal(t, rates1, rates2)
	assert.Equal(t, 1, repo.callCount, "expected no additional repo call on cache hit")
}

func TestRateCacheDifferentDates(t *testing.T) {
	repo := &MockExchangeRateRepo{}
	rc := NewRateCache(repo)

	_, err := rc.GetRatesForDate("2024-01-01")
	require.NoError(t, err)
	assert.Equal(t, 1, repo.callCount)

	_, err = rc.GetRatesForDate("2024-01-02")
	require.NoError(t, err)
	assert.Equal(t, 2, repo.callCount, "expected repo call for different date")
}

func TestRateCachePropagatesError(t *testing.T) {
	expectedErr := errors.New("connection refused")
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return nil, expectedErr
		},
	}
	rc := NewRateCache(repo)

	rates, err := rc.GetRatesForDate("2024-01-01")
	assert.Nil(t, rates)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRateCacheDoesNotCacheErrors(t *testing.T) {
	callCount := 0
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("temporary error")
			}
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"USD": 1.0},
			}, nil
		},
	}
	rc := NewRateCache(repo)

	// First call: error, should not be cached
	_, err := rc.GetRatesForDate("2024-01-01")
	assert.Error(t, err)

	// Second call: should retry the repo (error was not cached)
	rates, err := rc.GetRatesForDate("2024-01-01")
	require.NoError(t, err)
	require.NotNil(t, rates)
	assert.Equal(t, 2, callCount, "expected second repo call since error should not be cached")
}

// ==================== Service.NewRateCache Tests ====================

func TestServiceNewRateCache(t *testing.T) {
	repo := &MockExchangeRateRepo{}
	svc := New(repo, newTestLogger())

	rc := svc.NewRateCache()
	require.NotNil(t, rc)

	// Should use the service's repo
	rates, err := rc.GetRatesForDate("2024-01-01")
	require.NoError(t, err)
	require.NotNil(t, rates)
	assert.Equal(t, 1, repo.callCount)
}

// ==================== ConvertToBaseCurrency Tests ====================

func TestConvertToBaseCurrencySameCurrency(t *testing.T) {
	svc := New(&MockExchangeRateRepo{}, newTestLogger())
	rc := svc.NewRateCache()

	amount := decimal.NewFromFloat(100.50)
	result, err := svc.ConvertToBaseCurrency(amount, "USD", "USD", "2024-01-01", rc)
	require.NoError(t, err)
	assert.True(t, amount.Equal(result), "same-currency conversion should return original amount")
}

func TestConvertToBaseCurrencySqlErrNoRows(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.ErrorIs(t, err, ErrNoExchangeRates)
	assert.True(t, result.IsZero())
}

func TestConvertToBaseCurrencyNilRates(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return nil, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.ErrorIs(t, err, ErrNoExchangeRates)
	assert.True(t, result.IsZero())
}

func TestConvertToBaseCurrencyEmptyRates(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.ErrorIs(t, err, ErrNoExchangeRates)
	assert.True(t, result.IsZero())
}

func TestConvertToBaseCurrencyMissingSourceRate(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"USD": 1.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoExchangeRates, "missing source rate is a non-fatal error")
	assert.True(t, result.IsZero())
	assert.Contains(t, err.Error(), "currency rate missing")
}

func TestConvertToBaseCurrencyMissingTargetRate(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.85},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoExchangeRates, "missing target rate is a non-fatal error")
	assert.True(t, result.IsZero())
	assert.Contains(t, err.Error(), "currency rate missing")
}

func TestConvertToBaseCurrencyZeroSourceRate(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.0, "USD": 1.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoExchangeRates)
	assert.True(t, result.IsZero())
}

func TestConvertToBaseCurrencyZeroTargetRate(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.85, "USD": 0.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoExchangeRates)
	assert.True(t, result.IsZero())
}

func TestConvertToBaseCurrencyGenericError(t *testing.T) {
	genericErr := errors.New("connection timeout")
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return nil, genericErr
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(100), "EUR", "USD", "2024-01-01", rc)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoExchangeRates, "generic error should not be ErrNoExchangeRates")
	assert.True(t, result.IsZero())
	assert.Contains(t, err.Error(), "exchange rates unavailable")
	assert.ErrorIs(t, err, genericErr, "original error should be wrapped")
}

func TestConvertToBaseCurrencySuccessful(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.85, "USD": 1.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	// 85 EUR / 0.85 * 1.0 = 100 USD
	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(85), "EUR", "USD", "2024-01-01", rc)
	require.NoError(t, err)
	expected := decimal.NewFromInt(100)
	assert.True(t, expected.Equal(result), "expected %s, got %s", expected, result)
}

func TestConvertToBaseCurrencySuccessfulNonUSDTarget(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.85, "GBP": 0.73, "USD": 1.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	// 85 EUR -> GBP: 85 / 0.85 * 0.73 = 73 GBP
	result, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(85), "EUR", "GBP", "2024-01-01", rc)
	require.NoError(t, err)
	expected := decimal.NewFromInt(73)
	assert.True(t, expected.Equal(result), "expected %s, got %s", expected, result)
}

func TestConvertToBaseCurrencyUsesCache(t *testing.T) {
	repo := &MockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return &types.ExchangeRateSnapshot{
				Rates: map[string]float64{"EUR": 0.85, "USD": 1.0},
			}, nil
		},
	}
	svc := New(repo, newTestLogger())
	rc := svc.NewRateCache()

	// Two conversions for the same date should only call repo once
	_, err := svc.ConvertToBaseCurrency(decimal.NewFromInt(85), "EUR", "USD", "2024-01-01", rc)
	require.NoError(t, err)

	_, err = svc.ConvertToBaseCurrency(decimal.NewFromInt(73), "EUR", "USD", "2024-01-01", rc)
	require.NoError(t, err)

	assert.Equal(t, 1, repo.callCount, "expected only one repo call for same date")
}
