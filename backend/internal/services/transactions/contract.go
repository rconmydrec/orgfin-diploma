// Package transactions provides transaction management business logic including
// CRUD operations, transfer handling, balance recalculation, and template management.
//
// Invariants:
//   - All mutations verify user is active before proceeding.
//   - All mutations on existing transactions verify ownership via UserID.
//   - Transfers create linked transactions; deleting one deletes the other.
//   - Balance recalculation is performed on all affected accounts after mutations.
//   - Budget recalculation is enqueued (best-effort) after transaction changes.
//   - Self-transfers (same source and target account) are rejected.
package transactions

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the transactions service.
type ServiceInterface interface {
	CreateTransaction(userID int, input CreateTransactionInput) (*Transaction, error)
	GetTransactions(userID int, input GetTransactionsInput) ([]*Transaction, error)
	GetTransactionDetails(transactionID, userID int) (*Transaction, error)
	UpdateTransaction(userID int, input CreateTransactionInput, txID int) (*Transaction, error)
	DeleteTransaction(transactionID, userID int) (*Transaction, error)
	GetTemplates(userID int) ([]*TransactionTemplate, error)
	UpdateTemplate(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*TransactionTemplate, error)
	DeleteTemplates(userID int, ids []int) ([]*TransactionTemplate, error)
}

// Transaction represents a transaction in the transactions service domain.
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
	IsDeleted           bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	IsAdjustment        bool
	ExcludeFromReports  bool

	// Computed
	BaseCurrencyAmount *decimal.Decimal
	BaseCurrencyCode   *string

	// Relations
	Account  *TransactionAccount
	Category *TransactionCategory
	User     *TransactionUser
}

// TransactionAccount represents an account relation within a transaction.
type TransactionAccount struct {
	ID                    int
	UserID                int
	Name                  string
	AccountTypeID         int
	CurrencyID            int
	Balance               decimal.Decimal
	InitialBalance        decimal.Decimal
	Comment               *string
	ShowInReports         bool
	IsArchived            bool
	OpeningDate           *time.Time
	IsHidden              bool
	CreditLimit           *decimal.Decimal
	IsDeleted             bool
	BalanceInBaseCurrency *decimal.Decimal
	Currency              *TransactionCurrency
	AccountType           *TransactionAccountType
}

// TransactionCurrency represents a currency in the transactions service domain.
type TransactionCurrency struct {
	ID   int
	Code string
	Name string
}

// TransactionAccountType represents an account type in the transactions service domain.
type TransactionAccountType struct {
	ID       int
	TypeName string
	IsCredit bool
}

// TransactionCategory represents a category relation within a transaction.
type TransactionCategory struct {
	ID        int
	UserID    int
	Name      string
	ParentID  *int
	IsIncome  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Children  []*TransactionCategory
}

// TransactionUser represents a user relation within a transaction.
type TransactionUser struct {
	ID        int
	Email     string
	FirstName *string
	LastName  *string
}

// TransactionTemplate represents a transaction template in the service domain.
type TransactionTemplate struct {
	ID              int
	UserID          int
	CategoryID      *int
	TargetAccountID *int
	Label           string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Relations
	Category      *TransactionCategory
	TargetAccount *TransactionAccount
}

// CurrencyConverter converts amounts between currencies using exchange rates.
type CurrencyConverter interface {
	ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal
}

// TransactionRepository defines transaction data access methods needed by the service.
type TransactionRepository interface {
	Create(tx *models.Transaction) (*models.Transaction, error)
	GetByID(id int) (*models.Transaction, error)
	GetByAccountIDForRecalc(accountID int) ([]*models.Transaction, error)
	GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error)
	Update(tx *models.Transaction) error
	UpdateLinkedID(id int, linkedID int) error
	UpdateBalance(id int, newBalance decimal.Decimal) error
	Delete(id int) error
}

// AccountRepository defines account data access methods needed by the service.
type AccountRepository interface {
	GetByID(id int) (*models.Account, error)
	UpdateBalance(id int, balance decimal.Decimal) error
}

// CategoryRepository defines category data access methods needed by the service.
type CategoryRepository interface {
	GetByID(id int) (*models.UserCategory, error)
}

// UserRepository defines user data access methods needed by the service.
type UserRepository interface {
	GetByID(id int) (*models.User, error)
}

// TransactionTemplateRepository defines template data access methods needed by the service.
type TransactionTemplateRepository interface {
	Create(template *models.TransactionTemplate) error
	GetByID(id int) (*models.TransactionTemplate, error)
	GetByUserID(userID int) ([]*models.TransactionTemplate, error)
	Update(template *models.TransactionTemplate) error
	Delete(id int) error
}

// TaskEnqueuer is the interface for enqueuing async tasks.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// CreateTransactionInput holds the domain parameters for creating a transaction.
type CreateTransactionInput struct {
	AccountID          int
	TargetAccountID    *int
	CategoryID         *int
	Amount             decimal.Decimal
	TargetAmount       *decimal.Decimal
	Label              string
	Notes              *string
	DateTime           *time.Time
	IsIncome           bool
	IsTransfer         bool
	IsAdjustment       bool
	ExcludeFromReports bool
	IsTemplate         bool
}

// GetTransactionsInput holds the domain parameters for listing transactions.
type GetTransactionsInput struct {
	Page        int
	PerPage     int
	Types       []string
	AccountIDs  []int
	CurrencyIDs []int
	CategoryIDs []int
	FromDate    *string
	ToDate      *string
}
