# Plan Prices

**Table:** `plan_prices`
**Model:** `models.PlanPrice` (`backend/internal/models/plan_price.go`)

Maps subscription plans to external payment provider price IDs. Each plan can have one price per provider (e.g., one Stripe price ID per plan). This table decouples the internal plan definition from provider-specific pricing configuration.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `plan_id` | `int` | `INT` | No | FK to `subscription_plans.id` |
| `provider_type` | `string` | `TEXT` | No | Payment provider (e.g. `STRIPE`) |
| `external_price_id` | `string` | `TEXT` | No | Provider's price ID (e.g. Stripe price_xxx) |
| `is_active` | `bool` | `BOOLEAN` | No | Whether this price mapping is active (default: true) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Constraints

| Type | Name | Columns | Description |
|------|------|---------|-------------|
| Unique | `uq_plan_prices_plan_id_provider_type` | `plan_id, provider_type` | Each plan can have only one price per provider |
| Index | `idx_plan_prices_provider_type` | `provider_type` | Fast lookup by provider type |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [subscription_plans](subscription-plans.md) | `plan_id` | Many-to-One |

## Repository

**Interface:** `PlanPriceRepository` (`backend/internal/repositories/interfaces.go`)

| Method | Description |
|--------|-------------|
| `GetByPlanAndProvider(planID int, providerType string)` | Get the price for a specific plan and provider |
| `GetByExternalPriceID(externalPriceID string)` | Find the plan price by the provider's price ID (used in webhook handling) |
| `GetActivePricesByProvider(providerType string)` | Get all active prices for a given provider |

## Migration

**File:** `backend/migrations/00001_create_plan_prices.sql`
