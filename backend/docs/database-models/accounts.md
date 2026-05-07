# Accounts

**Table:** `accounts`
**Model:** `models.Account` (`backend/internal/models/account.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `name` | `string` | `TEXT` | No | Account name |
| `account_type_id` | `int` | `INT` | No | FK to `account_types.id` |
| `currency_id` | `int` | `INT` | No | FK to `currencies.id` |
| `balance` | `decimal.Decimal` | `NUMERIC` | No | Current balance |
| `initial_balance` | `decimal.Decimal` | `NUMERIC` | No | Initial balance at account creation |
| `comment` | `*string` | `TEXT` | Yes | Optional comment |
| `show_in_reports` | `bool` | `BOOLEAN` | No | Include in reports |
| `is_archived` | `bool` | `BOOLEAN` | No | Archived flag |
| `archived_at` | `*time.Time` | `TIMESTAMPTZ` | Yes | Archive timestamp |
| `opening_date` | `*time.Time` | `TIMESTAMPTZ` | Yes | Account opening date |
| `is_hidden` | `bool` | `BOOLEAN` | No | Hidden from UI |
| `credit_limit` | `*decimal.Decimal` | `NUMERIC` | Yes | Credit limit (for credit accounts) |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Computed Fields (not stored in DB)

| Field | Go Type | Description |
|-------|---------|-------------|
| `balance_in_base_currency` | `*decimal.Decimal` | Balance converted to user's base currency |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [account_types](account-types.md) | `account_type_id` | Many-to-One |
| Belongs to | [currencies](currencies.md) | `currency_id` | Many-to-One |
| Has many | [transactions](transactions.md) | `transactions.account_id` | One-to-Many |
