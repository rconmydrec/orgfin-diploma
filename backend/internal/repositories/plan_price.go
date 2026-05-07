package repositories

import (
	"database/sql"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

// PlanPriceRepositoryImpl provides database access for plan prices.
type PlanPriceRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPlanPriceRepository creates a new PlanPriceRepositoryImpl.
func NewPlanPriceRepository(db *sql.DB, logger *slog.Logger) *PlanPriceRepositoryImpl {
	return &PlanPriceRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// GetByPlanAndProvider returns the plan price for a given plan and provider combination.
func (r *PlanPriceRepositoryImpl) GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error) {
	query := `
		SELECT id, plan_id, provider_type, external_price_id, is_active, created_at, updated_at
		FROM plan_prices
		WHERE plan_id = $1 AND provider_type = $2 AND is_active = true
	`

	pp := &models.PlanPrice{}
	err := r.db.QueryRow(query, planID, providerType).Scan(
		&pp.ID, &pp.PlanID, &pp.ProviderType, &pp.ExternalPriceID,
		&pp.IsActive, &pp.CreatedAt, &pp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

// GetByExternalPriceID looks up a plan price by its external (e.g. Stripe) price ID.
func (r *PlanPriceRepositoryImpl) GetByExternalPriceID(externalPriceID string) (*models.PlanPrice, error) {
	query := `
		SELECT id, plan_id, provider_type, external_price_id, is_active, created_at, updated_at
		FROM plan_prices
		WHERE external_price_id = $1
	`

	pp := &models.PlanPrice{}
	err := r.db.QueryRow(query, externalPriceID).Scan(
		&pp.ID, &pp.PlanID, &pp.ProviderType, &pp.ExternalPriceID,
		&pp.IsActive, &pp.CreatedAt, &pp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

// GetActivePricesByProvider returns all active plan prices for the given provider type.
func (r *PlanPriceRepositoryImpl) GetActivePricesByProvider(providerType string) ([]*models.PlanPrice, error) {
	query := `
		SELECT id, plan_id, provider_type, external_price_id, is_active, created_at, updated_at
		FROM plan_prices
		WHERE provider_type = $1 AND is_active = true
		ORDER BY plan_id
	`

	rows, err := r.db.Query(query, providerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*models.PlanPrice
	for rows.Next() {
		pp := &models.PlanPrice{}
		if err := rows.Scan(
			&pp.ID, &pp.PlanID, &pp.ProviderType, &pp.ExternalPriceID,
			&pp.IsActive, &pp.CreatedAt, &pp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		prices = append(prices, pp)
	}

	return prices, rows.Err()
}
