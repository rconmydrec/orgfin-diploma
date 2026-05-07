package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type LanguageRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewLanguageRepository(db *sql.DB, logger *slog.Logger) *LanguageRepositoryImpl {
	return &LanguageRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *LanguageRepositoryImpl) GetAll() ([]*models.Language, error) {
	query := `
		SELECT id, code, name, is_deleted, created_at, updated_at
		FROM languages
		WHERE is_deleted = false
		ORDER BY name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var languages []*models.Language
	for rows.Next() {
		lang := &models.Language{}
		if err := rows.Scan(&lang.ID, &lang.Code, &lang.Name, &lang.IsDeleted, &lang.CreatedAt, &lang.UpdatedAt); err != nil {
			return nil, err
		}
		languages = append(languages, lang)
	}

	return languages, rows.Err()
}

func (r *LanguageRepositoryImpl) GetByCode(code string) (*models.Language, error) {
	query := `
		SELECT id, code, name, is_deleted, created_at, updated_at
		FROM languages
		WHERE code = $1 AND is_deleted = false
	`

	lang := &models.Language{}
	err := r.db.QueryRow(query, code).Scan(&lang.ID, &lang.Code, &lang.Name, &lang.IsDeleted, &lang.CreatedAt, &lang.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return lang, nil
}
