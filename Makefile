.PHONY: help dev dev-db dev-server dev-ui ui server test lint hash-password build release

.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

dev: dev-db ## Start PostgreSQL and print dev instructions
	@echo "Run 'make dev-server' and 'make dev-ui' in separate terminals"

dev-db: ## Start local PostgreSQL via Docker Compose
	COMPOSE_ENV_FILES=/dev/null docker compose up -d postgres

dev-server: dev-db ## Run the API server (loads .env)
	@test -f .env || (echo "Missing .env. Copy from .env.example: cp .env.example .env" && exit 1)
	@test -f server/internal/static/dist/index.html || (echo "Missing embedded UI placeholder. Run: git checkout -- server/internal/static/dist/index.html (or make ui for a full build)" && exit 1)
	@set -a && . ./.env && set +a && cd server && go run ./cmd/openlicensd

dev-ui: ## Run the Nuxt dev server
	cd ui && npm run dev

ui: ## Build static UI into server/internal/static/dist
	cd ui && npm run generate

server: ## Build binary to bin/openlicensd
	cd server && go build -o ../bin/openlicensd ./cmd/openlicensd

build: ui server ## Build UI and server binary

test: ## Run Go tests
	@set -a && [ -f .env ] && . ./.env; set +a && cd server && go test ./...

lint: ## Run go vet and ESLint
	cd server && go vet ./...
	cd ui && npm run lint

hash-password: ## Generate bcrypt password hash (PASSWORD=...)
	@test -n "$(PASSWORD)" || (echo "Usage: make hash-password PASSWORD=yourpassword" && exit 1)
	cd server && go run ./cmd/hashpassword $(PASSWORD)

release: ## Run GoReleaser release
	goreleaser release --clean
