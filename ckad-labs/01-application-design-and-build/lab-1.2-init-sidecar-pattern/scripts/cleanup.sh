#!/usr/bin/env bash
# Removes everything scripts/solve.sh created (Part B only — Part A never
# creates or mutates anything, it's read-only against the real bot-service).
set -euo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

echo "==> Deleting log-pipeline"
kubectl delete pod log-pipeline -n "$NAMESPACE" --ignore-not-found
