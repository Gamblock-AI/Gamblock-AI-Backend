ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: dev run build start generate key-generate migrate migrate-up migrate-down migrate-fresh seed seeder seed-education lint test test-cover verify

APP_NAME := api
BUILD_DIR := ./bin

dev:
	air

run:
	go run ./cmd/api

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api
	go build -o $(BUILD_DIR)/migrate-up ./cmd/migrate
	go build -o $(BUILD_DIR)/migrate-down ./cmd/migrate-down
	go build -o $(BUILD_DIR)/seeder ./cmd/seeder

generate:
	go run entgo.io/ent/cmd/ent generate ./ent/schema

# Creates a valid AES-256 journal key in .env. A valid existing key is never
# replaced without FORCE=1 because doing so makes already-encrypted local data
# unreadable.
key-generate:
	@set -eu; \
	if ! command -v openssl >/dev/null 2>&1; then \
		echo "openssl is required to generate JOURNAL_ENCRYPTION_KEY" >&2; \
		exit 1; \
	fi; \
	env_file=.env; \
	if [ ! -f "$$env_file" ]; then cp .env.example "$$env_file"; fi; \
	valid_key=$$(awk -F= '/^[[:space:]]*(export[[:space:]]+)?JOURNAL_ENCRYPTION_KEY[[:space:]]*=/ { value=$$0; sub(/^[^=]*=/, "", value); gsub(/^[[:space:]]+|[[:space:]]+$$/, "", value); if (length(value) == 64 && value ~ /^[0-9A-Fa-f]+$$/) valid=1 } END { print valid ? "yes" : "no" }' "$$env_file"); \
	if [ "$$valid_key" = "yes" ] && [ "$(FORCE)" != "1" ]; then \
		echo "A valid JOURNAL_ENCRYPTION_KEY already exists in .env; use 'make key-generate FORCE=1' only if no encrypted local data must be retained." >&2; \
		exit 1; \
	fi; \
	key=$$(openssl rand -hex 32); \
	temp_file=$$(mktemp "$$env_file.tmp.XXXXXX"); \
	awk -v key="$$key" '/^[[:space:]]*(export[[:space:]]+)?JOURNAL_ENCRYPTION_KEY[[:space:]]*=/ { if (!updated) { print "JOURNAL_ENCRYPTION_KEY=" key; updated=1 }; next } { print } END { if (!updated) print "JOURNAL_ENCRYPTION_KEY=" key }' "$$env_file" > "$$temp_file"; \
	mv "$$temp_file" "$$env_file"; \
	chmod 600 "$$env_file" 2>/dev/null || true; \
	echo "Generated a valid JOURNAL_ENCRYPTION_KEY in .env. Keep .env private and restart the API."

start: build
	$(BUILD_DIR)/$(APP_NAME)

migrate:
	go run ./cmd/migrate

migrate-up: migrate

migrate-down:
	@test "$(CONFIRM_MIGRATE_DOWN)" = "DROP_ALL_DATA" || { \
		echo "Refusing destructive migration. Re-run with CONFIRM_MIGRATE_DOWN=DROP_ALL_DATA." >&2; \
		exit 1; \
	}
	go run ./cmd/migrate-down

migrate-fresh:
	@echo "Dropping and recreating database..."
	psql "$(DATABASE_URL)" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" || true
	go run ./cmd/migrate
	@echo "Fresh migration complete."

seed:
	go run ./cmd/seed

seeder:
	go run ./cmd/seeder

seed-education:
	go run ./cmd/seed-education

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
