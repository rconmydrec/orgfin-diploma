# Transaction Templates

**Table:** `transaction_templates`
**Model:** `models.TransactionTemplate` (`backend/internal/models/transaction_template.go`)

Saved label+category pairs (or label+target account for transfers) for quick transaction creation.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `category_id` | `*int` | `INT` | Yes | FK to `user_categories.id` |
| `target_account_id` | `*int` | `INT` | Yes | FK to `accounts.id` (ON DELETE SET NULL). Set for transfer templates. |
| `label` | `string` | `TEXT` | No | Template label |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [user_categories](user-categories.md) | `category_id` | Many-to-One (nullable) |
| Belongs to | [accounts](accounts.md) | `target_account_id` | Many-to-One (nullable, ON DELETE SET NULL) |

## Template Types

- **Regular template**: Has `category_id` set, `target_account_id` is NULL. Used for income/expense transactions.
- **Transfer template**: Has `target_account_id` set. Used for transfer transactions. `category_id` may or may not be set.
