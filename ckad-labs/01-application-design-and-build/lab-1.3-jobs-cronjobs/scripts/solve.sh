#!/usr/bin/env bash
# Executable version of solution.md.
#
# Part A drives the REAL sync-service CronJob in namespace flight-tracker
# (present only if you've deployed the umbrella chart — see the lab
# README). It creates exactly one resource against the real system: a
# manually-triggered Job (sync-manual-1) via `--from=cronjob`. If
# sync-service isn't deployed, Part A is skipped with a warning.
#
# Parts B and C are not tied to any real flight-tracker resource — both
# create standalone Jobs/CronJobs from scratch, removed by cleanup.sh.
#
# Usage: bash scripts/solve.sh   (from the lab directory, or anywhere)
# Re-running: run scripts/cleanup.sh first — `kubectl create job`/
# `kubectl create cronjob` (Parts A and C) error on a name that already
# exists, unlike `kubectl apply`.
set -uo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }
warn() { echo "  [SKIP] $1"; }
FAILED=0

mkdir -p manifests

echo "=== Part A: sync-service (real CronJob) ==="
if ! kubectl get cronjob sync-service -n "$NAMESPACE" >/dev/null 2>&1; then
  warn "no sync-service CronJob in namespace ${NAMESPACE} — deploy it first (see README) to run Part A. Continuing with Parts B-D only."
else
  schedule=$(kubectl get cronjob sync-service -n "$NAMESPACE" -o jsonpath='{.spec.schedule}')
  concurrency=$(kubectl get cronjob sync-service -n "$NAMESPACE" -o jsonpath='{.spec.concurrencyPolicy}')
  backoff=$(kubectl get cronjob sync-service -n "$NAMESPACE" -o jsonpath='{.spec.jobTemplate.spec.backoffLimit}')
  deadline=$(kubectl get cronjob sync-service -n "$NAMESPACE" -o jsonpath='{.spec.jobTemplate.spec.activeDeadlineSeconds}')
  echo "    schedule=$schedule concurrencyPolicy=$concurrency backoffLimit=$backoff activeDeadlineSeconds=$deadline"
  [ -n "$schedule" ] && pass "read schedule" || fail "could not read schedule"
  [ -n "$backoff" ]  && pass "read backoffLimit" || fail "could not read backoffLimit"

  echo "==> Triggering a manual run: kubectl create job --from=cronjob/sync-service"
  kubectl create job sync-manual-1 -n "$NAMESPACE" --from=cronjob/sync-service 2>&1 | sed 's/^/    /' || true

  echo "==> Waiting up to 90s for it to reach Complete or Failed (either is a valid outcome here)"
  if kubectl wait --for=condition=Complete job/sync-manual-1 -n "$NAMESPACE" --timeout=90s >/dev/null 2>&1; then
    echo "    result: Complete"
  elif kubectl wait --for=condition=Failed job/sync-manual-1 -n "$NAMESPACE" --timeout=1s >/dev/null 2>&1; then
    echo "    result: Failed (backoffLimit reached — check logs for the missing dependency)"
  else
    echo "    result: still running after 90s, check manually with: kubectl get job sync-manual-1 -n $NAMESPACE"
  fi

  # Verified against a real cluster: --from=cronjob DOES set ownerReferences
  # (controller:true) back to the CronJob, same as an auto-scheduled Job —
  # see solution.md for the corrected explanation and its consequence
  # (kubectl delete cronjob cascades to this Job too).
  owner=$(kubectl get job sync-manual-1 -n "$NAMESPACE" -o jsonpath='{.metadata.ownerReferences[0].kind}')
  if [ "$owner" = "CronJob" ]; then
    pass "sync-manual-1 has an ownerReference to the CronJob (--from=cronjob does adopt)"
  else
    fail "expected an ownerReference of kind CronJob, got: '$owner'"
  fi
fi

echo
echo "=== Part B: fail-job (standalone) ==="
echo "==> Exporting base manifest with kubectl create job --dry-run=client (the deliverable for this step)"
kubectl create job fail-job -n "$NAMESPACE" --image=busybox --dry-run=client -o yaml \
  -- sh -c "exit 1" > manifests/fail-job.yaml
echo "    saved to manifests/fail-job.yaml"

echo "==> Writing the edited manifest (backoffLimit: 2 added — kubectl create job has no flag for this)"
cat > manifests/fail-job.yaml <<YAML
apiVersion: batch/v1
kind: Job
metadata:
  name: fail-job
  namespace: ${NAMESPACE}
spec:
  backoffLimit: 2
  template:
    spec:
      containers:
        - name: fail-job
          image: busybox
          command: ["sh", "-c", "exit 1"]
      restartPolicy: Never
YAML

kubectl apply -n "$NAMESPACE" -f manifests/fail-job.yaml

echo "==> Waiting up to 180s for backoffLimit to be exceeded (exponential backoff: ~10s/20s/40s between retries)"
kubectl wait --for=condition=Failed job/fail-job -n "$NAMESPACE" --timeout=180s >/dev/null 2>&1 \
  && pass "fail-job reached Failed (BackoffLimitExceeded)" \
  || fail "fail-job did not reach Failed within 180s — check manually"

pod_count=$(kubectl get pods -n "$NAMESPACE" -l job-name=fail-job --no-headers 2>/dev/null | wc -l | tr -d ' ')
echo "    Pod attempts observed: $pod_count"
[ "$pod_count" -ge 2 ] && pass "multiple Pod attempts observed (retry behavior)" || fail "expected >=2 Pod attempts, saw $pod_count"

echo
echo "=== Part C: hello-cron (standalone) ==="
kubectl create cronjob hello-cron -n "$NAMESPACE" \
  --image=busybox \
  --schedule="*/1 * * * *" \
  -- sh -c 'date; echo Hello from CKAD lab'

echo "==> Waiting up to 150s for >=2 child Jobs (schedule is */1 * * * *, so this takes a couple of minutes)"
child_count=0
for _ in $(seq 1 30); do
  child_count=$(kubectl get jobs -n "$NAMESPACE" --no-headers 2>/dev/null | awk '$1 ~ /^hello-cron-/' | wc -l | tr -d ' ')
  [ "$child_count" -ge 2 ] && break
  sleep 5
done
echo "    child Jobs observed: $child_count"
[ "$child_count" -ge 2 ] && pass "CronJob spawned >=2 child Jobs" || fail "expected >=2 child Jobs within 150s, saw $child_count"

first_child=$(kubectl get jobs -n "$NAMESPACE" --no-headers 2>/dev/null | awk '$1 ~ /^hello-cron-/ {print $1; exit}')
if [ -n "$first_child" ]; then
  echo "==> Log of $first_child:"
  kubectl logs "job/${first_child}" -n "$NAMESPACE" 2>&1 | sed 's/^/    /'
fi

echo
echo "=== Part D: Job vs CronJob vs Deployment (reference — not scriptable, see solution.md) ==="
cat <<'TABLE'
    |                                    | Job                    | CronJob                          | Deployment          |
    |------------------------------------|-------------------------|-----------------------------------|----------------------|
    | restartPolicy allowed in template  | Never / OnFailure       | same as Job (shares its template) | Always only          |
    | Pod owner chain                    | Job -> Pod               | CronJob -> Job -> Pod (scheduled) | Deployment -> ReplicaSet -> Pod |
    | Use when                           | run-to-completion, once | recurring on a fixed schedule     | always-on server     |
TABLE

echo
if [ "$FAILED" -eq 0 ]; then
  echo "All checks passed. Run scripts/cleanup.sh when done inspecting."
else
  echo "Some checks failed — see [FAIL] lines above." >&2
  exit 1
fi
