# Budgets

**Table:** `budgets`
**Model:** `models.Budget` (`backend/internal/models/budget.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `name` | `string` | `TEXT` | No | Budget name |
| `currency_id` | `int` | `INT` | No | FK to `currencies.id` |
| `target_amount` | `decimal.Decimal` | `NUMERIC` | No | Target spending limit |
| `collected_amount` | `decimal.Decimal` | `NUMERIC` | No | Amount spent so far |
| `period` | `string` | `TEXT` | No | Period type (stored as uppercase): `DAILY`, `WEEKLY`, `MONTHLY`, `YEARLY`, `CUSTOM` |
| `repeat` | `bool` | `BOOLEAN` | No | Whether the budget auto-repeats |
| `start_date` | `time.Time` | `TIMESTAMPTZ` | No | Budget period start |
| `end_date` | `time.Time` | `TIMESTAMPTZ` | No | Budget period end |
| `included_categories` | `string` | `TEXT` | No | Comma-separated category IDs to track |
| `is_archived` | `bool` | `BOOLEAN` | No | Archived flag |
| `comment` | `*string` | `TEXT` | Yes | Optional comment |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag (hidden from JSON) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [currencies](currencies.md) | `currency_id` | Many-to-One |

## Notes

- `included_categories` stores a comma-separated list of `user_categories.id` values (not a proper FK, handled in application logic).
- `collected_amount` is recalculated by a background worker task (`budget_update`).
- Expired repeating budgets are automatically archived and renewed by a scheduled task (`budgets`) and by the admin-only daily-processing HTTP endpoint.
- `period` values are normalized to uppercase by the budget service (`strings.ToUpper`) on both create and update. The daily processing logic compares against uppercase constants (`DAILY`, `WEEKLY`, `MONTHLY`, `YEARLY`).
