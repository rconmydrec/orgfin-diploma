package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type AccountTypeRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewAccountTypeRepository(db *sql.DB, logger *slog.Logger) *AccountTypeRepositoryImpl {
	return &AccountTypeRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *AccountTypeRepositoryImpl) GetAll() ([]*models.AccountType, error) {
	query := `
		SELECT id, type_name, is_credit, is_deleted, created_at, updated_at
		FROM account_types
		WHERE is_deleted = false
		ORDER BY id
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []*models.AccountType
	for rows.Next() {
		t := &models.AccountType{}
		if err := rows.Scan(&t.ID, &t.TypeName, &t.IsCredit, &t.IsDeleted, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		types = append(types, t)
	}

	return types, rows.Err()
}

func (r *AccountTypeRepositoryImpl) GetByID(id int) (*models.AccountType, error) {
	query := `
		SELECT id, type_name, is_credit, is_deleted, created_at, updated_at
		FROM account_types
		WHERE id = $1
	`

	t := &models.AccountType{}
	err := r.db.QueryRow(query, id).Scan(&t.ID, &t.TypeName, &t.IsCredit, &t.IsDeleted, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return t, nil
}
