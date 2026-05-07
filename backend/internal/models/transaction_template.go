package models

import "time"

type TransactionTemplate struct {
	ID              int       `json:"id" db:"id"`
	UserID          int       `json:"user_id" db:"user_id"`
	CategoryID      *int      `json:"category_id" db:"category_id"`
	TargetAccountID *int      `json:"target_account_id" db:"target_account_id"`
	Label           string    `json:"label" db:"label"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`

	// Relations
	Category      *UserCategory `json:"category,omitempty" db:"-"`
	TargetAccount *Account      `json:"target_account,omitempty" db:"-"`
}
