#!/usr/bin/env bash
# Deploys/upgrades the flight-tracker release for local personal testing
# (Docker Desktop Kubernetes + in-cluster Postgres/RabbitMQ/Redis).
#
# The --set-file flags seed postgresql's schema-creation scripts straight
# from docs/database/*.sql (the canonical source) via Bitnami's
# `primary.initdb.scripts` map — Helm's chart .Files can't read paths
# outside the chart directory, but --set-file (evaluated by the Helm CLI,
# not template execution) can, so this avoids keeping a second copy of the
# SQL in sync inside the chart.
#
# Usage: ./deploy-local.sh   (run from this directory)
set -euo pipefail
cd "$(dirname "$0")"

helm dependency update .

helm upgrade --install flight-tracker . \
  -f values.yaml \
  -f values-local.yaml \
  --set-file 'postgresql.primary.initdb.scripts.01-sync-service\.sql'=../../../docs/database/sync-service.sql \
  --set-file 'postgresql.primary.initdb.scripts.02-flight-service\.sql'=../../../docs/database/flight-service.sql \
  -n flight-tracker --create-namespace
