# User Settings

**Table:** `user_settings`
**Model:** `models.UserSettings` (`backend/internal/models/user_settings.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` (unique) |
| `settings` | `SettingsData` | `JSONB` | No | User preferences as JSON |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Settings JSON Structure

```json
{
  "language": "en",
  "projectionEndDate": "2026-12-31",
  "projectionPeriod": "monthly"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `language` | `string` | No | UI language code |
| `projectionEndDate` | `string` | No | End date for financial projections |
| `projectionPeriod` | `string` | No | Period for financial projections |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | One-to-One |
