package repositories

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

type TransactionRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTransactionRepository(db *sql.DB, logger *slog.Logger) *TransactionRepositoryImpl {
	return &TransactionRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *TransactionRepositoryImpl) Create(tx *models.Transaction) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (
			user_id, account_id, amount, new_balance, category_id, label,
			is_income, is_transfer, linked_transaction_id,
			notes, date_time, is_deleted, is_adjustment, exclude_from_reports
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, false, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		tx.UserID,
		tx.AccountID,
		tx.Amount,
		tx.NewBalance,
		tx.CategoryID,
		tx.Label,
		tx.IsIncome,
		tx.IsTransfer,
		tx.LinkedTransactionID,
		tx.Notes,
		tx.DateTime,
		tx.IsAdjustment,
		tx.ExcludeFromReports,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create transaction", "error", err)
		return nil, err
	}

	return tx, nil
}

func (r *TransactionRepositoryImpl) GetByID(id int) (*models.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.account_id, t.amount, t.new_balance, t.category_id, t.label,
		       t.is_income, t.is_transfer, t.linked_transaction_id,
		       t.notes, t.date_time, t.is_deleted, t.created_at, t.updated_at, t.is_adjustment,
		       t.exclude_from_reports,
		       a.id, a.user_id, a.name, a.account_type_id, a.currency_id, a.balance,
		       a.initial_balance, a.credit_limit, a.is_hidden, a.show_in_reports,
		       a.is_deleted, a.is_archived,
		       c.id, c.code, c.name,
		       at.id, at.type_name, at.is_credit,
		       cat.id, cat.user_id, cat.name, cat.parent_id, cat.is_income
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		JOIN currencies c ON a.currency_id = c.id
		JOIN account_types at ON a.account_type_id = at.id
		LEFT JOIN user_categories cat ON t.category_id = cat.id
		WHERE t.id = $1 AND t.is_deleted = false
	`

	tx := &models.Transaction{}
	account := &models.Account{}
	currency := &models.Currency{}
	accountType := &models.AccountType{}

	var catID, catUserID sql.NullInt64
	var catName sql.NullString
	var catParentID sql.NullInt64
	var catIsIncome sql.NullBool

	err := r.db.QueryRow(query, id).Scan(
		&tx.ID, &tx.UserID, &tx.AccountID, &tx.Amount, &tx.NewBalance, &tx.CategoryID,
		&tx.Label, &tx.IsIncome, &tx.IsTransfer, &tx.LinkedTransactionID,
		&tx.Notes, &tx.DateTime, &tx.IsDeleted,
		&tx.CreatedAt, &tx.UpdatedAt, &tx.IsAdjustment, &tx.ExcludeFromReports,
		&account.ID, &account.UserID, &account.Name, &account.AccountTypeID,
		&account.CurrencyID, &account.Balance, &account.InitialBalance,
		&account.CreditLimit, &account.IsHidden, &account.ShowInReports,
		&account.IsDeleted, &account.IsArchived,
		&currency.ID, &currency.Code, &currency.Name,
		&accountType.ID, &accountType.TypeName, &accountType.IsCredit,
		&catID, &catUserID, &catName, &catParentID, &catIsIncome,
	)
	if err != nil {
		return nil, err
	}

	account.Currency = currency
	account.AccountType = accountType
	tx.Account = account

	if catID.Valid {
		cat := &models.UserCategory{
			ID:       int(catID.Int64),
			UserID:   int(catUserID.Int64),
			Name:     catName.String,
			IsIncome: catIsIncome.Bool,
		}
		if catParentID.Valid {
			pid := int(catParentID.Int64)
			cat.ParentID = &pid
		}
		tx.Category = cat
	}

	return tx, nil
}

func (r *TransactionRepositoryImpl) GetByAccountID(accountID int, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, account_id, amount, new_balance, category_id, label,
		       is_income, is_transfer, linked_transaction_id,
		       notes, date_time, is_deleted, created_at, updated_at, is_adjustment,
		       exclude_from_reports
		FROM transactions
		WHERE account_id = $1 AND is_deleted = false
		ORDER BY date_time DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *TransactionRepositoryImpl) GetByUserID(userID int, filters types.TransactionFilters) ([]*models.Transaction, int, error) {
	// Build dynamic query with filters
	baseQuery := `
		SELECT t.id, t.user_id, t.account_id, t.amount, t.new_balance, t.category_id, t.label,
		       t.is_income, t.is_transfer, t.linked_transaction_id,
		       t.notes, t.date_time, t.is_deleted, t.created_at, t.updated_at, t.is_adjustment,
		       t.exclude_from_reports,
		       a.id, a.user_id, a.name, a.account_type_id, a.currency_id, a.balance,
		       a.initial_balance, a.credit_limit, a.is_hidden, a.show_in_reports,
		       a.is_deleted, a.is_archived,
		       c.id, c.code, c.name,
		       at.id, at.type_name, at.is_credit,
		       cat.id, cat.user_id, cat.name, cat.parent_id, cat.is_income
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		JOIN currencies c ON a.currency_id = c.id
		JOIN account_types at ON a.account_type_id = at.id
		LEFT JOIN user_categories cat ON t.category_id = cat.id
		WHERE t.user_id = $1 AND t.is_deleted = false AND a.is_deleted = false`

	countBase := `SELECT COUNT(*) FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.user_id = $1 AND t.is_deleted = false AND a.is_deleted = false`

	args := []interface{}{userID}
	countArgs := []interface{}{userID}
	argIdx := 2

	// Apply type filter
	if filters.Type != nil && *filters.Type != "" {
		switch *filters.Type {
		case "expense":
			baseQuery += " AND t.is_income = false AND t.is_transfer = false"
			countBase += " AND t.is_income = false AND t.is_transfer = false"
		case "income":
			baseQuery += " AND t.is_income = true AND t.is_transfer = false"
			countBase += " AND t.is_income = true AND t.is_transfer = false"
		case "transfer":
			baseQuery += " AND t.is_transfer = true"
			countBase += " AND t.is_transfer = true"
		}
	}

	// Apply account filter
	if len(filters.AccountIDs) > 0 {
		placeholders := ""
		for i, accID := range filters.AccountIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", argIdx)
			args = append(args, accID)
			countArgs = append(countArgs, accID)
			argIdx++
		}
		baseQuery += fmt.Sprintf(" AND t.account_id IN (%s)", placeholders)
		countBase += fmt.Sprintf(" AND t.account_id IN (%s)", placeholders)
	}

	// Apply category filter
	if len(filters.CategoryIDs) > 0 {
		placeholders := ""
		for i, catID := range filters.CategoryIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", argIdx)
			args = append(args, catID)
			countArgs = append(countArgs, catID)
			argIdx++
		}
		baseQuery += fmt.Sprintf(" AND t.category_id IN (%s)", placeholders)
		countBase += fmt.Sprintf(" AND t.category_id IN (%s)", placeholders)
	}

	// Apply date filters
	if filters.DateFrom != nil && *filters.DateFrom != "" {
		baseQuery += fmt.Sprintf(" AND t.date_time >= $%d", argIdx)
		countBase += fmt.Sprintf(" AND t.date_time >= $%d", argIdx)
		args = append(args, *filters.DateFrom)
		countArgs = append(countArgs, *filters.DateFrom)
		argIdx++
	}
	if filters.DateTo != nil && *filters.DateTo != "" {
		baseQuery += fmt.Sprintf(" AND t.date_time <= $%d", argIdx)
		countBase += fmt.Sprintf(" AND t.date_time <= $%d", argIdx)
		args = append(args, *filters.DateTo)
		countArgs = append(countArgs, *filters.DateTo)
		argIdx++
	}

	// Add ordering and pagination
	if filters.NoLimit {
		baseQuery += " ORDER BY t.date_time DESC"
	} else {
		limit := filters.Limit
		if limit == 0 {
			limit = 50
		}
		baseQuery += fmt.Sprintf(" ORDER BY t.date_time DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, limit, filters.Offset)
	}

	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	txs, err := r.scanTransactionsWithRelations(rows)
	if err != nil {
		return nil, 0, err
	}

	// Get total count with same filters
	var total int
	if err := r.db.QueryRow(countBase, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return txs, total, nil
}

func (r *TransactionRepositoryImpl) Update(tx *models.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id = $1, amount = $2, new_balance = $3, label = $4, notes = $5,
		    date_time = $6, is_transfer = $7, is_income = $8, is_adjustment = $9,
		    exclude_from_reports = $10,
		    linked_transaction_id = $11, account_id = $12, updated_at = NOW()
		WHERE id = $13
	`

	_, err := r.db.Exec(query,
		tx.CategoryID, tx.Amount, tx.NewBalance, tx.Label, tx.Notes,
		tx.DateTime, tx.IsTransfer, tx.IsIncome, tx.IsAdjustment,
		tx.ExcludeFromReports,
		tx.LinkedTransactionID, tx.AccountID, tx.ID)
	return err
}

func (r *TransactionRepositoryImpl) Delete(id int) error {
	query := `UPDATE transactions SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *TransactionRepositoryImpl) UpdateLinkedID(id int, linkedID int) error {
	query := `UPDATE transactions SET linked_transaction_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, linkedID, id)
	return err
}

func (r *TransactionRepositoryImpl) UpdateBalance(id int, newBalance decimal.Decimal) error {
	query := `UPDATE transactions SET new_balance = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, newBalance, id)
	return err
}

func (r *TransactionRepositoryImpl) GetByAccountIDForRecalc(accountID int) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, account_id, amount, new_balance, category_id, label,
		       is_income, is_transfer, linked_transaction_id,
		       notes, date_time, is_deleted, created_at, updated_at, is_adjustment,
		       exclude_from_reports
		FROM transactions
		WHERE account_id = $1 AND is_deleted = false
		ORDER BY date_time ASC, id ASC
	`

	rows, err := r.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *TransactionRepositoryImpl) GetForExport(userID int, startDate, endDate string) ([]*models.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.account_id, t.amount, t.new_balance, t.category_id, t.label,
		       t.is_income, t.is_transfer, t.linked_transaction_id,
		       t.notes, t.date_time, t.is_deleted, t.created_at, t.updated_at, t.is_adjustment,
		       t.exclude_from_reports,
		       a.id, a.user_id, a.name, a.account_type_id, a.currency_id, a.balance,
		       a.initial_balance, a.credit_limit, a.is_hidden, a.show_in_reports,
		       a.is_deleted, a.is_archived,
		       c.id, c.code, c.name,
		       at.id, at.type_name, at.is_credit,
		       cat.id, cat.user_id, cat.name, cat.parent_id, cat.is_income
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		JOIN currencies c ON a.currency_id = c.id
		JOIN account_types at ON a.account_type_id = at.id
		LEFT JOIN user_categories cat ON t.category_id = cat.id
		WHERE t.user_id = $1
		  AND t.is_deleted = false
		  AND a.is_deleted = false
		  AND t.is_transfer = false
		  AND t.is_adjustment = false
		  AND t.exclude_from_reports = false
		  AND t.date_time >= $2
		  AND t.date_time <= $3
		ORDER BY t.date_time ASC
		LIMIT 50000
	`

	rows, err := r.db.Query(query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactionsWithRelations(rows)
}

func (r *TransactionRepositoryImpl) scanTransactions(rows *sql.Rows) ([]*models.Transaction, error) {
	var txs []*models.Transaction
	for rows.Next() {
		tx := &models.Transaction{}
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.AccountID, &tx.Amount, &tx.NewBalance, &tx.CategoryID,
			&tx.Label, &tx.IsIncome, &tx.IsTransfer, &tx.LinkedTransactionID,
			&tx.Notes, &tx.DateTime, &tx.IsDeleted,
			&tx.CreatedAt, &tx.UpdatedAt, &tx.IsAdjustment, &tx.ExcludeFromReports,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *TransactionRepositoryImpl) scanTransactionsWithRelations(rows *sql.Rows) ([]*models.Transaction, error) {
	var txs []*models.Transaction
	for rows.Next() {
		tx := &models.Transaction{}
		account := &models.Account{}
		currency := &models.Currency{}
		accountType := &models.AccountType{}

		// Category fields (nullable)
		var catID, catUserID sql.NullInt64
		var catName sql.NullString
		var catParentID sql.NullInt64
		var catIsIncome sql.NullBool

		err := rows.Scan(
			// Transaction fields
			&tx.ID, &tx.UserID, &tx.AccountID, &tx.Amount, &tx.NewBalance, &tx.CategoryID,
			&tx.Label, &tx.IsIncome, &tx.IsTransfer, &tx.LinkedTransactionID,
			&tx.Notes, &tx.DateTime, &tx.IsDeleted,
			&tx.CreatedAt, &tx.UpdatedAt, &tx.IsAdjustment, &tx.ExcludeFromReports,
			// Account fields
			&account.ID, &account.UserID, &account.Name, &account.AccountTypeID,
			&account.CurrencyID, &account.Balance, &account.InitialBalance,
			&account.CreditLimit, &account.IsHidden, &account.ShowInReports,
			&account.IsDeleted, &account.IsArchived,
			// Currency fields
			&currency.ID, &currency.Code, &currency.Name,
			// AccountType fields
			&accountType.ID, &accountType.TypeName, &accountType.IsCredit,
			// Category fields (nullable)
			&catID, &catUserID, &catName, &catParentID, &catIsIncome,
		)
		if err != nil {
			return nil, err
		}

		account.Currency = currency
		account.AccountType = accountType
		tx.Account = account

		// Set category if present
		if catID.Valid {
			cat := &models.UserCategory{
				ID:       int(catID.Int64),
				UserID:   int(catUserID.Int64),
				Name:     catName.String,
				IsIncome: catIsIncome.Bool,
			}
			if catParentID.Valid {
				pid := int(catParentID.Int64)
				cat.ParentID = &pid
			}
			tx.Category = cat
		}

		txs = append(txs, tx)
	}
	return txs, rows.Err()
}
