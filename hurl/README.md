# Hurl endpoint tests

Start the backend from the repository root:

```sh
go run ./backend/cmd/server
```

With the server running, execute all scenarios with:

```sh
hurl --test hurl/*.hurl
```

The scenarios target `http://localhost:8080`, matching the server's current development address. Each workflow creates its own account and can be run independently against a running in-memory server.
