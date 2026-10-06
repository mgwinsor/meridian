# Repository Guidelines

## Project Structure & Module Organization

- `backend/cmd/server/` wires the Go HTTP server; `backend/internal/` contains feature packages (`account`, `cash`, `currency`, `money`, `health`) and colocated tests.
- `frontend/src/` contains React components, the API client, money validation, and CSS. Assets live in `frontend/src/assets/` and `frontend/public/`; Bun tests live in `frontend/tests/`.
- `hurl/` contains HTTP endpoint scenarios. `openapi/openapi.yaml` defines the API contract; `docs/` holds product and architecture decisions.

Keep backend logic in feature-owned packages with repository interfaces at storage boundaries. Consult implementation-status markers before assuming a documented endpoint exists.

## Build, Test, and Development Commands

Use the Go version declared in `backend/go.mod` and Bun for frontend dependencies.

From `backend/`:

- `go run ./cmd/db init` creates the database and applies goose migrations against the running Compose PostgreSQL service.
- `go run ./cmd/server` starts the API on port 8080.
- `sqlc generate` regenerates feature-owned queries (sqlc v1.31.1).
- `go test -tags=integration ./internal/database` checks PostgreSQL repositories in a temporary database.
- `go build ./...` compiles all packages.
- `go test ./...` runs backend tests.

From `frontend/`:

- `bun install` installs dependencies.
- `bun run dev` starts Vite, normally on port 5173.
- `bun run build` checks TypeScript and produces the production bundle.
- `bun run lint` runs ESLint; `bun run test` runs unit tests.
- `bun run test:integration` checks the API through Vite; both servers must be running.

From the repository root:

- `hurl --test hurl/*.hurl` checks endpoints against the running backend.
- `npx --yes @redocly/cli@latest lint openapi/openapi.yaml` validates the contract.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use lowercase package names, exported PascalCase identifiers, and `*_test.go` files. Match frontend conventions: two-space indentation, single quotes, no statement-ending semicolons, PascalCase components, and camelCase functions. Follow the configured ESLint rules.

Preserve money as decimal strings at API/UI boundaries and integer minor units internally; avoid floating-point conversion.

## Testing Guidelines

Use Go's `testing` package (`TestXxx`) and Bun's test runner (`*.test.ts`). Cover validation boundaries, repository/service behavior, HTTP errors, and exact monetary limits. No numeric coverage threshold is configured. Run relevant checks before submitting changes.

## Commit & Pull Request Guidelines

History uses short, lowercase subjects such as `add: account endpoints` and `docs: product design and architecture`. Follow that style and keep commits focused. PRs should describe behavior changes, list validation performed, link relevant issues, and include screenshots for UI changes. Update the API contract alongside endpoint changes.

## Local Configuration

Storage uses PostgreSQL from `compose.yaml`. Run `docker compose up -d --wait db` from the root and `go run ./cmd/db init` from `backend/` before starting the API; see `backend/README.md` for command working directories, migrations, and `DATABASE_URL`. Memory repositories support unit tests. Set `API_PROXY_TARGET` in `frontend/.env.local` when changing the backend address; see `.env.example`.
