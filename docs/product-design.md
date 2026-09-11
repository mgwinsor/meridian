# Wealth Dashboard — Product Design & MVP Requirements

**Status:** Active design baseline (v1.3)
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

The implementation currently models accounts and current cash balances. Concepts such as countries, instruments, positions, portfolio groupings, valuations, and rebalancing rules remain deferred until a vertical slice requires them.

### 3.2 Avoid premature classifications

An account should not acquire fields merely because they may be useful later. For example, the current model does not yet contain institution, country, currency, retirement status, or account type.

Those concepts will be evaluated independently when a real use case requires them.

### 3.3 Preserve domain meaning

When new concepts are added, domain types should express meaningful distinctions rather than relying on loosely typed primitive values. However, dedicated types should only be introduced when they protect a real invariant or clarify an actual requirement.

### 3.4 Prefer a useful vertical slice over broad scaffolding

Each implementation step should produce one complete behavior through the application's layers before adding another domain area.

## 4. Current domain model

### 4.1 Account

`Account` is the root entity introduced by the first vertical slice.

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

## 5. First vertical slice

The first complete vertical slice supports one small workflow:

> Create an account through the HTTP API and retrieve that account by ID.

The supported operations are:

```text
POST /api/v1/accounts
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

### 5.2 Retrieve account by ID

For:

```text
GET /api/v1/accounts/{id}
```

Expected behavior:

- a syntactically invalid ID returns `400 Bad Request`;
- a valid existing ID returns `200 OK` and the account representation;
- a valid missing ID returns `404 Not Found`;
- an unexpected application or repository error returns `500 Internal Server Error`.

### 5.3 Request errors and methods

The implemented HTTP behavior also includes:

- malformed create-account JSON returns `400 Bad Request`;
- an empty or whitespace-only account name returns `400 Bad Request`;
- an unexpected error while saving an account returns `500 Internal Server Error`;
- unsupported methods on a matched account route return `405 Method Not Allowed`.

Account data is currently stored only in memory. It survives requests within one server process but is lost when that process stops.

### 5.4 Operational health

The backend exposes two unversioned operational endpoints:

```text
GET /livez
GET /readyz
```

Both return `200 OK` during normal operation. When graceful shutdown begins, liveness continues to return `200 OK` while readiness returns `503 Service Unavailable`, allowing traffic to drain before the HTTP server stops.

## 6. Implemented first-slice capabilities

The backend currently provides and tests the following behavior:

- an account can be constructed with a generated `account.ID` and valid name;
- surrounding whitespace is removed from the account name;
- empty and whitespace-only names are rejected;
- account IDs can be generated and parsed through the public package API;
- an HTTP request can create an account;
- the returned ID can be used to retrieve the same account;
- successful creation returns `201 Created` and JSON containing `id` and normalized `name`;
- malformed IDs are rejected at the HTTP boundary;
- missing accounts are translated to HTTP 404;
- invalid request JSON and invalid names are translated to HTTP 400;
- unexpected repository errors are translated to HTTP 500;
- account routes reject unsupported methods with HTTP 405;
- liveness and readiness endpoints report server health, including shutdown readiness;
- tests use only the Go standard library and table-driven style where multiple cases exist.

## 7. Second vertical slice

The second complete vertical slice supports this workflow:

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

## 8. Explicitly deferred product concepts

The following are part of the broader product direction but are **not part of the current domain model**:

- account institution;
- account country or jurisdiction;
- account type;
- retirement/non-retirement classification;
- account-level display or base currency;
- country sentinel types;
- securities and other instruments;
- holdings or positions;
- prices and FX rates;
- asset classes;
- property and liabilities;
- portfolio groups or sleeves;
- allocation explorer calculations;
- rebalancing targets;
- historical snapshots;
- persistent account storage (the current repository is in-memory);
- frontend integration with the account API;
- authentication and deployment.

Their presence in the long-term product vision does not imply a particular future data model.

## 9. Next product decision

With account and cash workflows complete, the next product story should add the smallest capability that makes the data meaningfully usable. Candidates include a minimal frontend for entering and viewing accounts and cash, durable persistence, or the next asset type. That choice remains a product decision rather than an implied extension of the cash model.

## 10. MVP direction beyond the second slice

The intended MVP remains a useful personal wealth overview application. A later MVP should support enough concepts to show multiple assets across accounts and provide meaningful allocation exploration.

The exact sequence is intentionally not fixed beyond the second slice. Each new slice must earn any new domain concepts it introduces.

## 11. Non-goals

The product is not currently intended to become:

- a tax engine;
- a brokerage or trading interface;
- a transaction-ledger accounting system;
- a budgeting application;
- an automated financial adviser;
- a bank-credential aggregation service.

These boundaries should remain unless the product direction is deliberately changed later.
