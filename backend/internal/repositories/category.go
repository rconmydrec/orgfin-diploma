package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type CategoryRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewCategoryRepository(db *sql.DB, logger *slog.Logger) *CategoryRepositoryImpl {
	return &CategoryRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *CategoryRepositoryImpl) GetByUserID(userID int) ([]*models.UserCategory, error) {
	query := `
		SELECT id, user_id, name, parent_id, is_income, is_deleted, created_at, updated_at
		FROM user_categories
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY is_income, name
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.UserCategory
	for rows.Next() {
		c := &models.UserCategory{}
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Name, &c.ParentID, &c.IsIncome,
			&c.IsDeleted, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, rows.Err()
}

func (r *CategoryRepositoryImpl) GetByUserIDGrouped(userID int) ([]*models.UserCategory, error) {
	return r.GetByUserID(userID)
}

func (r *CategoryRepositoryImpl) GetByID(id int) (*models.UserCategory, error) {
	query := `
		SELECT id, user_id, name, parent_id, is_income, is_deleted, created_at, updated_at
		FROM user_categories
		WHERE id = $1 AND is_deleted = false
	`

	c := &models.UserCategory{}
	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.UserID, &c.Name, &c.ParentID, &c.IsIncome,
		&c.IsDeleted, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *CategoryRepositoryImpl) Create(category *models.UserCategory) (*models.UserCategory, error) {
	query := `
		INSERT INTO user_categories (user_id, name, parent_id, is_income, is_deleted)
		VALUES ($1, $2, $3, $4, false)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		category.UserID, category.Name, category.ParentID, category.IsIncome,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return category, nil
}

func (r *CategoryRepositoryImpl) Update(category *models.UserCategory) error {
	query := `
		UPDATE user_categories
		SET name = $1, parent_id = $2, is_income = $3, updated_at = NOW()
		WHERE id = $4 AND is_deleted = false
	`

	_, err := r.db.Exec(query, category.Name, category.ParentID, category.IsIncome, category.ID)
	return err
}

func (r *CategoryRepositoryImpl) Delete(id int) error {
	query := `UPDATE user_categories SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *CategoryRepositoryImpl) CopyDefaultCategories(userID int) error {
	// First, copy root categories (no parent)
	query := `
		INSERT INTO user_categories (user_id, name, parent_id, is_income, is_deleted)
		SELECT $1, name, NULL, is_income, false
		FROM default_categories
		WHERE parent_id IS NULL AND is_deleted = false
	`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		r.logger.Error("failed to copy default categories", "error", err, "user_id", userID)
		return err
	}

	// For child categories, we need to map parent IDs
	// This is a simplified approach - in production you might need a more sophisticated mapping
	childQuery := `
		WITH parent_mapping AS (
			SELECT dc.id as default_id, uc.id as user_id
			FROM default_categories dc
			JOIN user_categories uc ON uc.name = dc.name AND uc.user_id = $1 AND uc.parent_id IS NULL
			WHERE dc.parent_id IS NULL AND dc.is_deleted = false
		)
		INSERT INTO user_categories (user_id, name, parent_id, is_income, is_deleted)
		SELECT $1, dc.name, pm.user_id, dc.is_income, false
		FROM default_categories dc
		JOIN parent_mapping pm ON dc.parent_id = pm.default_id
		WHERE dc.is_deleted = false
	`

	_, err = r.db.Exec(childQuery, userID)
	if err != nil {
		r.logger.Error("failed to copy child categories", "error", err, "user_id", userID)
	}

	return err
}
