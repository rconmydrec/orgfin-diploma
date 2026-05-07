# Endpoint Migration Comparison — Historical Archive

Per-endpoint Python→Go comparison files generated during migration analysis (2026-02-21).

For a high-level summary with test count totals, see [`../migration-history.md`](../migration-history.md).

## Auth (01–06)

| File | Endpoint |
|------|----------|
| [`01-auth-register.md`](01-auth-register.md) | POST /auth/register/ |
| [`02-auth-login.md`](02-auth-login.md) | POST /auth/login/ |
| [`03-auth-profile.md`](03-auth-profile.md) | GET /auth/profile/ |
| [`04-auth-activate.md`](04-auth-activate.md) | GET /auth/activate/:token |
| [`05-auth-oauth.md`](05-auth-oauth.md) | POST /auth/oauth/ |
| [`06-auth-change-password.md`](06-auth-change-password.md) | POST /auth/change-password/ |

## Accounts (07–14)

| File | Endpoint |
|------|----------|
| [`07-accounts-create.md`](07-accounts-create.md) | POST /accounts/ |
| [`08-accounts-list.md`](08-accounts-list.md) | GET /accounts/ |
| [`09-accounts-types.md`](09-accounts-types.md) | GET /accounts/types/ |
| [`10-accounts-get-by-id.md`](10-accounts-get-by-id.md) | GET /accounts/:id |
| [`11-accounts-update.md`](11-accounts-update.md) | PUT /accounts/:id |
| [`12-accounts-delete.md`](12-accounts-delete.md) | DELETE /accounts/:id |
| [`13-accounts-set-archive-status.md`](13-accounts-set-archive-status.md) | PUT /accounts/set-archive-status |
| [`14-accounts-adjust-balance.md`](14-accounts-adjust-balance.md) | POST /accounts/adjust-balance |

## Transactions (15–23)

| File | Endpoint |
|------|----------|
| [`15-transactions-create.md`](15-transactions-create.md) | POST /transactions/ |
| [`16-transactions-list.md`](16-transactions-list.md) | GET /transactions/ |
| [`17-transactions-get-by-id.md`](17-transactions-get-by-id.md) | GET /transactions/:id |
| [`18-transactions-update.md`](18-transactions-update.md) | PUT /transactions/ |
| [`19-transactions-delete.md`](19-transactions-delete.md) | DELETE /transactions/:id |
| [`20-transactions-templates.md`](20-transactions-templates.md) | GET /transactions/templates |
| [`21-transactions-templates-update.md`](21-transactions-templates-update.md) | PUT /transactions/templates |
| [`22-transactions-templates-delete.md`](22-transactions-templates-delete.md) | DELETE /transactions/templates |
| [`23-transactions-templates-validate.md`](23-transactions-templates-validate.md) | DELETE /transactions/templates/validate |

## Categories (24–28)

| File | Endpoint |
|------|----------|
| [`24-categories-list.md`](24-categories-list.md) | GET /categories/ |
| [`25-categories-grouped.md`](25-categories-grouped.md) | GET /categories/grouped/ |
| [`26-categories-create.md`](26-categories-create.md) | POST /categories/ |
| [`27-categories-update.md`](27-categories-update.md) | PUT /categories/:id/ |
| [`28-categories-delete.md`](28-categories-delete.md) | DELETE /categories/:id/ |

## Currencies (29)

| File | Endpoint |
|------|----------|
| [`29-currencies-list.md`](29-currencies-list.md) | GET /currencies/ |

## Settings (30–34)

| File | Endpoint |
|------|----------|
| [`30-settings-languages.md`](30-settings-languages.md) | GET /settings/languages |
| [`31-settings-get.md`](31-settings-get.md) | GET /settings/ |
| [`32-settings-update.md`](32-settings-update.md) | POST /settings/ |
| [`33-settings-base-currency-get.md`](33-settings-base-currency-get.md) | GET /settings/base-currency/ |
| [`34-settings-base-currency-update.md`](34-settings-base-currency-update.md) | PUT /settings/base-currency/ |

## Exchange Rates (35–37)

| File | Endpoint |
|------|----------|
| [`35-exchange-rates-list.md`](35-exchange-rates-list.md) | GET /exchange-rates/ |
| [`36-exchange-rates-update.md`](36-exchange-rates-update.md) | GET /exchange-rates/update |
| [`37-exchange-rates-update-range.md`](37-exchange-rates-update-range.md) | GET /exchange-rates/update/from/:start/to/:end |

## Reports (38–43)

| File | Endpoint |
|------|----------|
| [`38-reports-cashflow.md`](38-reports-cashflow.md) | POST /reports/cashflow/ |
| [`39-reports-balance.md`](39-reports-balance.md) | POST /reports/balance/ |
| [`40-reports-balance-non-hidden.md`](40-reports-balance-non-hidden.md) | POST /reports/balance/non-hidden/ |
| [`41-reports-expenses-by-categories.md`](41-reports-expenses-by-categories.md) | POST /reports/expenses-by-categories/ |
| [`42-reports-diagram.md`](42-reports-diagram.md) | GET /reports/diagram/:type/:start/:end |
| [`43-reports-expenses-data.md`](43-reports-expenses-data.md) | POST /reports/expenses-data/ |

## Budgets (44–49)

| File | Endpoint |
|------|----------|
| [`44-budgets-create.md`](44-budgets-create.md) | POST /budgets/add/ |
| [`45-budgets-list.md`](45-budgets-list.md) | GET /budgets/ |
| [`46-budgets-update.md`](46-budgets-update.md) | PUT /budgets/:id/ |
| [`47-budgets-delete.md`](47-budgets-delete.md) | DELETE /budgets/:id/ |
| [`48-budgets-archive.md`](48-budgets-archive.md) | PUT /budgets/:id/archive/ |
| [`49-budgets-daily-processing.md`](49-budgets-daily-processing.md) | GET /budgets/daily-processing |

## Planned Transactions (50–55)

| File | Endpoint |
|------|----------|
| [`50-planned-transactions-create.md`](50-planned-transactions-create.md) | POST /planned-transactions/ |
| [`51-planned-transactions-list.md`](51-planned-transactions-list.md) | GET /planned-transactions/ |
| [`52-planned-transactions-upcoming.md`](52-planned-transactions-upcoming.md) | GET /planned-transactions/upcoming/occurrences |
| [`53-planned-transactions-get-by-id.md`](53-planned-transactions-get-by-id.md) | GET /planned-transactions/:id |
| [`54-planned-transactions-update.md`](54-planned-transactions-update.md) | PUT /planned-transactions/:id |
| [`55-planned-transactions-delete.md`](55-planned-transactions-delete.md) | DELETE /planned-transactions/:id |

## Financial Planning (56–57)

| File | Endpoint |
|------|----------|
| [`56-financial-planning-future-balance.md`](56-financial-planning-future-balance.md) | POST /financial-planning/future-balance |
| [`57-financial-planning-projection.md`](57-financial-planning-projection.md) | POST /financial-planning/projection |

## Analytics (58–59)

| File | Endpoint |
|------|----------|
| [`58-analytics-spending-trends.md`](58-analytics-spending-trends.md) | POST /analytics/spending-trends |
| [`59-analytics-expense-categorization.md`](59-analytics-expense-categorization.md) | POST /analytics/expense-categorization |

## Export (60–61)

| File | Endpoint |
|------|----------|
| [`60-export-download.md`](60-export-download.md) | POST /export/download |
| [`61-export-email.md`](61-export-email.md) | POST /export/email |

## Management (62)

| File | Endpoint |
|------|----------|
| [`62-management-backup.md`](62-management-backup.md) | GET /management/backup/ |

## Subscriptions (63–66)

| File | Endpoint |
|------|----------|
| [`63-subscriptions-status.md`](63-subscriptions-status.md) | GET /subscriptions/status |
| [`64-subscriptions-plans.md`](64-subscriptions-plans.md) | GET /subscriptions/plans |
| [`65-subscriptions-upgrade.md`](65-subscriptions-upgrade.md) | POST /subscriptions/upgrade |
| [`66-subscriptions-downgrade.md`](66-subscriptions-downgrade.md) | POST /subscriptions/downgrade |

## Payments (67–74)

| File | Endpoint |
|------|----------|
| [`67-payments-checkout.md`](67-payments-checkout.md) | POST /payments/checkout |
| [`68-payments-portal.md`](68-payments-portal.md) | POST /payments/portal |
| [`69-payments-status.md`](69-payments-status.md) | GET /payments/status |
| [`70-payments-session-status.md`](70-payments-session-status.md) | GET /payments/session/:session_id |
| [`71-payments-upgrade-price.md`](71-payments-upgrade-price.md) | GET /payments/upgrade-price/:plan_id |
| [`72-payments-change-plan.md`](72-payments-change-plan.md) | POST /payments/change-plan |
| [`73-payments-cancel-scheduled-change.md`](73-payments-cancel-scheduled-change.md) | POST /payments/cancel-scheduled-change |
| [`74-payments-webhook.md`](74-payments-webhook.md) | POST /payments/webhook |

## Cross-Domain Scenarios (S1–S6)

| File | Scenario |
|------|----------|
| [`S1-balance-adjustment-scenario.md`](S1-balance-adjustment-scenario.md) | Balance adjustment behavior and report exclusion |
| [`S2-running-balance-scenario.md`](S2-running-balance-scenario.md) | Running balance recalculation across transactions |
| [`S3-transfer-scenario.md`](S3-transfer-scenario.md) | Transfer between accounts (same and different currencies) |
| [`S4-subscription-plan-transitions-scenario.md`](S4-subscription-plan-transitions-scenario.md) | Subscription plan upgrade/downgrade transitions |
| [`S5-subscription-webhooks-scenario.md`](S5-subscription-webhooks-scenario.md) | Stripe webhook event processing |
| [`S6-trial-expired-scenario.md`](S6-trial-expired-scenario.md) | Trial expiry and free plan limits enforcement |
