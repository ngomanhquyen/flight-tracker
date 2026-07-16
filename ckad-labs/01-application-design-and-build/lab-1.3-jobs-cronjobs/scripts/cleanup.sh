#!/usr/bin/env bash
# Removes everything scripts/solve.sh created. Never touches the real
# sync-service CronJob — only the manual Job triggered from it
# (sync-manual-1), which is not owned by sync-service and won't be removed
# by anything else.
set -euo pipefail
cd "$(dirname "$0")/.."

NAMESPACE=flight-tracker

echo "==> Deleting fail-job, sync-manual-1 (if present)"
kubectl delete job fail-job sync-manual-1 -n "$NAMESPACE" --ignore-not-found

echo "==> Deleting hello-cron (cascades to its child Jobs/Pods)"
kubectl delete cronjob hello-cron -n "$NAMESPACE" --ignore-not-found
