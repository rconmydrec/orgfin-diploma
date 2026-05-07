package accounts

import (
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

type Service struct {
	accountRepo     AccountRepository
	accountTypeRepo AccountTypeRepository
	currencyRepo    CurrencyRepository
	userRepo        UserRepository
	transactionRepo TransactionRepository
	currencyService CurrencyConverter
	logger          *slog.Logger
}

func New(
	accountRepo AccountRepository,
	accountTypeRepo AccountTypeRepository,
	currencyRepo CurrencyRepository,
	userRepo UserRepository,
	transactionRepo TransactionRepository,
	currencyService CurrencyConverter,
	logger *slog.Logger,
) *Service {
	return &Service{
		accountRepo:     accountRepo,
		accountTypeRepo: accountTypeRepo,
		currencyRepo:    currencyRepo,
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
		currencyService: currencyService,
		logger:          logger,
	}
}

func (s *Service) CreateAccount(userID int, input CreateAccountInput) (*Account, error) {
	// Validate user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidUser
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActivated
	}

	// Validate currency
	curr, err := s.currencyRepo.GetByID(input.CurrencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCurrency
		}
		return nil, err
	}

	// Validate account type
	accType, err := s.accountTypeRepo.GetByID(input.AccountTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidAccountType
		}
		return nil, err
	}

	// TODO: Check account limit based on subscription

	openingDate := input.OpeningDate
	if openingDate == nil {
		now := time.Now().UTC()
		openingDate = &now
	}

	creditLimit := input.CreditLimit
	account := &models.Account{
		UserID:         userID,
		Name:           input.Name,
		CurrencyID:     input.CurrencyID,
		AccountTypeID:  input.AccountTypeID,
		InitialBalance: input.InitialBalance,
		Balance:        input.Balance,
		CreditLimit:    &creditLimit,
		IsHidden:       input.IsHidden,
		ShowInReports:  input.ShowInReports,
		OpeningDate:    openingDate,
		Comment:        &input.Comment,
	}

	createdAccount, err := s.accountRepo.Create(account)
	if err != nil {
		return nil, err
	}

	// Populate relations
	createdAccount.Currency = curr
	createdAccount.AccountType = accType

	// Calculate balance in base currency
	if user.BaseCurrency != nil {
		balanceInBase := s.currencyService.ConvertAmount(
			createdAccount.Balance,
			curr.Code,
			user.BaseCurrency.Code,
			time.Now().UTC(),
		)
		createdAccount.BalanceInBaseCurrency = &balanceInBase
	}

	return toAccount(createdAccount), nil
}

func (s *Service) GetUserAccounts(userID int, input GetAccountsInput) ([]*Account, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidUser
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActivated
	}

	accounts, err := s.accountRepo.GetByUserID(userID, types.AccountFilters{
		IncludeHidden:   input.IncludeHidden,
		IncludeArchived: input.IncludeArchived,
		ArchivedOnly:    input.ArchivedOnly,
	})
	if err != nil {
		return nil, err
	}

	// Calculate balance in base currency for each account
	for _, acc := range accounts {
		if user.BaseCurrency != nil && acc.Currency != nil {
			balanceInBase := s.currencyService.ConvertAmount(
				acc.Balance,
				acc.Currency.Code,
				user.BaseCurrency.Code,
				time.Now().UTC(),
			)
			acc.BalanceInBaseCurrency = &balanceInBase
		}
	}

	result := make([]*Account, len(accounts))
	for i, acc := range accounts {
		result[i] = toAccount(acc)
	}
	return result, nil
}

func (s *Service) GetAccountDetails(accountID, userID int) (*Account, error) {
	// Validate user is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidUser
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActivated
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Calculate balance in base currency
	if user.BaseCurrency != nil && account.Currency != nil {
		balanceInBase := s.currencyService.ConvertAmount(
			account.Balance,
			account.Currency.Code,
			user.BaseCurrency.Code,
			time.Now().UTC(),
		)
		account.BalanceInBaseCurrency = &balanceInBase
	}

	return toAccount(account), nil
}

func (s *Service) GetAccountTypes() ([]*AccountType, error) {
	types, err := s.accountTypeRepo.GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]*AccountType, len(types))
	for i, t := range types {
		result[i] = toAccountType(t)
	}
	return result, nil
}

func (s *Service) UpdateAccount(accountID, userID int, input UpdateAccountInput) (*Account, error) {
	// Validate user is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidUser
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActivated
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Validate currency
	curr, err := s.currencyRepo.GetByID(input.CurrencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCurrency
		}
		return nil, err
	}

	// Validate account type
	accType, err := s.accountTypeRepo.GetByID(input.AccountTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidAccountType
		}
		return nil, err
	}

	// Update fields (preserve balance!)
	creditLimit := input.CreditLimit
	account.Name = input.Name
	account.CurrencyID = input.CurrencyID
	account.AccountTypeID = input.AccountTypeID
	account.InitialBalance = input.InitialBalance
	account.CreditLimit = &creditLimit
	account.IsHidden = input.IsHidden
	account.ShowInReports = input.ShowInReports
	account.OpeningDate = &input.OpeningDate
	account.Comment = &input.Comment

	if err := s.accountRepo.Update(account); err != nil {
		return nil, err
	}

	// Populate relations
	account.Currency = curr
	account.AccountType = accType

	// Calculate balance in base currency
	if user.BaseCurrency != nil {
		balanceInBase := s.currencyService.ConvertAmount(
			account.Balance,
			curr.Code,
			user.BaseCurrency.Code,
			time.Now().UTC(),
		)
		account.BalanceInBaseCurrency = &balanceInBase
	}

	return toAccount(account), nil
}

func (s *Service) DeleteAccount(accountID, userID int) error {
	// Validate user is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidUser
		}
		return err
	}

	if !user.IsActive {
		return ErrUserNotActivated
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}
		return err
	}

	if account.UserID != userID {
		return ErrAccessDenied
	}

	return s.accountRepo.SoftDelete(accountID)
}

func (s *Service) SetArchiveStatus(accountID int, isArchived bool, userID int) error {
	// Validate user is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidUser
		}
		return err
	}

	if !user.IsActive {
		return ErrUserNotActivated
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}
		return err
	}

	if account.UserID != userID {
		return ErrAccessDenied
	}

	return s.accountRepo.SetArchiveStatus(accountID, isArchived)
}

func (s *Service) AdjustBalance(accountID int, newBalance decimal.Decimal, notes *string, userID int) (*Transaction, error) {
	// Validate user is active
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidUser
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActivated
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrAccessDenied
	}

	difference := newBalance.Sub(account.Balance)
	if difference.IsZero() {
		return nil, ErrBalanceUnchanged
	}

	isIncome := difference.GreaterThan(decimal.Zero)
	amount := difference.Abs()

	// Note: Currency conversion is done at read time in report handlers,
	// not at transaction creation time. This is because the user's base currency
	// can change at any time, which would make pre-computed values incorrect.

	now := time.Now().UTC()
	label := "Balance adjustment"
	excludeFromReports := true
	isAdjustment := true

	tx := &models.Transaction{
		UserID:             userID,
		AccountID:          accountID,
		Amount:             amount,
		NewBalance:         &newBalance,
		Label:              &label,
		Notes:              notes,
		DateTime:           &now,
		IsIncome:           isIncome,
		IsAdjustment:       isAdjustment,
		ExcludeFromReports: excludeFromReports,
	}

	createdTx, err := s.transactionRepo.Create(tx)
	if err != nil {
		return nil, err
	}

	// Update account balance
	if err := s.accountRepo.UpdateBalance(accountID, newBalance); err != nil {
		return nil, err
	}

	return toTransaction(createdTx), nil
}

// toAccount converts a models.Account to the service domain type.
func toAccount(m *models.Account) *Account {
	if m == nil {
		return nil
	}
	acc := &Account{
		ID:                    m.ID,
		UserID:                m.UserID,
		Name:                  m.Name,
		AccountTypeID:         m.AccountTypeID,
		CurrencyID:            m.CurrencyID,
		Balance:               m.Balance,
		InitialBalance:        m.InitialBalance,
		Comment:               m.Comment,
		ShowInReports:         m.ShowInReports,
		IsArchived:            m.IsArchived,
		ArchivedAt:            m.ArchivedAt,
		OpeningDate:           m.OpeningDate,
		IsHidden:              m.IsHidden,
		CreditLimit:           m.CreditLimit,
		IsDeleted:             m.IsDeleted,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
		BalanceInBaseCurrency: m.BalanceInBaseCurrency,
	}
	if m.AccountType != nil {
		acc.AccountType = toAccountType(m.AccountType)
	}
	if m.Currency != nil {
		acc.Currency = &Currency{
			ID:   m.Currency.ID,
			Code: m.Currency.Code,
			Name: m.Currency.Name,
		}
	}
	return acc
}

// toAccountType converts a models.AccountType to the service domain type.
func toAccountType(m *models.AccountType) *AccountType {
	if m == nil {
		return nil
	}
	return &AccountType{
		ID:       m.ID,
		TypeName: m.TypeName,
		IsCredit: m.IsCredit,
	}
}

// toTransaction converts a models.Transaction to the service domain type.
func toTransaction(m *models.Transaction) *Transaction {
	if m == nil {
		return nil
	}
	return &Transaction{
		ID:                  m.ID,
		UserID:              m.UserID,
		AccountID:           m.AccountID,
		Amount:              m.Amount,
		NewBalance:          m.NewBalance,
		CategoryID:          m.CategoryID,
		Label:               m.Label,
		IsIncome:            m.IsIncome,
		IsTransfer:          m.IsTransfer,
		LinkedTransactionID: m.LinkedTransactionID,
		Notes:               m.Notes,
		DateTime:            m.DateTime,
		IsAdjustment:        m.IsAdjustment,
		ExcludeFromReports:  m.ExcludeFromReports,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
		BaseCurrencyAmount:  m.BaseCurrencyAmount,
		BaseCurrencyCode:    m.BaseCurrencyCode,
	}
}
