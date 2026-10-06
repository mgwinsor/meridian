-- name: Create :exec
INSERT INTO properties (id, name, currency, minor_units) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name,
    currency = EXCLUDED.currency, minor_units = EXCLUDED.minor_units;

-- name: List :many
SELECT * FROM properties ORDER BY id;

-- name: ReplaceValue :one
UPDATE properties SET currency = $2, minor_units = $3 WHERE id = $1 RETURNING *;
