.PHONY: help build run test lint clean install-tools

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the volume manager binary
	go build -o bin/volume-manager ./cmd/volume-manager

run: ## Run the volume manager locally
	go run ./cmd/volume-manager

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	goimports -w .

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -t sistemica/docker-volume-manager:latest -f deploy/Dockerfile .

docker-run: ## Run in Docker (development)
	docker-compose -f deploy/docker-compose.yml up

docker-stop: ## Stop Docker containers
	docker-compose -f deploy/docker-compose.yml down

.DEFAULT_GOAL := help
