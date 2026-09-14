# Hurl endpoint tests

Start the backend from `backend/`:

```sh
go run ./cmd/server
```

With the server running, execute all scenarios from the repository root with:

```sh
hurl --test hurl/*.hurl
```

The scenarios target `http://localhost:8080`, matching the server's current development address. Each workflow creates its own required records and can be run independently against a running in-memory server. `prices.hurl` exercises price observation recording, history, timestamp normalization, exact amounts, and validation.
