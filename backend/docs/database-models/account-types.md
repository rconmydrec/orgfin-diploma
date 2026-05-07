# Account Types

**Table:** `account_types`
**Model:** `models.AccountType` (`backend/internal/models/account.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `type_name` | `string` | `TEXT` | No | Type name (e.g. cash, bank, credit card) |
| `is_credit` | `bool` | `BOOLEAN` | No | Whether accounts of this type are credit-based |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag (hidden from JSON) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Has many | [accounts](accounts.md) | `accounts.account_type_id` | One-to-Many |
