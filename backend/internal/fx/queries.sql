-- name: Save :exec
INSERT INTO fx_observations (currency, rate, observed_at) VALUES ($1, $2, $3);
-- name: List :many
SELECT * FROM fx_observations WHERE currency = $1 ORDER BY observed_at, id;
-- name: Latest :many
SELECT DISTINCT ON (currency) * FROM fx_observations WHERE observed_at <= $1
ORDER BY currency, observed_at DESC, id DESC;
