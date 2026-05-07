# Subscription Plans

**Table:** `subscription_plans`
**Model:** `models.SubscriptionPlan` (`backend/internal/models/subscription.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `name` | `string` | `TEXT` | No | Plan name |
| `translation_key` | `*string` | `TEXT` | Yes | i18n translation key |
| `plan_type` | `string` | `TEXT` | No | Type: `free`, `trial`, `premium` |
| `billing_period_id` | `*int` | `INT` | Yes | FK to `billing_periods.id` |
| `currency_id` | `int` | `INT` | No | FK to `currencies.id` |
| `currency_code` | `string` | `TEXT` | No | Currency ISO code (denormalized) |
| `price` | `decimal.Decimal` | `NUMERIC` | No | Plan price |
| `is_active` | `bool` | `BOOLEAN` | No | Plan is available for purchase |
| `is_featured` | `bool` | `BOOLEAN` | No | Highlighted/recommended plan |
| `sort_order` | `int` | `INT` | No | Display order |
| `description` | `*string` | `TEXT` | Yes | Plan description |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [billing_periods](billing-periods.md) | `billing_period_id` | Many-to-One (nullable) |
| Belongs to | [currencies](currencies.md) | `currency_id` | Many-to-One |
| Has many | [subscriptions](subscriptions.md) | `subscriptions.plan_id` | One-to-Many |
