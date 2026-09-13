# Wealth Dashboard — Technical Architecture

**Status:** Active implementation baseline (v1.4)
**Date:** 11 September 2026  
**Companion:** [Product Design & MVP Requirements](product-design.md)

## 1. Architecture objective

Build the application in small vertical slices with explicit domain boundaries and minimal abstractions.

The current account, cash, and property application is implemented end to end. A React/TypeScript frontend uses all of the Go backend's account, cash, property, and health operations through a shared same-origin HTTP boundary. The backend contains `account`, `cash`, `property`, `instrument`, `position`, `currency`, and `money` domain packages plus an operational `health` package. Instrument metadata has create/list/retrieve API operations, and positions have account-scoped set/list operations. Neither has a frontend interface yet.

Storage is still process-local and in memory. Broader architecture for other holdings, valuation, allocation, persistent storage, authentication, or deployment remains deferred until one of those requirements becomes the next vertical slice.

The guiding rule is:

> Introduce a domain concept or architectural abstraction only when the current vertical slice requires it.

## 2. Architectural style

The backend uses a feature-oriented package structure: packages communicate the capability they implement.

```text
internal/account
internal/cash
internal/property
internal/currency
internal/money
internal/health
```

Hexagonal principles apply at the account, cash, and property storage boundaries. Application logic depends on feature-owned repository interfaces rather than concrete in-memory implementations. The codebase does not use a repository-wide `domain/`, `ports/`, `adapters/`, and `infrastructure/` hierarchy; feature cohesion takes priority over architecture-layer folders.

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
│   ├── cash/
│   │   ├── cash.go
│   │   ├── http.go
│   │   ├── repository_memory.go
│   │   ├── service.go
│   │   └── *_test.go
│   ├── property/
│   │   ├── property.go
│   │   ├── http.go
│   │   ├── repository_memory.go
│   │   ├── service.go
│   │   └── *_test.go
│   ├── currency/
│   │   ├── currency.go
│   │   └── currency_test.go
│   ├── health/
│   │   ├── health.go
│   │   └── health_test.go
│   └── money/
│       ├── amount.go
│       └── amount_test.go
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

## 5. Currency and money values

`currency.Code` is an opaque, comparable value supporting USD, SGD, and VND. `Parse` trims and uppercases external values. `MinorUnitDigits` returns two for USD and SGD and zero for VND; a zero or otherwise invalid code returns `currency.ErrUnsupported`.

`money.Amount` combines an exported `Currency` code with an exact, non-negative `int64` minor-unit value. `money.Parse` accepts decimal strings, validates their syntax and currency precision, rejects overflow and negative values, and returns `money.ErrInvalidAmount` for invalid input. `Amount.String` emits the canonical currency-specific representation. The representation and constructors keep floating-point values out of the domain and HTTP boundary.

## 6. Cash balance slice

`cash.Balance` contains an `account.ID` and `money.Amount`. A balance has no generated identity: the in-memory repository keys it by account ID and currency, so `Save` replaces the previous current balance for that pair.

The cash dependency flow is:

```text
HTTP request
    ↓
cash.Handler
    ↓
cash.Service ──→ cash.Repository
    │
    └──────────→ cash.AccountFinder ──→ account.MemoryRepository
```

The service verifies account existence before reading or writing cash. `ListBalances` sorts results lexically by currency code, making the HTTP collection deterministic. The in-memory repository protects its nested account/currency maps with `sync.RWMutex` and returns an allocated empty slice when no balances exist.

The HTTP API is:

| Route | Success behavior |
|---|---|
| `PUT /api/v1/accounts/{id}/cash/{currency}` | `200` with the created or replaced balance |
| `GET /api/v1/accounts/{id}/cash` | `200` with `{"balances": [...]}` |

Amounts are JSON strings. Malformed account IDs, unsupported currencies, malformed requests, and invalid amounts return 400. A missing account returns 404. Unexpected account or cash repository errors return 500, and method-aware routing supplies 405 responses.

### Instrument metadata slice

`instrument.Instrument` contains its own UUID-backed `ID`, validated `Kind`,
trimmed nonempty `Symbol` and `Name`, and `currency.Code` quote currency.
It has no dependency on accounts, money, positions, or pricing. The service
depends on a feature-owned repository with `Save`, `FindByID`, and `List`.
The memory repository protects its flat ID-keyed map with an RWMutex and
returns value snapshots; the service orders lists by canonical UUID.

The server wires an independent instrument repository, service, and handler.
`POST /api/v1/instruments` creates metadata, `GET /api/v1/instruments` lists it,
and `GET /api/v1/instruments/{id}` retrieves it. JSON fields are `id`, `kind`,
`symbol`, `name`, and `quoteCurrency`. Kind values are `stock`, `etf`, `bond`,
`mutual_fund`, and `crypto`; currency input requires exact uppercase USD/SGD/VND.
Creation validates JSON, currency, kind, symbol, then name. Symbols preserve case
and punctuation and need not be unique. IDs require canonical lowercase
hyphenated UUIDs. Missing records return 404; validation errors return 400;
unexpected storage failures return a generic 500. Errors are plain text.

### 6.1 Standalone property slice

`property.Property` contains only its own `ID`, `Name`, and `money.Amount` value. It is a directly owned physical-asset aggregate and has no account ID or account lookup dependency. Accounts remain custody containers for cash and account-held positions.

The property dependency flow is:

```text
HTTP request
    ↓
property.Handler
    ↓
property.Service
    ↓
property.Repository
    ↓
property.MemoryRepository
```

The repository stores a flat map keyed by `property.ID`. `List` returns a snapshot of every property, which the service orders by canonical UUID. `ReplaceValue` holds the repository write lock while replacing the entire `money.Amount`, so currency and minor units change atomically; it never creates a missing property.

The HTTP API is:

| Route | Success behavior |
|---|---|
| `POST /api/v1/properties` | `201` with a generated canonical UUID, trimmed name, and canonical initial value |
| `GET /api/v1/properties` | `200` with every property ordered by canonical UUID |
| `PUT /api/v1/properties/{propertyId}/value` | `200` with the property after atomic currency-and-amount replacement |

Create validation order is JSON, name, value object, currency, then amount. Update validation order is canonical property ID, JSON, currency, amount, then property lookup. Errors remain plain text and unexpected repository failures are not exposed.

A future read-side Asset projection may combine properties, cash, and account-held positions for reporting. There is intentionally no persisted polymorphic base entity, `Asset` interface, generic asset repository, aggregation, or FX behavior in this slice.

## 7. Account slice and dependency flow

The account slice implements create-account, list-accounts, and retrieve-account-by-ID:

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

## 8. Account application service

The implemented service API is:

```go
type Service struct {
    repository Repository
}

func NewService(repository Repository) Service
func (s Service) CreateAccount(ctx context.Context, name string) (Account, error)
func (s Service) GetByID(ctx context.Context, id ID) (Account, error)
func (s Service) ListAccounts(ctx context.Context) ([]Account, error)
```

`CreateAccount` generates an ID, calls `New` to normalize and validate the account, and saves only a valid account. It returns constructor and repository errors unchanged. `GetByID` delegates to the repository and also propagates its result unchanged. `ListAccounts` obtains the complete repository collection and sorts it lexically by canonical account ID so API responses remain deterministic.

The domain constructor accepts an ID separately because constructing or reconstructing a domain entity may need to preserve an existing identity; callers of `Service.CreateAccount` do not supply one.

## 9. Account repository boundary and implementation

The account-owned storage contract is declared in `service.go`:

```go
type Repository interface {
    Save(ctx context.Context, account Account) error
    FindByID(ctx context.Context, id ID) (Account, error)
    List(ctx context.Context) ([]Account, error)
}

var ErrNotFound = errors.New("account not found")
```

`MemoryRepository`, implemented in `repository_memory.go`, is the only current repository. It stores accounts in a map keyed by `account.ID` and protects reads, list snapshots, and writes with `sync.RWMutex`. `FindByID` returns `ErrNotFound` when the key is absent. `List` returns an allocated snapshot, including a non-nil empty slice. Saving the same ID again replaces the stored value.

Storage is process-local and ephemeral. There is a PostgreSQL service in the root `compose.yaml`, but the backend has no PostgreSQL driver, repository implementation, migrations, or database wiring and does not currently use that service.

## 10. Account HTTP adapter

### 10.1 Handler and route registration

HTTP behavior is grouped in `account.Handler`:

```go
type Handler struct {
    service Service
}

func NewHandler(service Service) Handler
func (h Handler) RegisterRoutes(mux *http.ServeMux)
func (h Handler) CreateAccount(w http.ResponseWriter, r *http.Request)
func (h Handler) ListAccounts(w http.ResponseWriter, r *http.Request)
func (h Handler) GetAccountByID(w http.ResponseWriter, r *http.Request)
```

`RegisterRoutes` owns the method-aware Go 1.22-style route patterns:

```text
POST /api/v1/accounts
GET  /api/v1/accounts
GET  /api/v1/accounts/{id}
```

An unsupported method on a matched path is handled by `http.ServeMux` as `405 Method Not Allowed`.

### 10.2 Transport representations

The domain entity has no JSON tags. The HTTP adapter uses private request and response DTOs:

```go
type createAccountRequest struct {
    Name string `json:"name"`
}

type accountResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type accountsResponse struct {
    Accounts []accountResponse `json:"accounts"`
}
```

Successful responses set `Content-Type: application/json`. Account IDs are serialized through `ID.String()`.

### 10.3 Status and error mapping

| Operation or condition | HTTP status | Response |
|---|---:|---|
| account created | 201 | account JSON |
| accounts listed | 200 | `{"accounts": [...]}` ordered by canonical ID |
| no accounts | 200 | `{"accounts": []}` |
| malformed create JSON | 400 | `invalid request` |
| empty or whitespace-only name (`ErrInvalidName`) | 400 | `account name cannot be empty` |
| unexpected save error | 500 | `internal server error` |
| account found | 200 | account JSON |
| malformed account ID | 400 | `invalid account ID` |
| missing account (`ErrNotFound`) | 404 | `account not found` |
| unexpected lookup error | 500 | `internal server error` |
| unexpected list error | 500 | `internal server error` |

Error responses use `http.Error`, so they are plain text. Repository details are not exposed for unexpected failures.

## 11. Health adapter

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

## 12. Composition root and process lifecycle

`cmd/server/main.go` assembles concrete dependencies:

```go
accountRepository := account.NewMemoryRepository()
accountService := account.NewService(accountRepository)
accountHandler := account.NewHandler(accountService)
cashRepository := cash.NewMemoryRepository()
cashService := cash.NewService(accountRepository, cashRepository)
cashHandler := cash.NewHandler(cashService)
propertyRepository := property.NewMemoryRepository()
propertyService := property.NewService(propertyRepository)
propertyHandler := property.NewHandler(propertyService)
healthHandler := health.NewHandler()

router := http.NewServeMux()
healthHandler.RegisterRoutes(router)
accountHandler.RegisterRoutes(router)
cashHandler.RegisterRoutes(router)
propertyHandler.RegisterRoutes(router)
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

## 13. Testing strategy and current coverage

Tests use only the Go standard library. Table-driven tests are used where several cases share the same behavior.

The package choice reflects the level under test:

- `account_test` exercises the exported domain constructor and in-memory repository as an external consumer;
- `account` tests the service and HTTP adapter with package-private stubs and transport types;
- `cash_test` exercises the in-memory cash repository as an external consumer;
- `cash` tests service coordination and the HTTP adapter with package-private stubs;
- `property` tests its standalone aggregate, service and HTTP adapter, flat repository, and concurrency behavior;
- `currency_test` and `money_test` exercise their value types as external consumers;
- `health_test` exercises the health package through its exported API.

Current tests cover:

- valid, trimmed, empty, and whitespace-only account names;
- preservation of the supplied ID by `account.New`;
- in-memory save, lookup, list snapshots, and not-found behavior;
- service validation before save and repository-error propagation;
- create, list, and retrieve through the registered HTTP routes;
- deterministic account ordering and an allocated empty account collection;
- malformed JSON, invalid names, malformed IDs, missing accounts, repository failures, and unsupported account methods;
- currency normalization and minor-unit precision;
- exact amount parsing, canonical formatting, invalid values, negatives, excessive precision, and overflow;
- cash replacement, account isolation, empty collections, and concurrent repository access;
- account validation, deterministic currency ordering, and cash dependency-error propagation;
- create-account, set-and-replace balances, and list-balances through registered HTTP routes;
- cash HTTP validation, missing-account behavior, repository failures, and unsupported methods;
- standalone property creation, empty and ordered lists, duplicate names, canonical IDs, exact values, atomic concurrent replacement, missing properties, validation precedence, repository failures, and method behavior;
- normal liveness/readiness, readiness during shutdown, and unsupported health methods.

The frontend uses Bun's test runner for API-client failure behavior and exact money/name validation. Its integration script imports the same API client as React and runs it through the Vite proxy against the real Go server. That check exercises all ten operations, including account discovery, empty collections, amount normalization, cash and property replacement, exact numeric bounds, duplicate property names, and representative 400/404 responses.

`NewID` and `ParseID` are exercised indirectly by HTTP and domain tests; they do not currently have dedicated tests.

## 14. Frontend and end-to-end integration

The frontend is a React 19 and TypeScript 6 single-page application built by Vite 8 and managed with Bun. It is intentionally small: `App.tsx` composes sibling account and property workspace regions, account creation and selection, selected-account cash details, and backend connection status without a router or global state library.

```text
React components
    ↓
useResource ──→ loading, retained-data, error, and reload state
    ↓
API client ──→ /api/v1, /livez, /readyz
    ↓
Vite/reverse proxy
    ↓
Go HTTP handlers
```

`api.ts` owns the ten browser operations and the distinction between transport errors and status-bearing API errors. Requests have a ten-second timeout and are not retried automatically. Product data is loaded from the backend; the frontend does not keep a second durable store. `useResource` ignores results after a component or selected-account load becomes inactive, which prevents a late response from replacing newer state. The property region is mounted independently of account selection, so properties load with no account and account switching preserves property data and drafts.

`money.ts` mirrors the backend's whitespace, syntax, currency-precision, and signed-`int64` limit checks so invalid balances can be rejected before a request. Amounts remain strings throughout the form and API client. The backend remains authoritative and repeats all validation.

During development and preview, Vite proxies `/api`, `/livez`, and `/readyz` to `http://localhost:8080` by default; `API_PROXY_TARGET` can override that target. The browser therefore uses origin-relative URLs and the Go server does not currently need CORS handling. A static production build requires the deployment host or reverse proxy to provide equivalent routing because Vite's proxy is not embedded in built assets.

The frontend/backend integration is complete for the current account, cash, and standalone property scope. It does not imply that the larger wealth-dashboard reporting domain is implemented.

## 15. Decisions intentionally deferred

The backend does not yet fix an architecture for:

- persistent repository wiring, migrations, or a SQL schema;
- `country.Code`;
- account type, institution metadata, or retirement classification;
- instrument management in the frontend and holdings;
- the read-side Asset projection that may combine property, cash, and future positions;
- valuation and FX;
- portfolio grouping and allocation calculations;
- authentication and authorization;
- background workers and market-data integrations;
- runtime configuration beyond the current constants;
- production deployment.

## 16. Current baseline and next change

The account create/list/retrieve, cash set/list, and standalone property create/list/revalue vertical slices, their React interface, shared HTTP contract, in-memory repositories, operational health checks, graceful shutdown, and end-to-end integration are implemented and tested.

The next phase has two legitimate architectural directions:

1. **Database integration:** implement durable account, cash, and property repositories, migrations, connection lifecycle, configuration, and dependency-aware readiness while keeping feature-owned repository interfaces and the existing HTTP contract stable.
2. **Domain expansion:** select the smallest useful wealth workflow beyond current cash and property, then add only the domain types, API operations, and UI needed for that vertical slice. Likely candidates include position valuation or the read-side Asset reporting projection and require explicit decisions about instruments, valuation, FX, or classification before implementation.

The PostgreSQL service in `compose.yaml` is only preparatory infrastructure today; no driver, schema, migration, database repository, or server wiring exists. Until persistence is selected and implemented, all application data is lost when the Go process restarts.

## Current position slice

The backend supports current holdings of investment instruments in accounts through
`PUT /api/v1/accounts/{id}/positions/{instrumentID}` and
`GET /api/v1/accounts/{id}/positions`. Each account/instrument pair identifies one
position; PUT creates or replaces its quantity. Both references must exist.
Zero holdings remain visible. Lists are sorted by instrument UUID and empty
accounts return an empty array; unknown accounts return 404.

`position.Quantity` stores an exact nonnegative decimal without floating-point
conversion or currency-specific precision limits. Its zero value represents zero.
Input accepts whole and fractional digits, trims whitespace, and normalizes
redundant zeroes. Negative values, signs, exponents, and malformed decimals fail
validation. There is no fixed magnitude or precision limit.

The feature owns its repository interface and uses narrow account and instrument
lookup interfaces. In-memory storage is keyed by account and instrument, protected
by a mutex, and returns detached list snapshots. Writes are idempotent and the last
completed save wins. HTTP responses include accountId, instrumentId, and quantity.
Pricing, valuation, transactions, cash adjustments, and a frontend interface for
instruments and positions remain deferred.
