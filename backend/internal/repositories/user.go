package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type UserRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewUserRepository(db *sql.DB, logger *slog.Logger) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepositoryImpl) Create(user *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, is_active, base_currency_id, is_deleted)
		VALUES ($1, $2, $3, $4, $5, $6, false)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.IsActive,
		user.BaseCurrencyID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (r *UserRepositoryImpl) GetByID(id int) (*models.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name,
		       u.is_active, u.base_currency_id, u.is_deleted, u.created_at, u.updated_at,
		       c.id, c.code, c.name
		FROM users u
		LEFT JOIN currencies c ON u.base_currency_id = c.id
		WHERE u.id = $1 AND u.is_deleted = false
	`

	user := &models.User{}
	currency := &models.Currency{}

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.BaseCurrencyID, &user.IsDeleted, &user.CreatedAt, &user.UpdatedAt,
		&currency.ID, &currency.Code, &currency.Name,
	)
	if err != nil {
		return nil, err
	}

	user.BaseCurrency = currency
	return user, nil
}

func (r *UserRepositoryImpl) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name,
		       u.is_active, u.base_currency_id, u.is_deleted, u.created_at, u.updated_at,
		       c.id, c.code, c.name
		FROM users u
		LEFT JOIN currencies c ON u.base_currency_id = c.id
		WHERE u.email = $1 AND u.is_deleted = false
	`

	user := &models.User{}
	currency := &models.Currency{}

	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.BaseCurrencyID, &user.IsDeleted, &user.CreatedAt, &user.UpdatedAt,
		&currency.ID, &currency.Code, &currency.Name,
	)
	if err != nil {
		return nil, err
	}

	user.BaseCurrency = currency
	return user, nil
}

func (r *UserRepositoryImpl) Update(user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, first_name = $2, last_name = $3, is_active = $4,
		    base_currency_id = $5, updated_at = NOW()
		WHERE id = $6 AND is_deleted = false
	`

	_, err := r.db.Exec(query, user.Email, user.FirstName, user.LastName,
		user.IsActive, user.BaseCurrencyID, user.ID)
	return err
}

func (r *UserRepositoryImpl) Activate(userID int) error {
	query := `UPDATE users SET is_active = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *UserRepositoryImpl) UpdatePassword(userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, passwordHash, userID)
	return err
}
