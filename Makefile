CONFIG_PATH ?= assets/local.yaml
SEED_DIR ?= assets/seeds
MIGRATE := go run ./cmd/migrate -config $(CONFIG_PATH)

.PHONY: test test-integration buf-lint buf-generate migrate-up migrate-down migrate-version migrate-drop migrate-seeds-up migrate-seeds-down docker-up docker-down dev-up dev-down fmt tidy ci

## Run unit tests
test:
	go test ./...

## Run integration tests (requires PostgreSQL running)
test-integration:
	CONFIG_PATH=$(CONFIG_PATH) go test -tags=integration ./test/...

## Run buf lint
buf-lint:
	cd proto && buf lint

## Generate gRPC stubs
buf-generate:
	cd proto && buf generate

## Apply database migrations
migrate-up:
	$(MIGRATE) up

## Roll back database migrations
migrate-down:
	$(MIGRATE) down

## Show migration version
migrate-version:
	$(MIGRATE) version

## Drop all database objects managed by migrations
migrate-drop:
	$(MIGRATE) drop

## Apply seed data
migrate-seeds-up:
	go run ./cmd/migrate -config $(CONFIG_PATH) -dir $(SEED_DIR) up

## Roll back seed data
migrate-seeds-down:
	go run ./cmd/migrate -config $(CONFIG_PATH) -dir $(SEED_DIR) down

## Start local Docker services
docker-up:
	docker compose up -d postgres

## Stop local Docker services
docker-down:
	docker compose down

## Start development server with Air (foreground)
dev-up:
	docker compose --profile local up server

## Stop development server and related containers
dev-down:
	docker compose --profile local down

## Format Go files
fmt:
	gofmt -w $(shell go list -f '{{.Dir}}' ./...)

## Sync Go modules
tidy:
	go mod tidy

## Run CI-equivalent checks
ci:
	$(MAKE) fmt
	$(MAKE) test
	$(MAKE) buf-lint
	CONFIG_PATH=$(CONFIG_PATH) $(MAKE) migrate-up
	CONFIG_PATH=$(CONFIG_PATH) $(MAKE) test-integration
