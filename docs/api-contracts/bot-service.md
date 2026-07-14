# bot-service contract

`bot-service` exposes no public business REST API. It exposes:

- `POST /webhook/telegram/{secret}` — Telegram webhook receiver. The path
  contains a per-deployment random secret (in addition to Telegram's
  `X-Telegram-Bot-Api-Secret-Token` header check) so the endpoint isn't
  guessable even though it must be internet-reachable via Ingress.
- `GET /health`, `GET /ready`, `GET /metrics` — standard probes.

It is a **client** of `flight-service` and `subscription-service` (contracts
in [flight-service.yaml](flight-service.yaml) and
[subscription-service.yaml](subscription-service.yaml)).

## Webhook endpoint

```
POST /webhook/telegram/{secret}
Header: X-Telegram-Bot-Api-Secret-Token: <configured secret>
Body: Telegram Update JSON (https://core.telegram.org/bots/api#update)
```

Response: always `200 {"ok": true}` immediately after enqueuing the update
for processing (never blocks Telegram waiting on downstream REST calls),
so slow `flight-service`/`subscription-service` responses can't cause
Telegram to retry-storm the webhook. Processing happens asynchronously
in-process; the reply is sent back to the user via `sendMessage` once
downstream calls complete.

## Command grammar

| Command | Args | Behavior |
|---|---|---|
| `/start` | — | Upsert chat via `subscription-service` `POST /api/v1/chats`; send welcome + command list |
| `/help` | — | Send command list |
| `/flight` | `<FLIGHT_NUMBER>` | `GET flight-service /api/v1/flights/{flightNumber}` |
| `/route` | `<ORIGIN> <DEST>` | `GET flight-service /api/v1/flights/route?origin=&destination=` |
| `/subscribe flight` | `<FLIGHT_NUMBER>` | `POST subscription-service /api/v1/subscriptions {type: flight}` |
| `/subscribe route` | `<ORIGIN> <DEST>` | `POST subscription-service /api/v1/subscriptions {type: route}` |
| `/unsubscribe flight` | `<FLIGHT_NUMBER>` | Resolve subscription id, `DELETE /api/v1/subscriptions/{id}` |
| `/unsubscribe route` | `<ORIGIN> <DEST>` | Resolve subscription id, `DELETE /api/v1/subscriptions/{id}` |
| `/subscriptions` | — | `GET subscription-service /api/v1/subscriptions?telegram_chat_id=` and list them |

### Parsing rules

- Case-insensitive command keyword; arguments are case-normalized to
  uppercase IATA/flight-number form (`han` → `HAN`, `vn257` → `VN257`).
- Flight number pattern: `^[A-Z]{2}\d{1,4}[A-Z]?$` (2-letter/3-letter airline
  code + 1-4 digits, optional trailing letter).
- IATA airport code pattern: `^[A-Z]{3}$`.
- Malformed arguments → reply with usage hint for that command, no
  downstream call is made.
- Unknown command → reply pointing to `/help`.

### Example exchanges

```
User: /flight VN257
Bot:  ✈ VN257 (Vietnam Airlines)
      HAN → SGN
      Status: BOARDING
      Gate: 12  Terminal: T1
      Scheduled dep: 14:30  Est: 14:30

User: /subscribe route HAN SGN
Bot:  ✅ Subscribed to route HAN → SGN. You'll get a message when any
      matching flight's status changes.

User: /subscriptions
Bot:  Your subscriptions:
      1. Flight VN257
      2. Route HAN → SGN
      Use /unsubscribe flight <NUM> or /unsubscribe route <ORIG> <DEST> to remove.
```

### Push notification message format (sent by notification-service, not bot-service)

```
🔔 VN257 status changed: DELAYED
HAN → SGN
New estimated departure: 15:10 (was 14:30)
```
