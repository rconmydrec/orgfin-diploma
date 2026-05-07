# API Endpoints Reference

Go implementation reference for all 74 API endpoints. Each file covers a domain group.

## Endpoint Groups

| File | Endpoints | Count |
|------|-----------|-------|
| [`auth.md`](auth.md) | POST /auth/register/, POST /auth/login/, GET /auth/profile/, GET /auth/activate/:token, POST /auth/oauth/, POST /auth/change-password/ | 6 |
| [`accounts.md`](accounts.md) | POST/GET /accounts/, GET /accounts/types/, GET/PUT/DELETE /accounts/:id, PUT /accounts/set-archive-status, POST /accounts/adjust-balance | 8 |
| [`transactions.md`](transactions.md) | POST/GET /transactions/, GET/PUT/DELETE /transactions/:id, GET/PUT/DELETE /transactions/templates, DELETE /transactions/templates/validate | 9 |
| [`categories.md`](categories.md) | GET /categories/, GET /categories/grouped/, POST /categories/, PUT/DELETE /categories/:id/ | 5 |
| [`currencies.md`](currencies.md) | GET /currencies/ | 1 |
| [`settings.md`](settings.md) | GET /settings/languages, GET/POST /settings/, GET/PUT /settings/base-currency/ | 5 |
| [`exchange-rates.md`](exchange-rates.md) | GET /exchange-rates/, GET /exchange-rates/update, GET /exchange-rates/update/from/:start/to/:end | 3 |
| [`reports.md`](reports.md) | POST /reports/cashflow/, POST /reports/balance/, POST /reports/balance/non-hidden/, POST /reports/expenses-by-categories/, GET /reports/diagram/:type/:start/:end, POST /reports/expenses-data/ | 6 |
| [`budgets.md`](budgets.md) | POST /budgets/add/, GET /budgets/, PUT/DELETE /budgets/:id/, PUT /budgets/:id/archive/, GET /budgets/daily-processing | 6 |
| [`planned-transactions.md`](planned-transactions.md) | POST/GET /planned-transactions/, GET /planned-transactions/upcoming/occurrences, GET/PUT/DELETE /planned-transactions/:id | 6 |
| [`financial-planning.md`](financial-planning.md) | POST /financial-planning/future-balance, POST /financial-planning/projection | 2 |
| [`analytics.md`](analytics.md) | POST /analytics/spending-trends, POST /analytics/expense-categorization (both disabled — return 404) | 2 |
| [`export.md`](export.md) | POST /export/download, POST /export/email | 2 |
| [`management.md`](management.md) | GET /management/backup/ | 1 |
| [`subscriptions.md`](subscriptions.md) | GET /subscriptions/status, GET /subscriptions/plans, POST /subscriptions/upgrade, POST /subscriptions/downgrade | 4 |
| [`payments.md`](payments.md) | POST /payments/checkout, POST /payments/portal, GET /payments/status, GET /payments/session/:session_id, GET /payments/upgrade-price/:plan_id, POST /payments/change-plan, POST /payments/cancel-scheduled-change, POST /payments/webhook | 8 |
| [`scenarios.md`](scenarios.md) | Cross-domain integration scenarios: balance adjustment, running balance, transfers, subscription transitions, webhooks, trial expiry | 6 scenarios |

**Total: 74 endpoints + 6 cross-domain scenarios**

## Common Patterns

- **Auth**: All protected endpoints use `RequireAuth` middleware (JWT). Public endpoints are registered outside the protected group.
- **User active check**: The `RequireActiveUser` middleware is applied to all authenticated route groups. It verifies the user is active and not soft-deleted before the handler runs. Returns 401 "User not activated" if inactive or deleted.
- **Error responses**: All error messages are generic to clients. Internal errors are logged server-side only.
- **JSON naming**: Response fields use camelCase. Some request fields use snake_case to match frontend expectations.
