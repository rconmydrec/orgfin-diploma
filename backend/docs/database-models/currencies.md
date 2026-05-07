# Currencies

**Table:** `currencies`
**Model:** `models.Currency` (`backend/internal/models/currency.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `code` | `string` | `TEXT` | No | ISO 4217 currency code (e.g. USD, EUR, BGN) |
| `name` | `string` | `TEXT` | No | Full currency name |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Has many | [users](users.md) | `users.base_currency_id` | One-to-Many |
| Has many | [accounts](accounts.md) | `accounts.currency_id` | One-to-Many |
| Has many | [budgets](budgets.md) | `budgets.currency_id` | One-to-Many |
| Has many | [planned_transactions](planned-transactions.md) | `planned_transactions.currency_id` | One-to-Many |
| Has many | [subscription_plans](subscription-plans.md) | `subscription_plans.currency_id` | One-to-Many |
