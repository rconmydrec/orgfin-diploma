package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type CurrencyRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewCurrencyRepository(db *sql.DB, logger *slog.Logger) *CurrencyRepositoryImpl {
	return &CurrencyRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *CurrencyRepositoryImpl) GetAll() ([]*models.Currency, error) {
	query := `SELECT id, code, name, created_at, updated_at FROM currencies ORDER BY code`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var currencies []*models.Currency
	for rows.Next() {
		c := &models.Currency{}
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		currencies = append(currencies, c)
	}

	return currencies, rows.Err()
}

func (r *CurrencyRepositoryImpl) GetByCode(code string) (*models.Currency, error) {
	query := `SELECT id, code, name, created_at, updated_at FROM currencies WHERE code = $1`

	c := &models.Currency{}
	err := r.db.QueryRow(query, code).Scan(&c.ID, &c.Code, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *CurrencyRepositoryImpl) GetByID(id int) (*models.Currency, error) {
	query := `SELECT id, code, name, created_at, updated_at FROM currencies WHERE id = $1`

	c := &models.Currency{}
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.Code, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return c, nil
}
