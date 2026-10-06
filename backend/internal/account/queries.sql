-- name: Save :exec
INSERT INTO accounts (id, name) VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- name: FindByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: List :many
SELECT * FROM accounts ORDER BY id;
