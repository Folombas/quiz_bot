.PHONY: build run test clean docker-build docker-up docker-down lint

# Variables
BINARY_NAME=quiz-bot
CMD_PATH=cmd/quiz-bot/main.go
BUILD_DIR=.
CONFIG_PATH=configs/config.dev.yaml

# Build binary
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

# Run bot
run:
	go run $(CMD_PATH) -config $(CONFIG_PATH)

# Run with prod config
run-prod:
	go run $(CMD_PATH) -config configs/config.prod.yaml

# Run all tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
test-config:
	go test -v ./internal/config/...

test-logger:
	go test -v ./internal/logger/...

test-ratelimit:
	go test -v ./internal/ratelimit/...

test-models:
	go test -v ./internal/models/...

# Clean
clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean

# Lint
lint:
	golangci-lint run ./...

# Docker build
docker-build:
	docker build -f deploy/Dockerfile -t $(BINARY_NAME) .

# Docker run
docker-up:
	docker-compose -f deploy/docker-compose.yml up -d

# Docker stop
docker-down:
	docker-compose -f deploy/docker-compose.yml down

# Docker logs
docker-logs:
	docker-compose -f deploy/docker-compose.yml logs -f

# Migration create (manual)
migration-create:
	@echo "Create migration file in internal/storage/migrations/"
	@echo "-- Migration: $(name)" > internal/storage/migrations/$(version)_$(name).sql

# Help
help:
	@echo "Available commands:"
	@echo "  make build           - Build binary"
	@echo "  make run             - Run bot (dev)"
	@echo "  make run-prod        - Run bot (prod)"
	@echo "  make test            - Run all tests"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make lint            - Run linter"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-up       - Start Docker"
	@echo "  make docker-down     - Stop Docker"
	@echo "  make docker-logs     - View Docker logs"
