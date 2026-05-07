// Package export provides Excel export generation for user transactions.
//
// Invariants:
//   - Amounts are converted to the user's base currency using the currency service.
//   - The export includes headers, transaction rows, and summary rows.
package export

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the export service.
type ServiceInterface interface {
	GenerateExcel(userID int, startDate, endDate string) ([]byte, error)
}

// CurrencyConverter converts amounts between currencies using exchange rates.
type CurrencyConverter interface {
	ConvertAmount(amount decimal.Decimal, fromCurrency, toCurrency string, date time.Time) decimal.Decimal
}

// TransactionRepository defines transaction data access methods needed by the service.
type TransactionRepository interface {
	GetForExport(userID int, startDate, endDate string) ([]*models.Transaction, error)
}

// UserRepository defines user data access methods needed by the service.
type UserRepository interface {
	GetByID(id int) (*models.User, error)
}
