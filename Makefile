.PHONY: dev dev-server dev-ui ui server test lint hash-password build release

DATABASE_URL ?= postgres://openlicensd:openlicensd@localhost:5432/openlicensd?sslmode=disable
ADMIN_USER ?= admin
ADMIN_PASSWORD ?= admin
JWT_SECRET ?= dev-secret-change-me

export OPENLICENSD_DATABASE_URL ?= $(DATABASE_URL)
export OPENLICENSD_ADMIN_USER ?= $(ADMIN_USER)
export OPENLICENSD_ADMIN_PASSWORD_HASH ?= $$2a$$10$$r64NfJHy2Diu3lXHw7pSH.IHz97Ydz4Shf9M5LH7TvnmR3tfbT5.2
export OPENLICENSD_JWT_SECRET ?= $(JWT_SECRET)
export OPENLICENSD_ADDR ?= :8080

dev: dev-db
	@echo "Run 'make dev-server' and 'make dev-ui' in separate terminals"

dev-db:
	docker compose up -d postgres

dev-server: dev-db
	cd server && go run ./cmd/openlicensd

dev-ui:
	cd ui && npm run dev

ui:
	cd ui && npm run generate

server:
	cd server && go build -o ../bin/openlicensd ./cmd/openlicensd

build: ui server

test:
	cd server && go test ./...

lint:
	cd server && go vet ./...
	cd ui && npm run lint

hash-password:
	@test -n "$(PASSWORD)" || (echo "Usage: make hash-password PASSWORD=yourpassword" && exit 1)
	cd server && go run ./cmd/hashpassword $(PASSWORD)

release:
	goreleaser release --clean
