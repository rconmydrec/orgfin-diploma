package transactions

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/go-budget/backend/internal/workers/tasks"
	"github.com/shopspring/decimal"
)

type Service struct {
	transactionRepo TransactionRepository
	accountRepo     AccountRepository
	categoryRepo    CategoryRepository
	userRepo        UserRepository
	templateRepo    TransactionTemplateRepository
	currencyService CurrencyConverter
	enqueuer        TaskEnqueuer
	logger          *slog.Logger
}

func New(
	transactionRepo TransactionRepository,
	accountRepo AccountRepository,
	categoryRepo CategoryRepository,
	userRepo UserRepository,
	templateRepo TransactionTemplateRepository,
	currencyService CurrencyConverter,
	enqueuer TaskEnqueuer,
	logger *slog.Logger,
) *Service {
	return &Service{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
		userRepo:        userRepo,
		templateRepo:    templateRepo,
		currencyService: currencyService,
		enqueuer:        enqueuer,
		logger:          logger,
	}
}

func (s *Service) CreateTransaction(userID int, input CreateTransactionInput) (*Transaction, error) {
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

	// Validate account
	account, err := s.accountRepo.GetByID(input.AccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidAccount
		}
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Self-transfer guard
	if input.IsTransfer && input.TargetAccountID != nil && *input.TargetAccountID == input.AccountID {
		return nil, ErrSelfTransfer
	}

	// Validate target account for transfers
	var targetAccount *models.Account
	if input.IsTransfer && input.TargetAccountID != nil {
		targetAccount, err = s.accountRepo.GetByID(*input.TargetAccountID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrInvalidAccount
			}
			return nil, err
		}
		if targetAccount.UserID != userID {
			return nil, ErrAccessDenied
		}
	}

	// Validate category (optional for transfers)
	var category *models.UserCategory
	if input.CategoryID != nil {
		category, err = s.categoryRepo.GetByID(*input.CategoryID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// Set datetime
	dateTime := time.Now().UTC()
	if input.DateTime != nil {
		dateTime = *input.DateTime
	}

	// Note: Currency conversion is done at read time in report handlers,
	// not at transaction creation time. This is because the user's base currency
	// can change at any time, which would make pre-computed values incorrect.

	// Calculate new balance
	var newBalance decimal.Decimal
	if input.IsAdjustment {
		newBalance = input.Amount
	} else if input.IsIncome {
		newBalance = account.Balance.Add(input.Amount)
	} else {
		newBalance = account.Balance.Sub(input.Amount)
	}

	tx := &models.Transaction{
		UserID:             userID,
		AccountID:          input.AccountID,
		CategoryID:         input.CategoryID,
		Amount:             input.Amount,
		NewBalance:         &newBalance,
		Label:              &input.Label,
		Notes:              input.Notes,
		DateTime:           &dateTime,
		IsIncome:           input.IsIncome,
		IsTransfer:         input.IsTransfer,
		IsAdjustment:       input.IsAdjustment,
		ExcludeFromReports: input.ExcludeFromReports,
	}

	// Create transaction
	createdTx, err := s.transactionRepo.Create(tx)
	if err != nil {
		return nil, err
	}

	// Update account balance
	if err := s.accountRepo.UpdateBalance(input.AccountID, newBalance); err != nil {
		return nil, err
	}

	// Handle transfer - create linked transaction
	if input.IsTransfer && targetAccount != nil {
		targetAmount := input.Amount
		if input.TargetAmount != nil {
			targetAmount = *input.TargetAmount
		}

		targetNewBalance := targetAccount.Balance.Add(targetAmount)

		linkedTx := &models.Transaction{
			UserID:              userID,
			AccountID:           *input.TargetAccountID,
			CategoryID:          input.CategoryID,
			Amount:              targetAmount,
			NewBalance:          &targetNewBalance,
			Label:               &input.Label,
			Notes:               input.Notes,
			DateTime:            &dateTime,
			IsIncome:            true,
			IsTransfer:          true,
			LinkedTransactionID: &createdTx.ID,
		}

		linkedCreatedTx, err := s.transactionRepo.Create(linkedTx)
		if err != nil {
			return nil, err
		}

		// Update linked transaction ID on source
		createdTx.LinkedTransactionID = &linkedCreatedTx.ID
		if err := s.transactionRepo.UpdateLinkedID(createdTx.ID, linkedCreatedTx.ID); err != nil {
			s.logger.Error("failed to update linked transaction ID", "error", err)
		}

		// Update target account balance
		if err := s.accountRepo.UpdateBalance(*input.TargetAccountID, targetNewBalance); err != nil {
			return nil, err
		}
	}

	// Create template if requested
	if input.IsTemplate {
		template := &models.TransactionTemplate{
			UserID:     userID,
			CategoryID: input.CategoryID,
			Label:      input.Label,
		}
		if input.IsTransfer && input.TargetAccountID != nil {
			template.TargetAccountID = input.TargetAccountID
		}
		if err := s.templateRepo.Create(template); err != nil {
			s.logger.Error("failed to create template", "error", err)
		}
	}

	// Populate relations for response
	createdTx.Account = account
	createdTx.Category = category
	createdTx.User = user

	// Enqueue budget recalculation for the user
	s.enqueueBudgetUpdate(userID, input.AccountID)

	return toTransaction(createdTx), nil
}

func (s *Service) GetTransactions(userID int, input GetTransactionsInput) ([]*Transaction, error) {
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

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PerPage < 1 {
		input.PerPage = 30
	}

	filters := types.TransactionFilters{
		AccountIDs:  input.AccountIDs,
		CategoryIDs: input.CategoryIDs,
		DateFrom:    input.FromDate,
		DateTo:      input.ToDate,
		Limit:       input.PerPage,
		Offset:      (input.Page - 1) * input.PerPage,
	}

	if len(input.Types) > 0 {
		typeStr := input.Types[0]
		filters.Type = &typeStr
	}

	txs, _, err := s.transactionRepo.GetByUserID(userID, filters)
	if err != nil {
		return nil, err
	}

	// Populate base currency amount for each transaction
	for _, tx := range txs {
		tx.User = user
		if s.currencyService != nil && user.BaseCurrency != nil && tx.Account != nil && tx.Account.Currency != nil && tx.DateTime != nil {
			amount := s.currencyService.ConvertAmount(
				tx.Amount,
				tx.Account.Currency.Code,
				user.BaseCurrency.Code,
				*tx.DateTime,
			)
			tx.BaseCurrencyAmount = &amount
			tx.BaseCurrencyCode = &user.BaseCurrency.Code
		}
	}

	return toTransactions(txs), nil
}

func (s *Service) GetTransactionDetails(transactionID, userID int) (*Transaction, error) {
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

	tx, err := s.transactionRepo.GetByID(transactionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}

	if tx.UserID != userID {
		return nil, ErrAccessDenied
	}

	tx.User = user

	// Calculate base currency amount
	if s.currencyService != nil && user.BaseCurrency != nil && tx.Account != nil && tx.Account.Currency != nil && tx.DateTime != nil {
		amount := s.currencyService.ConvertAmount(
			tx.Amount,
			tx.Account.Currency.Code,
			user.BaseCurrency.Code,
			*tx.DateTime,
		)
		tx.BaseCurrencyAmount = &amount
		tx.BaseCurrencyCode = &user.BaseCurrency.Code
	}

	return toTransaction(tx), nil
}

func (s *Service) UpdateTransaction(userID int, input CreateTransactionInput, txID int) (*Transaction, error) {
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

	existingTx, err := s.transactionRepo.GetByID(txID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}

	if existingTx.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Self-transfer guard
	if input.IsTransfer && input.TargetAccountID != nil && *input.TargetAccountID == input.AccountID {
		return nil, ErrSelfTransfer
	}

	// Save old account ID before modification to recalculate its balance if account changes
	oldAccountID := existingTx.AccountID

	// Fetch existing linked transaction (if any) BEFORE modifying the source,
	// so we can capture the old target account ID for balance recalculation.
	var oldLinkedTx *models.Transaction
	if existingTx.IsTransfer && existingTx.LinkedTransactionID != nil {
		oldLinkedTx, err = s.transactionRepo.GetByID(*existingTx.LinkedTransactionID)
		if err != nil {
			s.logger.Error("failed to fetch linked transaction", "error", err, "linkedID", *existingTx.LinkedTransactionID)
			// Non-critical: the linked tx may have been manually deleted.
			// Continue with oldLinkedTx = nil.
			oldLinkedTx = nil
		}
	}

	// Validate account
	account, err := s.accountRepo.GetByID(input.AccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidAccount
		}
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccessDenied
	}

	dateTime := time.Now().UTC()
	if input.DateTime != nil {
		dateTime = *input.DateTime
	}

	wasTransfer := existingTx.IsTransfer
	nowTransfer := input.IsTransfer

	// Update source transaction fields
	existingTx.AccountID = input.AccountID
	existingTx.CategoryID = input.CategoryID
	existingTx.Amount = input.Amount
	existingTx.Label = &input.Label
	existingTx.Notes = input.Notes
	existingTx.DateTime = &dateTime
	existingTx.IsIncome = input.IsIncome
	existingTx.IsTransfer = input.IsTransfer
	existingTx.IsAdjustment = input.IsAdjustment
	existingTx.ExcludeFromReports = input.ExcludeFromReports

	// --- Handle transfer scenarios ---

	// Collect account IDs that need balance recalculation.
	// Using a map to avoid duplicate recalculations.
	accountsToRecalc := map[int]struct{}{
		input.AccountID: {},
	}
	if oldAccountID != input.AccountID {
		accountsToRecalc[oldAccountID] = struct{}{}
	}

	switch {
	case wasTransfer && !nowTransfer:
		// Scenario 4: Transfer becomes a regular transaction.
		// Soft-delete the linked transaction, clear the link.
		if oldLinkedTx != nil {
			if err := s.transactionRepo.Delete(oldLinkedTx.ID); err != nil {
				s.logger.Error("failed to delete linked transaction", "error", err, "linkedID", oldLinkedTx.ID)
			}
			accountsToRecalc[oldLinkedTx.AccountID] = struct{}{}
		}
		existingTx.LinkedTransactionID = nil

	case wasTransfer && nowTransfer:
		// Scenario 1/2/3: Update existing transfer (fields, target account, or source account).
		if oldLinkedTx != nil {
			// Capture old target account ID before any modifications
			oldTargetAccountID := oldLinkedTx.AccountID

			// Determine the new target account ID
			var newTargetAccountID int
			if input.TargetAccountID != nil {
				newTargetAccountID = *input.TargetAccountID
			} else {
				// If no target account specified in update, keep the existing one
				newTargetAccountID = oldTargetAccountID
			}

			// Validate the target account
			targetAccount, err := s.accountRepo.GetByID(newTargetAccountID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, ErrInvalidAccount
				}
				return nil, err
			}
			if targetAccount.UserID != userID {
				return nil, ErrAccessDenied
			}

			// Determine the linked transaction amount
			targetAmount := input.Amount
			if input.TargetAmount != nil {
				targetAmount = *input.TargetAmount
			}

			// Update linked transaction fields to mirror the source
			oldLinkedTx.AccountID = newTargetAccountID
			oldLinkedTx.Amount = targetAmount
			oldLinkedTx.Label = &input.Label
			oldLinkedTx.Notes = input.Notes
			oldLinkedTx.DateTime = &dateTime
			oldLinkedTx.CategoryID = input.CategoryID
			oldLinkedTx.ExcludeFromReports = input.ExcludeFromReports
			oldLinkedTx.IsIncome = true
			oldLinkedTx.IsTransfer = true

			if err := s.transactionRepo.Update(oldLinkedTx); err != nil {
				return nil, err
			}

			// Mark both old and new target accounts for recalculation
			accountsToRecalc[newTargetAccountID] = struct{}{}
			accountsToRecalc[oldTargetAccountID] = struct{}{}
		}

	case !wasTransfer && nowTransfer:
		// Scenario 5: Regular transaction becomes a transfer.
		if input.TargetAccountID != nil {
			targetAccount, err := s.accountRepo.GetByID(*input.TargetAccountID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, ErrInvalidAccount
				}
				return nil, err
			}
			if targetAccount.UserID != userID {
				return nil, ErrAccessDenied
			}

			targetAmount := input.Amount
			if input.TargetAmount != nil {
				targetAmount = *input.TargetAmount
			}

			targetNewBalance := targetAccount.Balance.Add(targetAmount)

			linkedTx := &models.Transaction{
				UserID:              userID,
				AccountID:           *input.TargetAccountID,
				CategoryID:          input.CategoryID,
				Amount:              targetAmount,
				NewBalance:          &targetNewBalance,
				Label:               &input.Label,
				Notes:               input.Notes,
				DateTime:            &dateTime,
				IsIncome:            true,
				IsTransfer:          true,
				LinkedTransactionID: &txID,
				ExcludeFromReports:  input.ExcludeFromReports,
			}

			linkedCreatedTx, err := s.transactionRepo.Create(linkedTx)
			if err != nil {
				return nil, err
			}

			existingTx.LinkedTransactionID = &linkedCreatedTx.ID

			accountsToRecalc[*input.TargetAccountID] = struct{}{}
		}
	}

	// Persist the source transaction
	if err := s.transactionRepo.Update(existingTx); err != nil {
		return nil, err
	}

	// Recalculate balances for all affected accounts
	for accID := range accountsToRecalc {
		if err := s.recalculateBalances(accID); err != nil {
			s.logger.Error("failed to recalculate balances", "error", err, "accountID", accID)
		}
	}

	// Create template if requested
	if input.IsTemplate {
		template := &models.TransactionTemplate{
			UserID:     userID,
			CategoryID: input.CategoryID,
			Label:      input.Label,
		}
		if input.IsTransfer && input.TargetAccountID != nil {
			template.TargetAccountID = input.TargetAccountID
		}
		if err := s.templateRepo.Create(template); err != nil {
			s.logger.Error("failed to create template", "error", err)
		}
	}

	existingTx.Account = account

	// Enqueue budget recalculation for all affected accounts
	for accID := range accountsToRecalc {
		s.enqueueBudgetUpdate(userID, accID)
	}

	return toTransaction(existingTx), nil
}

func (s *Service) DeleteTransaction(transactionID, userID int) (*Transaction, error) {
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

	tx, err := s.transactionRepo.GetByID(transactionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}

	if tx.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Fetch linked transaction BEFORE any deletes so we can get its AccountID
	var linkedAccountID int
	if tx.IsTransfer && tx.LinkedTransactionID != nil {
		linkedTx, err := s.transactionRepo.GetByID(*tx.LinkedTransactionID)
		if err == nil && linkedTx != nil {
			linkedAccountID = linkedTx.AccountID
		}
	}

	// Soft delete
	if err := s.transactionRepo.Delete(transactionID); err != nil {
		return nil, err
	}

	// Delete linked transaction if transfer
	if tx.IsTransfer && tx.LinkedTransactionID != nil {
		if err := s.transactionRepo.Delete(*tx.LinkedTransactionID); err != nil {
			s.logger.Error("failed to delete linked transaction", "error", err)
		}
	}

	// Recalculate source account balances
	if err := s.recalculateBalances(tx.AccountID); err != nil {
		s.logger.Error("failed to recalculate balances", "error", err)
	}

	// Recalculate target account balances using the saved AccountID
	if tx.IsTransfer && linkedAccountID != 0 {
		if err := s.recalculateBalances(linkedAccountID); err != nil {
			s.logger.Error("failed to recalculate target account balances", "error", err)
		}
	}

	// Enqueue budget recalculation for the source account
	s.enqueueBudgetUpdate(userID, tx.AccountID)

	// Enqueue budget recalculation for the linked/target account
	if linkedAccountID != 0 {
		s.enqueueBudgetUpdate(userID, linkedAccountID)
	}

	return toTransaction(tx), nil
}

func (s *Service) GetTemplates(userID int) ([]*TransactionTemplate, error) {
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

	templates, err := s.templateRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return toTransactionTemplates(templates), nil
}

func (s *Service) UpdateTemplate(userID, templateID int, label string, categoryID *int, targetAccountID *int) (*TransactionTemplate, error) {
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

	template, err := s.templateRepo.GetByID(templateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	if template.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Validate target account ownership if provided
	if targetAccountID != nil {
		account, err := s.accountRepo.GetByID(*targetAccountID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrInvalidAccount
			}
			return nil, err
		}
		if account.UserID != userID {
			return nil, ErrAccessDenied
		}
	}

	template.Label = label
	template.CategoryID = categoryID
	template.TargetAccountID = targetAccountID

	if err := s.templateRepo.Update(template); err != nil {
		return nil, err
	}

	// Re-fetch to get the populated TargetAccount relation from the JOIN
	updated, err := s.templateRepo.GetByID(templateID)
	if err != nil {
		return nil, err
	}

	return toTransactionTemplate(updated), nil
}

func (s *Service) DeleteTemplates(userID int, ids []int) ([]*TransactionTemplate, error) {
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

	for _, id := range ids {
		template, err := s.templateRepo.GetByID(id)
		if err != nil {
			continue
		}
		if template.UserID != userID {
			continue
		}
		if err := s.templateRepo.Delete(id); err != nil {
			s.logger.Error("failed to delete template", "error", err, "id", id)
		}
	}

	templates, err := s.templateRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return toTransactionTemplates(templates), nil
}

// enqueueBudgetUpdate enqueues a budget:user_update task if an enqueuer
// is configured. The method is nil-safe: if the enqueuer is nil, the call
// is a no-op. Errors are logged but not propagated because the budget
// update is best-effort and should not block the transaction operation.
func (s *Service) enqueueBudgetUpdate(userID, accountID int) {
	if s.enqueuer == nil {
		return
	}

	task, err := tasks.NewBudgetUserUpdateTask(tasks.BudgetUserUpdatePayload{
		UserID:    userID,
		AccountID: accountID,
	})
	if err != nil {
		s.logger.Error("failed to create budget user update task",
			"error", err,
			"userID", userID,
			"accountID", accountID,
		)
		return
	}

	if _, err := s.enqueuer.Enqueue(task); err != nil {
		s.logger.Error("failed to enqueue budget user update task",
			"error", err,
			"userID", userID,
			"accountID", accountID,
		)
	}
}

// RecalculateFromTransactions computes the final running balance for an
// account given its initial balance and an ordered (chronological) slice of
// transactions, applying the same rules used by recalculateBalances:
//
//   - For adjustment rows (IsAdjustment == true) the running balance is set
//     to *tx.NewBalance (the trusted target). tx.Amount is the absolute delta
//     and is intentionally ignored.
//   - For regular income rows the amount is added.
//   - For regular expense rows the amount is subtracted.
//
// If an adjustment row has a nil NewBalance it is treated as a hard error —
// every adjustment created by the codebase sets NewBalance, so a nil here
// indicates either data corruption or a row written by the buggy pre-fix
// code path.
//
// This helper is pure (no DB access, no side effects) so the CLI
// reconciliation tool can reuse it without dragging in service dependencies.
func RecalculateFromTransactions(initial decimal.Decimal, txs []*models.Transaction) (decimal.Decimal, error) {
	balance := initial
	for _, tx := range txs {
		switch {
		case tx.IsAdjustment:
			if tx.NewBalance == nil {
				return decimal.Zero, fmt.Errorf("adjustment transaction %d has nil new_balance", tx.ID)
			}
			balance = *tx.NewBalance
		case tx.IsIncome:
			balance = balance.Add(tx.Amount)
		default:
			balance = balance.Sub(tx.Amount)
		}
	}
	return balance, nil
}

func (s *Service) recalculateBalances(accountID int) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return err
	}

	txs, err := s.transactionRepo.GetByAccountIDForRecalc(accountID)
	if err != nil {
		return err
	}

	balance := account.InitialBalance
	for _, tx := range txs {
		switch {
		case tx.IsAdjustment:
			if tx.NewBalance == nil {
				err := fmt.Errorf("adjustment transaction %d has nil new_balance", tx.ID)
				s.logger.Error("recalculate balances: invariant violation", "error", err, "transactionID", tx.ID, "accountID", accountID)
				return err
			}
			balance = *tx.NewBalance
		case tx.IsIncome:
			balance = balance.Add(tx.Amount)
		default:
			balance = balance.Sub(tx.Amount)
		}
		tx.NewBalance = &balance
		if err := s.transactionRepo.UpdateBalance(tx.ID, balance); err != nil {
			return err
		}
	}

	return s.accountRepo.UpdateBalance(accountID, balance)
}

// toTransaction converts a models.Transaction to the service domain type.
func toTransaction(m *models.Transaction) *Transaction {
	if m == nil {
		return nil
	}
	t := &Transaction{
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
		IsDeleted:           m.IsDeleted,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
		IsAdjustment:        m.IsAdjustment,
		ExcludeFromReports:  m.ExcludeFromReports,
		BaseCurrencyAmount:  m.BaseCurrencyAmount,
		BaseCurrencyCode:    m.BaseCurrencyCode,
	}
	if m.Account != nil {
		t.Account = &TransactionAccount{
			ID:                    m.Account.ID,
			UserID:                m.Account.UserID,
			Name:                  m.Account.Name,
			AccountTypeID:         m.Account.AccountTypeID,
			CurrencyID:            m.Account.CurrencyID,
			Balance:               m.Account.Balance,
			InitialBalance:        m.Account.InitialBalance,
			Comment:               m.Account.Comment,
			ShowInReports:         m.Account.ShowInReports,
			IsArchived:            m.Account.IsArchived,
			OpeningDate:           m.Account.OpeningDate,
			IsHidden:              m.Account.IsHidden,
			CreditLimit:           m.Account.CreditLimit,
			IsDeleted:             m.Account.IsDeleted,
			BalanceInBaseCurrency: m.Account.BalanceInBaseCurrency,
		}
		if m.Account.Currency != nil {
			t.Account.Currency = &TransactionCurrency{
				ID:   m.Account.Currency.ID,
				Code: m.Account.Currency.Code,
				Name: m.Account.Currency.Name,
			}
		}
		if m.Account.AccountType != nil {
			t.Account.AccountType = &TransactionAccountType{
				ID:       m.Account.AccountType.ID,
				TypeName: m.Account.AccountType.TypeName,
				IsCredit: m.Account.AccountType.IsCredit,
			}
		}
	}
	if m.Category != nil {
		t.Category = toTransactionCategory(m.Category)
	}
	if m.User != nil {
		t.User = &TransactionUser{
			ID:        m.User.ID,
			Email:     m.User.Email,
			FirstName: m.User.FirstName,
			LastName:  m.User.LastName,
		}
	}
	return t
}

// toTransactions converts a slice of models.Transaction to domain types.
func toTransactions(ms []*models.Transaction) []*Transaction {
	result := make([]*Transaction, len(ms))
	for i, m := range ms {
		result[i] = toTransaction(m)
	}
	return result
}

// toTransactionCategory converts a models.UserCategory to the service domain type.
func toTransactionCategory(m *models.UserCategory) *TransactionCategory {
	if m == nil {
		return nil
	}
	cat := &TransactionCategory{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		ParentID:  m.ParentID,
		IsIncome:  m.IsIncome,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Children:  make([]*TransactionCategory, 0),
	}
	for _, child := range m.Children {
		cat.Children = append(cat.Children, toTransactionCategory(child))
	}
	return cat
}

// toTransactionTemplate converts a models.TransactionTemplate to the service domain type.
func toTransactionTemplate(m *models.TransactionTemplate) *TransactionTemplate {
	if m == nil {
		return nil
	}
	t := &TransactionTemplate{
		ID:              m.ID,
		UserID:          m.UserID,
		CategoryID:      m.CategoryID,
		TargetAccountID: m.TargetAccountID,
		Label:           m.Label,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.Category != nil {
		t.Category = toTransactionCategory(m.Category)
	}
	if m.TargetAccount != nil {
		t.TargetAccount = &TransactionAccount{
			ID:   m.TargetAccount.ID,
			Name: m.TargetAccount.Name,
		}
	}
	return t
}

// toTransactionTemplates converts a slice of models.TransactionTemplate to domain types.
func toTransactionTemplates(ms []*models.TransactionTemplate) []*TransactionTemplate {
	result := make([]*TransactionTemplate, len(ms))
	for i, m := range ms {
		result[i] = toTransactionTemplate(m)
	}
	return result
}
