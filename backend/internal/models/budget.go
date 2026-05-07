package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Budget struct {
	ID                 int             `json:"id" db:"id"`
	UserID             int             `json:"user_id" db:"user_id"`
	Name               string          `json:"name" db:"name"`
	CurrencyID         int             `json:"currency_id" db:"currency_id"`
	TargetAmount       decimal.Decimal `json:"target_amount" db:"target_amount"`
	CollectedAmount    decimal.Decimal `json:"collected_amount" db:"collected_amount"`
	Period             string          `json:"period" db:"period"` // daily, weekly, monthly, yearly, custom
	Repeat             bool            `json:"repeat" db:"repeat"`
	StartDate          time.Time       `json:"start_date" db:"start_date"`
	EndDate            time.Time       `json:"end_date" db:"end_date"`
	IncludedCategories string          `json:"included_categories" db:"included_categories"` // comma-separated IDs
	IsArchived         bool            `json:"is_archived" db:"is_archived"`
	Comment            *string         `json:"comment" db:"comment"`
	IsDeleted          bool            `json:"-" db:"is_deleted"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`

	// Relations
	Currency *Currency `json:"currency,omitempty" db:"-"`
}
