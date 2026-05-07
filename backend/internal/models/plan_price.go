package models

import (
	"time"
)

// PlanPrice maps a subscription plan to an external payment provider's price ID.
// Each plan can have one price per provider (e.g., one Stripe price ID per plan).
type PlanPrice struct {
	ID              int       `json:"id" db:"id"`
	PlanID          int       `json:"plan_id" db:"plan_id"`
	ProviderType    string    `json:"provider_type" db:"provider_type"`
	ExternalPriceID string    `json:"external_price_id" db:"external_price_id"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
