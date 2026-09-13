# Meridian API contract

[`openapi.yaml`](openapi.yaml) is the single OpenAPI 3.1 source for frontend and
backend implementation. It is based on the **v1.4**
[product design](../docs/product-design.md) and
[architecture](../docs/architecture.md), checked against the account, cash, property,
currency, money, and health code and the scenarios in `hurl/`.
Contract version **0.4.0** includes the implemented property-asset slice and
the instrument metadata and account-held position APIs.

## Flows and delivery status

| User flow | Operation | Status |
| --- | --- | --- |
| Enter a named account | `POST /api/v1/accounts` | Implemented |
| Discover accounts when opening the application | `GET /api/v1/accounts` | Implemented |
| Open an account | `GET /api/v1/accounts/{id}` | Implemented |
| View current cash in an account | `GET /api/v1/accounts/{id}/cash` | Implemented |
| Enter/correct cash, including replacing it with zero | `PUT /api/v1/accounts/{id}/cash/{currency}` | Implemented |
| Check liveness | `GET /livez` | Implemented |
| Check readiness | `GET /readyz` | Implemented |
| Enter a property and its current manual value | `POST /api/v1/properties` | Implemented |
| Discover all properties and current values | `GET /api/v1/properties` | Implemented |
| Replace a property's current manual value | `PUT /api/v1/properties/{propertyId}/value` | Implemented |
| Create instrument metadata | `POST /api/v1/instruments` | Implemented (backend) |
| Discover instruments | `GET /api/v1/instruments` | Implemented (backend) |
| Retrieve an instrument | `GET /api/v1/instruments/{id}` | Implemented (backend) |
| View an account’s positions | `GET /api/v1/accounts/{id}/positions` | Implemented (backend) |
| Set a holding quantity | `PUT /api/v1/accounts/{id}/positions/{instrumentID}` | Implemented (backend) |

Each operation has an `x-implementation-status` marker. All fifteen operations
are implemented; the ten account, cash, property, and health operations are used
by the frontend. Instrument and position management currently have no frontend interface.
Durable storage is a separate implementation concern and does not require a new
endpoint.

## Instruments

Instruments describe investment vehicles independently of accounts and holdings.
Each has a generated canonical UUID, kind (`stock`, `etf`, `bond`, `mutual_fund`,
or `crypto`), trimmed nonempty symbol and name, and quote currency (exact uppercase
`USD`, `SGD`, or `VND`). Symbols retain case and punctuation; symbols and names
need not be unique. Lists are ordered by UUID and empty lists return
`{"instruments":[]}`. Retrieve requires a canonical lowercase hyphenated UUID.
Instrument storage is in memory. Holdings belong to the separate position slice.
Instrument metadata contains no quantities, prices, or valuations.
Validation order is JSON, quote currency, kind, symbol, then name. Errors use
the existing plain-text convention. Creation is not idempotent.

## Property assets

Property adds one useful asset type beyond cash while reusing supported currencies
and exact money amounts. Manual valuation makes the slice usable
without first defining an instrument catalog, quantities, market prices, or FX.
It uses a feature-owned `property` backend package with its own identity and
repository and the existing currency/money types.

- A property record is a directly owned aggregate that represents the user's
  real-estate value independently of any account. Its asset class is implicitly Property.
  Enter the gross market value of the owned share, before mortgages or other
  liabilities; no ownership-percentage calculation or net-worth claim is made.
- Create with a trimmed, nonempty name and an initial value. A server-generated
  UUID identifies the property independently of its name. Duplicate names are
  allowed and physical-property deduplication is not provided.
- List returns complete property representations ordered by property UUID,
  including an explicit empty array. This supplies both discovery and current
  values without requiring a separate detail endpoint.
- Value PUT replaces currency and amount together on an existing property and
  returns the full property with `200`. Changing currency supplies a new estimate,
  not an FX conversion. Zero is retained. Writes are idempotent with the last
  completed save winning; creation is not idempotent.
- Property IDs require canonical UUIDs. Property value input requires uppercase
  USD/SGD/VND and uses
  existing money precision, normalization, and signed-int64 minor-unit limits.
  Responses use canonical amounts; errors retain the existing plain-text convention.
- Values are manual current estimates without timestamps or history. This slice
  adds no liabilities, price feeds, allocation calculations, or aggregate totals.
- Accounts are custody containers for cash and account-held positions;
  directly owned property is not account membership. A future read-side Asset
  projection may combine property, cash, and positions for reporting, but a
  shared Asset interface, generic asset storage, aggregation, and FX behavior
  remain deliberately deferred.

## Binding decisions for this scope

- One unauthenticated workspace, matching current routing. There is no login or
  implied per-user ownership. No security scheme exists to describe yet.
- Account identity is a UUID; responses are canonical lowercase hyphenated
  strings. The contract documents the existing parser's alternative input forms.
- Account names are trimmed, nonempty, and not unique. Creating an identical name
  again creates another account. No idempotency key is supported.
- Cash identity is `(account ID, currency)`. PUT creates/replaces synchronously
  with `200`; last completed save wins. Zero is retained, not deleted.
- USD/SGD/VND are the supported currencies. Preserve exact decimal **strings**
  across forms, clients, and handlers. USD/SGD have two fractional digits in
  responses, VND none; all are nonnegative and fit signed int64 minor units.
  Cross-parameter currency precision and numeric-string overflow need explicit
  validation beyond what generated schema validators can enforce.
- Account lists are complete and ordered by canonical UUID ascending; cash lists
  are complete and ordered by currency ascending. No pagination, filter, or sort
  arguments. This is a small personal-workspace assumption, not a scalable
  multi-tenant collection design.
- The server's plain-text errors are intentional compatibility constraints for
  this contract. Clients branch on status and the enumerated text (trimming the
  final newline for display); 400 is used for both decoding and validation.
  Account forms can associate the invalid-name message with `name`; cash forms
  associate invalid-amount with `amount` and unsupported-currency with currency.
  There are no structured field violations, 422 responses, or machine-code JSON
  envelopes. A future error-envelope migration must be explicit and coordinated.
- Preserve the existing decoder's tolerance of unknown fields. Clients use only
  documented fields. Response objects have fixed shapes and no nullable values.
- No server timestamps, uploads, deletes, background operations, request IDs,
  throttling headers, or conditional-write semantics are currently promised.

For UI implementation, each collection has populated and empty examples; each
write has success and validation examples. Handle missing accounts separately
from empty balances. Property loading is independent of account selection. A 500 or transport failure should leave entered values
available for correction/recovery. Cash and property-value PUT can be retried with awareness that doing so may
overwrite another edit; account and property POST cannot be retried safely after
an ambiguous response. The API provides no aggregate across currencies.

## Broader wealth-dashboard boundary

The property slice above is the selected domain expansion from the product
vision. Net worth, allocation drill-down, retirement classification, and target
drift remain outside this contract revision. The baseline's deferral language
does not prevent selecting future slices; each selection should resolve only
the domain decisions needed for its own workflow.

The following is a requirements map for extending this same contract when those
vertical slices are selected. These are **not reserved paths or available APIs**.
Frontend and backend implementations must agree on these decisions through a
contract revision rather than guess independently:

| Future flow | Decisions required before adding operations/schemas |
| --- | --- |
| Value instrument holdings | Price versus manually entered value; treatment of bonds and crypto; missing prices and FX |
| See net worth in a reporting currency | Liability/sign model; reporting currencies; FX source/direction/as-of timestamp; valuation source; rounding; stale/missing-rate and missing-price behavior; whether incomplete totals may be shown |
| Explore asset-class → currency/instrument/account allocation | Classification ownership; allowed dimensions and nesting; exclusive membership/double-count prevention; filters; grouping keys; denominator and zero-total behavior; percentage precision |
| Compare countries and retirement/non-retirement | Whether these classify accounts, holdings, or both; unknown/unclassified semantics; country code vocabulary and retirement membership rules |
| Set allocation targets and inspect drift | Target scope, total-sum invariant, threshold units, comparison timing, and whether alerts are computed on read or asynchronously |
| Inspect historical wealth | Snapshot cadence, revision policy, timestamps/timezone, range limits, and pagination |
| Edit/remove accounts or remove a cash entry | Whether needed beyond cash replacement; account deletion cascade/restriction policy; edit concurrency; distinction between absent and zero cash |
| Introduce authentication | Identity provider/session transport, workspace ownership, authorization boundaries, CSRF/CORS requirements, and coordinated 401/403/error behavior |
| Refresh external market data | Provider integrations, credentials, rate limits, synchronous versus asynchronous jobs, job polling and failure/retry states |

When timestamps are introduced, use RFC 3339 `date-time` values and explicitly
settle timezone/precision in that revision. Do not infer a current valuation
timestamp from when a balance was fetched. Upload/import is not a requirement of
the current baseline; define a transport only if an import story is selected.

## Validation

Run from the repository root:

```sh
npx --yes @redocly/cli@latest lint openapi/openapi.yaml
```

The frontend uses Bun (`frontend/bun.lock`) but has no Redocly dependency or
existing API-client generation convention. This one-off command avoids adding
a frontend dependency. `redocly.yaml` applies the recommended rules and treats
unused components and missing operation IDs/summaries/security as errors.
The license-metadata rule is intentionally disabled because the project has no
declared license. The server URL is origin-relative; configure generated clients
with `http://localhost:8080` for the current local backend.
There are no external schema references, so a separate bundle step is unnecessary.

## Positions

A position is a current instrument holding identified by `(account ID, instrument ID)`.
PUT accepts `{"quantity":"12.5"}` and creates or replaces the holding with 200.
Both references must exist. GET returns `{"positions":[]}` for an empty account,
otherwise full holdings (`accountId`, `instrumentId`, `quantity`) ordered by instrument UUID.
Zero remains a holding. Quantities use exact nonnegative decimal strings with no
fixed precision or magnitude limit. Whitespace and redundant zeroes are normalized;
signs, exponents, and malformed decimals are rejected. Last completed save wins.
Storage is in memory. Pricing, valuation, cash changes, history, and frontend
position management are deferred.
