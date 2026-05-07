package accounts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/shopspring/decimal"
)

// ValidationErrorResponse is returned when request validation fails with field-level details.
type ValidationErrorResponse struct {
	Detail string            `json:"detail"`
	Errors []ValidationField `json:"errors"`
}

// ValidationField describes a single validation error for a specific field.
type ValidationField struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Request DTOs

type CreateAccountRequest struct {
	Name           string          `json:"name" validate:"required"`
	CurrencyID     int             `json:"currencyId" validate:"required"`
	AccountTypeID  int             `json:"accountTypeId" validate:"required"`
	InitialBalance decimal.Decimal `json:"initialBalance"`
	Balance        decimal.Decimal `json:"balance"`
	CreditLimit    decimal.Decimal `json:"creditLimit"`
	IsHidden       bool            `json:"isHidden"`
	ShowInReports  *bool           `json:"showInReports"`
	OpeningDate    *time.Time      `json:"openingDate"`
	Comment        string          `json:"comment"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CreateAccountRequest.
// It accepts both camelCase and snake_case field names to support mixed frontend conventions.
func (r *CreateAccountRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Helper to get a value by camelCase key first, then snake_case fallback.
	get := func(camel, snake string) (json.RawMessage, bool) {
		if v, ok := raw[camel]; ok {
			return v, true
		}
		if v, ok := raw[snake]; ok {
			return v, true
		}
		return nil, false
	}

	// name (no alias needed)
	if v, ok := raw["name"]; ok {
		if err := json.Unmarshal(v, &r.Name); err != nil {
			return fmt.Errorf("invalid value for name: %w", err)
		}
	}

	// accountTypeId / account_type_id
	if v, ok := get("accountTypeId", "account_type_id"); ok {
		if err := json.Unmarshal(v, &r.AccountTypeID); err != nil {
			return fmt.Errorf("invalid value for accountTypeId: %w", err)
		}
	}

	// currencyId / currency_id
	if v, ok := get("currencyId", "currency_id"); ok {
		if err := json.Unmarshal(v, &r.CurrencyID); err != nil {
			return fmt.Errorf("invalid value for currencyId: %w", err)
		}
	}

	// initialBalance / initial_balance — decimal.Decimal, handle string or number
	if v, ok := get("initialBalance", "initial_balance"); ok {
		if err := unmarshalDecimal(v, &r.InitialBalance); err != nil {
			return fmt.Errorf("invalid value for initialBalance: %w", err)
		}
	}

	// balance (no alias needed)
	if v, ok := raw["balance"]; ok {
		if err := unmarshalDecimal(v, &r.Balance); err != nil {
			return fmt.Errorf("invalid value for balance: %w", err)
		}
	}

	// creditLimit / credit_limit
	if v, ok := get("creditLimit", "credit_limit"); ok {
		if err := unmarshalDecimal(v, &r.CreditLimit); err != nil {
			return fmt.Errorf("invalid value for creditLimit: %w", err)
		}
	}

	// isHidden / is_hidden
	if v, ok := get("isHidden", "is_hidden"); ok {
		if err := json.Unmarshal(v, &r.IsHidden); err != nil {
			return fmt.Errorf("invalid value for isHidden: %w", err)
		}
	}

	// showInReports / show_in_reports — *bool, leave nil if not present
	if v, ok := get("showInReports", "show_in_reports"); ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return fmt.Errorf("invalid value for showInReports: %w", err)
		}
		r.ShowInReports = &b
	}

	// openingDate / opening_date — *time.Time, try multiple formats
	if v, ok := get("openingDate", "opening_date"); ok {
		if err := unmarshalTime(v, &r.OpeningDate); err != nil {
			return fmt.Errorf("invalid value for openingDate: %w", err)
		}
	}

	// comment (no alias needed)
	if v, ok := raw["comment"]; ok {
		if err := json.Unmarshal(v, &r.Comment); err != nil {
			return fmt.Errorf("invalid value for comment: %w", err)
		}
	}

	return nil
}

// unmarshalDecimal handles JSON values that can be either a number or a quoted string.
func unmarshalDecimal(data json.RawMessage, d *decimal.Decimal) error {
	// Try unmarshaling as number first (decimal.Decimal handles both).
	if err := json.Unmarshal(data, d); err == nil {
		return nil
	}
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected number or string")
	}
	parsed, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("invalid decimal string %q: %w", s, err)
	}
	*d = parsed
	return nil
}

// unmarshalTime parses a JSON time value using the shared date parser.
// Preserves empty-string handling: returns nil for empty strings.
func unmarshalTime(data json.RawMessage, t **time.Time) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected string for time value")
	}
	if s == "" {
		return nil
	}

	parsed, err := dateutil.ParseDate(s)
	if err != nil {
		return err
	}
	*t = &parsed
	return nil
}

type UpdateAccountRequest struct {
	Name           string          `json:"name"`
	CurrencyID     int             `json:"currency_id"`
	AccountTypeID  int             `json:"account_type_id"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
	Balance        decimal.Decimal `json:"balance"`
	CreditLimit    decimal.Decimal `json:"creditLimit"`
	IsHidden       bool            `json:"is_hidden"`
	ShowInReports  bool            `json:"show_in_reports"`
	OpeningDate    string          `json:"opening_date"`
	Comment        string          `json:"comment"`
}

type ArchiveStatusRequest struct {
	AccountID  int  `json:"accountId" validate:"required"`
	IsArchived bool `json:"isArchived"`
}

type BalanceAdjustmentRequest struct {
	AccountID  int             `json:"accountId" validate:"required"`
	NewBalance decimal.Decimal `json:"newBalance" validate:"required"`
	Notes      *string         `json:"notes"`
}

// Response DTOs

type AccountResponse struct {
	UserID                int                `json:"userId"`
	AccountTypeID         int                `json:"accountTypeId"`
	CurrencyID            int                `json:"currencyId"`
	InitialBalance        float64            `json:"initialBalance"`
	Balance               float64            `json:"balance"`
	CreditLimit           float64            `json:"creditLimit"`
	Name                  string             `json:"name"`
	OpeningDate           *time.Time         `json:"openingDate"`
	Comment               string             `json:"comment"`
	IsHidden              bool               `json:"isHidden"`
	ShowInReports         bool               `json:"showInReports"`
	ID                    int                `json:"id"`
	Currency              common.CurrencyDTO `json:"currency"`
	AccountType           AccountTypeDTO     `json:"accountType"`
	IsDeleted             bool               `json:"isDeleted"`
	IsArchived            bool               `json:"isArchived"`
	BalanceInBaseCurrency float64            `json:"balanceInBaseCurrency"`
	ArchivedAt            *time.Time         `json:"archivedAt"`
}

// AccountTypeDTO uses snake_case to match original API
type AccountTypeDTO struct {
	ID       int    `json:"id"`
	TypeName string `json:"type_name"`
	IsCredit bool   `json:"is_credit"`
}

// TransactionResponse for balance adjustment
type TransactionResponse struct {
	ID                  int        `json:"id"`
	UserID              int        `json:"userId"`
	AccountID           int        `json:"accountId"`
	CategoryID          *int       `json:"categoryId"`
	Amount              float64    `json:"amount"`
	Label               string     `json:"label"`
	Notes               *string    `json:"notes"`
	DateTime            *time.Time `json:"dateTime"`
	IsTransfer          bool       `json:"isTransfer"`
	IsIncome            bool       `json:"isIncome"`
	IsAdjustment        bool       `json:"isAdjustment"`
	ExcludeFromReports  bool       `json:"excludeFromReports"`
	BaseCurrencyAmount  *float64   `json:"baseCurrencyAmount"`
	NewBalance          *float64   `json:"newBalance"`
	LinkedTransactionID *int       `json:"linkedTransactionId"`
}
