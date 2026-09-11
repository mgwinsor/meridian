# Meridian frontend

A minimal React + TypeScript frontend for the implemented operations in
[`../openapi/openapi.yaml`](../openapi/openapi.yaml). It uses the real Go backend:
create/list/select accounts, read account details, view and replace cash balances
in SGD/USD/VND, and check liveness/readiness.

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
It covers all seven API operations, account discovery, empty balances, decimal
normalization, replacement with zero, exact int64 limits, and 400/404 errors.
It creates a uniquely named test account that stays until the backend restarts.
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
