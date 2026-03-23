COMPOSE ?= docker compose
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= auth
POSTGRES_PORT ?= 5433
REDIS_PORT ?= 6379

MIGRATION ?= ./migrations/001_create_users.sql
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: db-up db-down db-reset db-logs db-psql redis-cli migrate-up migrate-down

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
	awk '/^-- \\+goose Down/{exit} {print}' $(MIGRATION) | \
	docker exec -i auth-service-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

migrate-down:
	awk 'found{print} /^-- \\+goose Down/{found=1}' $(MIGRATION) | \
	docker exec -i auth-service-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)
