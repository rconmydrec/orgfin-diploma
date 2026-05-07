# Billing Periods

**Table:** `billing_periods`
**Model:** `models.BillingPeriod` (`backend/internal/models/subscription.go`)

Reference table for subscription billing intervals.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `code` | `string` | `TEXT` | No | Period code (e.g. `monthly`, `yearly`) |
| `name` | `string` | `TEXT` | No | Display name |
| `duration_days` | `int` | `INT` | No | Duration in days |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Has many | [subscription_plans](subscription-plans.md) | `subscription_plans.billing_period_id` | One-to-Many |
| Has many | [subscriptions](subscriptions.md) | `subscriptions.current_billing_period_id` | One-to-Many |
