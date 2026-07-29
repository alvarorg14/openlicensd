.PHONY: dev dev-server dev-ui ui server test lint hash-password build release

dev: dev-db
	@echo "Run 'make dev-server' and 'make dev-ui' in separate terminals"

dev-db:
	COMPOSE_ENV_FILES=/dev/null docker compose up -d postgres

dev-server: dev-db
	@test -f .env || (echo "Missing .env. Copy from .env.example: cp .env.example .env" && exit 1)
	@test -f server/internal/static/dist/index.html || (echo "Missing embedded UI placeholder. Run: git checkout -- server/internal/static/dist/index.html (or make ui for a full build)" && exit 1)
	@set -a && . ./.env && set +a && cd server && go run ./cmd/openlicensd

dev-ui:
	cd ui && npm run dev

ui:
	cd ui && npm run generate

server:
	cd server && go build -o ../bin/openlicensd ./cmd/openlicensd

build: ui server

test:
	@set -a && [ -f .env ] && . ./.env; set +a && cd server && go test ./...

lint:
	cd server && go vet ./...
	cd ui && npm run lint

hash-password:
	@test -n "$(PASSWORD)" || (echo "Usage: make hash-password PASSWORD=yourpassword" && exit 1)
	cd server && go run ./cmd/hashpassword $(PASSWORD)

release:
	goreleaser release --clean
