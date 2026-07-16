#!/usr/bin/env bash
# Removes everything scripts/solve.sh created, including the label/
# annotation added to the real bot-service Deployment/Pod in Parts C5/D2
# (both additive-only changes, safe to remove independently).
set -uo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

echo "==> Deleting lab Pods"
# By exact name, not `-l tier=web` — Lab 1.1's speed-pod also carries
# tier=web (verified: an earlier version of this script deleted it by
# accident via that selector). Scoping by name keeps each lab's cleanup
# from reaching into another lab's resources.
kubectl delete pod pod-a1 pod-a2 pod-a3 pod-b1 pod-b2 pod-b3 pod-c1 -n "$NAMESPACE" --ignore-not-found

echo "==> Removing label/annotation from the real app, if C5/D2 were run"
BOT_POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=bot-service,lab=ckad-1.4 \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
if [ -n "$BOT_POD" ]; then
  kubectl label pod "$BOT_POD" -n "$NAMESPACE" lab- >/dev/null
  echo "    removed lab= from $BOT_POD"
fi
kubectl annotate deployment bot-service -n "$NAMESPACE" inspected-by- >/dev/null 2>&1 || true
