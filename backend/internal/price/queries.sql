-- name: Save :exec
INSERT INTO price_observations (instrument_id, currency, minor_units, observed_at)
VALUES ($1, $2, $3, $4);

-- name: ListByInstrument :many
SELECT * FROM price_observations WHERE instrument_id = $1
ORDER BY observed_at, id;
