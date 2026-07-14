#!/usr/bin/env bash
# Uninstalls the flight-tracker Helm release. PVCs (Postgres/Redis data)
# are intentionally left behind — Helm never auto-deletes them, to avoid
# silently losing data on a routine uninstall/reinstall cycle. Delete them
# explicitly (see the printed command at the end) for a fully clean slate.
set -euo pipefail
cd "$(dirname "$0")"

helm uninstall flight-tracker -n flight-tracker

echo
echo "Release uninstalled. Remaining PVCs (data preserved):"
kubectl get pvc -n flight-tracker 2>/dev/null || true
echo
echo "To also delete the namespace and all PVCs (irreversible — wipes Postgres/Redis data):"
echo "  kubectl delete namespace flight-tracker"
