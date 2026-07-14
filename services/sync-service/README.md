# sync-service

Polls the public flight-data API, detects status changes, ingests them into
`flight-service`, and publishes domain events to RabbitMQ
(`flight.events`). It is a **batch job**, not a long-running server: it
runs one poll cycle and exits, invoked on a schedule by a Kubernetes
`CronJob` rather than an in-process ticker (see
[../../docs/architecture.md](../../docs/architecture.md) section 2.4 and
[../../docs/deployment/helm-structure.md](../../docs/deployment/helm-structure.md)).

## Layout

```
cmd/main.go              wiring: config -> DB/RabbitMQ/provider/flight-service client -> service -> run once -> exit
internal/config          Viper-backed Config (env prefix SYNC_)
internal/model           FlightSnapshot: the canonical shape every provider normalizes into
internal/api             wire DTOs matching flight-service's ingest OpenAPI contract
internal/repository      SnapshotRepository/PollRunRepository (GORM, schema `sync`), FlightServiceClient, FlightDataProvider + FakeProvider
internal/service         SyncService: diff -> ingest -> map status transition to event type -> publish
```

## Provider

`FlightDataProvider` is the port to the upstream public flight API
(docs call it out generically since no real provider has been chosen yet).
Today the only implementation is `FakeProvider`
(`internal/repository/fake_provider.go`) — deterministic, no credentials
required. Its demo flight `VN257` (HAN→SGN) cycles through
`SCHEDULED -> DELAYED -> BOARDING -> DEPARTED -> LANDED` once per real
30-minute wall-clock window, so running this service repeatedly exercises
every status transition (and therefore every `flight.events` routing key)
against real infrastructure. A second demo flight, `VN9999`, is always
`CANCELLED`. Set `SYNC_PROVIDER_FAKE_NOW_OVERRIDE` (RFC3339) to jump
straight to a specific phase instead of waiting on the clock.

Wiring a real provider later means adding a new case in
`cmd/main.go`'s `newProvider` and a new `FlightDataProvider`
implementation — nothing else in this service changes.

## Running locally

Requires Postgres (schema `sync`, see
[../../docs/database/sync-service.sql](../../docs/database/sync-service.sql))
and RabbitMQ reachable, plus `flight-service` for the ingest call to
succeed (until that service exists, ingest calls fail and are logged —
the run still records its `poll_runs` row and exits non-zero).

```bash
cp .env.example .env
go run ./cmd
```

## Testing

```bash
make test    # go test ./... -cover
```

`internal/service` and `internal/repository` have full unit coverage of
the diff/status-transition/event-routing logic and the fake provider's
time-based state machine using fakes — no network, database, or broker
required.

## Build

```bash
make build          # local binary at bin/sync-service
make docker-build   # builds from the monorepo root (this module depends on ../../pkg)
```

## Configuration

All settings are environment variables prefixed `SYNC_` (see
`.env.example`); in Kubernetes they come from the `sync-service` chart's
ConfigMap/Secret (`deployments/helm/flight-tracker/charts/sync-service`),
never from a committed file.
