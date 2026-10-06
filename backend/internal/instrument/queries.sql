-- name: Save :exec
INSERT INTO instruments (id, kind, symbol, name, quote_currency) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET kind = EXCLUDED.kind, symbol = EXCLUDED.symbol,
    name = EXCLUDED.name, quote_currency = EXCLUDED.quote_currency;

-- name: FindByID :one
SELECT * FROM instruments WHERE id = $1;

-- name: List :many
SELECT * FROM instruments ORDER BY id;
