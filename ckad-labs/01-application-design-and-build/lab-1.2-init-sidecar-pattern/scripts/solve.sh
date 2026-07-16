#!/usr/bin/env bash
# Executable version of solution.md.
#
# Part A reads the REAL bot-service Deployment in namespace flight-tracker
# (created only if you've already run deploy-local.sh — see the lab
# README). It is entirely read-only: no resource is created or mutated.
# If bot-service isn't deployed, Part A is skipped with a warning instead
# of failing the script, exactly as the README allows.
#
# Part B is not tied to any real flight-tracker resource — it creates a
# standalone Pod (log-pipeline) from scratch, and cleanup.sh removes it.
#
# Usage: bash scripts/solve.sh   (from the lab directory, or anywhere)
# Safe to re-run without cleanup first: Part A is read-only, and Part B
# uses `kubectl apply` (create-or-update), not `kubectl run`.
set -uo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }
warn() { echo "  [SKIP] $1"; }
FAILED=0

echo "=== Part A: bot-service (real) ==="
POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=bot-service \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

if [ -z "$POD" ]; then
  warn "no bot-service Pod found in namespace ${NAMESPACE} — deploy it first (see README) to run Part A. Continuing with Part B only."
else
  echo "==> Pod: $POD"

  containers=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}')
  echo "    containers: $containers"
  for want in bot-service cloudflared webhook-registrar; do
    echo "$containers" | grep -qw "$want" && pass "container $want present" || fail "container $want missing (containers: $containers)"
  done

  volumes=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}')
  echo "    volumes: $volumes"
  echo "$volumes" | grep -qw "tunnel-log" && pass "emptyDir volume tunnel-log present" || fail "tunnel-log volume missing"

  echo "==> Log tail: cloudflared (last 5 lines)"
  kubectl logs "$POD" -n "$NAMESPACE" -c cloudflared --tail=5 2>&1 | sed 's/^/    /' || true

  echo "==> Log tail: webhook-registrar (last 5 lines)"
  kubectl logs "$POD" -n "$NAMESPACE" -c webhook-registrar --tail=5 2>&1 | sed 's/^/    /' || true

  echo "==> Confirming the two sidecars share a filesystem"
  EXEC_ERR=$(mktemp)
  if kubectl exec "$POD" -n "$NAMESPACE" -c webhook-registrar -- cat /shared/cloudflared.log >/dev/null 2>"$EXEC_ERR"; then
    pass "webhook-registrar can read cloudflared's log via the shared emptyDir"
  else
    fail "webhook-registrar could not read /shared/cloudflared.log — $(head -1 "$EXEC_ERR")"
    if grep -q "server gave HTTP response to HTTPS client" "$EXEC_ERR"; then
      echo "    ^ this is a known Docker Desktop Kubernetes bug in its exec/attach proxy,"
      echo "      not a problem with the Pod or this lab — restart Docker Desktop's"
      echo "      Kubernetes cluster (Settings > Kubernetes > Reset) and retry."
    fi
  fi
  rm -f "$EXEC_ERR"
fi

echo
echo "=== Part B: log-pipeline (standalone, not part of the real app) ==="
mkdir -p manifests
cat > manifests/log-pipeline.yaml <<'YAML'
apiVersion: v1
kind: Pod
metadata:
  name: log-pipeline
  labels:
    app: log-pipeline
spec:
  volumes:
    - name: shared-logs
      emptyDir: {}
  initContainers:
    - name: init-setup
      image: busybox:1.36
      command:
        - sh
        - -c
        - echo "Init completed at $(date)" | tee /var/log/app/init.log
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
  containers:
    - name: app
      image: busybox:1.36
      command:
        - sh
        - -c
        - |
          while true; do
            echo "App tick $(date)" >> /var/log/app/app.log
            sleep 5
          done
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
    - name: log-shipper
      image: busybox:1.36
      command:
        - sh
        - -c
        - touch /var/log/app/app.log && tail -n+1 -f /var/log/app/app.log
      volumeMounts:
        - name: shared-logs
          mountPath: /var/log/app
YAML

kubectl apply -n "$NAMESPACE" -f manifests/log-pipeline.yaml

echo "==> Waiting for Ready (2/2 — init container doesn't count)"
kubectl wait --for=condition=Ready pod/log-pipeline -n "$NAMESPACE" --timeout=60s

init_state=$(kubectl get pod log-pipeline -n "$NAMESPACE" -o jsonpath='{.status.initContainerStatuses[0].state.terminated.reason}' 2>/dev/null || true)
[ "$init_state" = "Completed" ] && pass "init container Completed" || fail "init container state=$init_state (expected Completed)"

kubectl logs log-pipeline -n "$NAMESPACE" -c init-setup 2>&1 | grep -q "Init completed at" \
  && pass "init-setup log has expected line" || fail "init-setup log missing expected line"

EXEC_ERR=$(mktemp)
if kubectl exec log-pipeline -n "$NAMESPACE" -c app -- cat /var/log/app/init.log >/dev/null 2>"$EXEC_ERR"; then
  pass "app container can read init.log via shared emptyDir"
else
  fail "app container could not read /var/log/app/init.log — $(head -1 "$EXEC_ERR")"
  if grep -q "server gave HTTP response to HTTPS client" "$EXEC_ERR"; then
    echo "    ^ known Docker Desktop Kubernetes exec-proxy bug, not a lab problem —"
    echo "      restart Docker Desktop's Kubernetes cluster and retry."
  fi
fi
rm -f "$EXEC_ERR"

echo "==> log-shipper tail (5s sample — portable timeout, no coreutils dependency)"
# Captured to a file rather than piped straight through `sed` in the
# background: a backgrounded pipe's stdio is block-buffered, so killing the
# tail end of the pipe early can silently drop output that was already
# read but not yet flushed. Writing to a file and formatting it after the
# process is confirmed dead avoids that.
TAIL_LOG=$(mktemp)
kubectl logs log-pipeline -n "$NAMESPACE" -c log-shipper -f > "$TAIL_LOG" 2>&1 &
TAIL_PID=$!
sleep 6
kill "$TAIL_PID" 2>/dev/null || true
wait "$TAIL_PID" 2>/dev/null || true
sed 's/^/    /' "$TAIL_LOG"
rm -f "$TAIL_LOG"

echo
if [ "$FAILED" -eq 0 ]; then
  echo "All checks passed. Run scripts/cleanup.sh when done inspecting."
else
  echo "Some checks failed — see [FAIL] lines above." >&2
  exit 1
fi
