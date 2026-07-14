# flight-service

System of record for the current, canonical status of a flight — the read
model `bot-service` searches against, and the only write target
`sync-service` ingests into (see
[../../docs/architecture.md](../../docs/architecture.md) section 2.3).
Unlike `sync-service`, this is a long-running server (Kubernetes
Deployment), not a batch job.

## Layout

```
cmd/main.go              wiring: config -> DB/Redis -> repositories -> services -> handler -> HTTP server
internal/config          Viper-backed Config (env prefix FLIGHT_)
internal/model           Flight: the canonical entity this service owns
internal/api             wire DTOs matching docs/api-contracts/flight-service.yaml
internal/repository      FlightRepository (GORM, schema `flight`) + CacheRepository (Redis, cache-aside)
internal/service         FlightSearchService (cache-aside reads) + IngestService (the only write path)
internal/handler         FlightHandler (Gin): the three routes below
api/openapi.yaml         copy of this service's transport contract
```

## Routes

- `GET /api/v1/flights/:flightNumber?date=YYYY-MM-DD` — search by flight number (defaults to today, UTC).
- `GET /api/v1/flights/route?origin=&destination=&date=&limit=` — search by route.
- `POST /internal/v1/flights/ingest` — the only write path into the `flight` schema, called exclusively by `sync-service`. Diffs the incoming snapshot against the stored row and returns `{flight, previous_status, changed}` so the caller can decide whether to publish a domain event.

Search results are cache-aside through Redis
(`flight:search:{flight_number}:{date}` / `flight:route:{origin}:{destination}:{date}`,
TTL `FLIGHT_CACHE_TTL`) and proactively invalidated on ingest. A Redis
failure degrades to a Postgres read/write rather than failing the request
— caching is an optimization, not a dependency correctness relies on.

## Running locally

Requires Postgres (schema `flight`, see
[../../docs/database/flight-service.sql](../../docs/database/flight-service.sql))
and Redis reachable — see `../../docker-compose.dev.yml` at the repo root.

```bash
cp .env.example .env
go run ./cmd
```

## Testing

```bash
make test    # go test ./... -cover
```

`internal/service` has full unit coverage of the cache-aside/invalidation
logic using fakes; `internal/repository` unit-tests the ingest diff rule
(`flightsDiffer`) directly. Both require no real Postgres/Redis — the
repository/cache implementations themselves are verified against real
infrastructure as part of this project's end-to-end smoke testing rather
than with `go test`.

## Build

```bash
make build          # local binary at bin/flight-service
make docker-build   # builds from the monorepo root (this module depends on ../../pkg)
```

## Configuration

All settings are environment variables prefixed `FLIGHT_` (see
`.env.example`); in Kubernetes they come from the `flight-service` chart's
ConfigMap/Secret (`deployments/helm/flight-tracker/charts/flight-service`),
never from a committed file.
