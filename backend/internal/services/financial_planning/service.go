package financial_planning

import (
	"errors"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/services/currency"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

// Service encapsulates all financial planning business logic.
type Service struct {
	plannedTxSvc PlannedTransactionService
	accountRepo  AccountRepository
	currencyRepo CurrencyRepository
	currencySvc  CurrencyService
	logger       *slog.Logger
}

// New creates a new financial planning Service with the given dependencies.
func New(
	plannedTxSvc PlannedTransactionService,
	accountRepo AccountRepository,
	currencyRepo CurrencyRepository,
	currencySvc CurrencyService,
	logger *slog.Logger,
) *Service {
	return &Service{
		plannedTxSvc: plannedTxSvc,
		accountRepo:  accountRepo,
		currencyRepo: currencyRepo,
		currencySvc:  currencySvc,
		logger:       logger,
	}
}

// getBaseCurrency resolves the base currency by ID.
func (s *Service) getBaseCurrency(baseCurrencyID int) (*models.Currency, error) {
	return s.currencyRepo.GetByID(baseCurrencyID)
}

// CalculateFutureBalance calculates the projected future balance for a user.
func (s *Service) CalculateFutureBalance(userID int, params FutureBalanceParams) (*FutureBalanceResult, error) {
	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Get user accounts: always require ShowInReports=true, exclude archived.
	// IncludeHidden controls whether hidden accounts are included.
	accountFilters := types.AccountFilters{
		IncludeHidden:     params.IncludeHidden,
		IncludeArchived:   false,
		OnlyShowInReports: true,
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

	// Calculate days until target date (use date-only math to avoid truncation)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	target := time.Date(params.TargetDate.Year(), params.TargetDate.Month(), params.TargetDate.Day(), 0, 0, 0, 0, now.Location())
	days := int(target.Sub(today).Hours()/24 + 0.5)
	if days < 0 {
		days = 0
	}

	// Get upcoming planned transaction occurrences via the planned_transactions service
	plannedTxs, err := s.plannedTxSvc.GetUpcoming(userID, days, params.IncludeInactive)
	if err != nil {
		s.logger.Error("failed to get planned transactions", "error", err)
		return nil, err
	}

	// Initialize exchange rate cache for currency conversion
	rc := s.currencySvc.NewRateCache()

	// Aggregate global totals from all planned transactions, converting to base currency
	globalPlannedIncome := decimal.Zero
	globalPlannedExpenses := decimal.Zero
	incomeCount := 0
	expensesCount := 0

	for _, ptx := range plannedTxs {
		// Look up the currency for this planned transaction
		ptxCurrency, currErr := s.currencyRepo.GetByID(ptx.CurrencyID)
		if currErr != nil {
			s.logger.Warn("failed to get currency for planned tx, skipping",
				"currency_id", ptx.CurrencyID, "error", currErr)
			continue
		}

		// Convert to base currency using the occurrence date
		dateStr := ptx.OccurrenceDate.Format("2006-01-02")
		convertedAmount, convErr := s.currencySvc.ConvertToBaseCurrency(
			ptx.Amount, ptxCurrency.Code, baseCurrency.Code, dateStr, rc,
		)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot calculate future balance")
				return nil, convErr
			}
			s.logger.Warn("skipping planned tx due to missing exchange rate",
				"planned_tx_id", ptx.PlannedTransactionID, "error", convErr)
			continue
		}

		if ptx.IsIncome {
			globalPlannedIncome = globalPlannedIncome.Add(convertedAmount)
			incomeCount++
		} else {
			globalPlannedExpenses = globalPlannedExpenses.Add(convertedAmount)
			expensesCount++
		}
	}

	// Build account projections, converting balances to base currency for the total
	totalCurrentBalance := decimal.Zero
	todayStr := time.Now().Format("2006-01-02")

	var accountProjections []AccountProjection

	for _, account := range accounts {
		if len(params.AccountIDs) > 0 && !accountIDSet[account.ID] {
			continue
		}

		acctCurrency, currErr := s.currencyRepo.GetByID(account.CurrencyID)
		if currErr != nil {
			continue
		}

		// Per-account projections show the account's own currency balance (not converted)
		accountProjections = append(accountProjections, AccountProjection{
			AccountID:            account.ID,
			AccountName:          account.Name,
			CurrencyCode:         acctCurrency.Code,
			CurrentBalance:       account.Balance.InexactFloat64(),
			ProjectedBalance:     account.Balance.InexactFloat64(),
			TotalPlannedIncome:   0,
			TotalPlannedExpenses: 0,
		})

		// Convert account balance to base currency for the total
		convertedBalance, convErr := s.currencySvc.ConvertToBaseCurrency(
			account.Balance, acctCurrency.Code, baseCurrency.Code, todayStr, rc,
		)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot calculate future balance")
				return nil, convErr
			}
			s.logger.Warn("could not convert account balance to base currency, using raw balance",
				"account_id", account.ID, "currency", acctCurrency.Code, "error", convErr)
			convertedBalance = account.Balance
		}
		totalCurrentBalance = totalCurrentBalance.Add(convertedBalance)
	}

	totalProjectedBalance := totalCurrentBalance.Add(globalPlannedIncome).Sub(globalPlannedExpenses)

	return &FutureBalanceResult{
		TargetDate:            params.TargetDate,
		BaseCurrencyCode:      baseCurrency.Code,
		TotalCurrentBalance:   totalCurrentBalance.InexactFloat64(),
		TotalProjectedBalance: totalProjectedBalance.InexactFloat64(),
		TotalPlannedIncome:    globalPlannedIncome.InexactFloat64(),
		TotalPlannedExpenses:  globalPlannedExpenses.InexactFloat64(),
		IncomeCount:           incomeCount,
		ExpensesCount:         expensesCount,
		Accounts:              accountProjections,
	}, nil
}

// GetBalanceProjection generates a balance projection over time for a user.
func (s *Service) GetBalanceProjection(userID int, params BalanceProjectionParams) (*BalanceProjectionResult, error) {
	// Default start date to now if not provided
	startDate := params.StartDate
	if startDate.IsZero() {
		startDate = time.Now()
	}

	// Default period to daily
	period := params.Period
	if period == "" {
		period = "daily"
	}

	baseCurrency, err := s.getBaseCurrency(params.BaseCurrencyID)
	if err != nil {
		s.logger.Error("failed to get base currency", "error", err)
		return nil, err
	}

	// Get user accounts: always require ShowInReports=true, exclude archived.
	// IncludeHidden controls whether hidden accounts are included.
	accountFilters := types.AccountFilters{
		IncludeHidden:     params.IncludeHidden,
		IncludeArchived:   false,
		OnlyShowInReports: true,
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

	// Initialize exchange rate cache for currency conversion
	rc := s.currencySvc.NewRateCache()
	todayStr := time.Now().Format("2006-01-02")

	// Calculate current balance, converting to base currency
	currentBalance := decimal.Zero
	for _, account := range accounts {
		if len(params.AccountIDs) > 0 && !accountIDSet[account.ID] {
			continue
		}

		acctCurrency, currErr := s.currencyRepo.GetByID(account.CurrencyID)
		if currErr != nil {
			s.logger.Warn("failed to get currency for account, using raw balance",
				"account_id", account.ID, "error", currErr)
			currentBalance = currentBalance.Add(account.Balance)
			continue
		}

		convertedBalance, convErr := s.currencySvc.ConvertToBaseCurrency(
			account.Balance, acctCurrency.Code, baseCurrency.Code, todayStr, rc,
		)
		if convErr != nil {
			if errors.Is(convErr, currency.ErrNoExchangeRates) {
				s.logger.Error("exchange rates table is empty, cannot generate projection")
				return nil, convErr
			}
			s.logger.Warn("could not convert account balance to base currency, using raw balance",
				"account_id", account.ID, "currency", acctCurrency.Code, "error", convErr)
			currentBalance = currentBalance.Add(account.Balance)
			continue
		}
		currentBalance = currentBalance.Add(convertedBalance)
	}

	// Calculate days using date-only math to avoid truncation
	projStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	projEnd := time.Date(params.EndDate.Year(), params.EndDate.Month(), params.EndDate.Day(), 0, 0, 0, 0, startDate.Location())
	days := int(projEnd.Sub(projStart).Hours()/24 + 0.5)
	if days < 0 {
		return nil, ErrEndDateBeforeStart
	}

	// Get planned transaction occurrences via the planned_transactions service
	plannedTxs, err := s.plannedTxSvc.GetUpcoming(userID, days, params.IncludeInactive)
	if err != nil {
		s.logger.Error("failed to get planned transactions", "error", err)
		return nil, err
	}

	// Generate date points using proper calendar-based algorithm
	datePoints := generateDatePoints(startDate, params.EndDate, period)

	// Generate projection points by iterating over each period
	var projectionPoints []ProjectionPoint
	runningBalance := currentBalance

	for i, point := range datePoints {
		// Determine the period boundaries for collecting transactions
		var periodStart, periodEnd time.Time

		if i == 0 {
			// First point: period_start = beginning of start_date
			periodStart = time.Date(point.Year(), point.Month(), point.Day(), 0, 0, 0, 0, point.Location())
		} else {
			// Subsequent points: period_start = day after previous point
			prev := datePoints[i-1]
			periodStart = time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location()).AddDate(0, 0, 1)
		}
		periodEnd = time.Date(point.Year(), point.Month(), point.Day(), 23, 59, 59, 999999999, point.Location())

		// Collect all planned transaction occurrences within this period
		periodIncome := decimal.Zero
		periodExpenses := decimal.Zero

		for _, ptx := range plannedTxs {
			occDate := time.Date(ptx.OccurrenceDate.Year(), ptx.OccurrenceDate.Month(), ptx.OccurrenceDate.Day(),
				0, 0, 0, 0, ptx.OccurrenceDate.Location())

			if (occDate.Equal(periodStart) || occDate.After(periodStart)) &&
				(occDate.Equal(periodEnd) || occDate.Before(periodEnd)) {

				// Convert to base currency using the occurrence date
				ptxCurrency, currErr := s.currencyRepo.GetByID(ptx.CurrencyID)
				if currErr != nil {
					s.logger.Warn("failed to get currency for planned tx, skipping",
						"currency_id", ptx.CurrencyID, "error", currErr)
					continue
				}

				dateStr := occDate.Format("2006-01-02")
				convertedAmount, convErr := s.currencySvc.ConvertToBaseCurrency(
					ptx.Amount, ptxCurrency.Code, baseCurrency.Code, dateStr, rc,
				)
				if convErr != nil {
					if errors.Is(convErr, currency.ErrNoExchangeRates) {
						s.logger.Error("exchange rates table is empty, cannot generate projection")
						return nil, convErr
					}
					s.logger.Warn("skipping planned tx due to missing exchange rate",
						"planned_tx_id", ptx.PlannedTransactionID, "error", convErr)
					continue
				}

				if ptx.IsIncome {
					periodIncome = periodIncome.Add(convertedAmount)
				} else {
					periodExpenses = periodExpenses.Add(convertedAmount)
				}
			}
		}

		runningBalance = runningBalance.Add(periodIncome).Sub(periodExpenses)

		projectionPoints = append(projectionPoints, ProjectionPoint{
			Date:     point,
			Balance:  runningBalance.InexactFloat64(),
			Income:   periodIncome.InexactFloat64(),
			Expenses: periodExpenses.InexactFloat64(),
		})
	}

	return &BalanceProjectionResult{
		StartDate:        startDate,
		EndDate:          params.EndDate,
		Period:           period,
		BaseCurrencyCode: baseCurrency.Code,
		ProjectionPoints: projectionPoints,
	}, nil
}

// generateDatePoints generates date points for the projection based on the period type.
// Python's algorithm:
//   - Daily: each day from start to end
//   - Weekly: first point = start_date, then find next Monday, generate end-of-week
//     (Sunday) dates until end_date, cap last point at end_date
//   - Monthly: first point = start_date, then find first of next month, generate
//     end-of-month dates until end_date, cap last point at end_date
func generateDatePoints(startDate, endDate time.Time, period string) []time.Time {
	// Normalize dates to start of day (strip time component)
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	var points []time.Time

	switch period {
	case "weekly":
		// First point is start_date
		points = append(points, start)

		// Find the next Monday after start_date
		daysUntilMonday := (8 - int(start.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		nextMonday := start.AddDate(0, 0, daysUntilMonday)

		// Generate Sunday (end-of-week) dates from that Monday onwards
		for {
			sunday := nextMonday.AddDate(0, 0, 6) // Sunday = Monday + 6
			if sunday.After(end) {
				// Cap at end date if we haven't already added it
				if !points[len(points)-1].Equal(end) {
					points = append(points, end)
				}
				break
			}
			points = append(points, sunday)
			nextMonday = nextMonday.AddDate(0, 0, 7)
		}

	case "monthly":
		// First point is start_date
		points = append(points, start)

		// Start iterating from the first day of the next month
		current := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, start.Location())

		// Generate end-of-month dates: for each month, end = first of NEXT month - 1 day
		for {
			endOfMonth := time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location()).AddDate(0, 0, -1)
			if endOfMonth.After(end) {
				// Cap at end date
				if !points[len(points)-1].Equal(end) {
					points = append(points, end)
				}
				break
			}
			points = append(points, endOfMonth)
			// Move to the next month
			current = current.AddDate(0, 1, 0)
		}

	default: // daily
		for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
			points = append(points, current)
		}
	}

	return points
}
