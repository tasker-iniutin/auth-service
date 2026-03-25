COMPOSE ?= docker compose
GOOSE ?= goose
MIGRATIONS_DIR ?= ./migrations
GO ?= go
GOCACHE ?= /tmp/go-build

POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= auth
POSTGRES_PORT ?= 5433
REDIS_PORT ?= 6379

DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: db-up db-down db-reset db-logs db-psql redis-cli migrate-up migrate-down migrate-status migrate-create test

db-up:
	$(COMPOSE) up -d

db-down:
	$(COMPOSE) down

db-reset:
	$(COMPOSE) down -v

db-logs:
	$(COMPOSE) logs -f postgres redis

db-psql:
	$(COMPOSE) exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

redis-cli:
	$(COMPOSE) exec redis redis-cli

migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

migrate-create:
ifndef NAME
	$(error NAME is required, use: make migrate-create NAME=create_users)
endif
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(NAME) sql

test:
	GOCACHE=$(GOCACHE) $(GO) test ./...
