package categories

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

// Service encapsulates all categories business logic.
type Service struct {
	categoryRepo CategoryRepository
	logger       *slog.Logger
}

// New creates a new categories Service with the given dependencies.
func New(
	categoryRepo CategoryRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		categoryRepo: categoryRepo,
		logger:       logger,
	}
}

// GetFlat returns a flat list of categories with formatted display names
// including hierarchy prefix and income/expense indicator.
func (s *Service) GetFlat(userID int) ([]FlatCategory, error) {
	categories, err := s.categoryRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	var result []FlatCategory

	var addChildren func(cat *models.UserCategory, parentName string)
	addChildren = func(cat *models.UserCategory, parentName string) {
		displayName := cat.Name
		if parentName != "" {
			displayName = parentName + " >> " + cat.Name
		}

		prefix := "-"
		if cat.IsIncome {
			prefix = "+"
		}
		formattedName := "(" + prefix + ") " + displayName

		fc := FlatCategory{
			ID:        cat.ID,
			UserID:    cat.UserID,
			Name:      formattedName,
			ParentID:  cat.ParentID,
			IsIncome:  cat.IsIncome,
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
		}
		result = append(result, fc)

		// Find and sort children by name
		var children []*models.UserCategory
		for _, c := range categories {
			if c.ParentID != nil && *c.ParentID == cat.ID {
				children = append(children, c)
			}
		}
		for i := 0; i < len(children); i++ {
			for j := i + 1; j < len(children); j++ {
				if children[i].Name > children[j].Name {
					children[i], children[j] = children[j], children[i]
				}
			}
		}
		for _, child := range children {
			addChildren(child, displayName)
		}
	}

	// Process root categories (no parent)
	for _, cat := range categories {
		if cat.ParentID == nil {
			addChildren(cat, "")
		}
	}

	return result, nil
}

// GetGrouped returns categories split into income and expenses lists as tree structures.
// It fetches a flat list from the repository and builds the parent-child hierarchy.
func (s *Service) GetGrouped(userID int) ([]*Category, []*Category, error) {
	categories, err := s.categoryRepo.GetByUserIDGrouped(userID)
	if err != nil {
		return nil, nil, err
	}

	// Build tree structure from flat list
	categoryMap := make(map[int]*models.UserCategory)
	for _, c := range categories {
		c.Children = []*models.UserCategory{}
		categoryMap[c.ID] = c
	}

	var roots []*models.UserCategory
	for _, c := range categories {
		if c.ParentID != nil {
			if parent, ok := categoryMap[*c.ParentID]; ok {
				parent.Children = append(parent.Children, c)
			}
		} else {
			roots = append(roots, c)
		}
	}

	// Split into income and expenses
	income := make([]*Category, 0)
	expenses := make([]*Category, 0)
	for _, cat := range roots {
		if cat.IsIncome {
			income = append(income, toCategory(cat))
		} else {
			expenses = append(expenses, toCategory(cat))
		}
	}

	return income, expenses, nil
}

// Create creates a new category for the given user.
func (s *Service) Create(userID int, params CreateParams) (*Category, error) {
	category := &models.UserCategory{
		UserID:   userID,
		Name:     params.Name,
		ParentID: params.ParentID,
		IsIncome: params.IsIncome,
	}

	created, err := s.categoryRepo.Create(category)
	if err != nil {
		return nil, err
	}

	return toCategory(created), nil
}

// Update modifies an existing category, verifying ownership.
func (s *Service) Update(userID, id int, params UpdateParams) (*Category, error) {
	existing, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrAccessDenied
	}

	existing.Name = params.Name
	existing.ParentID = params.ParentID
	existing.IsIncome = params.IsIncome

	if err := s.categoryRepo.Update(existing); err != nil {
		return nil, err
	}

	return toCategory(existing), nil
}

// Delete removes a category, verifying ownership. Returns the deleted entity.
func (s *Service) Delete(userID, id int) (*Category, error) {
	existing, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrAccessDenied
	}

	if err := s.categoryRepo.Delete(id); err != nil {
		return nil, err
	}

	return toCategory(existing), nil
}

// toCategory converts a models.UserCategory to the service domain type,
// recursively converting children.
func toCategory(m *models.UserCategory) *Category {
	if m == nil {
		return nil
	}
	cat := &Category{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		ParentID:  m.ParentID,
		IsIncome:  m.IsIncome,
		IsDeleted: m.IsDeleted,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.Parent != nil {
		cat.Parent = toCategory(m.Parent)
	}
	if m.Children != nil {
		cat.Children = make([]*Category, len(m.Children))
		for i, child := range m.Children {
			cat.Children[i] = toCategory(child)
		}
	}
	return cat
}
