package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

type PlannedTransactionRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewPlannedTransactionRepository(db *sql.DB, logger *slog.Logger) *PlannedTransactionRepositoryImpl {
	return &PlannedTransactionRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *PlannedTransactionRepositoryImpl) Create(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
	query := `
		INSERT INTO planned_transactions (user_id, currency_id, amount, label, notes,
		                                  is_income, planned_date, is_recurring, recurrence_rule,
		                                  is_executed, is_active, is_deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, true, false)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		tx.UserID, tx.CurrencyID, tx.Amount, tx.Label, tx.Notes,
		tx.IsIncome, tx.PlannedDate, tx.IsRecurring, tx.RecurrenceRule,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create planned transaction", "error", err)
		return nil, err
	}

	return tx, nil
}

func (r *PlannedTransactionRepositoryImpl) GetByID(id int) (*models.PlannedTransaction, error) {
	query := `
		SELECT id, user_id, currency_id, amount, label, notes,
		       is_income, planned_date, is_recurring, recurrence_rule,
		       is_executed, executed_transaction_id, execution_date,
		       is_active, is_deleted, created_at, updated_at
		FROM planned_transactions
		WHERE id = $1 AND is_deleted = false
	`

	tx := &models.PlannedTransaction{}
	err := r.db.QueryRow(query, id).Scan(
		&tx.ID, &tx.UserID, &tx.CurrencyID, &tx.Amount, &tx.Label, &tx.Notes,
		&tx.IsIncome, &tx.PlannedDate, &tx.IsRecurring, &tx.RecurrenceRule,
		&tx.IsExecuted, &tx.ExecutedTransactionID, &tx.ExecutionDate,
		&tx.IsActive, &tx.IsDeleted, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (r *PlannedTransactionRepositoryImpl) GetByUserID(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error) {
	query := `
		SELECT id, user_id, currency_id, amount, label, notes,
		       is_income, planned_date, is_recurring, recurrence_rule,
		       is_executed, executed_transaction_id, execution_date,
		       is_active, is_deleted, created_at, updated_at
		FROM planned_transactions
		WHERE user_id = $1 AND is_deleted = false
	`

	if !filters.IncludeInactive {
		query += " AND is_active = true"
	}

	if filters.IsRecurring == "true" {
		query += " AND is_recurring = true"
	} else if filters.IsRecurring == "false" {
		query += " AND is_recurring = false"
	}

	if filters.IsExecuted == "true" {
		query += " AND is_executed = true"
	} else if filters.IsExecuted == "false" {
		query += " AND is_executed = false"
	}

	query += " ORDER BY planned_date ASC"

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*models.PlannedTransaction
	for rows.Next() {
		tx := &models.PlannedTransaction{}
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.CurrencyID, &tx.Amount, &tx.Label, &tx.Notes,
			&tx.IsIncome, &tx.PlannedDate, &tx.IsRecurring, &tx.RecurrenceRule,
			&tx.IsExecuted, &tx.ExecutedTransactionID, &tx.ExecutionDate,
			&tx.IsActive, &tx.IsDeleted, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

func (r *PlannedTransactionRepositoryImpl) GetActiveByUserID(userID int, includeInactive bool) ([]*models.PlannedTransaction, error) {
	query := `
		SELECT id, user_id, currency_id, amount, label, notes,
		       is_income, planned_date, is_recurring, recurrence_rule,
		       is_executed, executed_transaction_id, execution_date,
		       is_active, is_deleted, created_at, updated_at
		FROM planned_transactions
		WHERE user_id = $1 AND is_deleted = false AND is_executed = false
	`

	if !includeInactive {
		query += " AND is_active = true"
	}

	query += " ORDER BY planned_date ASC"

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*models.PlannedTransaction
	for rows.Next() {
		tx := &models.PlannedTransaction{}
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.CurrencyID, &tx.Amount, &tx.Label, &tx.Notes,
			&tx.IsIncome, &tx.PlannedDate, &tx.IsRecurring, &tx.RecurrenceRule,
			&tx.IsExecuted, &tx.ExecutedTransactionID, &tx.ExecutionDate,
			&tx.IsActive, &tx.IsDeleted, &tx.CreatedAt, &tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

func (r *PlannedTransactionRepositoryImpl) Update(tx *models.PlannedTransaction) error {
	query := `
		UPDATE planned_transactions
		SET amount = $1, label = $2, notes = $3, is_income = $4,
		    planned_date = $5, is_recurring = $6, recurrence_rule = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9
	`

	_, err := r.db.Exec(query,
		tx.Amount, tx.Label, tx.Notes, tx.IsIncome,
		tx.PlannedDate, tx.IsRecurring, tx.RecurrenceRule, tx.IsActive, tx.ID)
	return err
}

func (r *PlannedTransactionRepositoryImpl) Delete(id int) error {
	query := `UPDATE planned_transactions SET is_deleted = true, is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
