# Planned Transactions

**Table:** `planned_transactions`
**Model:** `models.PlannedTransaction` (`backend/internal/models/planned_transaction.go`)

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `user_id` | `int` | `INT` | No | FK to `users.id` |
| `currency_id` | `int` | `INT` | No | FK to `currencies.id` |
| `amount` | `decimal.Decimal` | `NUMERIC` | No | Planned amount |
| `label` | `string` | `TEXT` | No | Transaction label |
| `notes` | `*string` | `TEXT` | Yes | Additional notes |
| `is_income` | `bool` | `BOOLEAN` | No | `true` = income, `false` = expense |
| `planned_date` | `time.Time` | `TIMESTAMPTZ` | No | Planned execution date |
| `is_recurring` | `bool` | `BOOLEAN` | No | Whether this is a recurring transaction |
| `recurrence_rule` | `*string` | `TEXT` | Yes | Recurrence rule as JSON string |
| `is_executed` | `bool` | `BOOLEAN` | No | Whether this has been executed |
| `executed_transaction_id` | `*int` | `INT` | Yes | FK to `transactions.id` (the actual transaction created) |
| `execution_date` | `*time.Time` | `TIMESTAMPTZ` | Yes | Actual execution date |
| `is_active` | `bool` | `BOOLEAN` | No | Active flag |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag (hidden from JSON) |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## RecurrenceRule JSON Structure

When `is_recurring = true`, the `recurrence_rule` field contains a JSON object:

```json
{
  "frequency": "monthly",
  "interval": 1,
  "end_date": "2026-12-31",
  "count": 12,
  "day_of_month": 15
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `frequency` | `string` | Yes | `daily`, `weekly`, `monthly`, `yearly` |
| `interval` | `int` | Yes | Repeat every N periods |
| `end_date` | `string` | No | End date for recurrence |
| `count` | `int` | No | Max number of occurrences |
| `day_of_month` | `int` | No | Specific day of month |

## Relationships

| Direction | Related Table | FK Column | Type |
|-----------|---------------|-----------|------|
| Belongs to | [users](users.md) | `user_id` | Many-to-One |
| Belongs to | [currencies](currencies.md) | `currency_id` | Many-to-One |
| Belongs to | [transactions](transactions.md) | `executed_transaction_id` | Many-to-One (nullable) |
