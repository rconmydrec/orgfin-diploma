package models

import (
	"time"
)

type Currency struct {
	ID        int       `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ExchangeRateHistory maps to the actual `exchange_rates` table in PostgreSQL.
// The table stores all currency rates as a JSONB object in the `rates` column,
// keyed by currency code (e.g., {"EUR": 0.8486, "BGN": 1.6598, ...}).
// The base currency is always USD.
type ExchangeRateHistory struct {
	ID               int                `json:"id" db:"id"`
	Rates            map[string]float64 `json:"rates" db:"rates"`
	ActualDate       time.Time          `json:"actual_date" db:"actual_date"`
	BaseCurrencyCode string             `json:"base_currency_code" db:"base_currency_code"`
	ServiceName      string             `json:"service_name" db:"service_name"`
	IsDeleted        bool               `json:"is_deleted" db:"is_deleted"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" db:"updated_at"`
}
