# Async Task System

Background task processing using [Asynq](https://github.com/hibiken/asynq) with Redis. The worker runs inside the same binary as the API server (single-binary deployment).

## Overview

- **Library**: Asynq (Go library for distributed task processing)
- **Broker**: Redis
- **Deployment**: Embedded in the main binary; disable with `WORKER_ENABLED=false`
- **Kubernetes note**: Database backups are handled by a dedicated CronJob (`deploy/kubernetes/backend/base/cronjob-backup.yaml`) that runs pg_dump + rclone in a separate container. Manual backup triggers are available via the `/management/backup` endpoint, which creates a one-off K8s Job from the CronJob spec.
- **Concurrency**: Configurable via `WORKER_CONCURRENCY` (default: 10)

## Queue Priorities

| Queue | Weight | Purpose |
|-------|--------|---------|
| `critical` | 6 | Emails, budget recalculation — must process quickly |
| `default` | 3 | Scheduled maintenance tasks |
| `export` | 1 | Data export jobs (longer-running, lower priority) |

Higher weight means the queue is polled more frequently relative to others.

## Task Types

### Scheduled Tasks (5)

These run on cron schedules configured via environment variables.

| Task Type | Constant | Schedule (default) | Description |
|-----------|----------|-------------------|-------------|
| `exchange_rate:update` | `TypeExchangeRateUpdate` | `0 13 * * *` (1 PM daily) | Fetch latest rates from CurrencyBeacon API, save to DB, send admin notification email |
| `budget:processing` | `TypeBudgetProcessing` | `1 0 * * *` (12:01 AM daily) | Process outdated budgets: create copies for repeating budgets with shifted dates, then archive all outdated budgets |
| `token:cleanup` | `TypeTokenCleanup` | `0 2 * * *` (2 AM daily) | Delete expired activation tokens |
| `subscription:renewal` | `TypeSubscriptionRenewal` | `0 1 * * *` (1 AM daily) | Process expired subscriptions (trial -> free, paid without Stripe -> deactivate) |
| `subscription:downgrade` | `TypeSubscriptionDowngrade` | `0 * * * *` (every hour) | Execute pending subscription plan downgrades with entity archiving |

### On-Demand Tasks (4)

These are enqueued by application code in response to user actions or other events.

| Task Type | Constant | Queue | Description |
|-----------|----------|-------|-------------|
| `email:send` | `TypeEmailSend` | critical | Send a generic email (with optional attachment) |
| `email:activation` | `TypeEmailActivation` | critical | Build and send account activation email (chains to `email:send`) |
| `budget:user_update` | `TypeBudgetUserUpdate` | critical | Recalculate a user's budget collected amounts with currency conversion |
| `export:email` | `TypeExportEmail` | export | Generate Excel export file and email it to the user |

## Adding a New Task

1. **Define the task type** in `internal/workers/tasks/types.go`:
   ```go
   const TypeMyNewTask = "my:new_task"
   ```

2. **Create a handler file** at `internal/workers/tasks/my_new_task.go`:
   - Define a payload struct (if the task needs input data)
   - Define any service/repository interfaces the handler needs
   - Implement a handler struct with a `ProcessTask(ctx context.Context, t *asynq.Task) error` method
   - Create a constructor: `NewMyNewTaskHandler(...) *MyNewTaskHandler`

3. **Register the handler** in `internal/workers/manager.go`:
   - Add dependencies to the `Dependencies` struct if needed
   - Register the handler in `registerHandlers()`:
     ```go
     mux.HandleFunc(tasks.TypeMyNewTask, myHandler.ProcessTask)
     ```
   - For scheduled tasks, add a cron entry in `registerSchedules()`:
     ```go
     m.scheduler.Register(m.cfg.ScheduleMyNewTask, asynq.NewTask(tasks.TypeMyNewTask, nil))
     ```

4. **Add a config variable** (for scheduled tasks) in `internal/config/config.go`:
   ```go
   ScheduleMyNewTask: getEnvOrDefault("SCHEDULE_MY_NEW_TASK", "0 0 * * *"),
   ```

5. **Write tests** in `internal/workers/tasks/my_new_task_test.go`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `WORKER_ENABLED` | `true` | Enable/disable worker processing (client always active for enqueueing) |
| `WORKER_CONCURRENCY` | `10` | Maximum concurrent task workers |
| `SCHEDULE_EXCHANGE_RATE_UPDATE` | `0 13 * * *` | Cron: exchange rate fetch |
| `SCHEDULE_BUDGET_PROCESSING` | `1 0 * * *` | Cron: budget daily processing (copy + archive) |
| `SCHEDULE_TOKEN_CLEANUP` | `0 2 * * *` | Cron: expired token cleanup |
| `SCHEDULE_SUBSCRIPTION_RENEWAL` | `0 1 * * *` | Cron: subscription renewal |
| `SCHEDULE_SUBSCRIPTION_DOWNGRADE` | `0 * * * *` | Cron: subscription downgrade |
| `MESSAGE_ID_DOMAIN` | _(empty)_ | FQDN used as the domain part of outbound email `Message-ID` headers. MUST contain at least one dot (RFC 5322 §3.6.4) to avoid Gmail bulk-mail spam heuristics. In production set to the public service domain (e.g. `orgfin`); when empty the email service falls back to `os.Hostname()` / `net.LookupCNAME()` and finally to `localhost`. |

## Outbound Email Rendering

All emails sent by the backend (admin notifications, account activation, data export, backup notifications) go through `internal/services/email/Service.Send` or `Service.SendWithAttachment` and are rendered as **`multipart/alternative`** (or `multipart/mixed` wrapping `multipart/alternative` when an attachment is present). Both a `text/plain` and a `text/html` part are always included; the `text/plain` part is auto-generated from the HTML body (tags stripped, whitespace collapsed). The HTML part is encoded with `quoted-printable` — never `7bit`. The `Message-ID` domain part is resolved per the `MESSAGE_ID_DOMAIN` chain documented above. These properties together align the rendered MIME with what Gmail's spam heuristics expect from legitimate transactional mail.

### Backup Notification Subjects

Subjects emitted by `services/backup_notify` are intentionally short, ASCII-only, and free of square brackets:

- success: `Database backup created`
- failure: `Database backup FAILED`

The environment label (`production`, `staging`, etc.) remains rendered in the email body for operator visibility.

## Architecture

```
internal/workers/
├── manager.go      - Orchestrates server, scheduler, and client
│                     Creates Asynq server (task processing),
│                     scheduler (cron), and client (enqueueing)
├── config.go       - Queue names/weights, Redis URL parsing
├── enqueuer.go     - Type aliases for task enqueueing
└── tasks/
    ├── types.go    - All task type string constants
    └── *.go        - One file per task (handler + payload + interfaces)
```

The `Manager` has three components:
- **Server**: Processes tasks from Redis queues (only when `WORKER_ENABLED=true`)
- **Scheduler**: Registers cron jobs that enqueue tasks on schedule (only when `WORKER_ENABLED=true`)
- **Client**: Always active — used by API handlers to enqueue on-demand tasks

## Task Details

### `exchange_rate:update` — ExchangeRateUpdateHandler

**Source**: `internal/workers/tasks/exchange_rates.go`

- **Payload**: No payload (scheduled task).
- **Behavior**:
  1. Delegate to `exchangerates.Service.FetchAndSaveRates(today)` which validates the API key, calls the CurrencyBeacon historical rates API (`/v1/historical`) for today's date, reads/parses the JSON response (body limited to 1 MB), and saves the rates to the DB via `ExchangeRateRepository.SaveRate()`.
  2. Send an admin notification email with update details (environment, timestamp, rate count, base currency).
- **Notifications**: Yes — enqueues `email:send` for each configured admin email on success.
- **Error handling**:
  - MaxRetry: **24**, Queue: `default`, Timeout: **10 minutes**.
  - HTTP client has a 30-second timeout per request (configured in the service).
  - Returns error on missing API key, API failure, or DB save failure (triggers retry).
  - Notification failures are logged but do not cause the task to fail.
- **Dependencies**: `*exchangerates.Service`, `TaskEnqueuer`, `NotificationConfig`.

---

### `budget:user_update` — BudgetUserUpdateHandler

**Source**: `internal/workers/tasks/budget_update.go`

- **Payload**: `BudgetUserUpdatePayload` — `{ user_id: int, account_id: int }`.
- **Behavior**:
  1. Unmarshal the JSON payload to get `UserID` and `AccountID`.
  2. Delegate to `BudgetService.RecalculateCollectedAmounts(userID)` which recalculates collected amounts for all active budgets belonging to the user with proper currency conversion.
- **Notifications**: No.
- **Error handling**:
  - MaxRetry: **10**, Queue: `critical`, Timeout: **5 minutes**.
  - Returns error on payload unmarshal failure or recalculation failure (triggers retry).
- **Dependencies**: `BudgetService`.

---

### `budget:processing` — BudgetProcessingHandler

**Source**: `internal/workers/tasks/budgets.go`

- **Payload**: No payload (scheduled task).
- **Behavior**:
  The task handler delegates all business logic to `BudgetService.DailyProcessing()`:
  1. Fetch all outdated budgets (budgets past their end date).
  2. If none found, return immediately.
  3. For each outdated budget:
     - If the budget is repeating (`Repeat = true`): create a new copy with shifted dates and "(copy)" name suffix.
     - Archive the original budget.
  4. Individual budget errors are logged and processing **continues** with remaining budgets (does not fail on first error).
- **Notifications**: No.
- **Error handling**:
  - MaxRetry: **10**, Queue: `default`, Timeout: **10 minutes**.
  - Returns error on initial DB read failure (triggers retry).
  - Individual budget processing errors are logged but do not cause the task to fail (partial failure handling in the service layer).
- **Dependencies**: `BudgetService`.

---

### `token:cleanup` — TokenCleanupHandler

**Source**: `internal/workers/tasks/token_cleanup.go`

- **Payload**: No payload (scheduled task).
- **Behavior**:
  1. Call `ActivationTokenRepository.DeleteExpired()` to remove all expired activation tokens from the database.
- **Notifications**: No.
- **Error handling**:
  - MaxRetry: **10**, Queue: `default`, Timeout: **5 minutes**.
  - Returns error on deletion failure (triggers retry).
- **Dependencies**: `ActivationTokenRepository`.

---

### `subscription:renewal` — SubscriptionRenewalHandler

**Source**: `internal/workers/tasks/subscription.go`

- **Payload**: No payload (scheduled task).
- **Behavior**:
  The task handler delegates all business logic to `SubscriptionService.ProcessRenewals()`:
  1. Fetch all expired subscriptions via `SubscriptionRepository.GetExpiredSubscriptions()`.
  2. If none found, return immediately.
  3. For each expired subscription, apply renewal logic based on plan type:
     - **Trial**: downgrade to the free plan (`SubscriptionPlanRepository.GetFreePlan()`), clear trial timestamps, keep active.
     - **Premium without Stripe** (`HasStripeSubscription = false`): deactivate the subscription (`IsActive = false`).
     - **Premium with Stripe** (`HasStripeSubscription = true`): skip — Stripe webhooks handle renewal.
  4. Save updated subscription via `SubscriptionRepository.Update()`.
  5. Individual subscription errors are logged and processing **continues** with remaining subscriptions (does not fail on first error).
- **Notifications**: No.
- **Error handling**:
  - MaxRetry: **10**, Queue: `default`, Timeout: **10 minutes**.
  - Returns error on initial DB read failure (triggers retry).
  - Individual subscription processing errors are logged but do not cause the task to fail.
- **Dependencies**: `SubscriptionService` (which internally uses `SubscriptionRepository`, `SubscriptionPlanRepository`).

---

### `subscription:downgrade` — SubscriptionDowngradeHandler

**Source**: `internal/workers/tasks/subscription.go`

- **Payload**: No payload (scheduled task).
- **Behavior**:
  The task handler delegates all business logic to `SubscriptionService.ProcessExpiredDowngrades()`:
  1. Fetch all subscriptions with pending downgrades via `SubscriptionRepository.GetPendingDowngrades()`.
  2. If none found, return immediately.
  3. For each pending downgrade:
     - Skip if `PendingPlanID` is nil.
     - Look up the target plan via `SubscriptionPlanRepository.GetByID()`.
     - Save the entity selection (`PendingDowngradeAccountIDs`, `PendingDowngradeBudgetID`) before clearing.
     - Update the subscription: set the new plan ID and type, clear `PendingPlanID`, `PendingDowngradeAccountIDs`, `PendingDowngradeBudgetID`, set `IsActive = true`.
     - Save via `SubscriptionRepository.UpdateSubscriptionFull()`.
     - If the new plan is "free", apply entity deactivation via `ApplyDowngradeEntitySelection()` — archives excess accounts and budgets.
  4. Individual subscription errors are logged and processing **continues** with remaining subscriptions (does not fail on first error).
- **Notifications**: No.
- **Error handling**:
  - MaxRetry: **10**, Queue: `default`, Timeout: **10 minutes**.
  - Returns error on initial DB read failure (triggers retry).
  - Individual subscription processing errors are logged but do not cause the task to fail.
  - Entity archiving failures are non-fatal (logged but subscription is already downgraded).
- **Dependencies**: `SubscriptionService` (which internally uses `SubscriptionRepository`, `SubscriptionPlanRepository`, `AccountArchiver`, `BudgetArchiver`).

---

### `email:send` — EmailSendHandler

**Source**: `internal/workers/tasks/email.go`

- **Payload**: `EmailSendPayload` — `{ to: string, subject: string, body: string, attachment_data?: []byte, attachment_name?: string }`.
- **Behavior**:
  1. Unmarshal the JSON payload.
  2. If `attachment_data` and `attachment_name` are both present, call `EmailService.SendWithAttachment()`.
  3. Otherwise, call `EmailService.Send()` for a plain email.
- **Notifications**: This task *is* the notification mechanism. Other tasks chain to it.
- **Error handling**:
  - MaxRetry: **10**, Queue: `critical`, Timeout: **2 minutes**.
  - Returns error on payload unmarshal failure or email send failure (triggers retry).
- **Dependencies**: `EmailService`.

---

### `email:activation` — ActivationEmailHandler

**Source**: `internal/workers/tasks/activation_email.go`

- **Payload**: `ActivationEmailPayload` — `{ user_id: int, email: string, token: string, app_url: string, frontend_url: string }`.
- **Behavior**:
  1. Unmarshal the JSON payload.
  2. Build the activation link: `{frontend_url}/activate/{token}`.
  3. Generate an HTML email body with a styled activation button and link.
  4. Create and enqueue an `email:send` task (task chaining) with the built subject/body and the user's email address.
- **Notifications**: Chains to `email:send` — does not send email directly.
- **Error handling**:
  - MaxRetry: **10**, Queue: `critical`, Timeout: **2 minutes**.
  - Returns error on payload unmarshal failure, email task creation failure, or enqueue failure (triggers retry).
- **Dependencies**: `TaskEnqueuer`.

---

### `export:email` — ExportEmailHandler

**Source**: `internal/workers/tasks/export_email.go`

- **Payload**: `ExportEmailPayload` — `{ user_id: int, start_date: string, end_date: string, email: string }`.
- **Behavior**:
  1. Unmarshal the JSON payload.
  2. Generate an Excel file via `ExportService.GenerateExcel(userID, startDate, endDate)`.
  3. Send the Excel file as an email attachment via `EmailService.SendWithAttachment()` with the filename `transactions_export.xlsx`.
- **Notifications**: Sends email directly via `EmailService` (does not chain to `email:send`).
- **Error handling**:
  - MaxRetry: **3**, Queue: `export`, Timeout: **5 minutes**.
  - Returns error on payload unmarshal failure, Excel generation failure, or email send failure (triggers retry).
- **Dependencies**: `ExportService`, `EmailService`.
