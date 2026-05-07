# Subscriptions

**Table:** `subscriptions`
**Model:** `models.Subscription` (`backend/internal/models/subscription.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `plan_id` | `int` | `INT` | No | FK to `subscription_plans.id` |
| `plan_type` | `string` | `TEXT` | No | Current plan type: `free`, `trial`, `premium` |
| `current_billing_period_id` | `*int` | `INT` | Yes | FK to `billing_periods.id` |
| `trial_started_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Trial start timestamp |
| `trial_ends_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Trial end timestamp |
| `subscribed_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Subscription start timestamp |
| `expires_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Subscription expiration timestamp |
| `auto_renew` | `bool` | `BOOLEAN` | No | Auto-renewal enabled |
| `canceled_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Cancellation timestamp |
| `pending_plan_id` | `*int` | `INT` | Yes | FK to `subscription_plans.id` (pending downgrade) |
| `pending_downgrade_account_ids` | `[]int` | `INT[]` / `JSONB` | Yes | Account IDs to keep on downgrade |
| `pending_downgrade_budget_id` | `*int` | `INT` | Yes | Budget ID to keep on downgrade |
| `is_active` | `bool` | `BOOLEAN` | No | Subscription is active |
| `has_stripe_subscription` | `bool` | `BOOLEAN` | No | Linked to Stripe subscription |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [subscription_plans](subscription-plans.md) | `plan_id` | Many-to-One |
| Belongs to | [subscription_plans](subscription-plans.md) | `pending_plan_id` | Many-to-One (nullable) |
| Belongs to | [billing_periods](billing-periods.md) | `current_billing_period_id` | Many-to-One (nullable) |
| Has one | [payment_provider_subscriptions](payment-provider-subscriptions.md) | `payment_provider_subscriptions.subscription_id` | One-to-One |
