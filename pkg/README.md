# pkg/

Shared Go libraries imported by every service in `services/`, kept
dependency-free of any single service's business logic. Implemented in
Step 2 (scaffolding), alongside each service's `go.mod`.

| Package | Purpose |
|---|---|
| `logger` | Zap setup: JSON in non-local envs, correlation-id-aware child loggers |
| `config` | Viper loader: ENV + optional file, typed struct binding, validation on load |
| `httpserver` | Gin bootstrap: graceful shutdown, `/health`, `/ready`, `/metrics` wiring, request-id middleware |
| `middleware` | Gin middleware: recovery, CORS (where applicable), request logging, correlation-id propagation |
| `errors` | Standard error types + HTTP status mapping, used by handlers to produce the `ErrorResponse` shape defined in the OpenAPI contracts |
| `response` | Standard success/error JSON envelope helpers |
| `eventbus` | RabbitMQ publisher/consumer wrapper: exchange/queue declaration, publisher confirms, retry/DLQ helpers matching [docs/events/event-catalog.md](../docs/events/event-catalog.md) |
| `telegram` | Thin Telegram Bot API client (used by both `bot-service` for webhook replies and `notification-service` for push sends) |
| `validator` | go-playground/validator setup + custom validators (IATA code, flight number pattern) |
| `jwtauth` | golang-jwt helpers, reserved for any future authenticated internal/admin endpoints |

Each package is a normal Go module-internal package (not its own Go module)
imported via the monorepo's shared module path, so a change to `pkg/logger`
is versioned and released together with the services that depend on it —
no separate release/versioning process for `pkg/` itself.
