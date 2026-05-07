package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type PlannedTransaction struct {
	ID                    int             `json:"id" db:"id"`
	UserID                int             `json:"user_id" db:"user_id"`
	CurrencyID            int             `json:"currency_id" db:"currency_id"`
	Amount                decimal.Decimal `json:"amount" db:"amount"`
	Label                 string          `json:"label" db:"label"`
	Notes                 *string         `json:"notes" db:"notes"`
	IsIncome              bool            `json:"is_income" db:"is_income"`
	PlannedDate           time.Time       `json:"planned_date" db:"planned_date"`
	IsRecurring           bool            `json:"is_recurring" db:"is_recurring"`
	RecurrenceRule        *string         `json:"recurrence_rule" db:"recurrence_rule"` // JSON string
	IsExecuted            bool            `json:"is_executed" db:"is_executed"`
	ExecutedTransactionID *int            `json:"executed_transaction_id" db:"executed_transaction_id"`
	ExecutionDate         *time.Time      `json:"execution_date" db:"execution_date"`
	IsActive              bool            `json:"is_active" db:"is_active"`
	IsDeleted             bool            `json:"-" db:"is_deleted"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`

	// Relations
	Currency *Currency `json:"currency,omitempty" db:"-"`
}

type RecurrenceRule struct {
	Frequency  string  `json:"frequency"` // daily, weekly, monthly, yearly
	Interval   int     `json:"interval"`
	EndDate    *string `json:"end_date,omitempty"`
	Count      *int    `json:"count,omitempty"`
	DayOfMonth *int    `json:"day_of_month,omitempty"`
}
