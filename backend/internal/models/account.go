package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID             int              `json:"id" db:"id"`
	UserID         int              `json:"user_id" db:"user_id"`
	Name           string           `json:"name" db:"name"`
	AccountTypeID  int              `json:"account_type_id" db:"account_type_id"`
	CurrencyID     int              `json:"currency_id" db:"currency_id"`
	Balance        decimal.Decimal  `json:"balance" db:"balance"`
	InitialBalance decimal.Decimal  `json:"initial_balance" db:"initial_balance"`
	Comment        *string          `json:"comment,omitempty" db:"comment"`
	ShowInReports  bool             `json:"show_in_reports" db:"show_in_reports"`
	IsArchived     bool             `json:"is_archived" db:"is_archived"`
	ArchivedAt     *time.Time       `json:"archived_at,omitempty" db:"archived_at"`
	OpeningDate    *time.Time       `json:"opening_date,omitempty" db:"opening_date"`
	IsHidden       bool             `json:"is_hidden" db:"is_hidden"`
	CreditLimit    *decimal.Decimal `json:"credit_limit,omitempty" db:"credit_limit"`
	IsDeleted      bool             `json:"is_deleted" db:"is_deleted"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`

	// Relations (populated by joins)
	AccountType *AccountType `json:"account_type,omitempty" db:"-"`
	Currency    *Currency    `json:"currency,omitempty" db:"-"`

	// Calculated fields
	BalanceInBaseCurrency *decimal.Decimal `json:"balance_in_base_currency,omitempty" db:"-"`
}

type AccountType struct {
	ID        int       `json:"id" db:"id"`
	TypeName  string    `json:"type_name" db:"type_name"`
	IsCredit  bool      `json:"is_credit" db:"is_credit"`
	IsDeleted bool      `json:"-" db:"is_deleted"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
