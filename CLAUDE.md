# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## `ckad-labs/`

Unrelated to the Flight Tracker app: a self-study lab series for the CKAD
certification, organized by exam domain (`ckad-labs/01-application-design-and-build/`,
more domains added as requested). Each lab has a `README.md` (tasks) and a
`solution.md` (answers + explanation) — don't conflate this with the
services under `services/` when reasoning about the app's architecture.

## What this is

Flight Tracker Bot — a Telegram bot (no web frontend) that lets users search
flights and subscribe to flight/route status changes. Go microservices
monorepo, currently mid-implementation: `bot-service`, `flight-service`, and
`sync-service` have real source code; `subscription-service` and
`notification-service` are still design-only (`api/README.placeholder`, no
`cmd`/`internal`). The root `README.md` says "Status: Step 1 — Design" —
that's stale; treat `docs/architecture.md` and the actual `services/*`
directories as the source of truth for what exists.

Read `docs/architecture.md` before making cross-service changes — it defines
service boundaries, the communication matrix, and the design principles
(Clean Architecture per service, repository pattern, event-driven integration)
that all the implemented services follow.

## Commands

There is no root build. Each service is an independent Go module joined by
`go.work` (workspace covers `pkg`, `bot-service`, `flight-service`,
`sync-service` — not the two unimplemented services). Run commands from
inside a service directory:

```bash
make build          # go build -o bin/<service> ./cmd
make run            # build + run
make test           # go test ./... -cover
make lint           # gofmt -l . && go vet ./...
make fmt            # gofmt -l -w .
make tidy           # go mod tidy
make docker-build   # must run from repo root context: cd ../.. && docker build -f services/<name>/Dockerfile -t ... .
```

Single test: `go test ./internal/service/... -run TestName -v` (run from the service directory).

Local infra (Postgres + RabbitMQ + Redis) for `flight-service`/`sync-service`: `docker compose -f docker-compose.dev.yml up`.

`bot-service` has its own PowerShell dev-loop scripts (`services/bot-service/scripts/start.ps1` / `stop.ps1`) that build it, open a Cloudflare quick tunnel, and call Telegram's `setWebhook` so you can message the real bot during local dev. Logs land in `services/bot-service/logs/` (gitignored) — `bot-service.err.log` is where Zap output actually goes (stderr), not `bot-service.log`.

Each service's `internal/service` (and relevant `internal/repository`) layer has full unit coverage using hand-written fakes — no real Postgres/Redis/RabbitMQ needed to run `make test`. Infra-backed behavior is verified via manual/end-to-end smoke testing against `docker-compose.dev.yml`, not `go test`.

## Cross-service architecture

- One Go module per service under `services/`, each with its own `go.mod`, `Dockerfile`, `Makefile`, `.env.example`, and Postgres schema. **No service reaches into another service's database** — all cross-service interaction is REST (`/internal/v1/...` for service-to-service, shared-secret/NetworkPolicy-protected) or RabbitMQ events.
- `pkg/` (module `github.com/flighttracker/pkg`) holds shared, business-logic-free libraries: `logger` (Zap + correlation-id-aware child loggers), `config` (Viper), `httpserver` (Gin bootstrap + `/health`/`/ready`/`/metrics`), `middleware`, `errors` (`AppError` → HTTP status mapping), `response` (JSON envelope helpers), `eventbus` (RabbitMQ publisher/consumer), `telegram` (Bot API client), `validator`. Each service's `go.mod` pulls it in via `replace github.com/flighttracker/pkg => ../../pkg`, so Docker builds must use the **repo root** as build context (see each `Dockerfile`'s comment) — building from inside a service directory will not resolve `pkg`.
- Per-service layering is uniform: `internal/model` (domain entities) → `internal/repository` (GORM behind interfaces, or HTTP clients to other services) → `internal/service` (use cases, depends on repository interfaces not GORM types) → `internal/handler` (Gin transport) → `internal/api` (wire DTOs kept separate from domain models, matching the OpenAPI contracts in `docs/api-contracts/`). Wiring is explicit in each `cmd/main.go` (config → logger → repositories → services → handlers → server), no DI framework.
- Data flow: `sync-service` polls a `FlightDataProvider` (currently only `FakeProvider`, a deterministic time-based state machine — see `services/sync-service/README.md`), diffs against its own `sync` schema snapshot, calls `flight-service`'s internal ingest endpoint (the *only* write path into the `flight` schema), then publishes to the `flight.events` RabbitMQ topic exchange based on the ingest response's previous/new status. `notification-service` (not yet built) will consume those events, resolve subscribers via `subscription-service`'s match endpoint, and send Telegram messages directly. `bot-service` is the only inbound entry point (Telegram webhook), stateless, calling `flight-service` and `subscription-service` over REST.
- Event contract: envelope + routing keys (`flight.created`, `flight.delayed`, etc.) are documented in `docs/events/event-catalog.md`. Versioning is additive-only (`version: MAJOR.MINOR`); breaking changes get a new routing-key namespace (`flight.v2.*`) rather than mutating `v1`.
- Deployment: umbrella Helm chart at `deployments/helm/flight-tracker`, one subchart per service plus a `common` library chart for shared Deployment/Service/HPA/probe templates. `sync-service` renders a `CronJob` (schedule `*/5 * * * *`), not a `Deployment` — it's a batch job, not a long-running server. Details in `docs/deployment/helm-structure.md`.

## Notable repo-specific conventions

- `deployments/helm/flight-tracker/values-demo.yaml` is **intentionally committed with real credentials** (Telegram bot token, DB passwords) — this is a throwaway public demo bot, not a product, so `./deploy-demo.sh` works turnkey for anyone who clones the repo. This is a deliberate exception to normal secret hygiene, documented in `.gitignore`; don't "fix" it by scrubbing or gitignoring it without being asked.
- Error handling flows through `pkg/errors.AppError` (stable `Code` + `HTTPStatus` + wrapped `Err`) → `pkg/response.Error` writes the standard `{code, message, correlation_id}` JSON envelope used by every handler.
- `context.Context` carries request-scoped deadlines/cancellation and a correlation ID from the first hop (Telegram webhook or CronJob run) through every downstream REST/AMQP call and into structured logs — preserve this when adding new call chains.
