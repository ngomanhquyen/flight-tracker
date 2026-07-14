# syntax=docker/dockerfile:1
#
# Build context must be the monorepo root (flight-tracker/), since this
# module depends on ../../pkg via a replace directive:
#   docker build -f services/flight-service/Dockerfile -t flight-tracker/flight-service:local .

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY pkg/ pkg/
COPY services/flight-service/ services/flight-service/
WORKDIR /src/services/flight-service

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/flight-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/flight-service /flight-service
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/flight-service"]
