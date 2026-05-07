# Payment Provider Subscriptions

**Table:** `payment_provider_subscriptions`
**Model:** `models.PaymentProviderSubscription` (`backend/internal/models/subscription.go`)

Stores external payment provider (Stripe) integration details for a subscription.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `subscription_id` | `int` | `INT` | No | FK to `subscriptions.id` |
| `provider_type` | `string` | `TEXT` | No | Payment provider (e.g. `stripe`) |
| `external_customer_id` | `*string` | `TEXT` | Yes | Provider's customer ID |
| `external_subscription_id` | `*string` | `TEXT` | Yes | Provider's subscription ID |
| `external_schedule_id` | `*string` | `TEXT` | Yes | Provider's schedule ID |
| `payment_method_id` | `*string` | `TEXT` | Yes | Provider's payment method ID |
| `last_payment_failed` | `bool` | `BOOLEAN` | No | Whether the last payment attempt failed |
| `provider_metadata` | `map[string]interface{}` | `JSONB` | No | Additional provider-specific metadata |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [subscriptions](subscriptions.md) | `subscription_id` | One-to-One |
