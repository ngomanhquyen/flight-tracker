-- notification-service owns this schema exclusively. Deliberately no Redis
-- dependency: this table is small (rolling window, pruned by TTL job) and
-- gives durable, restart-safe idempotency for at-least-once AMQP delivery.
CREATE SCHEMA IF NOT EXISTS notification;

CREATE TABLE notification.processed_events (
    event_id       UUID PRIMARY KEY,
    event_type     VARCHAR(40) NOT NULL,
    processed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- pruned by a periodic job (e.g. daily CronJob or in-process ticker):
--   DELETE FROM notification.processed_events WHERE processed_at < now() - interval '7 days';

CREATE TABLE notification.delivery_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id       UUID NOT NULL REFERENCES notification.processed_events (event_id) ON DELETE CASCADE,
    telegram_chat_id BIGINT NOT NULL,
    status         VARCHAR(20) NOT NULL, -- SENT | FAILED
    error          TEXT,
    sent_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_delivery_log_event_id ON notification.delivery_log (event_id);
CREATE INDEX idx_delivery_log_chat_id ON notification.delivery_log (telegram_chat_id, sent_at DESC);
