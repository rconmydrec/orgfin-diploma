// Package accounts provides account management business logic including
// CRUD operations, balance adjustments, and archive status management.
//
// Invariants:
//   - All mutations verify user is active before proceeding.
//   - All mutations on existing accounts verify ownership via UserID.
//   - Balance adjustments create a special transaction (IsAdjustment=true).
//   - Balance in base currency is calculated at read time using CurrencyConverter.
package accounts

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the accounts service.
type ServiceInterface interface {
	CreateAccount(userID int, input CreateAccountInput) (*Account, error)
	GetUserAccounts(userID int, input GetAccountsInput) ([]*Account, error)
	GetAccountDetails(accountID, userID int) (*Account, error)
	GetAccountTypes() ([]*AccountType, error)
	UpdateAccount(accountID, userID int, input UpdateAccountInput) (*Account, error)
	DeleteAccount(accountID, userID int) error
	SetArchiveStatus(accountID int, isArchived bool, userID int) error
	AdjustBalance(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*Transaction, error)
}

// Account represents an account in the accounts service domain.
type Account struct {
	ID             int
	UserID         int
	Name           string
	AccountTypeID  int
	CurrencyID     int
	Balance        decimal.Decimal
	InitialBalance decimal.Decimal
	Comment        *string
	ShowInReports  bool
	IsArchived     bool
	ArchivedAt     *time.Time
	OpeningDate    *time.Time
	IsHidden       bool
	CreditLimit    *decimal.Decimal
	IsDeleted      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Relations
	AccountType *AccountType
	Currency    *Currency

	// Computed
	BalanceInBaseCurrency *decimal.Decimal
}

// AccountType represents an account type in the accounts service domain.
type AccountType struct {
	ID       int
	TypeName string
	IsCredit bool
}

// Currency represents a currency in the accounts service domain.
type Currency struct {
	ID   int
	Code string
	Name string
}

// Transaction represents a transaction in the accounts service domain.
// Used for balance adjustment responses.
type Transaction struct {
	ID                  int
	UserID              int
	AccountID           int
	Amount              decimal.Decimal
	NewBalance          *decimal.Decimal
	CategoryID          *int
	Label               *string
	IsIncome            bool
	IsTransfer          bool
	LinkedTransactionID *int
	Notes               *string
	DateTime            *time.Time
	IsAdjustment        bool
	ExcludeFromReports  bool
	CreatedAt           time.Time
	UpdatedAt           time.Time

	// Computed
	BaseCurrencyAmount *decimal.Decimal
	BaseCurrencyCode   *string
}

// CurrencyConverter converts amounts between currencies using exchange rates.
type CurrencyConverter interface {
	ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal
}

// AccountRepository defines account data access methods needed by the service.
type AccountRepository interface {
	Create(account *models.Account) (*models.Account, error)
	GetByID(id int) (*models.Account, error)
	GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error)
	Update(account *models.Account) error
	SoftDelete(id int) error
	UpdateBalance(id int, balance decimal.Decimal) error
	SetArchiveStatus(id int, isArchived bool) error
}

// AccountTypeRepository defines account type data access methods needed by the service.
type AccountTypeRepository interface {
	GetAll() ([]*models.AccountType, error)
	GetByID(id int) (*models.AccountType, error)
}

// CurrencyRepository defines currency data access methods needed by the service.
type CurrencyRepository interface {
	GetByID(id int) (*models.Currency, error)
}

// UserRepository defines user data access methods needed by the service.
type UserRepository interface {
	GetByID(id int) (*models.User, error)
}

// TransactionRepository defines transaction data access methods needed by the service.
type TransactionRepository interface {
	Create(tx *models.Transaction) (*models.Transaction, error)
}

// CreateAccountInput holds the domain parameters for creating an account.
type CreateAccountInput struct {
	Name           string
	CurrencyID     int
	AccountTypeID  int
	InitialBalance decimal.Decimal
	Balance        decimal.Decimal
	CreditLimit    decimal.Decimal
	IsHidden       bool
	ShowInReports  bool
	OpeningDate    *time.Time
	Comment        string
}

// GetAccountsInput holds the domain parameters for listing accounts.
type GetAccountsInput struct {
	IncludeHidden   bool
	IncludeArchived bool
	ArchivedOnly    bool
}

// UpdateAccountInput holds the domain parameters for updating an account.
type UpdateAccountInput struct {
	Name           string
	CurrencyID     int
	AccountTypeID  int
	InitialBalance decimal.Decimal
	CreditLimit    decimal.Decimal
	IsHidden       bool
	ShowInReports  bool
	OpeningDate    time.Time
	Comment        string
}
