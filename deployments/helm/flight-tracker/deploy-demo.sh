#!/usr/bin/env bash
# Deploys/upgrades the flight-tracker release using images pulled from the
# public Docker Hub registry (docker.io/quyennm3/<service>:demo) instead of
# locally-built images — see values-demo.yaml.
#
# values-demo.yaml is committed with real credentials baked in (disposable
# demo bot, not a product — see the note at the top of that file), so this
# is self-contained: clone the repo, have a Kubernetes context + helm on
# PATH, and run this script — no manual secret setup needed.
#
# Usage: ./deploy-demo.sh   (run from this directory)
set -euo pipefail
cd "$(dirname "$0")"

helm dependency update .

helm upgrade --install flight-tracker . \
  -f values.yaml \
  -f values-demo.yaml \
  --set-file 'postgresql.primary.initdb.scripts.01-sync-service\.sql'=../../../docs/database/sync-service.sql \
  --set-file 'postgresql.primary.initdb.scripts.02-flight-service\.sql'=../../../docs/database/flight-service.sql \
  -n flight-tracker --create-namespace
