package transactions

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// FlexInt handles JSON values that can be either int or string
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		*fi = FlexInt(intVal)
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		if strVal == "" {
			*fi = 0
			return nil
		}
		intVal, err := strconv.Atoi(strVal)
		if err != nil {
			return err
		}
		*fi = FlexInt(intVal)
		return nil
	}

	return nil
}

func (fi FlexInt) ToIntPtr() *int {
	if fi == 0 {
		return nil
	}
	v := int(fi)
	return &v
}

// Request DTOs

type CreateTransactionRequest struct {
	AccountID          int              `json:"accountId" validate:"required"`
	TargetAccountID    *int             `json:"targetAccountId"`
	CategoryID         *FlexInt         `json:"categoryId"`
	Amount             decimal.Decimal  `json:"amount" validate:"required"`
	TargetAmount       *decimal.Decimal `json:"targetAmount"`
	Label              string           `json:"label" validate:"max=255"`
	Notes              *string          `json:"notes" validate:"omitempty,max=4000"`
	DateTime           *time.Time       `json:"dateTime"`
	IsIncome           bool             `json:"isIncome"`
	IsTransfer         bool             `json:"isTransfer"`
	IsAdjustment       bool             `json:"isAdjustment"`
	ExcludeFromReports bool             `json:"excludeFromReports"`
	IsTemplate         bool             `json:"isTemplate"`
}

type UpdateTransactionRequest struct {
	ID                 int              `json:"id" validate:"required"`
	AccountID          int              `json:"accountId" validate:"required"`
	TargetAccountID    *int             `json:"targetAccountId"`
	CategoryID         *FlexInt         `json:"categoryId"`
	Amount             decimal.Decimal  `json:"amount" validate:"required"`
	TargetAmount       *decimal.Decimal `json:"targetAmount"`
	Label              string           `json:"label" validate:"max=255"`
	Notes              *string          `json:"notes" validate:"omitempty,max=4000"`
	DateTime           *time.Time       `json:"dateTime"`
	IsIncome           bool             `json:"isIncome"`
	IsTransfer         bool             `json:"isTransfer"`
	IsAdjustment       bool             `json:"isAdjustment"`
	ExcludeFromReports bool             `json:"excludeFromReports"`
	IsTemplate         bool             `json:"isTemplate"`
}

type UpdateTemplateRequest struct {
	ID              int    `json:"id" validate:"required"`
	Label           string `json:"label" validate:"required,max=255"`
	CategoryID      *int   `json:"categoryId"`
	TargetAccountID *int   `json:"targetAccountId"`
}

// Response DTOs

type TransactionResponse struct {
	ID                  int          `json:"id"`
	AccountID           int          `json:"accountId"`
	TargetAccountID     *int         `json:"targetAccountId"`
	CategoryID          *int         `json:"categoryId"`
	Amount              float64      `json:"amount"`
	TargetAmount        *float64     `json:"targetAmount"`
	Label               string       `json:"label"`
	Notes               *string      `json:"notes"`
	DateTime            *time.Time   `json:"dateTime"`
	IsTransfer          bool         `json:"isTransfer"`
	IsIncome            bool         `json:"isIncome"`
	IsAdjustment        bool         `json:"isAdjustment"`
	ExcludeFromReports  bool         `json:"excludeFromReports"`
	UserID              int          `json:"userId"`
	User                *UserDTO     `json:"user"`
	Account             *AccountDTO  `json:"account"`
	BaseCurrencyAmount  *float64     `json:"baseCurrencyAmount"`
	BaseCurrencyCode    *string      `json:"baseCurrencyCode"`
	NewBalance          *float64     `json:"newBalance"`
	Category            *CategoryDTO `json:"category"`
	LinkedTransactionID *int         `json:"linkedTransactionId"`
}

type UserDTO struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
}

type AccountDTO struct {
	ID                    int             `json:"id"`
	UserID                int             `json:"userId"`
	Name                  string          `json:"name"`
	Balance               float64         `json:"balance"`
	InitialBalance        float64         `json:"initialBalance"`
	AccountTypeID         int             `json:"accountTypeId"`
	CurrencyID            int             `json:"currencyId"`
	CreditLimit           float64         `json:"creditLimit"`
	OpeningDate           *time.Time      `json:"openingDate"`
	Comment               *string         `json:"comment"`
	IsHidden              bool            `json:"isHidden"`
	ShowInReports         bool            `json:"showInReports"`
	IsDeleted             bool            `json:"isDeleted"`
	IsArchived            bool            `json:"isArchived"`
	BalanceInBaseCurrency float64         `json:"balanceInBaseCurrency"`
	Currency              *CurrencyDTO    `json:"currency,omitempty"`
	AccountType           *AccountTypeDTO `json:"accountType,omitempty"`
}

type CurrencyDTO struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type AccountTypeDTO struct {
	ID       int    `json:"id"`
	TypeName string `json:"typeName"`
	IsCredit bool   `json:"isCredit"`
}

type CategoryDTO struct {
	ID        int            `json:"id"`
	UserID    int            `json:"userId"`
	Name      string         `json:"name"`
	ParentID  *int           `json:"parentId"`
	IsIncome  bool           `json:"isIncome"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Children  []*CategoryDTO `json:"children"`
}

type TemplateResponse struct {
	ID                int          `json:"id"`
	CategoryID        *int         `json:"categoryId"`
	TargetAccountID   *int         `json:"targetAccountId"`
	TargetAccountName *string      `json:"targetAccountName"`
	Label             string       `json:"label"`
	Category          *CategoryDTO `json:"category,omitempty"`
}
