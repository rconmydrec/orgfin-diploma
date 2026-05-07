package budgets

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// Service implements budget business logic including CRUD operations,
// recalculation of collected amounts with proper currency conversion,
// and daily processing of outdated budgets.
type Service struct {
	budgetRepo      BudgetRepository
	transactionRepo TransactionRepository
	currencyRepo    CurrencyRepository
	currencySvc     CurrencyConverter
	logger          *slog.Logger
}

// New creates a new budget service.
func New(budgetRepo BudgetRepository, transactionRepo TransactionRepository, currencyRepo CurrencyRepository, currencySvc CurrencyConverter, logger *slog.Logger) *Service {
	return &Service{
		budgetRepo:      budgetRepo,
		transactionRepo: transactionRepo,
		currencyRepo:    currencyRepo,
		currencySvc:     currencySvc,
		logger:          logger,
	}
}

// Create creates a new budget after validating inputs and normalizing data.
func (s *Service) Create(userID int, params CreateParams) (*Budget, error) {
	startDate, err := dateutil.ParseDate(params.StartDate)
	if err != nil {
		return nil, ErrInvalidStartDate
	}

	endDate, err := dateutil.ParseDate(params.EndDate)
	if err != nil {
		return nil, ErrInvalidEndDate
	}

	currency, err := s.currencyRepo.GetByID(params.CurrencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCurrency
		}
		return nil, err
	}

	// Add 1 day to end date (as per original implementation)
	endDate = endDate.AddDate(0, 0, 1)

	budget := &models.Budget{
		UserID:             userID,
		Name:               params.Name,
		CurrencyID:         params.CurrencyID,
		TargetAmount:       params.TargetAmount,
		Period:             strings.ToUpper(params.Period),
		Repeat:             params.Repeat,
		StartDate:          startDate,
		EndDate:            endDate,
		IncludedCategories: categoriesToCSV(params.Categories),
		Comment:            params.Comment,
	}

	created, err := s.budgetRepo.Create(budget)
	if err != nil {
		return nil, err
	}

	created.Currency = currency

	// Recalculate collected_amount from existing matching transactions.
	// On failure, log and continue: persistence already succeeded, so the
	// HTTP request must still succeed. The stale collected_amount will be
	// self-healing on the next transaction CRUD or manual recalc.
	if err := s.recalculateBudget(userID, created); err != nil {
		s.logger.Error("recalculate budget after create failed",
			"budgetID", created.ID,
			"userID", userID,
			"error", err)
	}

	return toBudget(created), nil
}

// GetByUserID returns budgets for the given user. If include is empty,
// it defaults to "all".
func (s *Service) GetByUserID(userID int, include string) ([]*Budget, error) {
	if include == "" {
		include = "all"
	}
	budgets, err := s.budgetRepo.GetByUserID(userID, include)
	if err != nil {
		return nil, err
	}
	result := make([]*Budget, len(budgets))
	for i, b := range budgets {
		result[i] = toBudget(b)
	}
	return result, nil
}

// Update updates an existing budget after ownership check and input validation.
func (s *Service) Update(userID, budgetID int, params UpdateParams) (*Budget, error) {
	startDate, err := dateutil.ParseDate(params.StartDate)
	if err != nil {
		return nil, ErrInvalidStartDate
	}

	endDate, err := dateutil.ParseDate(params.EndDate)
	if err != nil {
		return nil, ErrInvalidEndDate
	}

	existing, err := s.budgetRepo.GetByID(budgetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrAccessDenied
	}

	currency, err := s.currencyRepo.GetByID(params.CurrencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCurrency
		}
		return nil, err
	}

	existing.Name = params.Name
	existing.CurrencyID = params.CurrencyID
	existing.TargetAmount = params.TargetAmount
	existing.Period = strings.ToUpper(params.Period)
	existing.Repeat = params.Repeat
	existing.StartDate = startDate
	existing.EndDate = endDate.AddDate(0, 0, 1)
	existing.IncludedCategories = categoriesToCSV(params.Categories)
	existing.Comment = params.Comment

	if err := s.budgetRepo.Update(existing); err != nil {
		return nil, err
	}

	existing.Currency = currency

	// Recalculate collected_amount from existing matching transactions
	// against the updated date window, categories, and currency. On failure,
	// log and continue: persistence already succeeded, so the HTTP request
	// must still succeed.
	if err := s.recalculateBudget(userID, existing); err != nil {
		s.logger.Error("recalculate budget after update failed",
			"budgetID", existing.ID,
			"userID", userID,
			"error", err)
	}

	return toBudget(existing), nil
}

// Delete deletes a budget after ownership check.
func (s *Service) Delete(userID, budgetID int) error {
	existing, err := s.budgetRepo.GetByID(budgetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if existing.UserID != userID {
		return ErrAccessDenied
	}

	return s.budgetRepo.Delete(budgetID)
}

// Archive archives a budget after ownership check.
func (s *Service) Archive(userID, budgetID int) error {
	existing, err := s.budgetRepo.GetByID(budgetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if existing.UserID != userID {
		return ErrAccessDenied
	}

	return s.budgetRepo.Archive(budgetID)
}

// DailyProcessing processes outdated budgets: creates copies for repeating
// budgets with new dates and archives all outdated budgets. It handles partial
// failures by logging errors and continuing to the next budget.
func (s *Service) DailyProcessing() (*DailyProcessingResult, error) {
	outdatedBudgets, err := s.budgetRepo.GetOutdatedBudgets()
	if err != nil {
		return nil, err
	}

	processed := 0
	for _, budget := range outdatedBudgets {
		copyCreated := false
		if budget.Repeat {
			newStart, newEnd := s.calculateNewDates(budget)

			newBudget := &models.Budget{
				UserID:             budget.UserID,
				Name:               budget.Name + " (copy)",
				CurrencyID:         budget.CurrencyID,
				TargetAmount:       budget.TargetAmount,
				CollectedAmount:    decimal.Zero,
				Period:             budget.Period,
				Repeat:             budget.Repeat,
				StartDate:          newStart,
				EndDate:            newEnd,
				IncludedCategories: budget.IncludedCategories,
				Comment:            budget.Comment,
			}

			created, err := s.budgetRepo.Create(newBudget)
			if err != nil {
				s.logger.Error("create repeated budget failed", "budgetID", budget.ID, "error", err)
				continue
			}
			copyCreated = true
			s.logger.Info("created copy of repeating budget", "originalBudgetID", budget.ID, "newBudgetID", created.ID)
		}

		if err := s.budgetRepo.Archive(budget.ID); err != nil {
			if copyCreated {
				s.logger.Error("archive budget failed after copy was created",
					"budgetID", budget.ID, "error", err,
					"warning", "duplicate budget may exist")
			} else {
				s.logger.Error("archive budget failed", "budgetID", budget.ID, "error", err)
			}
			continue
		}

		processed++
	}

	return &DailyProcessingResult{Processed: processed}, nil
}

// calculateNewDates computes new start and end dates for a repeating budget
// based on its period type.
func (s *Service) calculateNewDates(budget *models.Budget) (time.Time, time.Time) {
	duration := budget.EndDate.Sub(budget.StartDate)

	var newStart, newEnd time.Time
	switch budget.Period {
	case "DAILY":
		newStart = budget.StartDate.AddDate(0, 0, 1)
		newEnd = budget.EndDate.AddDate(0, 0, 1)
	case "WEEKLY":
		newStart = budget.StartDate.AddDate(0, 0, 7)
		newEnd = budget.EndDate.AddDate(0, 0, 7)
	case "MONTHLY":
		newStart = budget.StartDate.AddDate(0, 1, 0)
		newEnd = budget.EndDate.AddDate(0, 1, 0)
	case "YEARLY":
		newStart = budget.StartDate.AddDate(1, 0, 0)
		newEnd = budget.EndDate.AddDate(1, 0, 0)
	default:
		// For custom periods, shift by the same duration
		newStart = budget.EndDate
		newEnd = newStart.Add(duration)
	}

	return newStart, newEnd
}

// categoriesToCSV converts a slice of category IDs to a comma-separated string.
func categoriesToCSV(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// RecalculateCollectedAmounts recalculates the collected_amount for all
// active budgets belonging to the specified user. For each budget:
//  1. Parse included_categories (comma-separated string -> []int; empty = all categories).
//  2. Query matching expense transactions via TransactionRepository.
//  3. Convert each transaction amount to the budget's currency using exchange rates.
//  4. Sum all converted amounts and update the budget.
//
// The method continues processing remaining budgets even if individual
// recalculations fail, and returns an aggregate error.
func (s *Service) RecalculateCollectedAmounts(userID int) error {
	budgets, err := s.budgetRepo.GetByUserID(userID, "active")
	if err != nil {
		return fmt.Errorf("get user budgets: %w", err)
	}

	if len(budgets) == 0 {
		s.logger.Info("no active budgets for user, skipping recalculation",
			"userID", userID)
		return nil
	}

	var errs []error
	for _, budget := range budgets {
		if err := s.recalculateBudget(userID, budget); err != nil {
			s.logger.Error("failed to recalculate budget",
				"budgetID", budget.ID,
				"userID", userID,
				"error", err)
			errs = append(errs, fmt.Errorf("recalculate budget %d: %w", budget.ID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("budget recalculation errors: %w", errors.Join(errs...))
	}

	s.logger.Info("budget recalculation completed",
		"userID", userID,
		"budgetCount", len(budgets))

	return nil
}

// recalculateBudget handles recalculation for a single budget.
func (s *Service) recalculateBudget(userID int, budget *models.Budget) error {
	// Parse included categories
	categoryIDs := parseCategoryIDs(budget.IncludedCategories)

	// Determine the budget's currency code
	budgetCurrencyCode := ""
	if budget.Currency != nil {
		budgetCurrencyCode = budget.Currency.Code
	}
	if budgetCurrencyCode == "" {
		return fmt.Errorf("budget %d has no currency information", budget.ID)
	}

	// Query matching transactions
	transactions, err := s.transactionRepo.GetExpenseTransactionsForBudget(
		userID, budget.StartDate, budget.EndDate, categoryIDs)
	if err != nil {
		return fmt.Errorf("query transactions: %w", err)
	}

	// Sum amounts with currency conversion
	total := decimal.Zero
	for _, tx := range transactions {
		converted := s.currencySvc.ConvertAmount(
			tx.Amount, tx.CurrencyCode, budgetCurrencyCode, tx.DateTime)
		total = total.Add(converted)
	}

	// Update the budget's collected amount
	if err := s.budgetRepo.UpdateCollectedAmount(budget.ID, total); err != nil {
		return fmt.Errorf("update collected amount: %w", err)
	}

	// Refresh the in-memory value so callers (e.g., Create/Update) can return
	// the recalculated amount in the response without an extra DB round-trip.
	budget.CollectedAmount = total

	return nil
}

// toBudget converts a models.Budget to the service domain type.
func toBudget(m *models.Budget) *Budget {
	if m == nil {
		return nil
	}
	b := &Budget{
		ID:                 m.ID,
		UserID:             m.UserID,
		Name:               m.Name,
		CurrencyID:         m.CurrencyID,
		TargetAmount:       m.TargetAmount,
		CollectedAmount:    m.CollectedAmount,
		Period:             m.Period,
		Repeat:             m.Repeat,
		StartDate:          m.StartDate,
		EndDate:            m.EndDate,
		IncludedCategories: m.IncludedCategories,
		IsArchived:         m.IsArchived,
		Comment:            m.Comment,
		IsDeleted:          m.IsDeleted,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.Currency != nil {
		b.Currency = &Currency{
			ID:   m.Currency.ID,
			Code: m.Currency.Code,
			Name: m.Currency.Name,
		}
	}
	return b
}

// parseCategoryIDs converts a comma-separated string of category IDs
// into a slice of ints. Returns nil if the input is empty.
func parseCategoryIDs(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	var ids []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
