package transactions

import (
	txservice "github.com/go-budget/backend/internal/services/transactions"
)

// TransactionsService defines the interface for transaction operations
type TransactionsService interface {
	CreateTransaction(userID int, input txservice.CreateTransactionInput) (*txservice.Transaction, error)
	GetTransactions(userID int, input txservice.GetTransactionsInput) ([]*txservice.Transaction, error)
	GetTransactionDetails(transactionID, userID int) (*txservice.Transaction, error)
	UpdateTransaction(userID int, input txservice.CreateTransactionInput, txID int) (*txservice.Transaction, error)
	DeleteTransaction(transactionID, userID int) (*txservice.Transaction, error)
	GetTemplates(userID int) ([]*txservice.TransactionTemplate, error)
	UpdateTemplate(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*txservice.TransactionTemplate, error)
	DeleteTemplates(userID int, ids []int) ([]*txservice.TransactionTemplate, error)
}
