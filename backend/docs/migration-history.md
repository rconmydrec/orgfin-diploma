# Endpoint Migration Comparison: Python → Go

**Generated**: 2026-02-21
**Python project**: `/Users/Projects/budget-tracker/back-fastapi/`
**Go project**: `/Users/Projects/go-budget/backend/`

## Legend

- **Py Tests**: Total endpoint tests in Python
- **Go Unit**: Unit tests (mocked dependencies) in Go
- **Go Integ**: Integration tests (real DB) in Go
- **Go Total**: Unit + Integration
- **Ported OK**: Mark when verified that endpoint is ported correctly and without bugs

---

## AUTH

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 1 | POST | `/auth/register/` | `/auth/register/` | 6 | 3 | 10 | 13 | ✅ |
| 2 | POST | `/auth/login/` | `/auth/login/` | 7 | 3 | 9 | 12 | ✅ |
| 3 | GET | `/auth/profile/` | `/auth/profile/` | 5 | 3 | 4 | 7 | ✅ |
| 4 | GET | `/auth/activate/{token}` | `/auth/activate/:token` | 4 | 3 | 5 | 8 | ✅ |
| 5 | POST | `/auth/oauth/` | `/auth/oauth/` | 10 | 3 | 11 | 14 | ✅ |
| 6 | POST | `/auth/change-password/` | `/auth/change-password/` | 6 | 5 | 8 | 13 | ✅ |

**Auth totals**: Py 38 → Go 62

---

## ACCOUNTS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 7 | POST | `/accounts/` | `/accounts/` | 8 | 2 | 11 | 13 | ✅ |
| 8 | GET | `/accounts/` | `/accounts/` | 7 | 2 | 6 | 8 | ✅ |
| 9 | GET | `/accounts/types/` | `/accounts/types/` | 2 | 1 | 3 | 4 | ✅ |
| 10 | GET | `/accounts/{account_id}` | `/accounts/:id` | 7 | 1 | 4 | 5 | ✅ |
| 11 | PUT | `/accounts/{account_id}` | `/accounts/:id` | 5 | 4 | 7 | 11 | ✅ |
| 12 | DELETE | `/accounts/{account_id}` | `/accounts/:id` | 6 | 1 | 6 | 7 | ✅ |
| 13 | PUT | `/accounts/set-archive-status` | `/accounts/set-archive-status` | 6 | 4 | 7 | 11 | ✅ |
| 14 | POST | `/accounts/adjust-balance` | `/accounts/adjust-balance` | 9 | 5 | 9 | 14 | ✅ |

**Accounts totals**: Py 50 → Go 62

---

## TRANSACTIONS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 15 | POST | `/transactions/` | `/transactions/` | 10 | 16 | 18 | 34 | ✅ |
| 16 | GET | `/transactions/` | `/transactions/` | 5 | 3 | 10 | 13 | ✅ |
| 17 | GET | `/transactions/{transaction_id}` | `/transactions/:id` | 3 | 6 | 7 | 13 | ✅ |
| 18 | PUT | `/transactions/` | `/transactions/` | 11 | 7 | 8 | 15 | ✅ |
| 19 | DELETE | `/transactions/{transaction_id}` | `/transactions/:id` | 7 | 5 | 6 | 11 | ✅ |
| 20 | GET | `/transactions/templates` | `/transactions/templates` | 3 | 4 | 4 | 8 | ✅ |
| 21 | PUT | `/transactions/templates` | `/transactions/templates` | 3 | 7 | 4 | 11 | ✅ |
| 22 | DELETE | `/transactions/templates` | `/transactions/templates` | 3 | 5 | 6 | 11 | ✅ |
| 23 | DELETE | `/transactions/templates/validate` | `/transactions/templates/validate` | 2 | 0 | 5 | 5 | ✅ |

**Transactions totals**: Py 47 → Go 105

---

## CATEGORIES

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 24 | GET | `/categories/` | `/categories/` | 3 | 4 | 3 | 7 | ✅ |
| 25 | GET | `/categories/grouped/` | `/categories/grouped/` | 3 | 4 | 3 | 7 | ✅ |
| 26 | POST | `/categories/` | `/categories/` | 6 | 5 | 9 | 14 | ✅ |
| 27 | PUT | `/categories/{category_id}/` | `/categories/:id/` | 3 | 9 | 6 | 15 | ✅ |
| 28 | DELETE | `/categories/{category_id}/` | `/categories/:id/` | 4 | 7 | 5 | 12 | ✅ |

**Categories totals**: Py 19 → Go 55

---

## CURRENCIES

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 29 | GET | `/currencies/` | `/currencies/` | 3 | 4 | 11 | 15 | ✅ |

**Currencies totals**: Py 3 → Go 15

---

## SETTINGS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 30 | GET | `/settings/languages` | `/settings/languages` | 3 | 1 | 3 | 4 | ✅ |
| 31 | GET | `/settings/` | `/settings/` | 4 | 6 | 5 | 11 | ✅ |
| 32 | POST | `/settings/` | `/settings/` | 7 | 5 | 7 | 12 | ✅ |
| 33 | GET | `/settings/base-currency/` | `/settings/base-currency/` | 3 | 3 | 3 | 6 | ✅ |
| 34 | PUT | `/settings/base-currency/` | `/settings/base-currency/` | 7 | 3 | 7 | 10 | ✅ |

**Settings totals**: Py 24 → Go 44

---

## EXCHANGE RATES

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 35 | GET | `/exchange-rates/` | `/exchange-rates/` | 2 | 5 | 4 | 9 | ✅ |
| 36 | GET | `/exchange-rates/update/` | `/exchange-rates/update` | 3 | 2 | 4 | 6 | ✅ |
| 37 | GET | `/exchange-rates/update/from/{start}/to/{end}/` | `/exchange-rates/update/from/:start/to/:end` | 5 | 4 | 9 | 13 | ✅ |

**Exchange Rates totals**: Py 10 → Go 29

---

## REPORTS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 38 | POST | `/reports/cashflow/` | `/reports/cashflow/` | 6 | 9 | 10 | 19 | ✅ |
| 39 | POST | `/reports/balance/` | `/reports/balance/` | 5 | 7 | 9 | 16 | ✅ |
| 40 | POST | `/reports/balance/non-hidden/` | `/reports/balance/non-hidden/` | 4 | 4 | 4 | 8 | ✅ |
| 41 | POST | `/reports/expenses-by-categories/` | `/reports/expenses-by-categories/` | 4 | 8 | 7 | 15 | ✅ |
| 42 | GET | `/reports/diagram/{type}/{start}/{end}` | `/reports/diagram/:type/:start/:end` | 4 | 8 | 4 | 12 | ✅ |
| 43 | POST | `/reports/expenses-data/` | `/reports/expenses-data/` | 3 | 6 | 5 | 11 | ✅ |

**Reports totals**: Py 26 → Go 82

---

## BUDGETS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 44 | POST | `/budgets/add/` | `/budgets/add/` | 5 | 8 | 9 | 17 | ✅ |
| 45 | GET | `/budgets/` | `/budgets/` | 8 | 3 | 5 | 8 | ✅ |
| 46 | PUT | `/budgets/{id}/` | `/budgets/:id/` | 4 | 8 | 5 | 13 | ✅ |
| 47 | DELETE | `/budgets/{id}/` | `/budgets/:id/` | 5 | 6 | 4 | 10 | ✅ |
| 48 | PUT | `/budgets/{id}/archive/` | `/budgets/:id/archive/` | 4 | 6 | 5 | 11 | ✅ |
| 49 | GET | `/budgets/daily-processing/` | `/budgets/daily-processing` | 3 | 13 | **0** | 13 | ✅ |

**Budgets totals**: Py 29 → Go 73 (daily-processing has 0 integration tests)

---

## PLANNED TRANSACTIONS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 50 | POST | `/planned-transactions/` | `/planned-transactions/` | 10 | 8 | 6 | 14 | ✅ |
| 51 | GET | `/planned-transactions/` | `/planned-transactions/` | 9 | 3 | 7 | 10 | ✅ |
| 52 | GET | `/planned-transactions/upcoming/occurrences` | `/planned-transactions/upcoming/occurrences` | 5 | 3 | 5 | 8 | ✅ |
| 53 | GET | `/planned-transactions/{id}` | `/planned-transactions/:id` | 4 | 5 | 5 | 10 | ✅ |
| 54 | PUT | `/planned-transactions/{id}` | `/planned-transactions/:id` | 7 | 8 | 8 | 16 | ✅ |
| 55 | DELETE | `/planned-transactions/{id}` | `/planned-transactions/:id` | 5 | 6 | 5 | 11 | ✅ |

**Planned Transactions totals**: Py 40 → Go 71

---

## FINANCIAL PLANNING

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 56 | POST | `/financial-planning/future-balance` | `/financial-planning/future-balance` | 8 | 9 | 6 | 15 | ✅ |
| 57 | POST | `/financial-planning/projection` | `/financial-planning/projection` | 7 | 9 | 6 | 15 | ✅ |

**Financial Planning totals**: Py 15 → Go 30

---

## ANALYTICS (DISABLED)

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 58 | POST | `/analytics/spending-trends` | `/analytics/spending-trends` | 2 | 0 | 6 | 6 | ✅ |
| 59 | POST | `/analytics/expense-categorization` | `/analytics/expense-categorization` | 2 | 0 | 6 | 6 | ✅ |

**Analytics totals**: Py 4 → Go 12 (both disabled, return 404)

---

## EXPORT

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 60 | POST | `/export/download` | `/export/download` | **0** | 8 | 7 | 15 | ✅ |
| 61 | POST | `/export/email` | `/export/email` | **0** | 5 | 5 | 10 | ✅ |

**Export totals**: Py 0 → Go 25 (Python had no tests for export)

---

## MANAGEMENT

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 62 | GET | `/management/backup/` | `/management/backup/` | 5 | 4 | 7 | 11 | ✅ |

**Management totals**: Py 5 → Go 11

---

## SUBSCRIPTIONS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 63 | GET | `/subscriptions/status` | `/subscriptions/status` | 4 | 6 | 5 | 11 | ✅ |
| 64 | GET | `/subscriptions/plans` | `/subscriptions/plans` | 4 | 3 | 2 | 5 | ✅ |
| 65 | POST | `/subscriptions/upgrade` | `/subscriptions/upgrade` | 3 | 5 | 4 | 9 | ✅ |
| 66 | POST | `/subscriptions/downgrade` | `/subscriptions/downgrade` | 3 | 5 | 3 | 8 | ✅ |

**Subscriptions totals**: Py 14 → Go 33

---

## PAYMENTS

| # | Method | Python Endpoint | Go Endpoint | Py Tests | Go Unit | Go Integ | Go Total | Ported OK |
|---|--------|----------------|-------------|----------|---------|----------|----------|-----------|
| 67 | POST | `/payments/checkout` | `/payments/checkout` | 8 | 4 | 4 | 8 | ✅ |
| 68 | POST | `/payments/portal` | `/payments/portal` | 4 | 2 | 2 | 4 | ✅ |
| 69 | GET | `/payments/status` | `/payments/status` | 4 | 2 | 3 | 5 | ✅ |
| 70 | GET | `/payments/session/{session_id}` | `/payments/session/:session_id` | 4 | 1 | 2 | 3 | ✅ |
| 71 | GET | `/payments/upgrade-price/{plan_id}` | `/payments/upgrade-price/:plan_id` | 3 | 3 | 2 | 5 | ✅ |
| 72 | POST | `/payments/change-plan` | `/payments/change-plan` | 8 | 6 | 4 | 10 | ✅ |
| 73 | POST | `/payments/cancel-scheduled-change` | `/payments/cancel-scheduled-change` | 7 | 4 | 3 | 7 | ✅ |
| 74 | POST | `/payments/webhook` | `/payments/webhook` | 6 | 2 | 2 | 4 | ✅ |

**Payments totals**: Py 44 → Go 46

---

## GO-ONLY ENDPOINTS (no Python equivalent)

| # | Method | Go Endpoint | Go Unit | Go Integ | Go Total | Notes |
|---|--------|-------------|---------|----------|----------|-------|
| 75 | GET | `/health` | 0 | 0 | 0 | Health check, trivial inline handler |

---

## SCENARIO / CROSS-DOMAIN TESTS

| # | Python Scenario File | Go Scenario File | Py Tests | Go Tests | Ported OK |
|---|---------------------|------------------|----------|----------|-----------|
| S1 | `test_balance_adjustment_scenario.py` | `balance_adjustment_test.go` | 5 | 6 | ✅ |
| S2 | `test_transactions_running_balance_scenario.py` | `running_balance_test.go` | 1 | 1 | ✅ |
| S3 | `test_transactions_transfer_scenarios.py` | `transfer_test.go` | 2 | 2 | ✅ |
| S4 | `test_subscription_plan_transitions_scenario.py` | `subscription_plan_transitions_test.go` | 1 | 2 | ✅ |
| S5 | `test_subscription_webhooks_scenario.py` | `subscription_webhooks_test.go` | 1 | 5 | ✅ |
| S6 | `test_trial_expired_free_limits_enforced_scenario.py` | `trial_expired_test.go` | 1 | 3 | ✅ |

**Scenario totals**: Py 11 → Go 19

---

## GRAND TOTALS

| Metric | Python | Go |
|--------|--------|-----|
| Total endpoints | 74 | 75 (74 + health) |
| Endpoint tests | ~335 | ~793 (~300 unit + ~473 integ + 19 scenario) |
| Scenario test files | 6 | 6 |
| Scenario tests | 11 | 19 |
| Endpoints with 0 tests | 2 (export) | 2 (health, validate template IDs) |

---

## ATTENTION ITEMS

| # | Endpoint | Issue |
|---|----------|-------|
| 23 | `DELETE /transactions/templates/validate` | **0 tests in Go** (Python has 2) |
| 49 | `GET /budgets/daily-processing` | **0 integration tests** in Go (only unit tests with mocks) |
| 75 | `GET /health` | Go-only, no tests (trivial) |

---

## HOW TO USE THIS FILE

1. For each endpoint, manually or via testing verify that the Go implementation matches Python behavior
2. Mark `[x]` in the "Ported OK" column when verified
3. Use the "Attention Items" section as a starting point for investigation
4. The user suspects there may be porting errors — focus on endpoints where Go has significantly fewer integration tests than Python endpoint tests
