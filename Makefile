.PHONY: help dev dev-db dev-db-reset dev-server dev-ui stack-up stack-down stack-logs ui server test test-sdk lint lint-server lint-ui lint-sdk vuln hash-password build release

.DEFAULT_GOAL := help

# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2
# renovate: datasource=go depName=golang.org/x/vuln
GOVULNCHECK_VERSION ?= v1.7.0

TOOLS_BIN := $(CURDIR)/bin
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
COMPOSE_ENV_FILES ?= /dev/null

$(GOLANGCI_LINT):
	@mkdir -p $(TOOLS_BIN)
	GOBIN=$(TOOLS_BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@mv $(TOOLS_BIN)/golangci-lint $@

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

dev: dev-db ## Start PostgreSQL and print dev instructions
	@echo "Run 'make dev-server' and 'make dev-ui' in separate terminals"

dev-db: ## Start local PostgreSQL via Docker Compose
	COMPOSE_ENV_FILES=/dev/null docker compose -f docker-compose.yml up -d postgres

dev-db-reset: ## Reset local PostgreSQL volume and start fresh
	COMPOSE_ENV_FILES=/dev/null docker compose -f docker-compose.yml down -v
	$(MAKE) dev-db

stack-up: ## Start Postgres + openlicensd from the published image
	COMPOSE_ENV_FILES=$(COMPOSE_ENV_FILES) docker compose -f docker-compose.stack.yml pull
	COMPOSE_ENV_FILES=$(COMPOSE_ENV_FILES) docker compose -f docker-compose.stack.yml up -d

stack-down: ## Stop the full stack (add ARGS=-v to drop its data)
	COMPOSE_ENV_FILES=$(COMPOSE_ENV_FILES) docker compose -f docker-compose.stack.yml down $(ARGS)

stack-logs: ## Tail full stack logs
	COMPOSE_ENV_FILES=$(COMPOSE_ENV_FILES) docker compose -f docker-compose.stack.yml logs -f

dev-server: dev-db ## Run the API server (loads .env)
	@test -f .env || (echo "Missing .env. Copy from .env.example: cp .env.example .env" && exit 1)
	@test -f server/internal/static/dist/index.html || (echo "Missing embedded UI placeholder. Run: git checkout -- server/internal/static/dist/index.html (or make ui for a full build)" && exit 1)
	@set -a && . ./.env && set +a && cd server && go run ./cmd/openlicensd

dev-ui: ## Run the Nuxt dev server
	cd ui && npm run dev

ui: ## Build static UI into server/internal/static/dist
	cd ui && npm run generate

VERSION_LDFLAGS := -X github.com/alvarorg14/openlicensd/server/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

server: ## Build binary to bin/openlicensd
	cd server && go build -ldflags "$(VERSION_LDFLAGS)" -o ../bin/openlicensd ./cmd/openlicensd

build: ui server ## Build UI and server binary

test: test-sdk ## Run Go tests
	@set -a && [ -f .env ] && . ./.env; set +a && cd server && go test ./...

test-sdk: ## Run Go SDK tests
	cd sdk/go && go test ./...

lint: lint-server lint-ui lint-sdk ## Run all linters

lint-server: $(GOLANGCI_LINT) ## Run go vet and golangci-lint
	cd server && go vet ./...
	cd server && $(GOLANGCI_LINT) run ./...

lint-ui: ## Run ESLint
	cd ui && npm run lint

lint-sdk: $(GOLANGCI_LINT) ## Run go vet and golangci-lint on the Go SDK
	cd sdk/go && go vet ./...
	cd sdk/go && $(GOLANGCI_LINT) run ./...

vuln: ## Run govulncheck
	cd server && go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...
	cd sdk/go && go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

hash-password: ## Generate bcrypt password hash (PASSWORD=...)
	@test -n "$(PASSWORD)" || (echo "Usage: make hash-password PASSWORD=yourpassword" && exit 1)
	cd server && go run ./cmd/hashpassword $(PASSWORD)

release: ## Run GoReleaser release
	goreleaser release --clean
