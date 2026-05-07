package repositories

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
)

type AccountRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewAccountRepository(db *sql.DB, logger *slog.Logger) *AccountRepositoryImpl {
	return &AccountRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *AccountRepositoryImpl) Create(account *models.Account) (*models.Account, error) {
	query := `
		INSERT INTO accounts (
			user_id, name, account_type_id, currency_id, balance, initial_balance,
			comment, show_in_reports, is_archived, opening_date, is_hidden, credit_limit, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		account.UserID,
		account.Name,
		account.AccountTypeID,
		account.CurrencyID,
		account.Balance,
		account.InitialBalance,
		account.Comment,
		account.ShowInReports,
		account.IsArchived,
		account.OpeningDate,
		account.IsHidden,
		account.CreditLimit,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create account", "error", err)
		return nil, err
	}

	return account, nil
}

func (r *AccountRepositoryImpl) GetByID(id int) (*models.Account, error) {
	query := `
		SELECT a.id, a.user_id, a.name, a.account_type_id, a.currency_id,
		       a.balance, a.initial_balance, a.comment, a.show_in_reports,
		       a.is_archived, a.archived_at, a.opening_date, a.is_hidden,
		       a.credit_limit, a.is_deleted, a.created_at, a.updated_at,
		       c.id, c.code, c.name,
		       at.id, at.type_name, at.is_credit
		FROM accounts a
		LEFT JOIN currencies c ON a.currency_id = c.id
		LEFT JOIN account_types at ON a.account_type_id = at.id
		WHERE a.id = $1 AND a.is_deleted = false
	`

	account := &models.Account{}
	currency := &models.Currency{}
	accountType := &models.AccountType{}

	err := r.db.QueryRow(query, id).Scan(
		&account.ID, &account.UserID, &account.Name, &account.AccountTypeID, &account.CurrencyID,
		&account.Balance, &account.InitialBalance, &account.Comment, &account.ShowInReports,
		&account.IsArchived, &account.ArchivedAt, &account.OpeningDate, &account.IsHidden,
		&account.CreditLimit, &account.IsDeleted, &account.CreatedAt, &account.UpdatedAt,
		&currency.ID, &currency.Code, &currency.Name,
		&accountType.ID, &accountType.TypeName, &accountType.IsCredit,
	)
	if err != nil {
		return nil, err
	}

	account.Currency = currency
	account.AccountType = accountType
	return account, nil
}

func (r *AccountRepositoryImpl) GetByUserID(userID int, filters types.AccountFilters) ([]*models.Account, error) {
	query := `
		SELECT a.id, a.user_id, a.name, a.account_type_id, a.currency_id,
		       a.balance, a.initial_balance, a.comment, a.show_in_reports,
		       a.is_archived, a.archived_at, a.opening_date, a.is_hidden,
		       a.credit_limit, a.is_deleted, a.created_at, a.updated_at,
		       c.id, c.code, c.name,
		       at.id, at.type_name, at.is_credit
		FROM accounts a
		LEFT JOIN currencies c ON a.currency_id = c.id
		LEFT JOIN account_types at ON a.account_type_id = at.id
		WHERE a.user_id = $1
	`

	if filters.ArchivedOnly {
		query += " AND a.is_archived = true AND a.is_deleted = false"
	} else {
		if !filters.IncludeDeleted {
			query += " AND a.is_deleted = false"
		}
		if !filters.IncludeArchived {
			query += " AND a.is_archived = false"
		}
		if !filters.IncludeHidden {
			query += " AND a.is_hidden = false"
		}
		if filters.OnlyShowInReports {
			query += " AND a.show_in_reports = true"
		}
	}

	query += " ORDER BY a.name ASC"

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		account := &models.Account{}
		currency := &models.Currency{}
		accountType := &models.AccountType{}

		err := rows.Scan(
			&account.ID, &account.UserID, &account.Name, &account.AccountTypeID, &account.CurrencyID,
			&account.Balance, &account.InitialBalance, &account.Comment, &account.ShowInReports,
			&account.IsArchived, &account.ArchivedAt, &account.OpeningDate, &account.IsHidden,
			&account.CreditLimit, &account.IsDeleted, &account.CreatedAt, &account.UpdatedAt,
			&currency.ID, &currency.Code, &currency.Name,
			&accountType.ID, &accountType.TypeName, &accountType.IsCredit,
		)
		if err != nil {
			return nil, err
		}

		account.Currency = currency
		account.AccountType = accountType
		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}

func (r *AccountRepositoryImpl) Update(account *models.Account) error {
	query := `
		UPDATE accounts
		SET name = $1, account_type_id = $2, currency_id = $3, initial_balance = $4,
		    comment = $5, show_in_reports = $6, opening_date = $7, is_hidden = $8,
		    credit_limit = $9, updated_at = NOW()
		WHERE id = $10
	`

	_, err := r.db.Exec(query,
		account.Name, account.AccountTypeID, account.CurrencyID, account.InitialBalance,
		account.Comment, account.ShowInReports, account.OpeningDate, account.IsHidden,
		account.CreditLimit, account.ID)
	return err
}

func (r *AccountRepositoryImpl) SoftDelete(id int) error {
	query := `UPDATE accounts SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *AccountRepositoryImpl) UpdateBalance(id int, balance decimal.Decimal) error {
	query := `UPDATE accounts SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, balance, id)
	return err
}

func (r *AccountRepositoryImpl) SetArchiveStatus(id int, isArchived bool) error {
	query := `UPDATE accounts SET is_archived = $1, archived_at = $2, updated_at = NOW() WHERE id = $3`

	var archivedAt any
	if isArchived {
		archivedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(query, isArchived, archivedAt, id)
	return err
}

// CountActiveByUserID returns the number of active (non-deleted, non-archived)
// accounts belonging to the given user.
func (r *AccountRepositoryImpl) CountActiveByUserID(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM accounts WHERE user_id = $1 AND is_deleted = false AND is_archived = false`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}
