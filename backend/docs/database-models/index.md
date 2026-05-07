# Database Models

Documentation for all database tables used in the Budget Tracker backend.

**Source code:** `backend/internal/models/`

## Models

### Core

| Table | Model | Description |
|-------|-------|-------------|
| [users](users.md) | `User` | Application users |
| [accounts](accounts.md) | `Account` | Financial accounts (bank, cash, credit card, etc.) |
| [account_types](account-types.md) | `AccountType` | Account type definitions |
| [transactions](transactions.md) | `Transaction` | Financial transactions (income, expense, transfer) |
| [user_categories](user-categories.md) | `UserCategory` | Per-user transaction categories (tree structure) |
| [default_categories](default-categories.md) | `DefaultCategory` | Template categories for new users |
| [transaction_templates](transaction-templates.md) | `TransactionTemplate` | Saved label+category pairs for quick entry |

### Planning & Budgets

| Table | Model | Description |
|-------|-------|-------------|
| [budgets](budgets.md) | `Budget` | Spending budgets with category tracking |
| [planned_transactions](planned-transactions.md) | `PlannedTransaction` | Scheduled/recurring planned transactions |

### Currency

| Table | Model | Description |
|-------|-------|-------------|
| [currencies](currencies.md) | `Currency` | Currency definitions (ISO 4217) |
| [exchange_rates](exchange-rates.md) | `ExchangeRateHistory` | Historical exchange rates (JSONB, base = USD) |

### User Management

| Table | Model | Description |
|-------|-------|-------------|
| [user_settings](user-settings.md) | `UserSettings` | Per-user preferences (JSONB) |
| [activation_tokens](activation-tokens.md) | `ActivationToken` | Email activation tokens |
| [languages](languages.md) | `Language` | Supported UI languages |

### Subscriptions & Billing

| Table | Model | Description |
|-------|-------|-------------|
| [subscriptions](subscriptions.md) | `Subscription` | User subscription state |
| [subscription_plans](subscription-plans.md) | `SubscriptionPlan` | Available plans (free, trial, premium) |
| [billing_periods](billing-periods.md) | `BillingPeriod` | Billing intervals (monthly, yearly) |
| [plan_prices](plan-prices.md) | `PlanPrice` | Maps plans to external payment provider price IDs |
| [payment_provider_subscriptions](payment-provider-subscriptions.md) | `PaymentProviderSubscription` | Stripe integration details |

## Entity Relationship Diagram

```
                          currencies
                         +---------+
                         | id (PK) |
                         | code    |
                         | name    |
                         +----+----+
              +----------++---+----------++-------------+
              |           |   |           |              |
              v           v   v           v              v
           users      accounts  budgets  planned_     subscription_
         +-------+   +--------+         transactions    plans
         |id (PK)|   |id (PK) |                       +------+
         |email  |   |user_id +--+                     |      |
         +---+---+   |acct_   |  |                     v      |
     +---+---+---+---+type_id |  |             plan_prices    |
     |   |   |   |   |curr_id |  |             +----------+   |
     |   |   |   |   +---+----+  |             |plan_id   +---+
     |   |   |   |       |       |             |provider  |
     |   |   |   |       v       |             |price_id  |
     |   |   |   |  transactions |             +----------+
     |   |   |   |  +----------+ |
     |   |   |   |  |id (PK)   | |
     |   |   |   |  |user_id   +-+
     |   |   |   |  |acct_id   |
     |   |   |   |  |cat_id    +-------> user_categories
     |   |   |   |  |linked_   |        +--------------+
     |   |   |   |  | txn_id---|(self)  |id (PK)       |
     |   |   |   |  +----------+        |user_id       |
     |   |   |   |                      |parent_id ----|(self)
     |   |   |   |                      +--------------+
     |   |   |   |
     |   |   |   +---> user_settings
     |   |   |   +---> activation_tokens
     |   |   |   +---> transaction_templates
     |   |   |   +---> subscriptions ---> subscription_plans ---> billing_periods
     |   |   |              |
     |   |   |              +---> payment_provider_subscriptions
     |   |   |
     |   |   +---> budgets
     |   +---> planned_transactions
     +---> accounts

  account_types ---> accounts.account_type_id
  default_categories (self-ref tree, template for user_categories)
  languages (reference table, no FK)
  exchange_rates (reference table, no FK)
```

## Common Patterns

- **Soft delete**: Most tables use `is_deleted` flag instead of physical deletion
- **Timestamps**: All tables have `created_at` / `updated_at`
- **Monetary amounts**: Use `decimal.Decimal` (`shopspring/decimal`) for precision
- **Self-referencing FK**: Categories (`parent_id` tree) and transactions (`linked_transaction_id` for transfers)
- **JSONB columns**: `exchange_rates.rates`, `user_settings.settings`, `payment_provider_subscriptions.provider_metadata`
- **Transfers**: Implemented as a pair of transactions linked via `linked_transaction_id`
- **Payment provider mapping**: `plan_prices` maps internal plans to external provider price IDs; `payment_provider_subscriptions` maps internal subscriptions to external provider customer/subscription/schedule IDs
