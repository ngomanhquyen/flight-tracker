{{/*
Common library templates shared by every service subchart. Each subchart
invokes these via {{ include "common.xxx" . }} from its own
templates/*.yaml — see docs/deployment/helm-structure.md for the design.
*/}}

{{/* Standard labels applied to every resource. */}}
{{- define "common.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/part-of: flight-tracker
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels — must be a subset of common.labels and immutable. */}}
{{- define "common.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
{{- end -}}

{{/*
Fully qualified image reference: <image.registry>/<image.repository>:<image.tag>.
Deliberately a plain (non-global) per-subchart value, not
global.imageRegistry/imageTag — Bitnami's postgresql/redis/rabbitmq
subcharts declare their own defaults under that same bare "global" key
(e.g. global.imageRegistry: ""), and Helm's global-value merge across a
mixed dependency tree does not reliably deep-merge those with a custom
global key of ours, so anything under global.* here is unsafe to rely on.
*/}}
{{- define "common.image" -}}
{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end -}}

{{/* Liveness/readiness probes, included inside a container spec. */}}
{{- define "common.probes" -}}
livenessProbe:
  httpGet:
    path: {{ .Values.probes.liveness.path }}
    port: http
  initialDelaySeconds: {{ .Values.probes.liveness.initialDelaySeconds }}
  periodSeconds: {{ .Values.probes.liveness.periodSeconds }}
readinessProbe:
  httpGet:
    path: {{ .Values.probes.readiness.path }}
    port: http
  initialDelaySeconds: {{ .Values.probes.readiness.initialDelaySeconds }}
  periodSeconds: {{ .Values.probes.readiness.periodSeconds }}
{{- end -}}

{{/*
ConfigMap of plain (non-secret) env vars from .Values.env. Consumed via
envFrom in common.deployment / common.cronjob.
*/}}
{{- define "common.configmap" -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
data:
  {{- range $key, $value := .Values.env }}
  {{ $key }}: {{ $value | quote }}
  {{- end }}
{{- end -}}

{{/*
Secret of sensitive env vars from .Values.secrets. For this local/dev
deployment, values are passed directly (e.g. via a values-local.yaml kept
out of version control) — see deployments/helm/flight-tracker/values-local.yaml.example.
A real deployment replaces this template's values source with keys
populated by External Secrets Operator / Sealed Secrets / SOPS instead
(docs/deployment/helm-structure.md's Secrets section), without changing
any consumer of the Secret.
*/}}
{{- define "common.secret" -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
type: Opaque
stringData:
  {{- range $key, $value := .Values.secrets }}
  {{ $key }}: {{ $value | quote }}
  {{- end }}
{{- end -}}

{{/* Standard Service: ClusterIP (or overridden type) on .Values.service.port, named "http". */}}
{{- define "common.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - name: http
      port: {{ .Values.service.port }}
      targetPort: http
  selector:
    {{- include "common.selectorLabels" . | nindent 4 }}
{{- end -}}

{{/* HorizontalPodAutoscaler, rendered only when .Values.autoscaling.enabled. */}}
{{- define "common.hpa" -}}
{{- if .Values.autoscaling.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .Chart.Name }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
{{- end }}
{{- end -}}

{{/*
NetworkPolicy from .Values.networkPolicy.{ingressFrom,egressTo}. Entries
are either a plain service name (allow that service's pods), or one of the
special markers used in this project's values files:
ingressControllerOnly, prometheusOnly, external (a name/comment only —
egress to the public internet, left open since NetworkPolicy can't select
by DNS name). Not enforced by clusters without a policy-aware CNI (e.g.
Docker Desktop's default Kubernetes) — safe to include everywhere
regardless, since an unenforced NetworkPolicy is simply inert.
*/}}
{{- define "common.networkpolicy" -}}
{{- if .Values.networkPolicy }}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  podSelector:
    matchLabels:
      {{- include "common.selectorLabels" . | nindent 6 }}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    {{- range .Values.networkPolicy.ingressFrom }}
    {{- if kindIs "map" . }}
    {{- if .ingressControllerOnly }}
    - from:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              app.kubernetes.io/name: ingress-nginx
    {{- else if .prometheusOnly }}
    - from:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: prometheus
    {{- end }}
    {{- else }}
    - from:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: {{ . }}
    {{- end }}
    {{- end }}
  egress:
    # DNS is always allowed; every service needs it to resolve other
    # in-cluster Services and (for egress: external entries) the public internet.
    - to: []
      ports:
        - protocol: UDP
          port: 53
        - protocol: TCP
          port: 53
    {{- range .Values.networkPolicy.egressTo }}
    {{- if kindIs "map" . }}
    - to: [] # external (public internet) — see the entry's comment in values.yaml
    {{- else }}
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: {{ . }}
    {{- end }}
    {{- end }}
{{- end }}
{{- end -}}

{{/* Ingress, rendered only when .Values.ingress.enabled (bot-service only today). */}}
{{- define "common.ingress" -}}
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
  annotations:
    {{- if .Values.ingress.tlsSecretName }}
    cert-manager.io/cluster-issuer: letsencrypt
    {{- end }}
spec:
  {{- if .Values.ingress.tlsSecretName }}
  tls:
    - hosts:
        - {{ .Values.ingress.host }}
      secretName: {{ .Values.ingress.tlsSecretName }}
  {{- end }}
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: {{ .Values.ingress.path }}
            pathType: Prefix
            backend:
              service:
                name: {{ .Chart.Name }}
                port:
                  name: http
{{- end }}
{{- end -}}

{{/*
initContainers: a wait-for-db check and/or a log-shipping sidecar, both
gated per-subchart (dbCheck.enabled / logShipping.enabled — every subchart
using common.deployment/common.cronjob must declare both blocks, false is
fine, so this never hits a missing key).

The log-shipper is deliberately a *native* sidecar (restartPolicy: Always on
an initContainer, GA since k8s 1.29) rather than a regular container: it
tails this pod's log file (written by the main container to the shared
app-log emptyDir when LOG_FILE_PATH is set — see pkg/logger) and appends it
into a service-named file on the centrally-mounted PVC. `tail -F` never
exits on its own — as a regular sidecar in `containers:` that would block a
Job/CronJob pod from ever reaching Completed (restartPolicy: Never requires
ALL containers to exit), starving every later run once concurrencyPolicy:
Forbid kicks in. A native sidecar avoids this: the kubelet sends it SIGTERM
automatically once the pod's regular containers have all finished, letting
the Job complete normally. This matters for sync-service's CronJob; it's
harmless overhead for bot-service/flight-service's long-running Deployments.
*/}}
{{- define "common.initContainers" -}}
{{- if or .Values.dbCheck.enabled .Values.logShipping.enabled }}
initContainers:
  {{- if .Values.dbCheck.enabled }}
  - name: wait-for-db
    image: busybox:1.36
    command:
      - sh
      - -c
      - |
        until nc -z -w2 {{ .Values.dbCheck.host }} {{ .Values.dbCheck.port }}; do
          echo "waiting for db at {{ .Values.dbCheck.host }}:{{ .Values.dbCheck.port }}..."
          sleep 2
        done
    resources:
      requests: { cpu: 100m, memory: 100Mi }
  {{- end }}
  {{- if .Values.logShipping.enabled }}
  - name: log-shipper
    image: busybox:1.36
    restartPolicy: Always
    command:
      - sh
      - -c
      - |
        trap 'kill $TAIL_PID 2>/dev/null; exit 0' TERM
        touch /central-logs/{{ .Chart.Name }}.log
        tail -F /var/log/app/{{ .Chart.Name }}.log >> /central-logs/{{ .Chart.Name }}.log &
        TAIL_PID=$!
        wait $TAIL_PID
    volumeMounts:
      - name: app-log
        mountPath: /var/log/app
        readOnly: true
      - name: central-logs
        mountPath: /central-logs
    resources:
      requests: { cpu: 10m, memory: 16Mi }
      limits: { cpu: 50m, memory: 32Mi }
  {{- end }}
{{- end }}
{{- end -}}

{{/* Deployment: standard shape used by bot-service and flight-service. */}}
{{- define "common.deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      {{- include "common.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "common.selectorLabels" . | nindent 8 }}
      annotations:
        owner: vht
        purpose: ckad-lab
    spec:
      {{- include "common.initContainers" . | nindent 6 }}
      containers:
        - name: {{ .Chart.Name }}
          image: {{ include "common.image" . }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.port }}
          envFrom:
            - configMapRef:
                name: {{ .Chart.Name }}
            - secretRef:
                name: {{ .Chart.Name }}
                optional: true
          {{- if .Values.logShipping.enabled }}
          env:
            - name: LOG_FILE_PATH
              value: /var/log/app/{{ .Chart.Name }}.log
          {{- end }}
          {{- include "common.probes" . | nindent 10 }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- if .Values.logShipping.enabled }}
          volumeMounts:
            - name: app-log
              mountPath: /var/log/app
          {{- end }}
        {{- with .Values.extraContainers }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- if or .Values.extraVolumes .Values.logShipping.enabled }}
      volumes:
        {{- if .Values.logShipping.enabled }}
        - name: app-log
          emptyDir: {}
        - name: central-logs
          persistentVolumeClaim:
            claimName: {{ .Values.logShipping.centralLogsPVCName }}
        {{- end }}
        {{- with .Values.extraVolumes }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- end }}
{{- end -}}

{{/* CronJob: standard shape used by sync-service (a periodic batch job, not a Deployment). */}}
{{- define "common.cronjob" -}}
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{ .Chart.Name }}
  labels:
    {{- include "common.labels" . | nindent 4 }}
spec:
  schedule: {{ .Values.cronjob.schedule | quote }}
  concurrencyPolicy: {{ .Values.cronjob.concurrencyPolicy }}
  successfulJobsHistoryLimit: {{ .Values.cronjob.successfulJobsHistoryLimit }}
  failedJobsHistoryLimit: {{ .Values.cronjob.failedJobsHistoryLimit }}
  jobTemplate:
    spec:
      activeDeadlineSeconds: {{ .Values.cronjob.activeDeadlineSeconds }}
      backoffLimit: {{ .Values.cronjob.backoffLimit }}
      template:
        metadata:
          labels:
            {{- include "common.selectorLabels" . | nindent 12 }}
        spec:
          restartPolicy: Never
          {{- include "common.initContainers" . | nindent 10 }}
          containers:
            - name: {{ .Chart.Name }}
              image: {{ include "common.image" . }}
              imagePullPolicy: {{ .Values.image.pullPolicy }}
              ports:
                - name: http
                  containerPort: {{ .Values.service.port }}
              envFrom:
                - configMapRef:
                    name: {{ .Chart.Name }}
                - secretRef:
                    name: {{ .Chart.Name }}
                    optional: true
              {{- if .Values.logShipping.enabled }}
              env:
                - name: LOG_FILE_PATH
                  value: /var/log/app/{{ .Chart.Name }}.log
              {{- end }}
              resources:
                {{- toYaml .Values.resources | nindent 16 }}
              {{- if .Values.logShipping.enabled }}
              volumeMounts:
                - name: app-log
                  mountPath: /var/log/app
              {{- end }}
          {{- if .Values.logShipping.enabled }}
          volumes:
            - name: app-log
              emptyDir: {}
            - name: central-logs
              persistentVolumeClaim:
                claimName: {{ .Values.logShipping.centralLogsPVCName }}
          {{- end }}
{{- end -}}
