# Database and code-generation tasks. Run from backend/.
# Every target loads ../.env (git-ignored) so DATABASE_URL is never typed
# on the command line or stored in the repo.

SHELL := /bin/bash
ENV_FILE ?= ../.env
MIGRATIONS := db/migrations

# Load .env into the recipe environment, then run the command.
define with_env
	@set -euo pipefail; \
	if [ -f "$(ENV_FILE)" ]; then set -a; source "$(ENV_FILE)"; set +a; fi; \
	if [ -z "$${DATABASE_URL:-}" ]; then echo "DATABASE_URL is not set (expected in $(ENV_FILE))"; exit 1; fi; \
	$(1)
endef

.PHONY: help migrate-up migrate-down migrate-down-all migrate-version migrate-force \
        db-ping schema-snapshot sqlc-generate sqlc-vet sqlc check-schema

help:
	@echo "migrate-up        apply all pending migrations"
	@echo "migrate-down      roll back the most recent migration"
	@echo "migrate-down-all  roll back every migration (destroys all tables)"
	@echo "migrate-version   print current migration version"
	@echo "migrate-force V=n mark version n as applied and clean (after a failed migration)"
	@echo "db-ping           verify the connection string works"
	@echo "schema-snapshot   rebuild db/schema/schema.sql from migrations"
	@echo "sqlc-vet          prepare every query against DATABASE_URL"
	@echo "sqlc-generate     regenerate internal/postgres/sqlc"
	@echo "sqlc              snapshot + vet + generate"
	@echo "check-schema      fail if schema.sql is out of date with migrations"

migrate-up:
	$(call with_env,migrate -path $(MIGRATIONS) -database "$$DATABASE_URL" up)

migrate-down:
	$(call with_env,migrate -path $(MIGRATIONS) -database "$$DATABASE_URL" down 1)

migrate-down-all:
	$(call with_env,migrate -path $(MIGRATIONS) -database "$$DATABASE_URL" down -all)

migrate-version:
	$(call with_env,migrate -path $(MIGRATIONS) -database "$$DATABASE_URL" version)

migrate-force:
	@test -n "$(V)" || (echo "usage: make migrate-force V=<version>"; exit 1)
	$(call with_env,migrate -path $(MIGRATIONS) -database "$$DATABASE_URL" force $(V))

db-ping:
	$(call with_env,psql "$$DATABASE_URL" -Atc "select version()")

schema-snapshot:
	@sh scripts/schema-snapshot.sh

check-schema:
	@tmp=$$(mktemp); cp db/schema/schema.sql "$$tmp"; sh scripts/schema-snapshot.sh >/dev/null; \
	if ! diff -q "$$tmp" db/schema/schema.sql >/dev/null; then \
	  echo "db/schema/schema.sql was stale; regenerated. Commit the new snapshot."; rm -f "$$tmp"; exit 1; \
	fi; rm -f "$$tmp"; echo "schema snapshot is up to date"

sqlc-vet:
	$(call with_env,sqlc vet)

sqlc-generate:
	@sqlc generate && echo "generated internal/postgres/sqlc"

sqlc: schema-snapshot sqlc-vet sqlc-generate

.PHONY: seed run build test cover
seed:
	$(call with_env,go run ./cmd/seed)

run:
	$(call with_env,go run ./cmd/api)

build:
	@go build ./... && echo "build ok"

# Loads .env so TEST_DATABASE_URL is picked up; integration tests skip when it is unset.
test:
	@set -a; [ -f $(ENV_FILE) ] && source $(ENV_FILE); set +a; go test -race ./test/...

cover:
	@set -a; [ -f $(ENV_FILE) ] && source $(ENV_FILE); set +a; \
	go test -race -coverpkg=./api/... -coverprofile=coverage.out ./test/... && go tool cover -func=coverage.out | tail -1
