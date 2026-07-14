#!/usr/bin/env bash
# Deploys/upgrades the flight-tracker release using images pulled from the
# public Docker Hub registry (docker.io/quyennm3/<service>:demo) instead of
# locally-built images — see values-demo.yaml.
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
