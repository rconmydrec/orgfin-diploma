package categories

import (
	categoriesservice "github.com/go-budget/backend/internal/services/categories"
)

// CategoriesService defines the interface for category service operations
// needed by this handler.
type CategoriesService interface {
	GetFlat(userID int) ([]categoriesservice.FlatCategory, error)
	GetGrouped(userID int) ([]*categoriesservice.Category, []*categoriesservice.Category, error)
	Create(userID int, params categoriesservice.CreateParams) (*categoriesservice.Category, error)
	Update(userID, id int, params categoriesservice.UpdateParams) (*categoriesservice.Category, error)
	Delete(userID, id int) (*categoriesservice.Category, error)
}
