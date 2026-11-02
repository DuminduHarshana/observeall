# =============================================================
# ObserveAll Makefile
# =============================================================

GO          ?= $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)
BINARY      := observeall
BUILD_DIR   := ./bin
MAIN        := ./cmd/observeall/main.go
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-X main.Version=$(VERSION)"
PLATFORMS   := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build install test lint clean docker helm-lint helm-template local-dev

## Default target
all: build

## Build the CLI binary for the current OS/arch
build:
	@echo "Building $(BINARY) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(MAIN)
	@echo "✔ Binary: $(BUILD_DIR)/$(BINARY)"

## Install binary to GOPATH/bin
install:
	$(GO) install $(LDFLAGS) $(MAIN)
	@echo "✔ Installed $(BINARY) to GOPATH/bin"

## Run all tests
test:
	$(GO) test ./... -v -race -coverprofile=coverage.out
	@echo "✔ Tests passed"

## Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## Cross-compile for all platforms
build-all:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		output=$(BUILD_DIR)/$(BINARY)-$$os-$$arch; \
		[ "$$os" = "windows" ] && output=$$output.exe; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$output $(MAIN); \
	done
	@echo "✔ Cross-compiled binaries in $(BUILD_DIR)/"

## Build Docker image for the collector distribution
docker:
	docker build -t observeall/collector:$(VERSION) .
	@echo "✔ Docker image: observeall/collector:$(VERSION)"

## Lint the Helm chart
helm-lint:
	helm lint ./helm/observeall
	@echo "✔ Helm chart lint passed"

## Template the Helm chart (dry run)
helm-template:
	helm template observeall ./helm/observeall

## Deploy to local cluster (requires kind or minikube)
helm-install:
	kubectl create namespace observeall --dry-run=client -o yaml | kubectl apply -f -
	helm upgrade --install observeall ./helm/observeall \
		--namespace observeall \
		--set global.projectName=local-dev \
		--set global.environment=development

## Start local dev stack with docker compose
local-dev:
	docker compose up -d
	@echo ""
	@echo "✔ Local stack running:"
	@echo "  Prometheus:  http://localhost:9090"
	@echo "  Grafana:     http://localhost:3000"
	@echo "  Jaeger UI:   http://localhost:16686"
	@echo "  OTel gRPC:   localhost:4317"
	@echo "  OTel HTTP:   localhost:4318"

## Stop local dev stack
local-dev-stop:
	docker compose down

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out
	@echo "✔ Cleaned"

## Show version
version:
	@echo $(VERSION)

## Show help
help:
	@echo "ObserveAll — Universal OTel Observability Platform"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/^## /  /'
