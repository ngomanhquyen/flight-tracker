# Flight Tracker Bot

A Telegram bot that lets users search flights by number or route, subscribe to
flight numbers or routes, and receive automatic notifications on status
changes (boarding, delay, gate change, departure, landing, cancellation).

Users interact **only** through Telegram. There is no web frontend.

> **Status: Step 1 — Design.** This repository currently contains the
> architecture, API contracts, event definitions, database schemas, and Helm
> chart structure only. No service source code has been generated yet.
> See [`docs/architecture.md`](docs/architecture.md) for the full design and
> wait for sign-off before Step 2 (implementation) begins.

## Repository layout

```
flight-tracker/
├── services/                  # one Go module per microservice
│   ├── bot-service/            # Telegram webhook + command parser
│   ├── subscription-service/   # subscription CRUD (PostgreSQL)
│   ├── flight-service/         # flight search + cache (PostgreSQL + Redis)
│   ├── sync-service/           # polls public flight API, publishes events
│   └── notification-service/   # consumes events, sends Telegram messages
├── deployments/
│   ├── helm/                   # umbrella Helm chart (source of truth)
│   └── kubernetes/             # kustomize overlays / cluster-scoped extras
├── proto/                      # reserved for future gRPC contracts
├── pkg/                        # shared Go libraries (logger, config, eventbus, ...)
└── docs/                       # architecture, contracts, schemas, diagrams
```

Each service is self-contained: `cmd/`, `internal/{api,repository,service,handler,model,config}`,
its own `Dockerfile`, `Makefile`, and `README.md` (added in Step 2), and its
own database schema. **No service reaches into another service's database.**
All cross-service reads/writes go through REST APIs or RabbitMQ events.

## Documentation index

| Doc | Contents |
|---|---|
| [docs/architecture.md](docs/architecture.md) | System context, service responsibilities, communication matrix, design principles |
| [docs/diagrams/sequence-diagrams.md](docs/diagrams/sequence-diagrams.md) | Sequence diagrams for search, subscribe, sync, and notification flows |
| [docs/api-contracts/](docs/api-contracts/) | OpenAPI specs for flight-service and subscription-service; bot-service webhook/command contract |
| [docs/events/event-catalog.md](docs/events/event-catalog.md) | RabbitMQ topology, routing keys, event envelope, retry/DLQ policy |
| [docs/database/](docs/database/) | Per-service schema DDL + ERDs |
| [docs/deployment/helm-structure.md](docs/deployment/helm-structure.md) | Helm chart layout and rationale |
| [docs/deployment/deployment-guide.md](docs/deployment/deployment-guide.md) | Cluster prerequisites, install/upgrade steps, secrets handling |

## Tech stack

Go 1.24+ · Gin · GORM · Viper · Zap · go-playground/validator · golang-jwt ·
google/uuid · PostgreSQL · Redis · RabbitMQ · Docker · Kubernetes · Helm.

## Roadmap (incremental delivery)

1. **Design** (this step) — architecture, contracts, schemas, Helm structure.
2. `pkg/` shared libraries + service scaffolding (go.mod, config, health/ready/metrics).
3. `subscription-service` (CRUD + Postgres).
4. `flight-service` (search + Postgres + Redis cache).
5. `sync-service` (poller + change detection + event publishing).
6. `notification-service` (consumer + Telegram sender).
7. `bot-service` (webhook + command parser wired to the above).
8. Helm templates + Kubernetes manifests, CI/CD.

Each step is implemented and reviewed before moving to the next.
