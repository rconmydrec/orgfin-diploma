package accounts

import (
	accountsservice "github.com/go-budget/backend/internal/services/accounts"
	"github.com/shopspring/decimal"
)

// AccountsService defines the interface for accounts service operations
type AccountsService interface {
	CreateAccount(userID int, input accountsservice.CreateAccountInput) (*accountsservice.Account, error)
	GetUserAccounts(userID int, input accountsservice.GetAccountsInput) ([]*accountsservice.Account, error)
	GetAccountDetails(accountID, userID int) (*accountsservice.Account, error)
	GetAccountTypes() ([]*accountsservice.AccountType, error)
	UpdateAccount(accountID, userID int, input accountsservice.UpdateAccountInput) (*accountsservice.Account, error)
	DeleteAccount(accountID, userID int) error
	SetArchiveStatus(accountID int, isArchived bool, userID int) error
	AdjustBalance(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*accountsservice.Transaction, error)
}
