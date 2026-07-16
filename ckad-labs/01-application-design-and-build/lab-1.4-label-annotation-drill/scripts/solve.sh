#!/usr/bin/env bash
# Executable version of solution.md.
#
# Parts A/B1/C/D are standalone — 7 throwaway Pods created fresh, removed
# by cleanup.sh. Parts B2/C5/D2 read/label/annotate the REAL bot-service
# Deployment+Pods in namespace flight-tracker (present only if you've run
# deploy-local.sh — see the lab README); both are additive, non-destructive
# operations that never touch the Deployment's Pod template, so they can't
# trigger a rollout or affect matching/scheduling. Skipped with a warning
# if bot-service isn't deployed.
#
# Usage: bash scripts/solve.sh   (from the lab directory, or anywhere)
# Re-running: run scripts/cleanup.sh first — Part A uses `kubectl run`,
# which errors on a Pod name that already exists.
set -uo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }
warn() { echo "  [SKIP] $1"; }
FAILED=0

echo "=== Part A: bulk-create ==="
for i in 1 2 3; do
  kubectl run "pod-a${i}" -n "$NAMESPACE" --image=nginx:alpine --labels="env=dev,tier=web" >/dev/null
done
for i in 1 2 3; do
  kubectl run "pod-b${i}" -n "$NAMESPACE" --image=nginx:alpine --labels="env=prod,tier=web" >/dev/null
done
kubectl run pod-c1 -n "$NAMESPACE" --image=nginx:alpine --labels="env=dev,tier=db" >/dev/null
echo "    created pod-a1..a3, pod-b1..b3, pod-c1"

count=$(kubectl get pods -n "$NAMESPACE" -l 'tier in (web,db)' --no-headers 2>/dev/null | grep -c '^pod-')
[ "$count" -eq 7 ] && pass "7 pods created" || fail "expected 7 pods, found $count"

echo
echo "=== Part B1: selectors on the Pods you just created ==="
n() { kubectl get pods -n "$NAMESPACE" "$@" --no-headers 2>/dev/null | wc -l | tr -d ' '; }

echo "    env=dev            -> $(n -l env=dev) pods"
echo "    env in (dev,prod)  -> $(n -l 'env in (dev,prod)') pods"
echo "    tier=web,env=dev   -> $(n -l tier=web,env=dev) pods"
echo "    tier!=db (namespace-wide) -> $(n -l 'tier!=db') pods   # not asserted: also matches every other pod in the namespace lacking a tier label at all"
echo "    tier,tier!=db (scoped)    -> $(n -l 'tier,tier!=db') pods"
[ "$(n -l env=dev)" -eq 4 ] && pass "env=dev -> 4" || fail "env=dev count wrong"
[ "$(n -l 'env in (dev,prod)')" -eq 7 ] && pass "env in (dev,prod) -> 7" || fail "set-based count wrong"
[ "$(n -l tier=web,env=dev)" -eq 3 ] && pass "tier=web,env=dev -> 3" || fail "AND count wrong"
# Plain `tier!=db` is a namespace-wide match (also true for pods with no
# tier label at all — verified against a real cluster, see solution.md) so
# it can't be asserted against a fixed count here. `tier,tier!=db` (label
# must exist AND differ) is the scoped equivalent and is what we assert.
[ "$(n -l 'tier,tier!=db')" -eq 6 ] && pass "tier,tier!=db -> 6 (scoped negation)" || fail "scoped negation count wrong"

echo
echo "=== Part B2: selectors on the real app ==="
part_of_count=$(kubectl get deployments,services -n "$NAMESPACE" -l app.kubernetes.io/part-of=flight-tracker --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$part_of_count" -eq 0 ]; then
  warn "no Deployments/Services labeled app.kubernetes.io/part-of=flight-tracker — deploy the app first (see README) to run B2/C5/D2"
else
  echo "    deployments,services -l part-of=flight-tracker -> $part_of_count objects"
  pass "part-of selector matches Deployments/Services"

  pods_with_part_of=$(n -l app.kubernetes.io/part-of=flight-tracker)
  echo "    same selector against Pods                     -> $pods_with_part_of pods"
  [ "$pods_with_part_of" -eq 0 ] \
    && pass "part-of does NOT propagate to Pods (selectorLabels only carries 'name' — see solution.md)" \
    || fail "expected 0 Pods to match part-of, found $pods_with_part_of"

  echo "    app.kubernetes.io/name=bot-service        -> $(n -l app.kubernetes.io/name=bot-service) pods"
  echo "    app.kubernetes.io/name in (bot-service,flight-service,sync-service) -> $(n -l 'app.kubernetes.io/name in (bot-service,flight-service,sync-service)') pods"
  [ "$(n -l app.kubernetes.io/name=bot-service)" -ge 1 ] && pass "name=bot-service matches at least one Pod" || fail "name=bot-service matched no Pods"

  lab_pods_in_real_set=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name \
    -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep -c '^pod-' || true)
  [ "$lab_pods_in_real_set" -eq 0 ] \
    && pass "lab Pods (pod-a*/b*/c1) do not appear in the real-app selector (disjoint label sets, as expected)" \
    || fail "lab Pods unexpectedly matched the real-app selector"
fi

echo
echo "=== Part C: bulk label update ==="
kubectl label pods -n "$NAMESPACE" -l tier=web owner=team-a >/dev/null
pass "owner=team-a applied to tier=web pods"

ERR_FILE=$(mktemp)
if kubectl label pods -n "$NAMESPACE" -l tier=web owner=team-b >/dev/null 2>"$ERR_FILE"; then
  fail "expected an error without --overwrite, but the command succeeded"
else
  grep -q "overwrite is false" "$ERR_FILE" \
    && pass "overwrite without --overwrite correctly rejected" \
    || fail "command failed but not with the expected 'overwrite is false' message"
fi
rm -f "$ERR_FILE"

kubectl label pods -n "$NAMESPACE" -l tier=web owner=team-b --overwrite >/dev/null
owner_now=$(kubectl get pod pod-a1 -n "$NAMESPACE" -o jsonpath='{.metadata.labels.owner}')
[ "$owner_now" = "team-b" ] && pass "owner overwritten to team-b with --overwrite" || fail "owner is '$owner_now', expected team-b"

kubectl label pod pod-c1 -n "$NAMESPACE" tier- >/dev/null
tier_now=$(kubectl get pod pod-c1 -n "$NAMESPACE" -o jsonpath='{.metadata.labels.tier}')
[ -z "$tier_now" ] && pass "tier label removed from pod-c1" || fail "tier label still present: $tier_now"

BOT_POD=""
if [ "$part_of_count" -gt 0 ]; then
  echo
  echo "=== Part C5: label a real bot-service Pod (additive, safe) ==="
  BOT_POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  if [ -n "$BOT_POD" ]; then
    kubectl label pod "$BOT_POD" -n "$NAMESPACE" lab=ckad-1.4 >/dev/null
    live_label=$(kubectl get pod "$BOT_POD" -n "$NAMESPACE" -o jsonpath='{.metadata.labels.lab}')
    [ "$live_label" = "ckad-1.4" ] && pass "label applied to live Pod $BOT_POD" || fail "label not found on $BOT_POD"

    template_labels=$(kubectl get deployment bot-service -n "$NAMESPACE" -o jsonpath='{.spec.template.metadata.labels}')
    echo "$template_labels" | grep -q '"lab"' \
      && fail "unexpected: Deployment's pod template also has 'lab' — it should not" \
      || pass "Deployment's pod template does NOT have 'lab' (confirms the label won't survive a Pod recreate)"
  else
    warn "could not find a bot-service Pod, skipping C5"
  fi
fi

echo
echo "=== Part D: bulk annotate ==="
kubectl annotate pods -n "$NAMESPACE" -l env=prod description="Production workload - do not delete" >/dev/null
desc=$(kubectl get pod pod-b1 -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.description}')
[ "$desc" = "Production workload - do not delete" ] && pass "annotation applied to env=prod pods" || fail "annotation missing/wrong on pod-b1"

if [ "$part_of_count" -gt 0 ]; then
  echo
  echo "=== Part D2: annotate the real bot-service Deployment (safe — no rollout) ==="
  before=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[*].metadata.creationTimestamp}')
  kubectl annotate deployment bot-service -n "$NAMESPACE" inspected-by="ckad-lab-1.4" --overwrite >/dev/null
  sleep 2
  after=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[*].metadata.creationTimestamp}')
  [ "$before" = "$after" ] && pass "Pod creationTimestamps unchanged — no rollout was triggered" || fail "Pod creationTimestamps changed — a rollout happened unexpectedly"
fi

echo
if [ "$FAILED" -eq 0 ]; then
  echo "All checks passed. Run scripts/cleanup.sh when done inspecting."
else
  echo "Some checks failed — see [FAIL] lines above." >&2
  exit 1
fi
