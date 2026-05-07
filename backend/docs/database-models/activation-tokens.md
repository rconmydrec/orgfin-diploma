# Activation Tokens

**Table:** `activation_tokens`
**Model:** `models.ActivationToken` (`backend/internal/models/activation_token.go`)

Tokens for email-based account activation. Expired tokens are cleaned up by a scheduled worker task.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `token` | `string` | `TEXT` | No | Activation token string |
| `expires_at` | `time.Time` | `TIMESTAMPTZ` | No | Token expiration time |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
