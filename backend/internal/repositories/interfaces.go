package repositories

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(user *models.User) (*models.User, error)
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Activate(userID int) error
	UpdatePassword(userID int, passwordHash string) error
}

// ActivationTokenRepository defines the interface for activation token data access
type ActivationTokenRepository interface {
	Create(token *models.ActivationToken) error
	GetByToken(token string) (*models.ActivationToken, error)
	Delete(id int) error
	DeleteExpired() error
}

// UserSettingsRepository defines the interface for user settings data access
type UserSettingsRepository interface {
	CreateDefault(userID int) error
	GetByUserID(userID int) (*models.UserSettings, error)
	Update(settings *models.UserSettings) error
}

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	GetByUserID(userID int) ([]*models.UserCategory, error)
	GetByUserIDGrouped(userID int) ([]*models.UserCategory, error)
	GetByID(id int) (*models.UserCategory, error)
	Create(category *models.UserCategory) (*models.UserCategory, error)
	Update(category *models.UserCategory) error
	Delete(id int) error
	CopyDefaultCategories(userID int) error
}

// LanguageRepository defines the interface for language data access
type LanguageRepository interface {
	GetAll() ([]*models.Language, error)
	GetByCode(code string) (*models.Language, error)
}

// CurrencyRepository defines the interface for currency data access
type CurrencyRepository interface {
	GetAll() ([]*models.Currency, error)
	GetByCode(code string) (*models.Currency, error)
	GetByID(id int) (*models.Currency, error)
}

// AccountRepository defines the interface for account data access
type AccountRepository interface {
	Create(account *models.Account) (*models.Account, error)
	GetByID(id int) (*models.Account, error)
	GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error)
	Update(account *models.Account) error
	SoftDelete(id int) error
	UpdateBalance(id int, balance decimal.Decimal) error
	SetArchiveStatus(id int, isArchived bool) error
	CountActiveByUserID(userID int) (int, error)
}

// AccountTypeRepository defines the interface for account type data access
type AccountTypeRepository interface {
	GetAll() ([]*models.AccountType, error)
	GetByID(id int) (*models.AccountType, error)
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	Create(tx *models.Transaction) (*models.Transaction, error)
	GetByID(id int) (*models.Transaction, error)
	GetByAccountID(accountID int, limit, offset int) ([]*models.Transaction, error)
	GetByAccountIDForRecalc(accountID int) ([]*models.Transaction, error)
	GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error)
	GetForExport(userID int, startDate, endDate string) ([]*models.Transaction, error)
	Update(tx *models.Transaction) error
	UpdateLinkedID(id int, linkedID int) error
	UpdateBalance(id int, newBalance decimal.Decimal) error
	Delete(id int) error
}

// TransactionTemplateRepository defines the interface for transaction template data access
type TransactionTemplateRepository interface {
	Create(template *models.TransactionTemplate) error
	GetByID(id int) (*models.TransactionTemplate, error)
	GetByUserID(userID int) ([]*models.TransactionTemplate, error)
	Update(template *models.TransactionTemplate) error
	Delete(id int) error
}

// ExchangeRateRepository defines the interface for exchange rate data access
type ExchangeRateRepository interface {
	GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error)
	GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error)
	SaveRate(rate *models.ExchangeRateHistory) error
	GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error)
}

// BudgetRepository defines the interface for budget data access
type BudgetRepository interface {
	Create(budget *models.Budget) (*models.Budget, error)
	GetByID(id int) (*models.Budget, error)
	GetByUserID(userID int, include string) ([]*models.Budget, error)
	GetOutdatedBudgets() ([]*models.Budget, error)
	Update(budget *models.Budget) error
	Delete(id int) error
	Archive(id int) error
	UpdateCollectedAmount(budgetID int, amount decimal.Decimal) error
	GetExpenseTransactionsForBudget(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error)
	CountActiveByUserID(userID int) (int, error)
}

// PlannedTransactionRepository defines the interface for planned transaction data access
type PlannedTransactionRepository interface {
	Create(tx *models.PlannedTransaction) (*models.PlannedTransaction, error)
	GetByID(id int) (*models.PlannedTransaction, error)
	GetByUserID(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error)
	GetActiveByUserID(userID int, includeInactive bool) ([]*models.PlannedTransaction, error)
	Update(tx *models.PlannedTransaction) error
	Delete(id int) error
}

// SubscriptionRepository defines the interface for subscription data access
type SubscriptionRepository interface {
	GetByID(id int) (*models.Subscription, error)
	GetByUserID(userID int) (*models.Subscription, error)
	Create(sub *models.Subscription) error
	Update(sub *models.Subscription) error
	UpdateSubscriptionFull(sub *models.Subscription) error
	GetExpiredSubscriptions() ([]*models.Subscription, error)
	GetPendingDowngrades() ([]*models.Subscription, error)
	GetByPlanType(planType string) ([]*models.Subscription, error)
	GetProviderSubscription(subscriptionID int) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByExternalID(externalSubscriptionID string) (*models.PaymentProviderSubscription, error)
	GetProviderSubscriptionByScheduleID(externalScheduleID string) (*models.PaymentProviderSubscription, error)
	CreateProviderSubscription(pps *models.PaymentProviderSubscription) error
	UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error
}

// SubscriptionPlanRepository defines the interface for subscription plan data access
type SubscriptionPlanRepository interface {
	GetActivePlans() ([]*models.SubscriptionPlan, error)
	GetByID(id int) (*models.SubscriptionPlan, error)
	GetFreePlan() (*models.SubscriptionPlan, error)
}

// PlanPriceRepository defines the interface for plan price data access
type PlanPriceRepository interface {
	GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error)
	GetByExternalPriceID(externalPriceID string) (*models.PlanPrice, error)
	GetActivePricesByProvider(providerType string) ([]*models.PlanPrice, error)
}
