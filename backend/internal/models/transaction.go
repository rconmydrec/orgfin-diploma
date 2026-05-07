package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID                  int              `json:"id" db:"id"`
	UserID              int              `json:"user_id" db:"user_id"`
	AccountID           int              `json:"account_id" db:"account_id"`
	Amount              decimal.Decimal  `json:"amount" db:"amount"`
	NewBalance          *decimal.Decimal `json:"new_balance,omitempty" db:"new_balance"`
	CategoryID          *int             `json:"category_id,omitempty" db:"category_id"`
	Label               *string          `json:"label,omitempty" db:"label"`
	IsIncome            bool             `json:"is_income" db:"is_income"`
	IsTransfer          bool             `json:"is_transfer" db:"is_transfer"`
	LinkedTransactionID *int             `json:"linked_transaction_id,omitempty" db:"linked_transaction_id"`
	Notes               *string          `json:"notes,omitempty" db:"notes"`
	DateTime            *time.Time       `json:"date_time,omitempty" db:"date_time"`
	IsDeleted           bool             `json:"-" db:"is_deleted"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at" db:"updated_at"`
	IsAdjustment        bool             `json:"is_adjustment" db:"is_adjustment"`
	ExcludeFromReports  bool             `json:"exclude_from_reports" db:"exclude_from_reports"`

	// Computed fields (not from DB)
	BaseCurrencyAmount *decimal.Decimal `json:"base_currency_amount,omitempty" db:"-"`
	BaseCurrencyCode   *string          `json:"base_currency_code,omitempty" db:"-"`

	// Relations (populated by joins)
	Account  *Account      `json:"account,omitempty" db:"-"`
	Category *UserCategory `json:"category,omitempty" db:"-"`
	User     *User         `json:"user,omitempty" db:"-"`
}
