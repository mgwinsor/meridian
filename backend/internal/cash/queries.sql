-- name: Save :exec
INSERT INTO cash_balances (account_id, currency, minor_units) VALUES ($1, $2, $3)
ON CONFLICT (account_id, currency) DO UPDATE SET minor_units = EXCLUDED.minor_units;

-- name: ListByAccount :many
SELECT * FROM cash_balances WHERE account_id = $1 ORDER BY currency;
