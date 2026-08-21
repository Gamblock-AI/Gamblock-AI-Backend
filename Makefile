ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: dev run build start generate key-generate migrate migrate-up migrate-down migrate-fresh reset-storage seed seeder demo-seeder seed-accounts seed-education seed-learning-hub seed-scale seed-local-accounts lint test test-cover verify

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
	go build -o $(BUILD_DIR)/reset-storage ./cmd/reset-storage
	go build -o $(BUILD_DIR)/seeder ./cmd/seeder
	go build -o $(BUILD_DIR)/demo-seeder ./cmd/demo-seeder
	go build -o $(BUILD_DIR)/seed-accounts ./cmd/seed-accounts
	go build -o $(BUILD_DIR)/seed-learning-hub ./cmd/seed-learning-hub
	go build -o $(BUILD_DIR)/seed-scale ./cmd/seed-scale
	go build -o $(BUILD_DIR)/seed-local-accounts ./cmd/seed-local-accounts

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

# Runs the binary produced by `make build` with the local `.env` loaded into
# the environment (config reads env vars via viper). The shell `source` is
# used on top of the top-level include so values with spaces/comments survive.
start: build
	@test -f .env || { echo "Error: .env not found. Copy .env.example to .env (make key-generate creates it)." >&2; exit 1; }
	@echo "Starting $(BUILD_DIR)/$(APP_NAME) with .env..."
	@set -a; . ./.env; set +a; exec $(BUILD_DIR)/$(APP_NAME)

migrate:
	go run ./cmd/migrate

migrate-up: migrate

migrate-down:
	@test "$(CONFIRM_MIGRATE_DOWN)" = "DROP_ALL_DATA" || { \
		echo "Refusing destructive migration. Re-run with CONFIRM_MIGRATE_DOWN=DROP_ALL_DATA." >&2; \
		exit 1; \
	}
	CONFIRM_MIGRATE_DOWN="$(CONFIRM_MIGRATE_DOWN)" go run ./cmd/migrate-down
	CONFIRM_RESET_STORAGE=DELETE_DYNAMIC_STORAGE go run ./cmd/reset-storage

migrate-fresh:
	@test "$(CONFIRM_MIGRATE_DOWN)" = "DROP_ALL_DATA" || { \
		echo "Refusing destructive migration. Re-run with CONFIRM_MIGRATE_DOWN=DROP_ALL_DATA." >&2; \
		exit 1; \
	}
	@echo "Dropping and recreating database..."
	CONFIRM_MIGRATE_DOWN="$(CONFIRM_MIGRATE_DOWN)" go run ./cmd/migrate-down
	CONFIRM_RESET_STORAGE=DELETE_DYNAMIC_STORAGE go run ./cmd/reset-storage
	go run ./cmd/migrate
	@echo "Fresh migration complete."

# Empties the runtime storage directories (media/avatars/exports).
# Run after a destructive database reset; seed assets are recreated by `make seed`.
reset-storage:
	@test "$(CONFIRM_RESET_STORAGE)" = "DELETE_DYNAMIC_STORAGE" || { \
		echo "Refusing storage reset. Re-run with CONFIRM_RESET_STORAGE=DELETE_DYNAMIC_STORAGE." >&2; \
		exit 1; \
	}
	CONFIRM_RESET_STORAGE="$(CONFIRM_RESET_STORAGE)" go run ./cmd/reset-storage

seed:
	go run ./cmd/seed

seeder:
	go run ./cmd/seeder

demo-seeder:
	@test "$(CONFIRM_DEMO_SEED)" = "CREATE_FOUR_DEMO_ACCOUNTS" || { \
		echo "Refusing demo seed. Re-run with CONFIRM_DEMO_SEED=CREATE_FOUR_DEMO_ACCOUNTS." >&2; \
		exit 1; \
	}
	CONFIRM_DEMO_SEED="$(CONFIRM_DEMO_SEED)" go run ./cmd/demo-seeder

seed-accounts:
	@test "$(CONFIRM_SEED_ACCOUNTS)" = "CREATE_FOUR_DEMO_ACCOUNTS" || { \
		echo "Refusing account seed. Re-run with CONFIRM_SEED_ACCOUNTS=CREATE_FOUR_DEMO_ACCOUNTS." >&2; \
		exit 1; \
	}
	CONFIRM_SEED_ACCOUNTS="$(CONFIRM_SEED_ACCOUNTS)" go run ./cmd/seed-accounts

seed-education:
	go run ./cmd/seed-education

seed-learning-hub:
	go run ./cmd/seed-learning-hub

seed-scale:
	go run ./cmd/seed-scale

seed-local-accounts:
	go run ./cmd/seed-local-accounts

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
