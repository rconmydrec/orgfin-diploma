# Backend Documentation

This directory contains documentation for the Go backend of the Budget Tracker application.

## Contents

| Document | Description |
|----------|-------------|
| [`architecture-guide.md`](architecture-guide.md) | **Mandatory reading.** Layer responsibilities, DI rules, error handling, coding conventions |
| [`endpoints/README.md`](endpoints/README.md) | All 74 API endpoints grouped by domain — route, request/response, business logic, tests |
| [`database-models/index.md`](database-models/index.md) | Database models — all tables, fields, types, relationships, ER diagram |
| [`workers.md`](workers.md) | Async task system — Asynq + Redis, all 10 task types, queue priorities, cron schedules |
| [`cli-tools.md`](cli-tools.md) | One-shot operator CLIs under `cmd/` (e.g. `reconcile-balances`) — purpose, flags, output format, idempotency |
| [`migration-history.md`](migration-history.md) | Python→Go migration comparison summary: all 74 endpoints with test count totals and ported status |
| [`endpoint-migration-comparison/`](endpoint-migration-comparison/README.md) | Historical per-endpoint Python→Go comparison files (one file per endpoint) |
| [`deployment.md`](../../docs/deployment.md) | Deploy scripts — `deploy.sh` pipeline, `stage.sh` commands, DB sync, Stripe sanitization (moved to root `docs/`) |

## Quick Navigation

- Looking for **database schema and model details**? → [`database-models/`](database-models/index.md)
- Looking for **how an endpoint works in Go**? → [`endpoints/`](endpoints/README.md)
- Looking for **test counts or migration status**? → [`migration-history.md`](migration-history.md)
- Looking for **detailed Python vs Go differences**? → [`endpoint-migration-comparison/`](endpoint-migration-comparison/README.md)
