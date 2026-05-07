# User Categories

**Table:** `user_categories`
**Model:** `models.UserCategory` (`backend/internal/models/category.go`)

Per-user categories for classifying transactions. Initialized from [default_categories](default-categories.md) on registration.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `name` | `string` | `TEXT` | No | Category name |
| `parent_id` | `*int` | `INT` | Yes | FK to `user_categories.id` (self-referencing tree) |
| `is_income` | `bool` | `BOOLEAN` | No | `true` = income category, `false` = expense category |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Self-reference (parent) | user_categories | `parent_id` | Many-to-One (nullable) |
| Self-reference (children) | user_categories | `parent_id` | One-to-Many |
| Has many | [transactions](transactions.md) | `transactions.category_id` | One-to-Many |
| Has many | [transaction_templates](transaction-templates.md) | `transaction_templates.category_id` | One-to-Many |

## Notes

- Forms a tree structure via `parent_id` self-reference (parent/child categories).
- The `Parent` and `Children` relations are populated in Go code via joins, not by the DB directly.
