.DEFAULT_GOAL := help
.PHONY: help dev build run clean migrate-up migrate-down migrate-status migrate-create tidy watch test test-fresh setup-test-db test-cover test-cover-total test-cover-handlers lint

## Development
dev: ## Run with hot reload (requires air)
	@air

watch: ## Run with hot reload (auto-installs air if missing)
	@if command -v air > /dev/null; then \
	    air; \
	    echo "Watching...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/air-verse/air@latest; \
	        air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

## Build
build: ## Build binaries
	@echo "Building..."
	@go build -o bin/web ./cmd/web
	@go build -o bin/migrate ./cmd/migrate

run: build ## Build and run
	@./bin/web

clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -rf tmp/

tidy: ## Run go mod tidy
	@go mod tidy

## Migrations
migrate-up: ## Run migrations up
	@go run ./cmd/migrate up

migrate-down: ## Run migrations down
	@go run ./cmd/migrate down

migrate-status: ## Show migration status
	@go run ./cmd/migrate status

migrate-create: ## Create migration (usage: make migrate-create name=<name>)
	@go run ./cmd/migrate create $(name)

## Testing
test: ## Run all tests (verbose)
	@go test -v ./...

test-fresh: ## Recreate test DB from scratch and run all tests
	@echo "Recreating test database..."
	@bash scripts/db-setup/setup-test-db.sh
	@echo ""
	@go test -v ./...

setup-test-db: ## Create/recreate test database (no tests)
	@bash scripts/db-setup/setup-test-db.sh

test-cover: ## Run tests with per-package coverage
	@go test -cover ./... 2>&1 | grep --line-buffered -v "no test files" | grep --line-buffered -v "coverage: 0.0%"

test-cover-total: ## Run tests and show total coverage
	@go test -coverpkg=./... -coverprofile=coverage.out ./... 2>&1 | grep --line-buffered -v "no test files" | grep --line-buffered -v "coverage: 0.0%"
	@echo ""
	@echo "=== Total coverage ==="
	@go tool cover -func=coverage.out | grep "total:" | awk '{print $$3}'
	@rm -f coverage.out

test-cover-handlers: ## Coverage for handlers only
	@go test -coverprofile=coverage.out ./internal/handlers/... 2>&1 | grep --line-buffered -v "no test files"
	@echo ""
	@echo "=== Handlers total coverage ==="
	@go tool cover -func=coverage.out | grep "total:" | awk '{print $$3}'
	@rm -f coverage.out

## Quality
lint: ## Run golangci-lint
	@golangci-lint run

## Help
help: ## Show this help
	@echo "Usage: make <target>"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; section = ""} \
		/^## / { section = substr($$0, 4); next } \
		/^[a-zA-Z_-]+:.*?## / { \
			if (section != "") { printf "\n\033[1m%s\033[0m\n", section; section = "" } \
			printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 \
		}' $(MAKEFILE_LIST)
	@echo ""
