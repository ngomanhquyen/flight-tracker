-- subscription-service owns this schema exclusively. No other service connects to it.
CREATE SCHEMA IF NOT EXISTS subscription;

CREATE TYPE subscription.subscription_type AS ENUM ('flight', 'route');

CREATE TABLE subscription.chats (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_chat_id  BIGINT NOT NULL UNIQUE,
    username          VARCHAR(64),
    first_name        VARCHAR(120),
    language_code     VARCHAR(10),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscription.subscriptions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id           UUID NOT NULL REFERENCES subscription.chats (id) ON DELETE CASCADE,
    type              subscription.subscription_type NOT NULL,
    flight_number     VARCHAR(8),
    origin_iata       CHAR(3),
    destination_iata  CHAR(3),
    active            BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_subscription_shape CHECK (
        (type = 'flight' AND flight_number IS NOT NULL AND origin_iata IS NULL AND destination_iata IS NULL)
        OR
        (type = 'route' AND flight_number IS NULL AND origin_iata IS NOT NULL AND destination_iata IS NOT NULL)
    )
);

-- prevent duplicate active subscriptions per chat (partial unique indexes, since NULLs
-- would otherwise make a plain UNIQUE constraint ineffective for one of the two shapes)
CREATE UNIQUE INDEX uq_active_flight_subscription
    ON subscription.subscriptions (chat_id, flight_number)
    WHERE type = 'flight' AND active = true;

CREATE UNIQUE INDEX uq_active_route_subscription
    ON subscription.subscriptions (chat_id, origin_iata, destination_iata)
    WHERE type = 'route' AND active = true;

-- fast lookup path for notification-service's /internal/v1/subscriptions/match
CREATE INDEX idx_subscriptions_flight_lookup
    ON subscription.subscriptions (flight_number)
    WHERE type = 'flight' AND active = true;

CREATE INDEX idx_subscriptions_route_lookup
    ON subscription.subscriptions (origin_iata, destination_iata)
    WHERE type = 'route' AND active = true;

CREATE INDEX idx_subscriptions_chat_id ON subscription.subscriptions (chat_id) WHERE active = true;
