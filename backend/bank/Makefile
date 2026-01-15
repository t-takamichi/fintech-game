# Simple Makefile for local DB and migrations

DATABASE_URL ?= postgres://bank_user:bank_pass@localhost:5432/bank_db?sslmode=disable
# When running migrate inside Docker, use the internal service hostname `db`.
DATABASE_URL_DOCKER ?= postgres://bank_user:bank_pass@db:5432/bank_db?sslmode=disable

.PHONY: help up db-up migrate-up migrate-down migrate-status psql new-migration down logs build run seed

help:
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  up               Start Postgres container"
	@echo "  db-up            Alias for up"
	@echo "  migrate-up       Apply migrations (up)"
	@echo "  migrate-down     Rollback last migration (down 1)"
	@echo "  migrate-status   Show migration status"
	@echo "  psql             Open psql shell on the DB container"
	@echo "  new-migration    Create new migration file: NAME=your_name"
	@echo "  down             Stop containers"

up db-up:
	docker compose up -d db


migrate-up:
	@echo "Starting db service (if not running)..."
	docker compose up -d db
	@echo "Waiting for db to be ready..."
	@# wait for pg to accept connections inside the db container
	@count=0; \
	while ! docker compose exec db pg_isready -U bank_user -d bank_db >/dev/null 2>&1; do \
	  count=$$((count+1)); \
	  if [ $$count -gt 60 ]; then echo "Timed out waiting for db"; exit 1; fi; \
	  echo "  waiting for postgres... ($$count)"; sleep 1; \
	done;
	@echo "Applying migrations inside docker (connecting to db service)..."
	docker compose run --rm -e DATABASE_URL="$(DATABASE_URL_DOCKER)" migrate -path=/migrations -database "$(DATABASE_URL_DOCKER)" up


migrate-down:
	docker compose run --rm -e DATABASE_URL="$(DATABASE_URL_DOCKER)" migrate -path=/migrations -database "$(DATABASE_URL_DOCKER)" down 1


migrate-status:
	docker compose run --rm -e DATABASE_URL="$(DATABASE_URL_DOCKER)" migrate -path=/migrations -database "$(DATABASE_URL_DOCKER)" version

psql:
	docker compose exec db psql -U bank_user -d bank_db

new-migration:
	@if [ -z "$(NAME)" ]; then echo "NAME is required. Usage: make new-migration NAME=description"; exit 1; fi
	@ts=$$(date +%Y%m%d%H%M%S)
	@file="migrations/$$ts-$(NAME).sql"
	@mkdir -p migrations
	@echo "-- migration: $(NAME)" > $$file
	@echo "-- write SQL here" >> $$file
	@echo Created $$file

down:
	docker compose down

logs:
	docker compose logs -f db

build:
	@echo "Building server..."
	@mkdir -p bin
	go build -o bin/server ./cmd/server

run:
	@echo "Building (if needed) and running server in foreground..."
	@mkdir -p bin
	$(MAKE) build
	./bin/server

seed:
	@echo "Starting db service (if not running)..."
	docker compose up -d db
	@echo "Applying seed SQL into db container..."
	# copy seed into container path and execute
	docker compose exec -T db psql -U bank_user -d bank_db -f /migrations/seed.sql || \
	  docker compose exec -T db psql -U bank_user -d bank_db -f - < migrations/seed.sql
