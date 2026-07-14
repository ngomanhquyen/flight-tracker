-- sync-service owns this schema exclusively. It is a private ingestion cache
-- used only to cheaply detect "did this flight change since the last poll" —
-- it is NOT a copy of flight-service's canonical data and no other service
-- may depend on it.
CREATE SCHEMA IF NOT EXISTS sync;

CREATE TABLE sync.flight_snapshots (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flight_number           VARCHAR(8)  NOT NULL,
    scheduled_departure     TIMESTAMPTZ NOT NULL,
    raw_hash                CHAR(64)    NOT NULL, -- sha256 of normalized payload, for cheap change detection
    raw_payload             JSONB       NOT NULL,
    last_polled_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_changed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_sync_flight_departure UNIQUE (flight_number, scheduled_departure)
);

CREATE INDEX idx_sync_flight_snapshots_polled_at ON sync.flight_snapshots (last_polled_at);

-- one row per CronJob run, for observability/troubleshooting of poll cycles
CREATE TABLE sync.poll_runs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at       TIMESTAMPTZ,
    flights_seen       INTEGER NOT NULL DEFAULT 0,
    flights_changed    INTEGER NOT NULL DEFAULT 0,
    events_published   INTEGER NOT NULL DEFAULT 0,
    error              TEXT
);
