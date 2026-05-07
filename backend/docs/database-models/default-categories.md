# Default Categories

**Table:** `default_categories`
**Model:** `models.DefaultCategory` (`backend/internal/models/category.go`)

Template categories that are copied to new users upon registration.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `name` | `string` | `TEXT` | No | Category name |
| `parent_id` | `*int` | `INT` | Yes | FK to `default_categories.id` (self-referencing tree) |
| `is_income` | `bool` | `BOOLEAN` | No | `true` = income category, `false` = expense category |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Self-reference (parent) | default_categories | `parent_id` | Many-to-One (nullable) |
| Self-reference (children) | default_categories | `parent_id` | One-to-Many |

## Notes

- Forms a tree structure via `parent_id` self-reference.
- Used as a template; actual user categories are stored in [user_categories](user-categories.md).
