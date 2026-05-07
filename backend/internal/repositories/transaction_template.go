package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type TransactionTemplateRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTransactionTemplateRepository(db *sql.DB, logger *slog.Logger) *TransactionTemplateRepositoryImpl {
	return &TransactionTemplateRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *TransactionTemplateRepositoryImpl) Create(template *models.TransactionTemplate) error {
	query := `
		INSERT INTO transaction_templates (user_id, category_id, target_account_id, label)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, template.UserID, template.CategoryID, template.TargetAccountID, template.Label).
		Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create transaction template", "error", err)
		return err
	}

	return nil
}

func (r *TransactionTemplateRepositoryImpl) GetByID(id int) (*models.TransactionTemplate, error) {
	query := `
		SELECT t.id, t.user_id, t.category_id, t.target_account_id, t.label, t.created_at, t.updated_at,
		       c.id, c.user_id, c.name, c.parent_id, c.is_income, c.created_at, c.updated_at,
		       a.id, a.name
		FROM transaction_templates t
		LEFT JOIN user_categories c ON t.category_id = c.id
		LEFT JOIN accounts a ON t.target_account_id = a.id
		WHERE t.id = $1
	`

	template := &models.TransactionTemplate{}
	var category models.UserCategory
	var catID, catUserID sql.NullInt64
	var catName sql.NullString
	var catParentID sql.NullInt64
	var catIsIncome sql.NullBool
	var catCreatedAt, catUpdatedAt sql.NullTime
	var accID sql.NullInt64
	var accName sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&template.ID, &template.UserID, &template.CategoryID, &template.TargetAccountID,
		&template.Label, &template.CreatedAt, &template.UpdatedAt,
		&catID, &catUserID, &catName, &catParentID, &catIsIncome, &catCreatedAt, &catUpdatedAt,
		&accID, &accName,
	)
	if err != nil {
		return nil, err
	}

	if catID.Valid {
		category.ID = int(catID.Int64)
		category.UserID = int(catUserID.Int64)
		category.Name = catName.String
		if catParentID.Valid {
			parentID := int(catParentID.Int64)
			category.ParentID = &parentID
		}
		category.IsIncome = catIsIncome.Bool
		category.CreatedAt = catCreatedAt.Time
		category.UpdatedAt = catUpdatedAt.Time
		template.Category = &category
	}

	if accID.Valid {
		template.TargetAccount = &models.Account{
			ID:   int(accID.Int64),
			Name: accName.String,
		}
	}

	return template, nil
}

func (r *TransactionTemplateRepositoryImpl) GetByUserID(userID int) ([]*models.TransactionTemplate, error) {
	query := `
		SELECT t.id, t.user_id, t.category_id, t.target_account_id, t.label, t.created_at, t.updated_at,
		       c.id, c.user_id, c.name, c.parent_id, c.is_income, c.created_at, c.updated_at,
		       a.id, a.name
		FROM transaction_templates t
		LEFT JOIN user_categories c ON t.category_id = c.id
		LEFT JOIN accounts a ON t.target_account_id = a.id
		WHERE t.user_id = $1
		ORDER BY t.label ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.TransactionTemplate
	for rows.Next() {
		template := &models.TransactionTemplate{}
		var category models.UserCategory
		var catID, catUserID sql.NullInt64
		var catName sql.NullString
		var catParentID sql.NullInt64
		var catIsIncome sql.NullBool
		var catCreatedAt, catUpdatedAt sql.NullTime
		var accID sql.NullInt64
		var accName sql.NullString

		err := rows.Scan(
			&template.ID, &template.UserID, &template.CategoryID, &template.TargetAccountID,
			&template.Label, &template.CreatedAt, &template.UpdatedAt,
			&catID, &catUserID, &catName, &catParentID, &catIsIncome, &catCreatedAt, &catUpdatedAt,
			&accID, &accName,
		)
		if err != nil {
			return nil, err
		}

		if catID.Valid {
			category.ID = int(catID.Int64)
			category.UserID = int(catUserID.Int64)
			category.Name = catName.String
			if catParentID.Valid {
				parentID := int(catParentID.Int64)
				category.ParentID = &parentID
			}
			category.IsIncome = catIsIncome.Bool
			category.CreatedAt = catCreatedAt.Time
			category.UpdatedAt = catUpdatedAt.Time
			template.Category = &category
		}

		if accID.Valid {
			template.TargetAccount = &models.Account{
				ID:   int(accID.Int64),
				Name: accName.String,
			}
		}

		templates = append(templates, template)
	}

	return templates, rows.Err()
}

func (r *TransactionTemplateRepositoryImpl) Update(template *models.TransactionTemplate) error {
	query := `
		UPDATE transaction_templates
		SET label = $1, category_id = $2, target_account_id = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(query, template.Label, template.CategoryID, template.TargetAccountID, template.ID)
	return err
}

func (r *TransactionTemplateRepositoryImpl) Delete(id int) error {
	query := `DELETE FROM transaction_templates WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
