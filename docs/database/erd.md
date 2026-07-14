# Entity Relationship Diagrams

Each schema below lives in a **separate logical database/schema per service**
(one PostgreSQL cluster can host all schemas for a small deployment, or each
service can point at its own instance — the application layer never assumes
either). No foreign keys cross schema boundaries; cross-service references
(e.g. `flight_number` appearing in both `subscription` and `flight`) are
plain values, resolved via REST, never joined in SQL.

## flight schema

```mermaid
erDiagram
    FLIGHTS ||--o{ FLIGHT_STATUS_HISTORY : "has history"
    FLIGHTS {
        uuid id PK
        varchar flight_number
        varchar airline_iata
        char origin_iata
        char destination_iata
        timestamptz scheduled_departure
        timestamptz estimated_departure
        timestamptz actual_departure
        timestamptz scheduled_arrival
        enum status
        varchar gate
        varchar terminal
        timestamptz last_synced_at
    }
    FLIGHT_STATUS_HISTORY {
        uuid id PK
        uuid flight_id FK
        enum old_status
        enum new_status
        timestamptz changed_at
        jsonb raw_payload
    }
```

## subscription schema

```mermaid
erDiagram
    CHATS ||--o{ SUBSCRIPTIONS : "subscribes"
    CHATS {
        uuid id PK
        bigint telegram_chat_id
        varchar username
        varchar language_code
    }
    SUBSCRIPTIONS {
        uuid id PK
        uuid chat_id FK
        enum type
        varchar flight_number
        char origin_iata
        char destination_iata
        boolean active
    }
```

## sync schema

```mermaid
erDiagram
    FLIGHT_SNAPSHOTS {
        uuid id PK
        varchar flight_number
        timestamptz scheduled_departure
        char raw_hash
        jsonb raw_payload
        timestamptz last_polled_at
        timestamptz last_changed_at
    }
    POLL_RUNS {
        uuid id PK
        timestamptz started_at
        timestamptz finished_at
        int flights_seen
        int flights_changed
        int events_published
        text error
    }
```

## notification schema

```mermaid
erDiagram
    PROCESSED_EVENTS ||--o{ DELIVERY_LOG : "resulted in"
    PROCESSED_EVENTS {
        uuid event_id PK
        varchar event_type
        timestamptz processed_at
    }
    DELIVERY_LOG {
        uuid id PK
        uuid event_id FK
        bigint telegram_chat_id
        varchar status
        timestamptz sent_at
    }
```
