# Go Budget Tracker - Backend

Backend for budget tracker application built with Go and Echo framework, migrated from Python/FastAPI.

> **Note**: This is a sub-project within the [go-budget monorepo](../README.md). All commands below should be run from the `backend/` directory.

## Requirements

- Go 1.22+
- PostgreSQL 15+
- Redis 7+ (for async task queue)
- Air (for hot reload, optional)

## Quick Start

```bash
cd backend

# Install dependencies
go mod download

# Copy environment file and configure
cp .env.sample .env

# Start Redis (required for task queue)
redis-server

# Run database migrations
make migrate-up

# Start development server with hot reload
make watch
```

## Make Commands

### Development

| Command | Description |
|---------|-------------|
| `make watch` | Run with hot reload (installs air if needed) |
| `make dev` | Run with air (requires air installed) |
| `make run` | Build and run |
| `make build` | Build binaries to `bin/` |
| `make clean` | Remove build artifacts |
| `make tidy` | Run go mod tidy |
| `make lint` | Run golangci-lint |

### Database Migrations

| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-status` | Show migration status |
| `make migrate-create name=create_users` | Create new migration |

### Testing

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (verbose) |
| `make test-cover` | Run all tests with coverage |
| `make test-cover-total` | Run tests and show total coverage percentage |
| `make test-cover-handlers` | Coverage for handlers only |

## Running Tests

Tests use a separate database `budgeter_test`. Make sure it exists:

```sql
CREATE DATABASE budgeter_test;
```

### Run all tests

```bash
make test
```

### Run with coverage

```bash
make test-cover
```

### Run specific tests

```bash
# Specific module
go test -v ./internal/handlers/auth/...

# Single test by name
go test -v ./internal/handlers/auth/... -run TestLoginSuccess
```

### Test database config

See `internal/testutil/testutil.go`:
- Host: localhost
- Port: 5432
- User: postgres
- Password: 123
- Database: budgeter_test

## Project Structure

```
backend/
├── cmd/
│   ├── web/          # Main application entry point
│   └── migrate/      # Database migration tool
├── internal/
│   ├── auth/         # JWT utilities
│   ├── config/       # Configuration
│   ├── database/     # Database connection
│   ├── handlers/     # HTTP handlers
│   ├── middleware/    # Auth, validation
│   ├── models/       # Domain models
│   ├── repositories/ # Data access layer
│   ├── services/     # Business logic
│   ├── testutil/     # Test helpers
│   └── workers/      # Background jobs
├── migrations/       # SQL migrations
├── .air.toml         # Air hot-reload config
├── .env.sample       # Environment template
├── Makefile          # Build and run targets
└── go.mod            # Go module definition
```

## API Endpoints

### Auth
- `POST /auth/register/` - Register new user
- `POST /auth/login/` - Login
- `GET /auth/profile/` - Get user profile (auth required)
- `GET /auth/activate/:token` - Activate account
- `POST /auth/oauth/` - Google OAuth login
- `POST /auth/change-password/` - Change password (auth required)

### Accounts
- `POST /accounts/` - Create account
- `GET /accounts/` - List accounts
- `GET /accounts/types/` - List account types
- `GET /accounts/:id` - Get account details
- `PUT /accounts/:id` - Update account
- `DELETE /accounts/:id` - Delete account
- `PUT /accounts/set-archive-status` - Archive/unarchive
- `POST /accounts/adjust-balance` - Adjust balance

### Other endpoints
- `/transactions/*` - Transaction management
- `/categories/*` - Category management
- `/budgets/*` - Budget management
- `/currencies/*` - Currency list
- `/settings/*` - User settings
- `/reports/*` - Financial reports
- `/subscriptions/*` - Subscription plans
- `/payments/*` - Stripe payments

## Environment Variables

See `.env.sample` for all available options:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=budget_tracker

# Server
PORT=8080
ENVIRONMENT=development

# Security
SECRET_KEY=your-secret-key
JWT_EXPIRATION_MINUTES=30

# Google OAuth
GOOGLE_CLIENT_ID=your-client-id

# Stripe (optional)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Redis & Workers
REDIS_URL=redis://localhost:6379
WORKER_ENABLED=true
WORKER_CONCURRENCY=10
```

## Deployment

The project includes full deployment infrastructure in `deploy/`:

```
deploy/
├── docker/
│   └── backend.Dockerfile          # Multi-stage build, distroless runtime
├── kubernetes/
│   ├── backend/
│   │   ├── base/                    # Deployment, Service, Ingress, Redis, CronJob backup
│   │   └── overlays/
│   │       ├── dev/                 # orgfin-api.local, no TLS
│   │       ├── staging/             # staging-api.orgfin, cert-manager TLS
│   │       └── prod/                # api.orgfin, cert-manager TLS
│   └── web/                         # Placeholder for future frontend
└── scripts/
    ├── build-and-push.sh            # Build and push Docker image
    ├── deploy.sh                    # Deploy to any environment
    └── stage.sh                     # Staging management (up/down/destroy/status/sync-db)
```

### Prerequisites

Create `.deploy.env` in the repo root (gitignored) with SSH connection details:

```bash
SSH_HOST=kuber@65.108.248.92
PLATFORM=linux/arm64
REMOTE_REPO=/home/kuber/go-budget
```

### Full deploy (build + push + deploy)

`deploy.sh` performs the complete pipeline in one command:
1. Builds Docker image via `build-and-push.sh`
2. Pushes to Docker Hub (versioned tag only, no `latest`)
3. Updates `version.txt` and image tag in base `deployment.yaml`
4. Commits and pushes to current git branch
5. SSH to server: `git pull` → `kubectl apply -k` → waits for rollout

```bash
# Deploy to staging
./deploy/scripts/deploy.sh --env staging --tag 1.0.0

# Deploy to production (with confirmation prompt)
./deploy/scripts/deploy.sh --env prod --tag 1.0.0

# Skip build (deploy existing image)
./deploy/scripts/deploy.sh --env staging --tag 1.0.0 --skip-build
```

### Build only (without deploy)

```bash
# Build Docker image locally
./deploy/scripts/build-and-push.sh 1.0.0

# Build and push to Docker Hub
./deploy/scripts/build-and-push.sh 1.0.0 --push
```

### Version tracking

Current deployed version is stored in `version.txt` (tracked in git). It is updated automatically by `build-and-push.sh` after a successful build.

### Staging management

```bash
./deploy/scripts/stage.sh up                  # Start staging environment
./deploy/scripts/stage.sh deploy --tag=1.0.0  # Update staging image
./deploy/scripts/stage.sh status              # Show staging state
./deploy/scripts/stage.sh down                # Scale to 0 (stop)
./deploy/scripts/stage.sh sync-db             # Sync DB from prod to staging
./deploy/scripts/stage.sh destroy             # Delete staging namespace
```

### Environment configuration

Each environment has its own `.env` file loaded via Kustomize `configMapGenerator`:

- **Dev**: `deploy/kubernetes/backend/overlays/dev/.env.dev` (copy from `.env.dev.sample`)
- **Staging**: `deploy/kubernetes/backend/overlays/staging/.env.staging` (copy from `.env.staging.sample`)
- **Prod**: `deploy/kubernetes/backend/overlays/prod/.env.prod` (copy from `.env.prod.sample`, gitignored)

See `.env.prod.sample` for the full list of required variables.

## License

MIT
