# Users

**Table:** `users`
**Model:** `models.User` (`backend/internal/models/user.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `email` | `string` | `TEXT` | No | Unique email address |
| `password_hash` | `string` | `TEXT` | No | Bcrypt password hash (hidden from JSON) |
| `is_active` | `bool` | `BOOLEAN` | No | Whether the account is activated |
| `first_name` | `*string` | `TEXT` | Yes | First name |
| `last_name` | `*string` | `TEXT` | Yes | Last name |
| `base_currency_id` | `int` | `INT` | No | FK to `currencies.id` |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag (hidden from JSON) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [currencies](currencies.md) | `base_currency_id` | Many-to-One |
| Has many | [accounts](accounts.md) | `accounts.user_id` | One-to-Many |
| Has many | [transactions](transactions.md) | `transactions.user_id` | One-to-Many |
| Has many | [user_categories](user-categories.md) | `user_categories.user_id` | One-to-Many |
| Has many | [budgets](budgets.md) | `budgets.user_id` | One-to-Many |
| Has many | [planned_transactions](planned-transactions.md) | `planned_transactions.user_id` | One-to-Many |
| Has many | [transaction_templates](transaction-templates.md) | `transaction_templates.user_id` | One-to-Many |
| Has one | [user_settings](user-settings.md) | `user_settings.user_id` | One-to-One |
| Has many | [activation_tokens](activation-tokens.md) | `activation_tokens.user_id` | One-to-Many |
| Has one | [subscriptions](subscriptions.md) | `subscriptions.user_id` | One-to-One |

## Methods

- `SetPassword(password string) error` — hashes password with bcrypt and stores in `password_hash`
- `CheckPassword(password string) bool` — compares password against stored hash
