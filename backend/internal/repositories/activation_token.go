package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type ActivationTokenRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewActivationTokenRepository(db *sql.DB, logger *slog.Logger) *ActivationTokenRepositoryImpl {
	return &ActivationTokenRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *ActivationTokenRepositoryImpl) Create(token *models.ActivationToken) error {
	query := `
		INSERT INTO activation_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, token.UserID, token.Token, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create activation token", "error", err)
		return err
	}

	return nil
}

func (r *ActivationTokenRepositoryImpl) GetByToken(token string) (*models.ActivationToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, updated_at
		FROM activation_tokens
		WHERE token = $1
	`

	at := &models.ActivationToken{}
	err := r.db.QueryRow(query, token).Scan(
		&at.ID, &at.UserID, &at.Token, &at.ExpiresAt, &at.CreatedAt, &at.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return at, nil
}

func (r *ActivationTokenRepositoryImpl) Delete(id int) error {
	query := `DELETE FROM activation_tokens WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *ActivationTokenRepositoryImpl) DeleteExpired() error {
	query := `DELETE FROM activation_tokens WHERE expires_at < NOW()`
	_, err := r.db.Exec(query)
	return err
}
