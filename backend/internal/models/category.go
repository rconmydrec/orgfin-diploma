package models

import "time"

type DefaultCategory struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ParentID  *int      `json:"parentId,omitempty" db:"parent_id"`
	IsIncome  bool      `json:"isIncome" db:"is_income"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type UserCategory struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"userId" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	ParentID  *int      `json:"parentId,omitempty" db:"parent_id"`
	IsIncome  bool      `json:"isIncome" db:"is_income"`
	IsDeleted bool      `json:"isDeleted" db:"is_deleted"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// Relations
	Parent   *UserCategory   `json:"parent,omitempty" db:"-"`
	Children []*UserCategory `json:"children,omitempty" db:"-"`
}
