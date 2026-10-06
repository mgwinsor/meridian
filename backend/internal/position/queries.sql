-- name: Save :exec
INSERT INTO positions (account_id, instrument_id, quantity) VALUES ($1, $2, $3)
ON CONFLICT (account_id, instrument_id) DO UPDATE SET quantity = EXCLUDED.quantity;

-- name: ListByAccount :many
SELECT * FROM positions WHERE account_id = $1 ORDER BY instrument_id;
