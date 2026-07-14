# syntax=docker/dockerfile:1
#
# Build context must be the monorepo root (flight-tracker/), since this
# module depends on ../../pkg via a replace directive:
#   docker build -f services/sync-service/Dockerfile -t flight-tracker/sync-service:local .

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY pkg/ pkg/
COPY services/sync-service/ services/sync-service/
WORKDIR /src/services/sync-service

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sync-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sync-service /sync-service
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["/sync-service"]
