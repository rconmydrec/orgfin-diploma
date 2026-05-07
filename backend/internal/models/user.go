package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int       `json:"id" db:"id"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	FirstName      *string   `json:"first_name,omitempty" db:"first_name"`
	LastName       *string   `json:"last_name,omitempty" db:"last_name"`
	BaseCurrencyID int       `json:"base_currency_id" db:"base_currency_id"`
	IsDeleted      bool      `json:"-" db:"is_deleted"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`

	// Relations (populated by joins)
	BaseCurrency *Currency `json:"base_currency,omitempty" db:"-"`
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// DisplayName returns the user's full name built from FirstName and LastName.
// If both are nil/empty, it returns an empty string.
func (u *User) DisplayName() string {
	var name string
	if u.FirstName != nil {
		name = *u.FirstName
	}
	if u.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *u.LastName
	}
	return name
}
