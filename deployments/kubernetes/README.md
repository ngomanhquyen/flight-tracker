# deployments/kubernetes

The Helm chart in `../helm/flight-tracker` is the source of truth for all
application resources (Deployments, Services, ConfigMaps, Secrets, HPAs,
CronJob, NetworkPolicies). This directory does **not** duplicate those
templates. It holds two things instead:

- `base/` — cluster-scoped resources that are not part of the Helm release
  (e.g. `Namespace`, `IngressClass`/`ClusterIssuer` for cert-manager, any
  CRDs required by the RabbitMQ/Postgres operators if the org runs operators
  instead of the Bitnami subcharts).
- `overlays/{dev,staging,prod}/` — Kustomize overlays that patch the
  **rendered output** of `helm template` for environments where GitOps
  tooling (e.g. Argo CD with Kustomize) is preferred over a live `helm
  upgrade`. Each overlay's `kustomization.yaml` will reference
  `helm template deployments/helm/flight-tracker -f values-<env>.yaml` as
  its base once Step 8 lands.

This structure is intentionally thin at this stage — see
[docs/deployment/helm-structure.md](../../docs/deployment/helm-structure.md)
for the actual resource design and
[docs/deployment/deployment-guide.md](../../docs/deployment/deployment-guide.md)
for how these pieces fit together at deploy time.
