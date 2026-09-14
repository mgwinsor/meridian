# Meridian frontend

A React + TypeScript frontend for the implemented operations in
[`../openapi/openapi.yaml`](../openapi/openapi.yaml). The frontend and Go backend
are fully integrated for the current product scope: create/list/select accounts,
read account details, view and replace cash balances in SGD/USD/VND, create and
list properties, replace manual property values, create/list/select instruments,
set account holdings, record/list price observations, and check liveness/readiness.

This completes the browser-to-backend account, cash, property, instrument,
position, and price workflows. The next project
phase is either durable database integration for the existing repositories or a
new end-to-end domain slice that makes the wealth dashboard more useful. The
current UI supports manual property valuation and investment holdings; position
valuation, FX conversion, combined net worth, and allocation remain outside this scope.

## Run locally

From the repository root, start the backend in one terminal:

```sh
cd backend
go run ./cmd/server
```

In a second terminal, from the repository root:

```sh
cd frontend
bun install
bun run dev
```

Open the URL Vite prints (normally <http://localhost:5173>). The connection
indicator should say **Backend connected**. Create an account, save a cash
balance, then reload the page and select the account to confirm server state.

Vite proxies `/api`, `/livez`, and `/readyz` to `http://localhost:8080`, so browser
requests stay on the frontend origin and do not need backend CORS changes.
To change the backend address, copy `.env.example` to `.env.local`, change
`API_PROXY_TARGET`, and restart Vite. This variable is only used by the Vite
server; it is not embedded in the browser bundle.

The backend currently uses in-memory storage and clears all data when restarted.
No database or Prisma setup is needed. The existing Prisma development dependency
is not used by this frontend.

## Checks

```sh
bun run test             # Client error handling and exact amount validation
bun run lint
bun run build
bun run test:integration # Requires both Go and Vite to be running
```

The integration check runs the same API client as React through the Vite proxy.
It covers all 17 API operations, account discovery, empty collections, decimal
normalization, replacement with zero, exact int64 limits, duplicate property names,
standalone property operations, all instrument types, exact fractional holdings,
price timestamp normalization, duplicate observations, and 400/404 errors. It creates
test accounts, properties, instruments, holdings, and prices that stay until the backend restarts.
For a different Vite port, use:

```sh
API_TEST_ORIGIN=http://localhost:5174 bun run test:integration
```

`bun run preview` also proxies to the backend for local production-build checks.
For a deployed static build, configure the host/reverse proxy to route `/api`,
`/livez`, and `/readyz` to Go; Vite's proxy is not part of the built files.

## Client behavior

- Money stays in decimal strings, with currency-specific precision and exact
  upper-bound validation. There is no floating-point conversion or FX total.
- Loading, empty, missing-account, connection, and server-error states are shown.
  Reload failures retain the last loaded data with an error message.
- Account IDs distinguish duplicate names. Selecting a different account resets
  its form, and late responses from the previous selection are ignored.
- Failed saves preserve the entered values. Reloading balances refreshes the
  table while preserving an in-progress form; changing currency loads that
  currency's last fetched amount into the form.
- Writes are never automatically retried. After an uncertain account creation,
  reload accounts before submitting again to avoid duplicates. After an
  uncertain cash save, reload balances before deciding whether to overwrite.

## Properties

Properties are directly owned and independent of custody accounts. Create a named
property with a currency and current manual value even when no account exists or
is selected. Enter the gross market value of your owned share before liabilities.
Duplicate names are allowed, so the
UI displays property IDs. Use **Update value** to replace currency and amount
together, and **Reload properties** to retrieve current values. Reload keeps an
open draft; cancel and reopen the editor to use the latest fetched value.

Property values use the same exact money validation as cash. Zero remains listed.
Creation and value updates are never automatically retried. Failed writes retain
the draft with recovery guidance. Property loading/errors are shown separately
from cash. Account selection and switching do not reload properties or reset
property creation and value-edit drafts.

## Instruments, holdings, and prices

**Instruments & prices** is a shared catalog independent of account selection.
Create a stock, ETF, bond, mutual fund, or crypto instrument with a name, symbol,
and quote currency. Search by name, symbol, type, currency, or ID, then select an
instrument to retrieve its details and price history. IDs distinguish duplicate
names and symbols. Creating an instrument makes it available in account holdings.

Select an account and use **Set a holding** to choose an instrument and enter the
total units owned. **Edit holding** loads the last fetched quantity into the form.
Saving replaces that account/instrument quantity; it does not change cash.
Quantities stay exact decimal strings with no currency precision or magnitude
limit. Zero remains visible. Reloading keeps drafts; selecting another instrument
loads its last fetched quantity, and switching accounts resets the holding form.

**Record a price** adds an observation in the instrument's fixed quote currency.
Prices use the same exact money limits as cash. The optional date/time uses the
browser's displayed local timezone and converts to UTC; blank uses server time.
History shows the full server timestamps in UTC, newest first, including duplicate
observations. The latest observed price is highlighted, including future observations
if entered. Switching instruments resets price drafts and ignores late responses.
Account switching preserves the selected instrument and its price draft.

Loading, empty, failed-load, and failed-save states have recovery guidance.
Reload failures retain the last fetched data; failed writes retain drafts and are
never automatically retried. Uncertain price saves advise checking history before
resubmitting because each POST adds another observation. Prices and holdings are
independent: the UI does not calculate position values or portfolio totals.
