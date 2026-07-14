# Deployment Guide

This describes how the design in [helm-structure.md](helm-structure.md) will
be deployed once Step 8 (Helm templates + CI/CD) lands. It's written now so
the target operating model is clear before implementation starts.

## Prerequisites

- Kubernetes cluster (1.28+) with an Ingress controller (nginx or cloud LB)
  and cert-manager for TLS.
- Helm 3.14+.
- A Telegram bot token from [@BotFather](https://t.me/BotFather) and a
  public HTTPS URL for the webhook (the cluster's Ingress host).
- Credentials for the chosen public flight-data API.
- Either the bundled dev dependencies (Bitnami PostgreSQL/Redis/RabbitMQ,
  enabled by default) or managed equivalents (RDS/CloudSQL, ElastiCache,
  Amazon MQ/CloudAMQP) for staging/prod.
- A secrets mechanism wired into the cluster: External Secrets Operator,
  Sealed Secrets, or SOPS — `values.yaml` only ever references secret
  name/key, never plaintext (see helm-structure.md → Secrets).

## First-time install (dev/local)

```bash
kubectl create namespace flight-tracker
kubectl -n flight-tracker create secret generic telegram-credentials \
  --from-literal=bot-token=<TELEGRAM_BOT_TOKEN> \
  --from-literal=webhook-secret=<RANDOM_SECRET>
kubectl -n flight-tracker create secret generic flight-api-credentials \
  --from-literal=api-key=<PUBLIC_FLIGHT_API_KEY>

helm dependency update deployments/helm/flight-tracker
helm install flight-tracker deployments/helm/flight-tracker \
  --namespace flight-tracker \
  -f deployments/helm/flight-tracker/values.yaml
```

Then register the webhook with Telegram (one-time, or automated as a Helm
post-install hook in Step 8):

```bash
curl -F "url=https://bot.flight-tracker.local/webhook/telegram/<path-secret>" \
     -F "secret_token=<RANDOM_SECRET>" \
     "https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook"
```

## Staging / production

```bash
helm upgrade --install flight-tracker deployments/helm/flight-tracker \
  --namespace flight-tracker \
  -f deployments/helm/flight-tracker/values.yaml \
  -f deployments/helm/flight-tracker/values-prod.yaml
```

`values-prod.yaml` (added in Step 8) is expected to:

- Set `postgresql.enabled: false`, `redis.enabled: false`,
  `rabbitmq.enabled: false` and point each service's env at managed
  endpoints instead.
- Raise HPA `minReplicas`/`maxReplicas` per service based on observed load.
- Set the real Ingress host + `cert-manager.io/cluster-issuer` annotation
  for `bot-service`.
- Tighten `NetworkPolicy` egress to the managed DB/cache/broker CIDRs.

## Rollout safety

- Each Deployment uses `RollingUpdate` (`maxUnavailable: 0`,
  `maxSurge: 1`) so a bad image never drops available replicas to zero.
- `readinessProbe` (`/ready`) gates traffic — a pod that can't reach its
  database/cache/broker is never added to the Service's endpoints.
- `sync-service`'s CronJob uses `concurrencyPolicy: Forbid` so a slow run
  never overlaps the next scheduled trigger.
- Database schema migrations (per service, via a migration tool chosen in
  Step 2) run as a Helm pre-upgrade hook `Job`, before the Deployment
  rollout, so new code never runs against an un-migrated schema.

## Observability

- Every service exposes `/metrics` (Prometheus format); a
  `ServiceMonitor`/`PodMonitor` (if Prometheus Operator is present) or plain
  scrape annotations are added per subchart in Step 8.
- Structured JSON logs (Zap) include `correlation_id`, making it possible to
  trace a single Telegram command or sync run across services in a log
  aggregator.
- `sync.poll_runs` (see [database/sync-service.sql](../database/sync-service.sql))
  and `notification.delivery_log` (see
  [database/notification-service.sql](../database/notification-service.sql))
  give an in-database audit trail for "did the 07:10 poll run, and did chat
  X get notified" without needing to grep logs.

## Rollback

`helm rollback flight-tracker <revision>` reverts all subcharts to a prior
release atomically. Because each service owns its own schema and migrations
are additive/backward-compatible within a release line (per the versioning
rule in [events/event-catalog.md](../events/event-catalog.md)), a rollback
does not require a corresponding down-migration in the common case — only
breaking schema changes need a documented rollback migration, called out
explicitly in that change's PR.
