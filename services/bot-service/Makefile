BINARY := bin/bot-service
IMAGE  := flight-tracker/bot-service:local

.PHONY: build run test lint tidy fmt docker-build

build:
	go build -o $(BINARY) ./cmd

run: build
	./$(BINARY)

test:
	go test ./... -cover

lint:
	gofmt -l .
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

# Build context is the monorepo root because this module depends on
# ../../pkg (see Dockerfile).
docker-build:
	cd ../.. && docker build -f services/bot-service/Dockerfile -t $(IMAGE) .
