package reports

import (
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// Service encapsulates all reports business logic.
type Service struct {
	transactionRepo TransactionRepository
	accountRepo     AccountRepository
	categoryRepo    CategoryRepository
	currencyRepo    CurrencyRepository
	currencySvc     CurrencyService
	logger          *slog.Logger
}

// New creates a new reports Service with the given dependencies.
func New(
	transactionRepo TransactionRepository,
	accountRepo AccountRepository,
	categoryRepo CategoryRepository,
	currencyRepo CurrencyRepository,
	currencySvc CurrencyService,
	logger *slog.Logger,
) *Service {
	return &Service{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		categoryRepo:    categoryRepo,
		currencyRepo:    currencyRepo,
		currencySvc:     currencySvc,
		logger:          logger,
	}
}

// catInfo is an internal type for category hierarchy building.
type catInfo struct {
	ID       int
	Name     string
	ParentID *int
}

// expenseEntry is an internal type for the shared aggregation pipeline.
type expenseEntry struct {
	Label        string
	Amount       float64
	CategoryID   int
	CategoryName string
}

// getBaseCurrency resolves the base currency by ID.
func (s *Service) getBaseCurrency(baseCurrencyID int) (*models.Currency, error) {
	return s.currencyRepo.GetByID(baseCurrencyID)
}

// getTransactionAmount returns the base currency amount for a transaction by
// always converting via exchange rates at read time.
//
// Error semantics follow CurrencyService.ConvertToBaseCurrency:
//   - currency.ErrNoExchangeRates: fatal, table is empty, report should fail.
//   - Other errors: non-fatal, specific currency missing, skip the transaction.
func (s *Service) getTransactionAmount(
	tx *models.Transaction,
	baseCurrencyCode string,
	startDate string,
	rc *currency.RateCache,
) (decimal.Decimal, error) {
	// Determine source currency from the transaction's account
	sourceCurrency := ""
	if tx.Account != nil && tx.Account.Currency != nil {
		sourceCurrency = tx.Account.Currency.Code
	}

	if sourceCurrency == "" {
		// Cannot determine currency; return raw amount without error
		// since this is a data quality issue, not a missing exchange rate.
		return tx.Amount, nil
	}

	// Use the report's start date for exchange rate lookup, matching the Python
	// implementation. This ensures consistent currency conversion within a report.
	return s.currencySvc.ConvertToBaseCurrency(tx.Amount, sourceCurrency, baseCurrencyCode, startDate, rc)
}

// aggregateChildrenIntoParents rolls up child category expenses into their
// parent totals.
func (s *Service) aggregateChildrenIntoParents(
	categoryExpenses map[int]decimal.Decimal,
	categoryMap map[int]*catInfo,
) map[int]decimal.Decimal {
	parentTotals := make(map[int]decimal.Decimal)
	for catID, total := range categoryExpenses {
		cat := categoryMap[catID]
		if cat == nil {
			continue
		}
		var parentID int
		if cat.ParentID == nil {
			parentID = catID
		} else {
			parentID = *cat.ParentID
		}
		current := parentTotals[parentID]
		parentTotals[parentID] = current.Add(total)
	}
	return parentTotals
}

// applyOtherThreshold combines entries below the given threshold fraction
// (e.g., 0.02 = 2%) into a single "Other" entry, then sorts by amount
// descending.
func (s *Service) applyOtherThreshold(entries []expenseEntry, threshold float64) []expenseEntry {
	var totalSum float64
	for _, entry := range entries {
		totalSum += entry.Amount
	}

	if totalSum <= 0 {
		return entries
	}

	var otherAmount float64
	filtered := make([]expenseEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Amount/totalSum < threshold {
			otherAmount += entry.Amount
		} else {
			filtered = append(filtered, entry)
		}
	}
	if otherAmount > 0 {
		filtered = append(filtered, expenseEntry{
			Label:        "Other",
			Amount:       otherAmount,
			CategoryID:   0,
			CategoryName: "Other",
		})
	}

	// Sort by amount descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Amount > filtered[j].Amount
	})

	return filtered
}

// CashFlowReport generates a cash flow report for the given user and parameters.
func (s *Service) CashFlowReport(userID int, params CashFlowParams) (*CashFlowResult, error) {
	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Default date range when startDate/endDate are not provided
	now := time.Now()
	var fromDate, toDate *string

	if params.StartDate != nil {
		d := params.StartDate.Format("2006-01-02")
		fromDate = &d
	} else {
		var defaultStart string
		switch params.Period {
		case "daily":
			defaultStart = now.AddDate(0, 0, -30).Format("2006-01-02")
		default: // "monthly" and others
			defaultStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -12, 0).Format("2006-01-02")
		}
		fromDate = &defaultStart
	}

	if params.EndDate != nil {
		// Make endDate inclusive by adding 1 day
		d := dateutil.MakeEndDateInclusive(params.EndDate.Format("2006-01-02"))
		toDate = &d
	} else {
		// Default end date: tomorrow (to make today inclusive)
		defaultEnd := now.AddDate(0, 0, 1).Format("2006-01-02")
		toDate = &defaultEnd
	}

	filters := types.TransactionFilters{
		DateFrom: fromDate,
		DateTo:   toDate,
		NoLimit:  true,
	}

	transactions, _, err := s.transactionRepo.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Error("failed to get transactions", "error", err)
		return nil, err
	}

	rc := s.currencySvc.NewRateCache()

	totalIncome := make(map[string]float64)
	totalExpenses := make(map[string]float64)
	netFlow := make(map[string]float64)

	for _, tx := range transactions {
		if tx.ExcludeFromReports || tx.IsTransfer {
			continue
		}

		var periodKey string
		switch params.Period {
		case "monthly":
			periodKey = tx.DateTime.Format("2006-01")
		case "daily":
			periodKey = tx.DateTime.Format("2006-01-02")
		default:
			periodKey = tx.DateTime.Format("2006-01")
		}

		txAmount, convErr := s.getTransactionAmount(tx, baseCurrency.Code, *fromDate, rc)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot generate report")
				return nil, convErr
			}
			// Currency missing from rates map -- skip this transaction
			s.logger.Warn("skipping transaction due to missing exchange rate",
				"tx_id", tx.ID, "error", convErr)
			continue
		}
		amount, _ := txAmount.Float64()
		if tx.IsIncome {
			totalIncome[periodKey] += amount
			netFlow[periodKey] += amount
		} else {
			totalExpenses[periodKey] += amount
			netFlow[periodKey] -= amount
		}
	}

	return &CashFlowResult{
		CurrencyCode:  baseCurrency.Code,
		TotalIncome:   totalIncome,
		TotalExpenses: totalExpenses,
		NetFlow:       netFlow,
	}, nil
}

// BalanceReport generates a balance report for the given user and parameters.
func (s *Service) BalanceReport(userID int, params BalanceParams) ([]BalanceItem, error) {
	s.logger.Info("Getting balance report",
		"user_id", userID,
		"account_ids", params.AccountIDs,
		"balance_date", params.BalanceDate,
		"exclude_hidden", params.ExcludeHidden,
	)

	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	accountFilters := types.AccountFilters{
		IncludeHidden:   !params.ExcludeHidden,
		IncludeArchived: false,
	}

	accounts, err := s.accountRepo.GetByUserID(userID, accountFilters)
	if err != nil {
		s.logger.Error("failed to get accounts", "error", err)
		return nil, err
	}

	// Filter by account IDs if provided
	accountIDSet := make(map[int]bool)
	for _, id := range params.AccountIDs {
		accountIDSet[id] = true
	}

	rc := s.currencySvc.NewRateCache()

	// Parse balance date for historical balance calculation
	balanceDate, balanceDateErr := dateutil.ParseDate(params.BalanceDate)

	result := make([]BalanceItem, 0)
	for _, account := range accounts {
		// Filter by account IDs if specified
		if len(params.AccountIDs) > 0 && !accountIDSet[account.ID] {
			continue
		}

		// Skip hidden accounts for non-hidden report
		if params.ExcludeHidden && account.IsHidden {
			continue
		}

		// Get currency for account
		acctCurrency, currErr := s.currencyRepo.GetByID(account.CurrencyID)
		if currErr != nil {
			continue
		}

		// Calculate balance at the requested date by reversing
		// transactions that occurred after that date.
		balance := account.Balance
		if balanceDateErr == nil {
			dayAfter := balanceDate.AddDate(0, 0, 1).Format("2006-01-02")
			txFilters := types.TransactionFilters{
				AccountIDs: []int{account.ID},
				DateFrom:   &dayAfter,
				NoLimit:    true,
			}
			txsAfter, _, txErr := s.transactionRepo.GetByUserID(userID, txFilters)
			if txErr != nil {
				s.logger.Error("failed to get transactions for balance calculation",
					"account_id", account.ID,
					"error", txErr,
				)
			} else {
				for _, tx := range txsAfter {
					if tx.AccountID != account.ID {
						continue
					}
					// Reverse the effect of this transaction
					if tx.IsIncome {
						// Income or incoming transfer: subtract to go back
						balance = balance.Sub(tx.Amount)
					} else {
						// Expense or outgoing transfer: add back
						balance = balance.Add(tx.Amount)
					}
				}
			}
		}

		balanceFloat, _ := balance.Float64()

		// Convert balance to base currency using exchange rates
		baseCurrencyBalance, convErr := s.currencySvc.ConvertToBaseCurrency(
			balance, acctCurrency.Code, baseCurrency.Code, params.BalanceDate, rc,
		)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot generate report")
				return nil, convErr
			}
			// Currency missing from rates -- use raw balance as fallback for this account
			s.logger.Warn("could not convert balance to base currency",
				"account_id", account.ID, "currency", acctCurrency.Code, "error", convErr)
			baseCurrencyBalance = balance
		}
		baseCurrencyBalanceFloat, _ := baseCurrencyBalance.Float64()

		result = append(result, BalanceItem{
			AccountID:           account.ID,
			AccountName:         account.Name,
			AccountTypeID:       account.AccountTypeID,
			CurrencyCode:        acctCurrency.Code,
			Balance:             balanceFloat,
			BaseCurrencyBalance: baseCurrencyBalanceFloat,
			BaseCurrencyCode:    baseCurrency.Code,
			ReportDate:          params.BalanceDate,
		})
	}

	return result, nil
}

// ExpensesByCategories generates an expenses-by-categories report for the given user.
func (s *Service) ExpensesByCategories(userID int, params ExpensesParams) ([]ExpensesCategoryItem, error) {
	s.logger.Info("Getting expenses by categories",
		"user_id", userID,
		"start_date", params.StartDate,
		"end_date", params.EndDate,
	)

	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Get categories (only expense categories)
	allCategories, err := s.categoryRepo.GetByUserID(userID)
	if err != nil {
		s.logger.Error("failed to get categories", "error", err)
		return nil, err
	}

	// Build category filter set
	categoryFilterSet := make(map[int]bool)
	for _, id := range params.Categories {
		categoryFilterSet[id] = true
	}

	// Build category structures: parent categories first, then direct children only
	type categoryInfo struct {
		ID            int
		Name          string
		ParentID      *int
		ParentName    *string
		DisplayName   string
		IsParent      bool
		TotalExpenses decimal.Decimal
	}

	// First pass: identify top-level parent categories (no parent_id)
	parentNames := make(map[int]string)
	for _, cat := range allCategories {
		if cat.ParentID == nil && !cat.IsIncome {
			parentNames[cat.ID] = cat.Name
		}
	}

	// Build flat category list: only top-level parents and their direct children
	flatCategories := make(map[int]*categoryInfo)
	for _, cat := range allCategories {
		if cat.IsIncome {
			continue // Skip income categories
		}

		// Apply categories filter if specified
		if len(categoryFilterSet) > 0 {
			if cat.ParentID == nil && !categoryFilterSet[cat.ID] {
				continue // Skip parent categories not in filter
			}
			if cat.ParentID != nil && !categoryFilterSet[cat.ID] && !categoryFilterSet[*cat.ParentID] {
				continue // Skip child categories whose parent is not in filter either
			}
		}

		if cat.ParentID == nil {
			// Top-level parent category - always include
			flatCategories[cat.ID] = &categoryInfo{
				ID:            cat.ID,
				Name:          cat.Name,
				DisplayName:   cat.Name,
				IsParent:      true,
				TotalExpenses: decimal.Zero,
			}
		} else if parentName, ok := parentNames[*cat.ParentID]; ok {
			// Direct child of a top-level parent - include with "Parent >> Child" name
			pName := parentName
			flatCategories[cat.ID] = &categoryInfo{
				ID:            cat.ID,
				Name:          cat.Name,
				ParentID:      cat.ParentID,
				ParentName:    &pName,
				DisplayName:   parentName + " >> " + cat.Name,
				IsParent:      false,
				TotalExpenses: decimal.Zero,
			}
		}
		// Skip categories that are grandchildren (parent is not a top-level category)
	}

	// Make endDate inclusive
	inclusiveEndDate := dateutil.MakeEndDateInclusive(params.EndDate)

	// Get transactions for date range
	filters := types.TransactionFilters{
		DateFrom: &params.StartDate,
		DateTo:   &inclusiveEndDate,
		NoLimit:  true,
	}

	// Filter by expense type
	expenseType := "expense"
	filters.Type = &expenseType

	transactions, _, err := s.transactionRepo.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Error("failed to get transactions", "error", err)
		return nil, err
	}

	rc := s.currencySvc.NewRateCache()

	// Aggregate expenses by category using base currency amount
	for _, tx := range transactions {
		if tx.ExcludeFromReports || tx.IsIncome || tx.IsTransfer || tx.CategoryID == nil {
			continue
		}
		if cat, ok := flatCategories[*tx.CategoryID]; ok {
			amount, convErr := s.getTransactionAmount(tx, baseCurrency.Code, params.StartDate, rc)
			if convErr != nil {
				if errors.Is(convErr, currency.ErrNoExchangeRates) {
					s.logger.Error("exchange rates table is empty, cannot generate report")
					return nil, convErr
				}
				s.logger.Warn("skipping transaction due to missing exchange rate",
					"tx_id", tx.ID, "error", convErr)
				continue
			}
			cat.TotalExpenses = cat.TotalExpenses.Add(amount)
		}
	}

	// Build result
	result := make([]ExpensesCategoryItem, 0)
	for _, cat := range flatCategories {
		if params.HideEmptyCategories && cat.TotalExpenses.IsZero() {
			continue
		}

		item := ExpensesCategoryItem{
			ID:            cat.ID,
			Name:          cat.DisplayName,
			ParentID:      cat.ParentID,
			ParentName:    cat.ParentName,
			TotalExpenses: cat.TotalExpenses.InexactFloat64(),
			CurrencyCode:  baseCurrency.Code,
			IsParent:      cat.IsParent,
		}

		result = append(result, item)
	}

	// Sort result: parents alphabetically by name, children grouped under
	// their parent and sorted alphabetically within each group.
	sort.SliceStable(result, func(i, j int) bool {
		// Determine the group key for each item: parents use own name,
		// children use their parent's name.
		groupI := result[i].Name
		if result[i].ParentName != nil {
			groupI = *result[i].ParentName
		}
		groupJ := result[j].Name
		if result[j].ParentName != nil {
			groupJ = *result[j].ParentName
		}

		if groupI != groupJ {
			return groupI < groupJ
		}
		// Within the same group, parents come before children
		if result[i].IsParent != result[j].IsParent {
			return result[i].IsParent
		}
		// Among children of the same parent, sort alphabetically by name
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// DiagramData generates diagram data for the given user and parameters.
func (s *Service) DiagramData(userID int, params DiagramParams) (*DiagramResult, error) {
	s.logger.Info("Getting diagram",
		"user_id", userID,
		"diagram_type", params.DiagramType,
		"start_date", params.StartDate,
		"end_date", params.EndDate,
	)

	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Get categories
	categories, err := s.categoryRepo.GetByUserID(userID)
	if err != nil {
		s.logger.Error("failed to get categories", "error", err)
		return nil, err
	}

	// Make endDate inclusive
	inclusiveEndDate := dateutil.MakeEndDateInclusive(params.EndDate)

	// Get transactions for date range
	expenseType := "expense"
	filters := types.TransactionFilters{
		DateFrom: &params.StartDate,
		DateTo:   &inclusiveEndDate,
		Type:     &expenseType,
		NoLimit:  true,
	}

	transactions, _, err := s.transactionRepo.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Error("failed to get transactions", "error", err)
		return nil, err
	}

	rc := s.currencySvc.NewRateCache()

	// Aggregate expenses by category using base currency amount
	categoryExpenses := make(map[int]decimal.Decimal)
	for _, tx := range transactions {
		if tx.ExcludeFromReports || tx.IsIncome || tx.IsTransfer || tx.CategoryID == nil {
			continue
		}
		amount, convErr := s.getTransactionAmount(tx, baseCurrency.Code, params.StartDate, rc)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot generate report")
				return nil, convErr
			}
			s.logger.Warn("skipping transaction due to missing exchange rate",
				"tx_id", tx.ID, "error", convErr)
			continue
		}
		current := categoryExpenses[*tx.CategoryID]
		categoryExpenses[*tx.CategoryID] = current.Add(amount)
	}

	// Build category structures for parent aggregation
	categoryMap := make(map[int]*catInfo)
	parentCategories := make(map[int]*catInfo)
	for _, cat := range categories {
		if cat.IsIncome {
			continue
		}
		info := &catInfo{ID: cat.ID, Name: cat.Name, ParentID: cat.ParentID}
		categoryMap[cat.ID] = info
		if cat.ParentID == nil {
			parentCategories[cat.ID] = info
		}
	}

	// Aggregate children into parents
	parentTotals := s.aggregateChildrenIntoParents(categoryExpenses, categoryMap)

	// Build aggregated entries
	aggregated := make([]expenseEntry, 0)
	for parentID, total := range parentTotals {
		if total.IsZero() {
			continue
		}
		parent := parentCategories[parentID]
		if parent == nil {
			continue
		}
		f, _ := total.Float64()
		aggregated = append(aggregated, expenseEntry{
			Label:        parent.Name,
			Amount:       f,
			CategoryID:   parentID,
			CategoryName: parent.Name,
		})
	}

	// Apply 2% "Other" threshold and sort by amount descending
	filtered := s.applyOtherThreshold(aggregated, 0.02)

	labels := make([]string, 0, len(filtered))
	data := make([]float64, 0, len(filtered))
	for _, entry := range filtered {
		labels = append(labels, entry.Label)
		data = append(data, entry.Amount)
	}

	return &DiagramResult{
		Labels:       labels,
		Data:         data,
		CurrencyCode: baseCurrency.Code,
	}, nil
}

// ExpensesData generates expenses data with category IDs for the given user.
func (s *Service) ExpensesData(userID int, params ExpensesParams) ([]ExpenseDataEntry, error) {
	s.logger.Info("Getting expenses data for diagram",
		"user_id", userID,
		"start_date", params.StartDate,
		"end_date", params.EndDate,
	)

	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Get categories
	categories, err := s.categoryRepo.GetByUserID(userID)
	if err != nil {
		s.logger.Error("failed to get categories", "error", err)
		return nil, err
	}

	// Build category structures
	categoryMap := make(map[int]*catInfo)
	parentCategories := make(map[int]*catInfo) // Only top-level expense categories

	// Build category filter set
	categoryFilterSet := make(map[int]bool)
	for _, id := range params.Categories {
		categoryFilterSet[id] = true
	}

	for _, cat := range categories {
		if cat.IsIncome {
			continue
		}
		info := &catInfo{ID: cat.ID, Name: cat.Name, ParentID: cat.ParentID}
		categoryMap[cat.ID] = info
		if cat.ParentID == nil {
			parentCategories[cat.ID] = info
		}
	}

	// Make endDate inclusive
	inclusiveEndDate := dateutil.MakeEndDateInclusive(params.EndDate)

	// Get transactions for date range
	expenseType := "expense"
	filters := types.TransactionFilters{
		DateFrom: &params.StartDate,
		DateTo:   &inclusiveEndDate,
		Type:     &expenseType,
		NoLimit:  true,
	}

	transactions, _, err := s.transactionRepo.GetByUserID(userID, filters)
	if err != nil {
		s.logger.Error("failed to get transactions", "error", err)
		return nil, err
	}

	rc := s.currencySvc.NewRateCache()

	// Aggregate expenses by category using base currency amount
	categoryExpenses := make(map[int]decimal.Decimal)
	for _, tx := range transactions {
		if tx.ExcludeFromReports || tx.IsIncome || tx.IsTransfer || tx.CategoryID == nil {
			continue
		}

		// Apply categories filter if specified
		if len(categoryFilterSet) > 0 {
			catID := *tx.CategoryID
			cat := categoryMap[catID]
			if cat == nil {
				continue
			}
			// Check if this category or its parent is in the filter
			if !categoryFilterSet[catID] {
				if cat.ParentID == nil || !categoryFilterSet[*cat.ParentID] {
					continue
				}
			}
		}

		amount, convErr := s.getTransactionAmount(tx, baseCurrency.Code, params.StartDate, rc)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot generate report")
				return nil, convErr
			}
			s.logger.Warn("skipping transaction due to missing exchange rate",
				"tx_id", tx.ID, "error", convErr)
			continue
		}
		current := categoryExpenses[*tx.CategoryID]
		categoryExpenses[*tx.CategoryID] = current.Add(amount)
	}

	// Aggregate children into parents
	parentTotals := s.aggregateChildrenIntoParents(categoryExpenses, categoryMap)

	// Build aggregated results
	aggregated := make([]expenseEntry, 0)
	for parentID, total := range parentTotals {
		if total.IsZero() {
			continue
		}
		parent := parentCategories[parentID]
		if parent == nil {
			continue
		}
		f, _ := total.Float64()
		aggregated = append(aggregated, expenseEntry{
			Label:        parent.Name,
			Amount:       f,
			CategoryID:   parentID,
			CategoryName: parent.Name,
		})
	}

	// Apply 2% "Other" threshold and sort by amount descending
	filtered := s.applyOtherThreshold(aggregated, 0.02)

	// Convert to domain result type
	result := make([]ExpenseDataEntry, 0, len(filtered))
	for _, entry := range filtered {
		result = append(result, ExpenseDataEntry{
			Label:        entry.Label,
			Amount:       entry.Amount,
			CategoryID:   entry.CategoryID,
			CategoryName: entry.CategoryName,
		})
	}

	return result, nil
}
