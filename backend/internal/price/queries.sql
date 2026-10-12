-- name: Save :exec
INSERT INTO price_observations (instrument_id, currency, minor_units, observed_at)
VALUES ($1, $2, $3, $4);

-- name: ListByInstrument :many
SELECT * FROM price_observations WHERE instrument_id = $1
ORDER BY observed_at, id;

-- name: Latest :many
SELECT DISTINCT ON (instrument_id) * FROM price_observations
WHERE instrument_id = ANY($1::uuid[]) AND observed_at <= $2
ORDER BY instrument_id, observed_at DESC, id DESC;
