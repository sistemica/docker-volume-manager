.PHONY: help build build-manager build-csi run test lint clean install-tools plugin-build plugin-enable plugin-disable plugin-remove

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build both volume manager and CSI plugin binaries
	go build -o bin/volume-manager ./cmd/volume-manager
	go build -o bin/csi-plugin ./cmd/csi-plugin

build-manager: ## Build only the volume manager binary
	go build -o bin/volume-manager ./cmd/volume-manager

build-csi: ## Build only the CSI plugin binary
	go build -o bin/csi-plugin ./cmd/csi-plugin

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

plugin-build: ## Build Docker managed CSI plugin
	./plugin/build-plugin.sh

plugin-enable: ## Enable the CSI plugin
	docker plugin enable sistemica/docker-volume-manager-csi:latest

plugin-disable: ## Disable the CSI plugin
	docker plugin disable sistemica/docker-volume-manager-csi:latest

plugin-remove: ## Remove the CSI plugin
	docker plugin rm -f sistemica/docker-volume-manager-csi:latest

.DEFAULT_GOAL := help
