package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

type ExchangeRateRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewExchangeRateRepository(db *sql.DB, logger *slog.Logger) *ExchangeRateRepositoryImpl {
	return &ExchangeRateRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// scanRateRow scans a single row from the exchange_rates table, parsing the JSONB rates column.
func scanRateRow(row *sql.Row) (*models.ExchangeRateHistory, error) {
	var rate models.ExchangeRateHistory
	var ratesJSON []byte
	err := row.Scan(&rate.ID, &ratesJSON, &rate.ActualDate, &rate.BaseCurrencyCode,
		&rate.ServiceName, &rate.IsDeleted, &rate.CreatedAt, &rate.UpdatedAt)
	if err != nil {
		return nil, err
	}

	rate.Rates = make(map[string]float64)
	if len(ratesJSON) > 0 {
		if err := json.Unmarshal(ratesJSON, &rate.Rates); err != nil {
			return nil, fmt.Errorf("failed to parse JSONB rates: %w", err)
		}
	}

	return &rate, nil
}

// GetRate returns the latest exchange rate record and extracts the specific
// target currency rate. If the currency is not found in the rates map, returns an error.
func (r *ExchangeRateRepositoryImpl) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	query := `
		SELECT id, rates, actual_date, base_currency_code, service_name, is_deleted, created_at, updated_at
		FROM exchange_rates
		WHERE is_deleted = false
		ORDER BY actual_date DESC
		LIMIT 1
	`

	rate, err := scanRateRow(r.db.QueryRow(query))
	if err != nil {
		return nil, err
	}

	// Verify the requested currency exists in the rates map
	if _, ok := rate.Rates[targetCurrencyCode]; !ok {
		return nil, fmt.Errorf("currency %s not found in exchange rates for %s",
			targetCurrencyCode, rate.ActualDate.Format("2006-01-02"))
	}

	return rate, nil
}

// GetRateForDate returns the exchange rate record for the given date (back-fill: nearest past date,
// then forward-fill: nearest future date). Extracts and validates the specific target currency rate.
func (r *ExchangeRateRepositoryImpl) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	rate, err := r.getAllRatesWithFallback(date)
	if err != nil {
		return nil, err
	}

	// Verify the requested currency exists in the rates map
	if _, ok := rate.Rates[targetCurrencyCode]; !ok {
		return nil, fmt.Errorf("currency %s not found in exchange rates for %s",
			targetCurrencyCode, rate.ActualDate.Format("2006-01-02"))
	}

	// For backward compatibility with the currency service, set a single Rate field.
	// The caller can access rate.Rates[targetCurrencyCode] directly if needed.
	return rate, nil
}

// SaveRate writes exchange rates to the exchange_rates table using UPSERT on (service_name, actual_date).
func (r *ExchangeRateRepositoryImpl) SaveRate(rate *models.ExchangeRateHistory) error {
	ratesJSON, err := json.Marshal(rate.Rates)
	if err != nil {
		return fmt.Errorf("failed to marshal rates to JSON: %w", err)
	}

	query := `
		INSERT INTO exchange_rates (rates, actual_date, base_currency_code, service_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (service_name, actual_date)
		DO UPDATE SET rates = $1, updated_at = now()
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRow(query, ratesJSON, rate.ActualDate, rate.BaseCurrencyCode, rate.ServiceName).
		Scan(&rate.ID, &rate.CreatedAt, &rate.UpdatedAt)
	return err
}

// GetAllRatesForDate returns all exchange rates for the given date, implementing
// back-fill (nearest past date) and forward-fill (nearest future date if no past data).
// Returns an error when the exchange_rates table is completely empty.
func (r *ExchangeRateRepositoryImpl) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	rate, err := r.getAllRatesWithFallback(date)
	if err != nil {
		return nil, err
	}

	return &types.ExchangeRateSnapshot{
		ID:               rate.ID,
		Rates:            rate.Rates,
		ActualDate:       rate.ActualDate.Format("2006-01-02"),
		BaseCurrencyCode: rate.BaseCurrencyCode,
	}, nil
}

// getAllRatesWithFallback implements the back-fill + forward-fill logic:
// 1. Try to find rates for actual_date <= requested date (back-fill)
// 2. If none found, try actual_date >= requested date (forward-fill)
// 3. If still none found, the table is empty -- return sql.ErrNoRows
func (r *ExchangeRateRepositoryImpl) getAllRatesWithFallback(date string) (*models.ExchangeRateHistory, error) {
	// Back-fill: nearest past or exact date
	backfillQuery := `
		SELECT id, rates, actual_date, base_currency_code, service_name, is_deleted, created_at, updated_at
		FROM exchange_rates
		WHERE actual_date <= $1 AND is_deleted = false
		ORDER BY actual_date DESC
		LIMIT 1
	`
	rate, err := scanRateRow(r.db.QueryRow(backfillQuery, date))
	if err == nil {
		return rate, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Forward-fill: nearest future date
	forwardfillQuery := `
		SELECT id, rates, actual_date, base_currency_code, service_name, is_deleted, created_at, updated_at
		FROM exchange_rates
		WHERE actual_date >= $1 AND is_deleted = false
		ORDER BY actual_date ASC
		LIMIT 1
	`
	rate, err = scanRateRow(r.db.QueryRow(forwardfillQuery, date))
	if err == nil {
		return rate, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Table is completely empty
	return nil, sql.ErrNoRows
}
