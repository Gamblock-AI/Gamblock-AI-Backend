ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: dev run build start generate migrate migrate-fresh seed lint test test-cover verify

APP_NAME := api
BUILD_DIR := ./bin

dev:
	air

run:
	go run ./cmd/api

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

generate:
	go run entgo.io/ent/cmd/ent generate ./ent/schema

start: build
	$(BUILD_DIR)/$(APP_NAME)

migrate:
	go run ./cmd/migrate

migrate-fresh:
	@echo "Dropping and recreating database..."
	psql "$(DATABASE_URL)" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" || true
	go run ./cmd/migrate
	@echo "Fresh migration complete."

seed:
	go run ./cmd/seed

lint:
	go vet ./...

test:
	go test ./...

test-cover:
	go test -cover ./...

verify:
	go build ./...
	go vet ./...
	go test -race ./...
