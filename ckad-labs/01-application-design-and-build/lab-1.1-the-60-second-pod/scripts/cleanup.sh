#!/usr/bin/env bash
# Removes everything scripts/solve.sh created. Does not touch the
# flight-tracker namespace itself (shared with later labs).
set -euo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

echo "==> Deleting speed-pod / speed-pod-v2"
kubectl delete pod speed-pod speed-pod-v2 -n "$NAMESPACE" --ignore-not-found
