-- flight-service owns this schema exclusively. No other service connects to it.
CREATE SCHEMA IF NOT EXISTS flight;

CREATE TYPE flight.flight_status AS ENUM (
    'SCHEDULED', 'BOARDING', 'DELAYED', 'DEPARTED', 'LANDED', 'CANCELLED'
);

CREATE TABLE flight.flights (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flight_number        VARCHAR(8)   NOT NULL,
    airline_iata         VARCHAR(3)   NOT NULL,
    airline_name         VARCHAR(120),
    origin_iata          CHAR(3)      NOT NULL,
    destination_iata     CHAR(3)      NOT NULL,
    scheduled_departure  TIMESTAMPTZ  NOT NULL,
    estimated_departure  TIMESTAMPTZ,
    actual_departure     TIMESTAMPTZ,
    scheduled_arrival    TIMESTAMPTZ  NOT NULL,
    estimated_arrival    TIMESTAMPTZ,
    actual_arrival       TIMESTAMPTZ,
    gate                 VARCHAR(10),
    terminal             VARCHAR(10),
    status               flight.flight_status NOT NULL DEFAULT 'SCHEDULED',
    aircraft_type        VARCHAR(40),
    last_synced_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- one row per physical flight departure, keyed by flight number + its own scheduled day
    CONSTRAINT uq_flight_number_departure UNIQUE (flight_number, scheduled_departure)
);

CREATE INDEX idx_flights_flight_number ON flight.flights (flight_number);
CREATE INDEX idx_flights_route_date ON flight.flights (origin_iata, destination_iata, scheduled_departure);
CREATE INDEX idx_flights_status ON flight.flights (status);

-- audit trail of every status transition, also used to answer "what changed" for support/debugging
CREATE TABLE flight.flight_status_history (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flight_id     UUID NOT NULL REFERENCES flight.flights (id) ON DELETE CASCADE,
    old_status    flight.flight_status,
    new_status    flight.flight_status NOT NULL,
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    source        VARCHAR(40) NOT NULL DEFAULT 'sync-service',
    raw_payload   JSONB
);

CREATE INDEX idx_flight_status_history_flight_id ON flight.flight_status_history (flight_id, changed_at DESC);

-- Redis (not DDL, documented here for completeness):
--   key:   flight:search:{flight_number}:{date}         value: FlightResponse JSON, TTL 60s
--   key:   flight:route:{origin}:{destination}:{date}   value: FlightResponse[] JSON, TTL 60s
-- Cache is invalidated proactively on ingest (delete the relevant keys) in addition to TTL expiry.
