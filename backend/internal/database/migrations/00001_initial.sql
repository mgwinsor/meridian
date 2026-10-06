-- +goose Up
CREATE TABLE accounts (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> '')
);

CREATE TABLE instruments (
    id uuid PRIMARY KEY,
    kind text NOT NULL CHECK (kind IN ('stock', 'etf', 'bond', 'mutual_fund', 'crypto')),
    symbol text NOT NULL CHECK (btrim(symbol) <> ''),
    name text NOT NULL CHECK (btrim(name) <> ''),
    quote_currency text NOT NULL CHECK (quote_currency IN ('USD', 'SGD', 'VND'))
);

CREATE TABLE cash_balances (
    account_id uuid NOT NULL REFERENCES accounts(id),
    currency text NOT NULL CHECK (currency IN ('USD', 'SGD', 'VND')),
    minor_units bigint NOT NULL CHECK (minor_units >= 0),
    PRIMARY KEY (account_id, currency)
);

CREATE TABLE properties (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> ''),
    currency text NOT NULL CHECK (currency IN ('USD', 'SGD', 'VND')),
    minor_units bigint NOT NULL CHECK (minor_units >= 0)
);

CREATE TABLE positions (
    account_id uuid NOT NULL REFERENCES accounts(id),
    instrument_id uuid NOT NULL REFERENCES instruments(id),
    quantity numeric(38,18) NOT NULL
        CHECK (quantity >= 0 AND quantity < 100000000000000000000),
    PRIMARY KEY (account_id, instrument_id)
);
CREATE INDEX positions_instrument_id_idx ON positions (instrument_id);

CREATE TABLE price_observations (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    instrument_id uuid NOT NULL REFERENCES instruments(id),
    currency text NOT NULL CHECK (currency IN ('USD', 'SGD', 'VND')),
    minor_units bigint NOT NULL CHECK (minor_units >= 0),
    observed_at timestamptz NOT NULL CHECK (
        observed_at >= TIMESTAMPTZ '0001-01-01 00:00:00+00 BC'
        AND observed_at < TIMESTAMPTZ '10000-01-01 00:00:00+00'
        AND observed_at <> TIMESTAMPTZ '0001-01-01 00:00:00+00'
    )
);
CREATE INDEX price_observations_history_idx
    ON price_observations (instrument_id, observed_at, id);

-- +goose Down
DROP TABLE price_observations;
DROP TABLE positions;
DROP TABLE properties;
DROP TABLE cash_balances;
DROP TABLE instruments;
DROP TABLE accounts;
