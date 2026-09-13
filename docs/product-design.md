# Wealth Dashboard — Product Design & MVP Requirements

**Status:** Active design baseline (v1.4)
**Date:** 11 September 2026  
**Companion:** [Technical Architecture](architecture.md)

## 1. Product summary

Wealth Dashboard is a personal finance application for understanding total net worth and asset allocation across accounts, countries, currencies, and asset types.

The long-term product goal is to provide a bird's-eye view of wealth while allowing progressively finer exploration. A user should eventually be able to move from a macro allocation such as Cash, Bonds, Equities, Property, and Crypto down to dimensions such as currency, instrument, and account.

The product is analytical. It is not intended to perform tax calculations, execute trades, or provide automated investment advice.

## 2. Product goals

The application should eventually help answer questions such as:

- What is my total net worth in a selected reporting currency?
- How is my wealth divided between major asset classes?
- How much exposure do I have to USD, SGD, VND, or other currencies?
- Which instruments make up my equity or crypto allocation?
- Which accounts contain those assets?
- How much of my wealth is retirement versus non-retirement?
- When has an allocation moved far enough from a target that I should review it?

The allocation explorer remains the central product idea. The same underlying wealth should eventually be viewable through different dimensions without double counting.

## 3. Product principles

### 3.1 Build the domain incrementally

The domain model will not be designed exhaustively up front. New domain concepts are introduced only when a concrete product requirement requires them.

The implemented application currently models custody accounts, current cash balances, directly owned properties, and instrument metadata. Accounts, cash, and properties are wired end to end: the React frontend uses the Go HTTP API to manage accounts and cash and, independently, to create, list, and revalue properties. Instrument creation, listing, and retrieval are available through the backend API. Concepts such as countries, positions, portfolio groupings, aggregate reporting, FX, and rebalancing rules remain deferred until a vertical slice requires them.

### 3.2 Avoid premature classifications

An account should not acquire fields merely because they may be useful later. For example, the current model does not yet contain institution, country, currency, retirement status, or account type.

Those concepts will be evaluated independently when a real use case requires them.

### 3.3 Preserve domain meaning

When new concepts are added, domain types should express meaningful distinctions rather than relying on loosely typed primitive values. However, dedicated types should only be introduced when they protect a real invariant or clarify an actual requirement.

### 3.4 Prefer a useful vertical slice over broad scaffolding

Each implementation step should produce one complete behavior through the application's layers before adding another domain area.

## 4. Current domain model

### 4.1 Account

`Account` is the custody-container aggregate introduced by the first vertical slice.

An account currently contains two exported fields:

| Field | Meaning |
|---|---|
| `ID` | Stable identity of the account |
| `Name` | Human-readable account name |

Example:

```text
Account
├── ID   = <generated account ID>
└── Name = "HSBC Premier"
```

### 4.2 Account identity

Account identity is represented by the package-specific sentinel type `account.ID`.

The public behavior is:

- `account.NewID()` creates a new ID.
- `account.ParseID(string)` reconstructs an ID from its external string representation.
- `ID.String()` exposes the external representation.

The current implementation uses UUIDs internally. The UUID implementation remains encapsulated by the `account` package; consumers work with `account.ID`, not `uuid.UUID`.

### 4.3 Account name

An account name is represented as a string rather than a separate value type.

Accounts should be created with `account.New`, which owns the current name invariant:

- leading and trailing whitespace is removed;
- an empty name is rejected;
- a whitespace-only name is rejected.

There is currently no requirement for name uniqueness.

### 4.4 Currency

`currency.Code` represents one of the currencies currently supported by the product: USD, SGD, or VND. Parsing trims whitespace and normalizes case. Each valid code also defines the number of minor-unit digits used for exact monetary amounts: two for USD and SGD, and zero for VND.

### 4.5 Money amount

`money.Amount` pairs a currency with a non-negative quantity stored as integer minor units. Amounts enter the application as decimal strings, avoiding floating-point ambiguity. Parsing enforces the currency's supported precision, rejects malformed or overflowing values, and produces a canonical string representation.

### 4.6 Cash balance

A cash balance represents the current amount of one currency held in an account. Its natural identity is the combination of account ID and currency; it has no independent generated ID. Setting the same account and currency again replaces the previous amount, including when the new amount is zero.

### Instrument metadata

An instrument describes an investment vehicle independently of ownership and
valuation. It contains a server-generated `instrument.ID`, kind (`stock`, `etf`,
`bond`, `mutual_fund`, or `crypto`), a trimmed nonempty symbol and name, and a
`currency.Code` quote currency using the existing USD/SGD/VND support. Symbols
preserve case and punctuation. Duplicate symbols and names are allowed because
the UUID provides identity; exchange-specific deduplication is not implemented.

The backend supports `POST /api/v1/instruments`, `GET /api/v1/instruments`, and
`GET /api/v1/instruments/{id}`. Lists are ordered by canonical UUID and include
an explicit empty array. IDs require lowercase hyphenated UUIDs. Storage is
process-local and in memory, and the frontend does not yet expose this catalog.
This slice contains no account association, position, quantity, price, or value.

### 4.7 Property

A property is a standalone physical-asset aggregate, not a member of an account. It contains a stable, server-generated `property.ID`, a trimmed nonempty name, and one current manual `money.Amount` value. Duplicate names are allowed because identity is independent of display name.

The value is the estimated gross market value of the user's owned share before mortgages or other liabilities. Revaluation replaces currency and amount together; changing currency is a new estimate, not an FX conversion. Zero remains an explicit value.

Accounts continue to contain cash and will likely provide custody for future account-held positions. A future read-side Asset projection may combine properties, cash, and positions for reporting. This revision does not define an `Asset` interface, generic asset persistence, aggregation, or FX behavior.

## 5. Account vertical slice

The first backend vertical slice supports the account workflow required by the frontend:

> Discover accounts, create an account, and retrieve an account by ID.

The supported operations are:

```text
POST /api/v1/accounts
GET  /api/v1/accounts
GET  /api/v1/accounts/{id}
```

### 5.1 Create account

Input:

```json
{
  "name": "  HSBC Premier  "
}
```

Expected behavior:

1. The HTTP layer decodes the request.
2. `Service.CreateAccount` generates an `account.ID`.
3. The account domain constructor validates and normalizes the name.
4. The account is stored through the account repository boundary.
5. The created account is returned with `201 Created`.

Expected representation:

```json
{
  "id": "<uuid>",
  "name": "HSBC Premier"
}
```

### 5.2 List accounts

For:

```text
GET /api/v1/accounts
```

Expected behavior:

- every account in the current workspace is returned;
- accounts are ordered lexically by canonical UUID;
- an empty workspace returns `{"accounts": []}` rather than `null`;
- an unexpected repository error returns `500 Internal Server Error`.

The complete, unpaginated collection is appropriate for the current small personal workspace. Pagination, filtering, and sorting options are not yet required.

### 5.3 Retrieve account by ID

For:

```text
GET /api/v1/accounts/{id}
```

Expected behavior:

- a syntactically invalid ID returns `400 Bad Request`;
- a valid existing ID returns `200 OK` and the account representation;
- a valid missing ID returns `404 Not Found`;
- an unexpected application or repository error returns `500 Internal Server Error`.

### 5.4 Request errors and methods

The implemented HTTP behavior also includes:

- malformed create-account JSON returns `400 Bad Request`;
- an empty or whitespace-only account name returns `400 Bad Request`;
- an unexpected error while saving an account returns `500 Internal Server Error`;
- unsupported methods on a matched account route return `405 Method Not Allowed`.

Account data is currently stored only in memory. It survives requests within one server process but is lost when that process stops.

### 5.5 Operational health

The backend exposes two unversioned operational endpoints:

```text
GET /livez
GET /readyz
```

Both return `200 OK` during normal operation. When graceful shutdown begins, liveness continues to return `200 OK` while readiness returns `503 Service Unavailable`, allowing traffic to drain before the HTTP server stops.

## 6. Cash vertical slice

The second backend vertical slice supports this workflow:

> Set current cash balances for an account and retrieve all of that account's cash balances.

The supported operations are:

```text
PUT /api/v1/accounts/{id}/cash/{currency}
GET /api/v1/accounts/{id}/cash
```

The PUT request accepts an exact decimal string:

```json
{
  "amount": "1234.56"
}
```

It creates or replaces the current balance and returns `200 OK` with the canonical amount. The GET operation returns balances ordered by currency and uses an empty JSON array when an existing account has no balances. Both operations return 404 for a missing account. Invalid account IDs, unsupported currencies, and invalid amounts return 400; unexpected repository failures return 500.

Cash data, like account data, is held in memory and is lost when the server stops. Negative cash is excluded because this slice models assets; overdrafts and other liabilities remain deferred.

### 6.1 Property vertical slice

The property slice supports a directly owned physical asset independently of account existence or selection:

```text
POST /api/v1/properties
GET  /api/v1/properties
PUT  /api/v1/properties/{propertyId}/value
```

Create accepts a trimmed, nonempty name and initial exact value and returns a generated canonical UUID. Listing returns every property ordered by canonical UUID, including an explicit empty array. Value replacement atomically changes currency and amount without changing identity or name and returns 404 for a valid missing property ID. Property IDs accept only canonical lowercase, hyphenated UUIDs.

Create validates JSON, name, value object, currency, then amount. Update validates property ID, JSON, currency, amount, then property existence. Errors are plain text, writes are not retried automatically, and storage is process-local and in memory.

## 7. Completed frontend integration

The frontend and backend are fully connected for the implemented account, cash, and property scope. On application load, the frontend checks backend liveness/readiness and requests the account and property collections independently. A user can then:

1. create an account and immediately select it;
2. reload and select any account returned by the backend;
3. retrieve the selected account and its current cash balances;
4. create or replace a USD, SGD, or VND balance;
5. reload balances and observe the backend's canonical amount representation.
6. create, list, reload, and revalue properties without creating or selecting an account.

The browser client uses the shared HTTP contract for all ten implemented operations: three account operations, two cash operations, three property operations, and two health operations. Vite proxies `/api`, `/livez`, and `/readyz` to the Go server during local development and preview. A deployed frontend likewise requires an origin or reverse proxy that sends those paths to the backend.

The interface handles loading, empty, unavailable-backend, missing-account, validation, and server-error states. It retains user input after failed writes, avoids automatic write retries, and prevents late responses for a previously selected account from replacing the current view. Monetary values remain decimal strings throughout the browser and API; the client validates the same currency precision and signed-`int64` bounds as the backend.

Accounts and Properties are sibling workspace regions. Account selection affects only account details and cash; property data and in-progress drafts remain mounted and intact when selection changes. The current UI does not calculate a combined balance, net worth, FX conversion, or allocation because the read-side projection does not yet exist.

## 8. Current delivered baseline

The application currently provides and tests:

- account creation, discovery, selection, and retrieval;
- deterministic account, cash, and property collections, including explicit empty arrays;
- current cash creation and replacement in SGD, USD, and VND;
- standalone property creation and atomic current-value replacement in SGD, USD, and VND;
- exact string-based monetary validation and canonical formatting;
- end-to-end frontend calls through the Vite proxy to the real Go server;
- visible connection, loading, validation, recovery, and empty states;
- liveness and readiness behavior, including readiness changes during graceful shutdown;
- in-memory, concurrency-safe repositories for accounts, cash, and property.

The frontend/backend boundary for this scope is complete. Persistence and a broader wealth domain are not.

## 9. Explicitly deferred product concepts

The following are part of the broader product direction but are **not part of the current domain model**:

- account institution;
- account country or jurisdiction;
- account type;
- retirement/non-retirement classification;
- account-level display or base currency;
- country sentinel types;
- instrument management in the frontend and account-held securities positions;
- holdings or positions;
- prices and FX rates;
- a cross-asset classification system;
- liabilities;
- an Asset reporting projection, aggregate totals, and FX conversion;
- portfolio groups or sleeves;
- allocation explorer calculations;
- rebalancing targets;
- historical snapshots;
- persistent account, cash, and property storage (the current repositories are in-memory);
- authentication and deployment.

Their presence in the long-term product vision does not imply a particular future data model.

## 10. Next product decision

With the current frontend and backend fully wired, frontend integration is no longer a candidate next step. The next phase should choose one of two directions:

1. **Durable persistence.** Replace or supplement the in-memory repositories with database-backed account, cash, and property repositories, define migrations and schema ownership, wire the database into startup/readiness/shutdown, and preserve the existing HTTP behavior.
2. **A more useful wealth domain.** Add the smallest end-to-end capability beyond cash and property that moves the product toward a wealth overview—for example, an instrument/position slice or the read-side Asset projection. This path must first settle only the ownership, valuation, classification, and FX decisions needed by that slice.

These paths can eventually converge, but the next story should have one primary outcome. Database integration improves durability without expanding what the product can express; domain expansion improves usefulness while data remains ephemeral unless persistence is addressed alongside it.

## 11. MVP direction beyond the current application

The intended MVP remains a useful personal wealth overview application. A later MVP should support enough concepts to show multiple assets across accounts and provide meaningful allocation exploration.

The exact sequence is intentionally not fixed beyond the completed account/cash/property application. Each new slice must earn any new domain concepts it introduces.

## 12. Non-goals

The product is not currently intended to become:

- a tax engine;
- a brokerage or trading interface;
- a transaction-ledger accounting system;
- a budgeting application;
- an automated financial adviser;
- a bank-credential aggregation service.

These boundaries should remain unless the product direction is deliberately changed later.
