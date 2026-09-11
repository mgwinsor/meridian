# Meridian API contract

[`openapi.yaml`](openapi.yaml) is the single OpenAPI 3.1 source for frontend and
backend implementation. It is based on the **v1.3**
[product design](../docs/product-design.md) and
[architecture](../docs/architecture.md), checked against the account, cash,
currency, money, and health code and the scenarios in `hurl/`.

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

Each operation has an `x-implementation-status` marker. **Planned does not mean
available on the running server.** The account list enables the minimal
account/cash frontend contemplated by product-design §9. Its contract deliberately
uses the existing account resource, not speculative account metadata. Durable
storage is a separate implementation concern and does not require a new endpoint.

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
from empty balances. A 500 or transport failure should leave entered values
available for correction/recovery. Cash PUT can be retried with awareness that
doing so may overwrite another edit; account POST cannot be retried safely after
an ambiguous response. The API provides no aggregate across currencies.

## Broader wealth-dashboard boundary

The product vision includes net worth, allocation drill-down, retirement
classification, and target drift. However, product-design §§3, 8–10 and
architecture §14 explicitly defer their domain models and say that the vision
does not imply a future data model. Assigning implementation-ready paths and
payloads to those concepts now would resolve product decisions the baseline
intentionally leaves open.

The following is a requirements map for extending this same contract when those
vertical slices are selected. These are **not reserved paths or available APIs**.
Frontend and backend implementations must agree on these decisions through a
contract revision rather than guess independently:

| Future flow | Decisions required before adding operations/schemas |
| --- | --- |
| Enter instruments and holdings | Supported asset types; instrument identity and deduplication; account/position relationship; quantity precision; price versus manually entered value; treatment of bonds, property, crypto, and cash overlap |
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
