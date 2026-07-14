# bot-service

Telegram webhook receiver and command dispatcher for Flight Tracker Bot.
Stateless — owns no database. It is a REST client of `flight-service`
(search) and `subscription-service` (subscriptions), and the only inbound
entry point for users (see [../../docs/architecture.md](../../docs/architecture.md)).

## Layout

```
cmd/main.go              wiring: config -> logger -> clients -> service -> handlers -> HTTP server
internal/config          Viper-backed Config (env prefix BOT_)
internal/model           domain types: ParsedCommand, FlightInfo, Subscription
internal/api             wire DTOs matching flight-service/subscription-service OpenAPI specs
internal/repository      FlightClient / SubscriptionClient — HTTP ports to the two upstream services
internal/service         CommandParser (text -> ParsedCommand) + BotService (orchestration, dispatch, formatting)
internal/handler         WebhookHandler (Gin)
api/openapi.yaml         this service's own transport contract (webhook + probes)
```

## Command grammar

See [docs/api-contracts/bot-service.md](../../docs/api-contracts/bot-service.md)
for the full table. Summary: `/start`, `/help`, `/flight <NUM>`,
`/route <ORIG> <DEST>`, `/subscribe flight|route ...`,
`/unsubscribe flight|route ...`, `/subscriptions`.

## Running locally

```bash
cp .env.example .env   # fill in a real bot token + secrets
go run ./cmd
```

Requires `flight-service` and `subscription-service` reachable at the URLs
in `.env` (`/ready` will report unhealthy otherwise; the bot still starts
and will reply with a friendly error to commands that need them).

Point Telegram at your local instance during development with a tunnel
(e.g. `ngrok http 8080`), then:

```bash
curl -F "url=https://<tunnel-host>/webhook/telegram/<BOT_TELEGRAM_WEBHOOK_PATH_SECRET>" \
     -F "secret_token=<BOT_TELEGRAM_WEBHOOK_SECRET>" \
     "https://api.telegram.org/bot<BOT_TELEGRAM_BOT_TOKEN>/setWebhook"
```

## Testing

```bash
make test    # go test ./... -cover
```

`internal/service` has full unit coverage of command parsing and dispatch
using fakes for `FlightClient`/`SubscriptionClient`/`telegram.Sender` — no
network or other services required.

## Build

```bash
make build          # local binary at bin/bot-service
make docker-build   # builds from the monorepo root (this module depends on ../../pkg)
```

## Configuration

All settings are environment variables prefixed `BOT_` (see
`.env.example`); in Kubernetes they come from the `bot-service` chart's
ConfigMap/Secret (`deployments/helm/flight-tracker/charts/bot-service`),
never from a committed file.
