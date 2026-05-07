package repositories

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type BudgetRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewBudgetRepository(db *sql.DB, logger *slog.Logger) *BudgetRepositoryImpl {
	return &BudgetRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *BudgetRepositoryImpl) Create(budget *models.Budget) (*models.Budget, error) {
	query := `
		INSERT INTO budgets (user_id, name, currency_id, target_amount, collected_amount, period, repeat,
		                     start_date, end_date, included_categories, is_archived, comment, is_deleted)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $9, false, $10, false)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		budget.UserID, budget.Name, budget.CurrencyID, budget.TargetAmount,
		budget.Period, budget.Repeat, budget.StartDate, budget.EndDate,
		budget.IncludedCategories, budget.Comment,
	).Scan(&budget.ID, &budget.CreatedAt, &budget.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create budget", "error", err)
		return nil, err
	}

	return budget, nil
}

func (r *BudgetRepositoryImpl) GetByID(id int) (*models.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.name, b.currency_id, b.target_amount, b.collected_amount,
		       b.period, b.repeat, b.start_date, b.end_date, b.included_categories,
		       b.is_archived, b.comment, b.is_deleted, b.created_at, b.updated_at,
		       c.id, c.code, c.name
		FROM budgets b
		LEFT JOIN currencies c ON b.currency_id = c.id
		WHERE b.id = $1 AND b.is_deleted = false
	`

	budget := &models.Budget{}
	currency := &models.Currency{}

	err := r.db.QueryRow(query, id).Scan(
		&budget.ID, &budget.UserID, &budget.Name, &budget.CurrencyID, &budget.TargetAmount,
		&budget.CollectedAmount, &budget.Period, &budget.Repeat, &budget.StartDate, &budget.EndDate,
		&budget.IncludedCategories, &budget.IsArchived, &budget.Comment, &budget.IsDeleted,
		&budget.CreatedAt, &budget.UpdatedAt,
		&currency.ID, &currency.Code, &currency.Name,
	)
	if err != nil {
		return nil, err
	}

	budget.Currency = currency
	return budget, nil
}

func (r *BudgetRepositoryImpl) GetByUserID(userID int, include string) ([]*models.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.name, b.currency_id, b.target_amount, b.collected_amount,
		       b.period, b.repeat, b.start_date, b.end_date, b.included_categories,
		       b.is_archived, b.comment, b.is_deleted, b.created_at, b.updated_at,
		       c.id, c.code, c.name
		FROM budgets b
		LEFT JOIN currencies c ON b.currency_id = c.id
		WHERE b.user_id = $1 AND b.is_deleted = false
	`

	switch include {
	case "active":
		query += " AND b.is_archived = false"
	case "archived":
		query += " AND b.is_archived = true"
	}

	query += " ORDER BY b.start_date DESC"

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []*models.Budget
	for rows.Next() {
		budget := &models.Budget{}
		currency := &models.Currency{}

		err := rows.Scan(
			&budget.ID, &budget.UserID, &budget.Name, &budget.CurrencyID, &budget.TargetAmount,
			&budget.CollectedAmount, &budget.Period, &budget.Repeat, &budget.StartDate, &budget.EndDate,
			&budget.IncludedCategories, &budget.IsArchived, &budget.Comment, &budget.IsDeleted,
			&budget.CreatedAt, &budget.UpdatedAt,
			&currency.ID, &currency.Code, &currency.Name,
		)
		if err != nil {
			return nil, err
		}

		budget.Currency = currency
		budgets = append(budgets, budget)
	}

	return budgets, rows.Err()
}

func (r *BudgetRepositoryImpl) GetOutdatedBudgets() ([]*models.Budget, error) {
	query := `
		SELECT b.id, b.user_id, b.name, b.currency_id, b.target_amount, b.collected_amount,
		       b.period, b.repeat, b.start_date, b.end_date, b.included_categories,
		       b.is_archived, b.comment, b.is_deleted, b.created_at, b.updated_at,
		       c.id, c.code, c.name
		FROM budgets b
		LEFT JOIN currencies c ON b.currency_id = c.id
		WHERE b.end_date < NOW() AND b.is_archived = false AND b.is_deleted = false
		ORDER BY b.end_date ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []*models.Budget
	for rows.Next() {
		budget := &models.Budget{}
		currency := &models.Currency{}

		err := rows.Scan(
			&budget.ID, &budget.UserID, &budget.Name, &budget.CurrencyID, &budget.TargetAmount,
			&budget.CollectedAmount, &budget.Period, &budget.Repeat, &budget.StartDate, &budget.EndDate,
			&budget.IncludedCategories, &budget.IsArchived, &budget.Comment, &budget.IsDeleted,
			&budget.CreatedAt, &budget.UpdatedAt,
			&currency.ID, &currency.Code, &currency.Name,
		)
		if err != nil {
			return nil, err
		}

		budget.Currency = currency
		budgets = append(budgets, budget)
	}

	return budgets, rows.Err()
}

func (r *BudgetRepositoryImpl) Update(budget *models.Budget) error {
	query := `
		UPDATE budgets
		SET name = $1, currency_id = $2, target_amount = $3, period = $4, repeat = $5,
		    start_date = $6, end_date = $7, included_categories = $8, comment = $9, updated_at = NOW()
		WHERE id = $10
	`

	_, err := r.db.Exec(query,
		budget.Name, budget.CurrencyID, budget.TargetAmount, budget.Period, budget.Repeat,
		budget.StartDate, budget.EndDate, budget.IncludedCategories, budget.Comment, budget.ID)
	return err
}

func (r *BudgetRepositoryImpl) Delete(id int) error {
	query := `UPDATE budgets SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BudgetRepositoryImpl) Archive(id int) error {
	query := `UPDATE budgets SET is_archived = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// UpdateCollectedAmount sets the collected_amount for a budget.
// This is a simple CRUD operation; the actual calculation logic
// belongs in the budget service layer.
func (r *BudgetRepositoryImpl) UpdateCollectedAmount(budgetID int, amount decimal.Decimal) error {
	query := `UPDATE budgets SET collected_amount = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, amount, budgetID)
	if err != nil {
		r.logger.Error("failed to update collected amount", "budgetID", budgetID, "error", err)
		return err
	}
	return nil
}

// GetExpenseTransactionsForBudget returns expense (non-income, non-transfer,
// non-deleted, not excluded from reports) transactions within the given date
// range, optionally filtered by category IDs.
// If categoryIDs is empty, all matching expense transactions are returned.
func (r *BudgetRepositoryImpl) GetExpenseTransactionsForBudget(userID int, startDate, endDate time.Time, categoryIDs []int) ([]types.BudgetTransactionRow, error) {
	query := `
		SELECT t.amount, c.code, t.date_time
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		JOIN currencies c ON a.currency_id = c.id
		WHERE t.user_id = $1
		  AND t.is_income = false
		  AND t.is_transfer = false
		  AND t.is_deleted = false
		  AND t.exclude_from_reports = false
		  AND t.date_time >= $2
		  AND t.date_time < $3
	`
	args := []any{userID, startDate, endDate}

	if len(categoryIDs) > 0 {
		query += fmt.Sprintf(" AND t.category_id = ANY($%d::int[])", len(args)+1)
		args = append(args, pq.Array(categoryIDs))
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.logger.Error("failed to query expense transactions for budget",
			"userID", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var result []types.BudgetTransactionRow
	for rows.Next() {
		var row types.BudgetTransactionRow
		if err := rows.Scan(&row.Amount, &row.CurrencyCode, &row.DateTime); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

// CountActiveByUserID returns the number of active (non-deleted, non-archived)
// budgets belonging to the given user.
func (r *BudgetRepositoryImpl) CountActiveByUserID(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM budgets WHERE user_id = $1 AND is_deleted = false AND is_archived = false`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}
