# Exchange Rates

**Table:** `exchange_rates`
**Model:** `models.ExchangeRateHistory` (`backend/internal/models/currency.go`)

Stores historical exchange rates as a JSONB object. The base currency is always USD.

## Fields

| Column | Go Type | DB Type | Nullable | Description |
|--------|---------|---------|----------|-------------|
| `id` | `int` | `SERIAL PK` | No | Primary key |
| `rates` | `map[string]float64` | `JSONB` | No | Currency rates keyed by code (e.g. `{"EUR": 0.85, "BGN": 1.66}`) |
| `actual_date` | `time.Time` | `TIMESTAMPTZ` | No | Date the rates are valid for |
| `base_currency_code` | `string` | `TEXT` | No | Base currency (always `USD`) |
| `service_name` | `string` | `TEXT` | No | Data source service name |
| `is_deleted` | `bool` | `BOOLEAN` | No | Soft delete flag |
| `created_at` | `time.Time` | `TIMESTAMPTZ` | No | Creation timestamp |
| `updated_at` | `time.Time` | `TIMESTAMPTZ` | No | Last update timestamp |

## Relationships

No foreign key relationships. This is a reference/lookup table.
