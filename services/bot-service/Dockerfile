# syntax=docker/dockerfile:1
#
# Build context must be the monorepo root (flight-tracker/), since this
# module depends on ../../pkg via a replace directive:
#   docker build -f services/bot-service/Dockerfile -t flight-tracker/bot-service:local .

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY pkg/ pkg/
COPY services/bot-service/ services/bot-service/
WORKDIR /src/services/bot-service

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bot-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/bot-service /bot-service
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bot-service"]
