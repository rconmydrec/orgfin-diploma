// Package categories provides categories CRUD with hierarchy building and
// ownership checks.
//
// Invariants:
//   - All mutations (Create, Update, Delete) verify ownership via UserID.
//   - GetFlat returns a flat list sorted alphabetically with hierarchy prefix notation.
//   - GetGrouped returns two lists (income, expenses) with tree structure.
package categories

import (
	"time"

	"github.com/go-budget/backend/internal/models"
)

// ServiceInterface defines the public API of the categories service.
type ServiceInterface interface {
	GetFlat(userID int) ([]FlatCategory, error)
	GetGrouped(userID int) ([]*Category, []*Category, error)
	Create(userID int, params CreateParams) (*Category, error)
	Update(userID, id int, params UpdateParams) (*Category, error)
	Delete(userID, id int) (*Category, error)
}

// Category represents a category in the categories service domain.
type Category struct {
	ID        int
	UserID    int
	Name      string
	ParentID  *int
	IsIncome  bool
	IsDeleted bool
	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	Parent   *Category
	Children []*Category
}

// CategoryRepository defines category data access methods needed by the service.
type CategoryRepository interface {
	GetByUserID(userID int) ([]*models.UserCategory, error)
	GetByUserIDGrouped(userID int) ([]*models.UserCategory, error)
	GetByID(id int) (*models.UserCategory, error)
	Create(category *models.UserCategory) (*models.UserCategory, error)
	Update(category *models.UserCategory) error
	Delete(id int) error
}

// CreateParams holds the domain parameters for creating a category.
type CreateParams struct {
	Name     string
	ParentID *int
	IsIncome bool
}

// UpdateParams holds the domain parameters for updating a category.
type UpdateParams struct {
	Name     string
	ParentID *int
	IsIncome bool
}

// FlatCategory represents a category with a formatted display name
// including hierarchy prefix and income/expense indicator.
type FlatCategory struct {
	ID        int
	UserID    int
	Name      string // Formatted name, e.g. "(-) Food >> Groceries"
	ParentID  *int
	IsIncome  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
