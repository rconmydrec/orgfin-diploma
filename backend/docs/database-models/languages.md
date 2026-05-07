# Languages

**Table:** `languages`
**Model:** `models.Language` (`backend/internal/models/language.go`)

Reference table of supported UI languages.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `code` | `string` | `TEXT` | No | ISO 639-1 language code (e.g. `en`, `ru`, `uk`, `bg`) |
| `name` | `string` | `TEXT` | No | Language name |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

No foreign key relationships. This is a reference/lookup table.
