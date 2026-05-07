# Architecture Guide

Mandatory reading for anyone writing backend code — human or AI agent.
This document defines the architectural rules and coding conventions for the Go backend.

## Core Principles

### 1. Simplicity Over Cleverness

Follow the Go philosophy: write clear, obvious code.

- **No premature abstractions.** Three similar lines are better than a generic helper used once.
- **No speculative generality.** Don't build for hypothetical future requirements. Solve the current problem.
- **No unnecessary indirection.** If a wrapper/adapter/factory adds no real value, don't create it.
- **Flat is better than nested.** Prefer early returns over deeply nested if/else chains.
- **Standard library first.** Reach for third-party packages only when the standard library is clearly insufficient.

### 2. Layer Isolation

Each layer (handler, service, repository) must be **independently replaceable**. You should be able to rewrite any single service or repository without touching other parts of the application.

**Why this matters:** AI-generated code degrades in quality over time. Tight coupling makes it impossible to fix one area without breaking others. Strict isolation keeps the blast radius small — a bad service can be rewritten without a full rewrite of the application.

Rules:

- **No cross-service imports.** Service A must not import Service B's package directly. If Service A needs functionality from Service B, inject it as an interface.
- **No repository-to-repository calls.** Each repository talks only to the database, never to another repository.
- **No shared mutable state.** Services communicate through method calls, not shared variables or package-level state.
- **Interfaces at the consumer side.** The package that *uses* a dependency defines the interface it needs (Go convention). This keeps packages decoupled even if the implementation changes.

---

## Layer Responsibilities

### Handlers (`internal/handlers/`)

Handlers are a **thin translation layer** between HTTP and the service layer.

**Allowed:**
- Extract values from `echo.Context` (path params, query params, user ID, request body)
- Bind and validate request DTOs
- Call **one** service method
- Map service errors to HTTP status codes
- Convert domain models to response DTOs
- Log bind/validation errors

**Forbidden:**
- Business logic of any kind (calculations, conditional branching based on domain rules, data enrichment)
- Direct database or repository calls
- Calling multiple service methods to compose a result (that composition belongs in the service)
- Passing `echo.Context` to services

**Typical handler method flow:**
```
extract user ID → bind request → validate → build service input → call service → handle error or convert to DTO → return JSON
```

### Services (`internal/services/`)

Services contain **all business logic**.

**Allowed:**
- Input validation (domain-level: "does this currency exist?", "does the user own this resource?")
- Orchestrating multiple repository calls
- Data transformation and calculations
- Defining and returning sentinel/structured errors
- Calling other services through injected interfaces

**Forbidden:**
- Importing `echo` or any HTTP framework package
- Knowing about HTTP status codes, request/response DTOs, or JSON tags
- Direct SQL queries (that belongs in repositories)
- Accessing package-level mutable state

### Repositories (`internal/repositories/`)

Repositories are a **pure data access layer**.

**Allowed:**
- SQL queries (SELECT, INSERT, UPDATE, DELETE)
- Scanning results into domain models
- Building dynamic WHERE clauses from filter structs
- Populating relations via JOINs
- Logging database errors

**Forbidden:**
- Business logic (calculations, validations, conditional domain rules)
- Calling other repositories
- Importing service packages
- Knowing about HTTP or API concerns

---

## Dependency Injection

### Constructor Pattern

Every handler, service, and repository uses the same constructor pattern:

```go
type Service struct {
    accountRepo AccountRepository  // interface
    logger      *slog.Logger
}

func New(accountRepo AccountRepository, logger *slog.Logger) *Service {
    return &Service{
        accountRepo: accountRepo,
        logger:      logger,
    }
}
```

### What to Inject as Interfaces vs Concrete Types

| Dependency | Inject as | Why |
|---|---|---|
| Repositories | Interface | Swappable for testing, keeps layers decoupled |
| Other services (cross-domain) | Interface | Prevents direct package imports between services |
| Logger (`*slog.Logger`) | Concrete | Stable API, no need for abstraction |
| Database (`*sql.DB`) | Concrete (repositories only) | Only repositories talk to the DB |

### Where to Define Interfaces

**Consumer defines the interface** (standard Go practice):

- A service defines the repository interface it needs, inside its own `contract.go` file
- A handler defines the service interface it needs, inside its own `service_interface.go` file
- The `repositories/interfaces.go` file defines shared repository interface types
- The `internal/types/` package defines shared data transfer structs (filters, snapshots)

This prevents import cycles and keeps packages independently compilable.

### Service Package Structure

Every service package follows a canonical file layout:

```
service_name/
├── contract.go       # Public interface, dependency interfaces, input/output types, invariants
├── errors.go         # All sentinel and structured errors
├── service.go        # Implementation (disposable — can be rewritten)
├── contract_test.go  # PERMANENT — black-box tests against ServiceInterface
└── service_test.go   # DISPOSABLE — implementation-level unit tests
```

Some services have additional files for specific concerns (e.g., `rate_cache.go`, `webhook_handler.go`, `occurrences.go`).

#### `contract.go` Pattern

Each service's `contract.go` is the single file that describes the service's full public surface. An LLM or developer reads `contract.go` + `errors.go` to understand the service without loading the implementation.

**What goes in `contract.go`:**
- **`ServiceInterface`** — the public interface (the methods handlers call)
- **Dependency interfaces** — local repository interfaces, cross-service interfaces (only methods the service actually uses)
- **Input/output types** — `CreateParams`, `UpdateParams`, result types, etc.
- **Documented invariants** — as package-level comments

**What does NOT go in `contract.go`:**
- Error sentinels (those go in `errors.go`)
- Implementation logic (stays in `service.go`)
- Private/internal types

**Naming convention:** The service interface is always named `ServiceInterface`.

#### `errors.go` Pattern

All sentinel errors and structured error types for a service live in `errors.go`. Each sentinel uses `serviceerrors.New()` with the appropriate `Kind`. Reference: `payment/errors.go`.

```go
package myservice

import "github.com/go-budget/backend/internal/serviceerrors"

var (
    ErrNotFound     = serviceerrors.New(serviceerrors.NotFound, "entity not found")
    ErrAccessDenied = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
)
```

#### Service Domain Types

Services return their own domain types from `ServiceInterface` methods, not `*models.X`. Each service defines types like `accounts.Account`, `transactions.Transaction` in its `contract.go`.

**Pattern:**
- `ServiceInterface` methods return service-local types (e.g., `*accounts.Account`)
- Repository interfaces inside `contract.go` still use `models.*` types (Option B — pragmatic intermediate step)
- `service.go` contains `toXxx()` conversion methods from `*models.X` to the local domain type
- Handler `service_interface.go` references service domain types, not `models`

**What `models` is used for:** The `models` package is now repository-internal. It is imported only by:
- `repositories/*.go` (repo implementations)
- Service `contract.go` (in repo interface signatures — Option B)
- Service `service.go` (in conversion logic)

Handlers and `ServiceInterface` signatures do NOT import `models`.

#### Shared Data Transfer Types (`internal/types/`)

Pure value structs used by both repositories and services for filter criteria and query results:
- `types.AccountFilters`, `types.TransactionFilters`, `types.PlannedTxFilters` — query filter parameters
- `types.ExchangeRateSnapshot` — exchange rate lookup result
- `types.BudgetTransactionRow` — simplified transaction row for budget recalculation

These are infrastructure types (no business logic, no methods). Both services and repositories import them.

#### Contract Tests

Each service has a `contract_test.go` file — permanent black-box tests that validate `ServiceInterface` behavior:

```
service_name/
├── contract.go        # Public interface, dependency interfaces, types, invariants
├── errors.go          # All sentinel and structured errors
├── service.go         # Implementation (disposable — can be rewritten)
├── contract_test.go   # PERMANENT — black-box tests against ServiceInterface
└── service_test.go    # DISPOSABLE — implementation-level unit tests
```

**Contract test rules:**
- External test package (`package <service>_test`) — true black-box
- Test only through `ServiceInterface` methods
- Use real DB via `testutil` (not mocks for data layer)
- Mock only external services (HTTP APIs, SMTP, etc.)
- Cover every error sentinel from `errors.go` and every invariant from `contract.go`
- These tests survive any implementation rewrite — they are sacred

**Coverage:** 12 of 18 services have `contract_test.go`. Deferred (require complex external mocks): `backup` (K8s), `backup_notify` (task enqueueing), `email` (SMTP), `export` (Excel generation), `payment` (Stripe), `reports` (complex multi-table queries).

#### Cross-Service Type Dependencies

Some services reference types from other services' contracts:
- `financial_planning` imports `planned_transactions.Occurrence` and `currency.RateCache`
- `reports` imports `currency.RateCache` and `currency.ErrNoExchangeRates`

This is acceptable — importing a contract type from a sibling service is the same pattern as importing `ServiceInterface` for DI. The key rule: only import from `contract.go`/`errors.go`, never from `service.go`.

---

## Error Handling

### Structured Error Types (`internal/serviceerrors/`)

All service sentinel errors use `serviceerrors.ServiceError`, a structured error type with a `Kind` field. Handlers use the `Kind` to determine HTTP status codes without importing individual service packages.

**Error Kinds:**

| Kind | HTTP Status | Description |
|------|------------|-------------|
| `NotFound` | 404 | Resource not found |
| `AccessDenied` | 403 | User does not own the resource |
| `Conflict` | 409 | Duplicate or state conflict |
| `InvalidInput` | 422 | Validation failure |
| `Unauthorized` | 401 | User not activated or not authenticated |
| `ProviderError` | 502 | External service failure |
| `LimitExceeded` | 403 | Plan or resource limit exceeded |
| `NoChange` | 409 | Operation would have no effect |
| `Other` | 500 | Uncategorized/system error |

**Service error definition pattern (`errors.go`):**
```go
var ErrAccountNotFound = serviceerrors.New(serviceerrors.NotFound, "account not found")
var ErrAccessDenied    = serviceerrors.New(serviceerrors.AccessDenied, "access denied")
```

**Level 2 — Structured errors (domain, when extra context is needed):**
```go
type DateError struct {
    Date      string
    Operation string
    Err       error
}

func (e *DateError) Error() string { ... }
func (e *DateError) Unwrap() error { return e.Err }
```

**Level 3 — Database/infrastructure errors (technical):**
Raw errors from `database/sql`, HTTP clients, etc.

### Error Flow Between Layers

```
Repository → returns raw DB error (e.g., sql.ErrNoRows)
Service    → converts to domain error (e.g., ErrAccountNotFound), or bubbles up unexpected errors
Handler    → maps domain error Kind to HTTP status code, logs unexpected errors, returns generic message to client
```

**Rules:**
- Services MUST convert `sql.ErrNoRows` to a domain sentinel error. Never let database errors leak to handlers as-is.
- Handlers MUST NOT expose internal error messages to clients. Log the real error, return a generic message.
- Use `serviceerrors.GetKind()` or `serviceerrors.IsKind()` for Kind-based error matching in handlers.
- Use `errors.As()` for structured errors like `DateError`.
- Use `serviceerrors.Message()` when multiple errors of the same Kind need different client messages.
- Unexpected errors (not matching any Kind) -> log + return 500 with generic message.

### Reusable Error Handlers

When a handler has many methods with the same error mapping, extract a `handleServiceError` helper method on the handler struct:

```go
func (h *Handler) handleServiceError(c echo.Context, err error) error {
    switch serviceerrors.GetKind(err) {
    case serviceerrors.NotFound:
        return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Not found"})
    case serviceerrors.AccessDenied:
        return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Access denied"})
    default:
        h.logger.Error("operation failed", "error", err)
        return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Internal error"})
    }
}
```

### Error Response Body Shape (`detail` + optional `errorCode`/`params`)

`common.ErrorResponse` is the single error body type returned by every handler. It has three JSON fields:

```go
type ErrorResponse struct {
    Detail    string         `json:"detail"`
    ErrorCode string         `json:"errorCode,omitempty"`
    Params    map[string]any `json:"params,omitempty"`
}
```

- `detail` — human-readable English fallback message. Always present.
- `errorCode` — stable, machine-readable i18n key (e.g. `errors.transaction.labelTooLong`). Optional. Frontends use it to look up a localized message; if the key is missing in the locale file, they fall back to `detail`.
- `params` — optional structured values the frontend interpolates into the localized string (e.g. `{"max": 255}`).

Both `ErrorCode` and `Params` are tagged `omitempty`, so endpoints that have not opted in continue to emit byte-identical JSON (`{"detail": "..."}`).

**Hybrid state — which endpoints emit `errorCode` today?**

The `errorCode` convention is being introduced **incrementally**. As of 2026-05-01, only the following endpoints emit `errorCode` / `params`:

| Endpoint | Emitted error codes |
|----------|---------------------|
| `POST /transactions/` | `errors.transaction.labelTooLong`, `errors.transaction.notesTooLong`, `errors.transaction.validationFailed` |
| `PUT /transactions/` | same as above |

Every other endpoint still returns the legacy `{"detail": "..."}` shape. Both shapes are valid; new endpoints SHOULD adopt the `errorCode` convention going forward, but retrofitting existing endpoints is not required.

**i18n key convention.** Error codes follow `errors.<domain>.<camelCase>`:

- `errors.` — fixed top-level namespace
- `<domain>` — singular noun matching the domain area (`transaction`, not `transactions`)
- `<camelCase>` — short, descriptive identifier (`labelTooLong`, `notesTooLong`, `validationFailed`)

The exact same string is used as both the JSON `errorCode` value AND the lookup key in `web/src/locales/<lang>.json` (placed at the **root** of the locale file, NOT inside the `message` namespace). This 1:1 binding is the contract — never alias or rewrap the key on either side.

**No-internals-leak rule.** The error response body MUST NEVER contain:

- Go struct names (`CreateTransactionRequest`, `UpdateTransactionRequest`, capital-L `Label`, capital-N `Notes`)
- Validator tag literals (`max`, `required`, `omitempty`) as standalone values
- Raw `validator.FieldError.Error()` strings (e.g. `"Field validation for 'Label' failed on the 'max' tag"`)
- The phrase `"Field validation"`, `"failed on the"`, `"Tag="`, `"Field="`

The mapping from a validator failure to an `errorCode` lives **server-side** in the handler (typically a small private helper like `handlers/transactions/validation_errors.go::mapValidationError`). Only the JSON-tag-cased field name (e.g. `label`, `notes`) may appear, and only inside `errorCode` / `detail` as part of a human English sentence.

**How to opt in (recipe for new endpoints).**

1. Add a `mapValidationError` (or equivalent) helper next to the handler that takes `validator.ValidationErrors` and returns `(errorCode string, detail string, params map[string]any)`. Map only the cases you have actual i18n strings for; fall through to a generic `errors.<domain>.validationFailed`.
2. Register the i18n keys in **all** locale files at the same path (`errors.<domain>.<case>`). Run `npm run lint:i18n` to verify parity.
3. Add an integration test that asserts both the positive case (correct `errorCode` + `params`) and the no-leak negative assertions (response body does not contain any of the forbidden substrings above).
4. Document the new error codes in the endpoint's `backend/docs/endpoints/<domain>.md` row.

### Middleware Context Primitives

The `RequireActiveUser` middleware stores primitive values in the echo context instead of `*models.User`:

| Context Key | Type | Description |
|------------|------|-------------|
| `user_id` | `int` | User ID (set by auth middleware) |
| `active_user_email` | `string` | User's email |
| `active_user_base_currency_id` | `int` | User's base currency ID |
| `active_user_display_name` | `string` | User's display name (FirstName + LastName) |

Handlers access these via helpers in `internal/handlers/common/context.go`:
```go
userID := common.ActiveUserID(c)
email := common.ActiveUserEmail(c)
baseCurrencyID := common.ActiveUserBaseCurrencyID(c)
displayName := common.ActiveUserDisplayName(c)
```

---

## DTOs vs Domain Models

### Domain Models (`internal/models/`)

- Represent database entities
- Include all database fields, relations, calculated fields
- Used inside services and repositories
- Never returned directly from HTTP handlers

### Request DTOs (in handler packages)

- Defined in `dto.go` next to the handler
- Include JSON tags and validation tags (`validate:"required"`)
- May include custom `UnmarshalJSON` for flexible parsing (camelCase + snake_case)
- Exist only in the handler layer — services never see them

### Response DTOs (in handler packages)

- Defined in `dto.go` next to the handler
- Use camelCase for JSON fields
- Convert types as needed (`decimal.Decimal` → `float64`, `*time.Time` → `string`)
- Omit internal fields (`is_deleted`, `updated_at`)

### Shared DTOs (`internal/handlers/common/`)

- Reusable across multiple handlers: `ErrorResponse`, `DeleteResponse`, `CurrencyDTO`, etc.

### Conversion

Handlers convert between DTOs and models using private helper methods (`toAccountResponse`, etc.). This conversion logic lives **only in handlers** — services do not know about DTOs.

---

## Service Input Types

Services define their own input structs. Handlers build these from request DTOs.

```go
// In service package
type CreateAccountInput struct {
    Name           string
    CurrencyID     int
    AccountTypeID  int
    InitialBalance decimal.Decimal
}

// In handler
input := accountsservice.CreateAccountInput{
    Name:       req.Name,
    CurrencyID: int(req.CurrencyID),
    // ...
}
account, err := h.service.CreateAccount(userID, input)
```

**Why:** Services must not depend on HTTP request shapes. If the API format changes, only the handler changes.

---

## Context Usage

`echo.Context` is **handler-only**. Services and repositories never receive it.

Handlers extract all needed values and pass them as explicit arguments:

```go
// Handler
userID := c.Get("user_id").(int)
account, err := h.service.GetAccount(userID, accountID)

// Service method signature — no echo dependency
func (s *Service) GetAccount(userID, accountID int) (*models.Account, error)
```

**Why:** This keeps services framework-agnostic and independently testable.

---

## Logging

- **Logger:** `*slog.Logger` (Go standard library, structured JSON output)
- **Injection:** Passed via constructor to all layers
- **Error paths:** Always log the actual error before returning a generic message to the client
- **Debug level:** Framework-level details (JWT validation, middleware decisions)
- **Info level:** Request lifecycle (handled by middleware)
- **Error level:** Failed operations, unexpected errors
- **Never log:** Passwords, tokens, API keys, or other sensitive data
- **Format:** Structured key-value pairs, not string formatting

```go
// Good
h.logger.Error("create account failed", "error", err, "userID", userID)

// Bad
h.logger.Error(fmt.Sprintf("create account failed: %v for user %d", err, userID))
```

---

## Checklist for New Code

Before submitting code, verify:

- [ ] Handlers contain no business logic — only bind, validate, delegate, convert
- [ ] Services contain no HTTP concerns — no `echo` imports, no status codes, no DTOs
- [ ] Repositories contain no business logic — only SQL and model scanning
- [ ] `echo.Context` is not passed beyond handlers
- [ ] `sql.ErrNoRows` is converted to a domain sentinel error in the service, not checked in the handler
- [ ] Internal errors are logged, not exposed to the client
- [ ] Cross-service dependencies are injected as interfaces, not direct imports
- [ ] No unnecessary abstractions — code is as simple as it can be while being correct
