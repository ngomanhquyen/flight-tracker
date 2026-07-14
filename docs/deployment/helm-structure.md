# Helm Chart Structure

## Layout

```
deployments/helm/flight-tracker/          # umbrella chart, deployed as one release
├── Chart.yaml                             # declares Bitnami postgresql/redis/rabbitmq as dependencies (dev/staging only)
├── values.yaml                            # global values: image registry, environment name, domain
├── templates/
│   └── namespace.yaml                     # optional: namespace + default-deny NetworkPolicy baseline
└── charts/
    ├── common/                            # type: library — reusable template snippets, no rendered resources of its own
    │   ├── Chart.yaml
    │   └── templates/
    │       ├── _deployment.tpl            # common.deployment — used by bot/flight/subscription/notification
    │       ├── _service.tpl                # common.service
    │       ├── _hpa.tpl                    # common.hpa
    │       ├── _configmap.tpl              # common.configmap
    │       ├── _probes.tpl                 # common.livenessProbe / common.readinessProbe (hits /health, /ready)
    │       └── _networkpolicy.tpl          # common.networkpolicy
    │
    ├── bot-service/                        # Deployment + Service + Ingress (public webhook) + HPA + ConfigMap + Secret + NetworkPolicy
    ├── flight-service/                      # Deployment + Service + HPA + ConfigMap + Secret + NetworkPolicy
    ├── subscription-service/                # Deployment + Service + HPA + ConfigMap + Secret + NetworkPolicy
    ├── notification-service/                # Deployment + Service + HPA + ConfigMap + Secret + NetworkPolicy
    └── sync-service/                        # CronJob (not Deployment) + ConfigMap + Secret + NetworkPolicy
```

Each service subchart has its own `Chart.yaml` and `values.yaml` so it can be
templated/tested in isolation (`helm template charts/flight-service`), while
`helm install flight-tracker deployments/helm/flight-tracker` deploys
everything as one release with one set of global values (image tag,
environment, hostnames) cascading down via Helm's parent/child values
resolution.

## Why an umbrella chart with a `common` library chart

- All five services need the same shape of Deployment/Service/HPA/probes —
  a `library`-type subchart (`type: library` in its `Chart.yaml`, no
  templates of its own get rendered) holds that shape once and each
  service's templates call `{{ include "common.deployment" . }}`. This
  avoids five near-identical copies of the same YAML drifting apart.
- `sync-service` is the one exception: it renders a `CronJob`, not a
  `Deployment`/`HPA` (a periodic batch job has no steady-state replica count
  to autoscale). Its subchart still reuses `common.configmap` /
  `common.networkpolicy` / `common.secret`.
- One umbrella release means one `helm upgrade` rolls out consistent config
  (e.g. a shared `RABBITMQ_URL` or image tag bump) across services, while
  each subchart's own `values.yaml` still lets a single service's replica
  count/resources be overridden independently
  (`--set flight-service.replicaCount=3`).

## Kubernetes resources produced per chart

| Chart | Deployment | CronJob | Service | Ingress | ConfigMap | Secret | HPA | NetworkPolicy | PVC |
|---|---|---|---|---|---|---|---|---|---|
| bot-service | ✅ | | ✅ (ClusterIP) | ✅ (public, webhook path only) | ✅ | ✅ (bot token, webhook secret) | ✅ | ✅ (allow ingress from Ingress controller only; egress to flight-service, subscription-service, Telegram API) | — |
| flight-service | ✅ | | ✅ (ClusterIP) | — | ✅ | ✅ (DB/Redis creds) | ✅ | ✅ (ingress from bot-service + sync-service only; egress to Postgres/Redis) | — |
| subscription-service | ✅ | | ✅ (ClusterIP) | — | ✅ | ✅ (DB creds) | ✅ | ✅ (ingress from bot-service + notification-service only; egress to Postgres) | — |
| notification-service | ✅ | | ✅ (ClusterIP, metrics only) | — | ✅ | ✅ (DB creds, bot token) | ✅ | ✅ (ingress from RabbitMQ mgmt only for health; egress to subscription-service, RabbitMQ, Telegram API, Postgres) | — |
| sync-service | | ✅ (`schedule: "*/5 * * * *"`) | ✅ (ClusterIP, metrics only, for scrape between runs) | — | ✅ | ✅ (DB creds, public flight API key) | — (batch job, HPA N/A) | ✅ (egress to flight-service, RabbitMQ, Postgres, public API; no ingress needed) | — |

`PVC`: none of our own services are stateful — PostgreSQL, Redis, and
RabbitMQ are declared as chart **dependencies** (Bitnami `postgresql`,
`redis`, `rabbitmq` charts) for local/dev/staging convenience; their PVCs are
managed by those subcharts. Production is expected to point at managed
services (RDS/CloudSQL for Postgres, ElastiCache/Memorystore for Redis, or a
managed RabbitMQ/Amazon MQ) via `values-prod.yaml` overriding
`postgresql.enabled: false` + external connection secrets, so no PVC
ownership exists in our own templates.

## Values override strategy

```
values.yaml            # defaults (used as-is for local/dev)
values-staging.yaml     # staging overrides: resource requests, external DB hosts, ingress host
values-prod.yaml         # prod overrides: HPA min/max, external managed DB/Redis/RabbitMQ endpoints, TLS issuer
```

Deploy with `helm upgrade --install flight-tracker . -f values.yaml -f values-prod.yaml`.

## Secrets

No secret material lives in `values.yaml` or ConfigMaps. Each subchart's
`Secret` template renders keys from `.Values.<service>.secretRefs`, which in
a real cluster is populated by **External Secrets Operator** / Sealed
Secrets / SOPS-encrypted values files (tool choice left to the deploying
org) rather than plaintext in the chart repo. `values.yaml` only ever
contains **references** (secret name/key), never values, for:
`TELEGRAM_BOT_TOKEN`, `TELEGRAM_WEBHOOK_SECRET`, Postgres passwords, Redis
password, RabbitMQ credentials, public flight API key.

## NetworkPolicy default posture

`templates/namespace.yaml` at the umbrella level installs a default-deny
policy for the namespace; each subchart's own `NetworkPolicy` explicitly
allow-lists the exact caller→callee edges from the communication matrix in
[architecture.md](../architecture.md#3-communication-matrix) — e.g.
`subscription-service` only accepts ingress from `bot-service` and
`notification-service` pods (by label selector), never from `sync-service`
or the public Ingress.
