# Wealth Dashboard — Technical Architecture

**Status:** Active implementation baseline (v1.2)  
**Date:** 11 September 2026  
**Companion:** [Product Design & MVP Requirements](product-design.md)

## 1. Architecture objective

Build the application in small vertical slices with explicit domain boundaries and minimal abstractions.

The implemented backend contains an `account` feature slice and an operational `health` package. Broader architecture for cash, holdings, valuation, allocation, persistent storage, or deployment remains deferred until those requirements are implemented.

The guiding rule is:

> Introduce a domain concept or architectural abstraction only when the current vertical slice requires it.

## 2. Architectural style

The backend uses a feature-oriented package structure: packages communicate the capability they implement.

```text
internal/account
internal/health
```

Hexagonal principles apply at the account storage boundary. Account application logic depends on the account-owned `Repository` interface rather than the concrete in-memory implementation. The codebase does not use a repository-wide `domain/`, `ports/`, `adapters/`, and `infrastructure/` hierarchy; feature cohesion takes priority over architecture-layer folders.

## 3. Current backend structure

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── account/
│   │   ├── account.go
│   │   ├── http.go
│   │   ├── repository_memory.go
│   │   ├── service.go
│   │   └── *_test.go
│   └── health/
│       ├── health.go
│       └── health_test.go
├── go.mod
└── go.sum
```

The module path is `github.com/mgwinsor/meridian/backend`. It currently declares Go 1.27.1 and depends directly on `github.com/google/uuid` v1.6.0.

## 4. Account domain model

### 4.1 Account entity

The domain entity is intentionally minimal:

```go
type Account struct {
    ID   ID
    Name string
}

func New(id ID, name string) (Account, error)
```

`ID` and `Name` are exported fields; there are no accessor methods. Code should use `New` when creating an account so the name invariant is applied. The constructor trims surrounding whitespace and returns `ErrInvalidName` for an empty or whitespace-only name.

There are no fields for institution, country, currency, account type, retirement classification, or other future metadata.

### 4.2 Account ID

Account identity is a package-specific wrapper around `uuid.UUID`:

```go
type ID struct {
    value uuid.UUID
}

func NewID() ID
func ParseID(value string) (ID, error)
func (id ID) String() string
```

The wrapped value is unexported, so consumers use `account.ID` without depending directly on the UUID library. `NewID` generates a UUID, `ParseID` reconstructs an ID from a supported UUID string, and `String` produces its external representation.

There is no shared global ID type. A future entity can define a distinct identity type even if it uses the same underlying representation.

## 5. Account slice and dependency flow

The account slice implements create-account and retrieve-account-by-ID:

```text
HTTP request
    ↓
account.Handler
    ↓
account.Service
    ↓
account.Repository
    ↓
account.MemoryRepository
```

The HTTP adapter converts transport values and errors, the service coordinates use cases, and the repository owns storage. Request contexts flow from HTTP through the service to repository calls.

## 6. Application service

The implemented service API is:

```go
type Service struct {
    repository Repository
}

func NewService(repository Repository) Service
func (s Service) CreateAccount(ctx context.Context, name string) (Account, error)
func (s Service) GetByID(ctx context.Context, id ID) (Account, error)
```

`CreateAccount` generates an ID, calls `New` to normalize and validate the account, and saves only a valid account. It returns constructor and repository errors unchanged. `GetByID` delegates to the repository and also propagates its result unchanged.

The domain constructor accepts an ID separately because constructing or reconstructing a domain entity may need to preserve an existing identity; callers of `Service.CreateAccount` do not supply one.

## 7. Repository boundary and implementation

The account-owned storage contract is declared in `service.go`:

```go
type Repository interface {
    Save(ctx context.Context, account Account) error
    FindByID(ctx context.Context, id ID) (Account, error)
}

var ErrNotFound = errors.New("account not found")
```

`MemoryRepository`, implemented in `repository_memory.go`, is the only current repository. It stores accounts in a map keyed by `account.ID` and protects reads and writes with `sync.RWMutex`. `FindByID` returns `ErrNotFound` when the key is absent. Saving the same ID again replaces the stored value.

Storage is process-local and ephemeral. There is a PostgreSQL service in the root `compose.yaml`, but the backend has no PostgreSQL driver, repository implementation, migrations, or database wiring and does not currently use that service.

## 8. Account HTTP adapter

### 8.1 Handler and route registration

HTTP behavior is grouped in `account.Handler`:

```go
type Handler struct {
    service Service
}

func NewHandler(service Service) Handler
func (h Handler) RegisterRoutes(mux *http.ServeMux)
func (h Handler) CreateAccount(w http.ResponseWriter, r *http.Request)
func (h Handler) GetAccountByID(w http.ResponseWriter, r *http.Request)
```

`RegisterRoutes` owns the method-aware Go 1.22-style route patterns:

```text
POST /api/v1/accounts
GET  /api/v1/accounts/{id}
```

An unsupported method on a matched path is handled by `http.ServeMux` as `405 Method Not Allowed`.

### 8.2 Transport representations

The domain entity has no JSON tags. The HTTP adapter uses private request and response DTOs:

```go
type createAccountRequest struct {
    Name string `json:"name"`
}

type accountResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

Successful responses set `Content-Type: application/json`. Account IDs are serialized through `ID.String()`.

### 8.3 Status and error mapping

| Operation or condition | HTTP status | Response |
|---|---:|---|
| account created | 201 | account JSON |
| malformed create JSON | 400 | `invalid request` |
| empty or whitespace-only name (`ErrInvalidName`) | 400 | `account name cannot be empty` |
| unexpected save error | 500 | `internal server error` |
| account found | 200 | account JSON |
| malformed account ID | 400 | `invalid account ID` |
| missing account (`ErrNotFound`) | 404 | `account not found` |
| unexpected lookup error | 500 | `internal server error` |

Error responses use `http.Error`, so they are plain text. Repository details are not exposed for unexpected failures.

## 9. Health adapter

The `health` package owns liveness and readiness independently of the versioned product API:

```go
func NewHandler() *Handler
func (h *Handler) RegisterRoutes(mux *http.ServeMux)
func (h *Handler) BeginShutdown()
```

It registers:

| Route | Normal operation | During shutdown |
|---|---:|---:|
| `GET /livez` | 200 `OK` | 200 `OK` |
| `GET /readyz` | 200 `OK` | 503 `shutting down` |

Responses use `text/plain; charset=utf-8`. Shutdown state is held in an `atomic.Bool`, allowing concurrent health requests to observe the transition safely. Unsupported methods on these routes receive HTTP 405 from `http.ServeMux`.

## 10. Composition root and process lifecycle

`cmd/server/main.go` assembles concrete dependencies:

```go
accountRepository := account.NewMemoryRepository()
accountService := account.NewService(accountRepository)
accountHandler := account.NewHandler(accountService)
healthHandler := health.NewHandler()

router := http.NewServeMux()
healthHandler.RegisterRoutes(router)
accountHandler.RegisterRoutes(router)
```

The server currently:

- listens on the hard-coded address `:8080`;
- identifies its environment as the hard-coded value `dev` in startup logging;
- writes structured JSON logs to standard output with `log/slog`;
- listens for `SIGINT` and `SIGTERM`;
- marks readiness unavailable when shutdown begins;
- waits five seconds for load-balancer deregistration;
- gives `http.Server.Shutdown` ten seconds to drain active connections;
- exits with status 1 on an unexpected serve error or failed graceful shutdown.

Liveness remains available during the drain period. Port, environment, deregistration delay, and drain timeout are compile-time constants rather than configuration inputs.

## 11. Testing strategy and current coverage

Tests use only the Go standard library. Table-driven tests are used where several cases share the same behavior.

The package choice reflects the level under test:

- `account_test` exercises the exported domain constructor and in-memory repository as an external consumer;
- `account` tests the service and HTTP adapter with package-private stubs and transport types;
- `health_test` exercises the health package through its exported API.

Current tests cover:

- valid, trimmed, empty, and whitespace-only account names;
- preservation of the supplied ID by `account.New`;
- in-memory save, lookup, and not-found behavior;
- service validation before save and repository-error propagation;
- create-then-retrieve through the registered HTTP routes;
- malformed JSON, invalid names, malformed IDs, missing accounts, repository failures, and unsupported account methods;
- normal liveness/readiness, readiness during shutdown, and unsupported health methods.

`NewID` and `ParseID` are exercised indirectly by HTTP and domain tests; they do not currently have dedicated tests.

## 12. Decisions intentionally deferred

The backend does not yet fix an architecture for:

- persistent repository wiring, migrations, or a SQL schema;
- `currency.Code` or `country.Code`;
- money or decimal value types;
- cash balances;
- account type, institution metadata, or retirement classification;
- instruments and holdings;
- valuation and FX;
- portfolio grouping and allocation calculations;
- authentication and authorization;
- background workers and market-data integrations;
- runtime configuration beyond the current constants;
- production deployment.

The repository contains a frontend scaffold, but frontend product architecture and integration with the account API remain outside the implemented backend slice.

## 13. Current baseline and next change

The account create/retrieve vertical slice, in-memory repository, HTTP routing, operational health checks, and graceful shutdown are implemented and tested. The next product story should determine the next domain concept or infrastructure boundary rather than expanding the model speculatively.
