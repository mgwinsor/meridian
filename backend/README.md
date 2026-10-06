# Meridian backend

The server stores accounts, cash, properties, instruments, positions, and price
observations in PostgreSQL using pgx. Each feature owns its repository and sqlc
queries; memory repositories remain available for unit tests.

## Run locally

Use the Go version in `go.mod`. From the repository root:

```sh
docker compose up -d --wait db
cd backend
go run ./cmd/db init
go run ./cmd/server
```

`init` uses a goose Go migration to create the application database if absent,
then applies the embedded goose SQL migrations. It also works when the original
Compose configuration already created `meridian`. The development role must have
`CREATEDB`; Compose provisions that role automatically. No reset is required for
an existing empty development database.

Both commands read `DATABASE_URL`, defaulting to:

```text
postgres://meridian:dev_password@127.0.0.1:5432/meridian?sslmode=disable
```

Export `DATABASE_URL` to use a different database. `init` connects to the
`postgres` maintenance database on the same server using the same credentials.
For an already provisioned database, use `up`, which needs schema permissions but
does not require `CREATEDB` or maintenance-database access.

The server checks connectivity and migration versions before listening on port
8080. `/readyz` checks PostgreSQL with a one-second timeout and returns 503 when
it is unavailable or shutdown has started. `/livez` remains independent of the
database. The connection pool closes after HTTP shutdown.

## Migrations and queries

From `backend/`:

```sh
go run ./cmd/db status   # Show goose migration state
go run ./cmd/db up       # Apply pending migrations
go run ./cmd/db down     # Roll back ONE migration; the initial rollback deletes all feature data
```

Goose is pinned in `go.mod`; the command embeds SQL from
`internal/database/migrations/`. The single `00001_initial.sql` migration creates
the complete current schema on a clean database, with `-- +goose Up` and
`-- +goose Down` sections. The SQL migration runs transactionally and uses
a PostgreSQL advisory lock to serialize migration runners. Database creation is a
separate, idempotent, nontransactional goose migration. Server startup does not
apply application migrations.

Install the pinned generator, then regenerate after changing schemas or queries:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
"$(go env GOPATH)/bin/sqlc" generate
"$(go env GOPATH)/bin/sqlc" vet
```

`sqlc.yaml` reads the migration schema and each feature's `queries.sql`, generating
pgx queries into that feature's `postgres/` subpackage. Commit generated files;
do not edit them manually. Domain conversion and missing-record error mapping
belong in the feature's `repository_postgres.go`.

Money uses checked, nonnegative `bigint` minor units. Position quantities use
NUMERIC(38,18), with the backend and frontend enforcing 20 integer and 18 fractional
digits after removing redundant zeroes. Out-of-range values are rejected without
rounding. Price times use one timestamptz column; the backend normalizes to UTC
and truncates to microseconds before validation and persistence. UTC years
0000–9999 remain supported. Price rows have independent sequence IDs so
identical observations remain distinct and equal timestamps keep insertion order.

## Checks

```sh
go build ./...
go test ./...
go test -tags=integration ./internal/database -count=1
```

Integration tests require the running development PostgreSQL service. They use
`TEST_DATABASE_URL` (falling back to `DATABASE_URL` and then the local default) for
connection credentials, create a uniquely named database, and drop only that
database afterward. The role needs `CREATEDB`. Tests exercise goose creation,
repeated migration, rollback/reapply, all six repositories, reconnect persistence,
exact numeric boundaries and microsecond timestamps, concurrent replacement, foreign keys,
constraints, and cancellation. Ordinary unit tests need no database.

For HTTP scenarios, start the migrated server and run `hurl --test hurl/*.hurl`
from the repository root. Test-created records persist across server restarts.

## Reset the development database

To delete all local data, run these commands from the repository root:

```sh
docker compose down -v
docker compose up -d --wait db
cd backend
go run ./cmd/db init
```

The named Compose volume survives ordinary container restarts and `compose down`.
Only the explicit `-v` reset removes it. The supplied credentials are for local
development; production provisioning, authentication, and backups remain outside
this milestone.
