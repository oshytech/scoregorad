.PHONY: run build migrate test lint

# Variables
BINARY=bin/api
DB_URL?=$(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)

run:
	go run ./cmd/api

build:
	go build -o $(BINARY) ./cmd/api

migrate:
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql "$(DB_URL)" -f $$f; \
	done

test:
	go test ./...

lint:
	go vet ./...

.DEFAULT_GOAL := run
