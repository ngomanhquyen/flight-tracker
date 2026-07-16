#!/usr/bin/env bash
# Executable version of solution.md. Not tied to any real flight-tracker
# resource (Lab 1.1 is a standalone Pod exercise) — everything here is
# created fresh and torn down by cleanup.sh.
#
# Usage: bash scripts/solve.sh   (from the lab directory, or anywhere)
# Re-running: run scripts/cleanup.sh first — this script isn't idempotent
# (kubectl run errors on a Pod that already exists).
set -euo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker
POD=speed-pod

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }
FAILED=0

echo "==> Ensuring namespace ${NAMESPACE} exists (idempotent)"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - >/dev/null

mkdir -p manifests

echo "==> Exporting base manifest with kubectl run --dry-run=client (the deliverable for step 3)"
kubectl run "$POD" -n "$NAMESPACE" \
  --image=nginx:1.25 \
  --labels="app=speed,tier=web,course=ckad" \
  --env="APP_ENV=production" \
  --env="LOG_LEVEL=debug" \
  --dry-run=client -o yaml > manifests/speed-pod.yaml
echo "    saved to manifests/speed-pod.yaml"

echo "==> Writing the edited manifest (resources block added — what step 4 does by hand)"
cat > manifests/speed-pod.yaml <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: ${POD}
  namespace: ${NAMESPACE}
  labels:
    app: speed
    tier: web
    course: ckad
spec:
  containers:
    - name: ${POD}
      image: nginx:1.25
      env:
        - name: APP_ENV
          value: production
        - name: LOG_LEVEL
          value: debug
      resources:
        requests:
          cpu: "100m"
          memory: "128Mi"
        limits:
          cpu: "250m"
          memory: "256Mi"
YAML

echo "==> Applying"
kubectl apply -f manifests/speed-pod.yaml

echo "==> Waiting for Ready"
kubectl wait --for=condition=Ready "pod/${POD}" -n "$NAMESPACE" --timeout=60s

echo "==> Verifying (no editor, jsonpath/get/describe only)"

status=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
[ "$status" = "Running" ] && pass "phase=Running" || fail "phase=$status (expected Running)"

labels=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.metadata.labels}')
echo "$labels" | grep -q '"app":"speed"'   && pass "label app=speed"   || fail "label app missing/wrong"
echo "$labels" | grep -q '"tier":"web"'    && pass "label tier=web"    || fail "label tier missing/wrong"
echo "$labels" | grep -q '"course":"ckad"' && pass "label course=ckad" || fail "label course missing/wrong"

resources=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources}')
echo "$resources" | grep -q '"cpu":"100m"'    && pass "requests.cpu=100m"    || fail "requests.cpu wrong: $resources"
echo "$resources" | grep -q '"memory":"128Mi"' && pass "requests.memory=128Mi" || fail "requests.memory wrong: $resources"
echo "$resources" | grep -q '"cpu":"250m"'    && pass "limits.cpu=250m"      || fail "limits.cpu wrong: $resources"
echo "$resources" | grep -q '"memory":"256Mi"' && pass "limits.memory=256Mi"  || fail "limits.memory wrong: $resources"

echo
echo "==> Bonus: the zero-editor --overrides one-liner (creates speed-pod-v2)"
kubectl run speed-pod-v2 -n "$NAMESPACE" \
  --image=nginx:1.25 \
  --overrides='{
    "metadata": {"labels": {"app":"speed","tier":"web","course":"ckad"}},
    "spec": {
      "containers": [{
        "name": "speed-pod-v2",
        "image": "nginx:1.25",
        "env": [
          {"name":"APP_ENV","value":"production"},
          {"name":"LOG_LEVEL","value":"debug"}
        ],
        "resources": {
          "requests": {"cpu":"100m","memory":"128Mi"},
          "limits": {"cpu":"250m","memory":"256Mi"}
        }
      }]
    }
  }'
kubectl wait --for=condition=Ready pod/speed-pod-v2 -n "$NAMESPACE" --timeout=60s

echo
if [ "$FAILED" -eq 0 ]; then
  echo "All checks passed. Run scripts/cleanup.sh when done inspecting."
else
  echo "Some checks failed — see [FAIL] lines above." >&2
  exit 1
fi
