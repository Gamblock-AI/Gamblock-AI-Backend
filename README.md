# Gamblock-AI Backend

Go/Gin API service for the Gamblock-AI Flutter client and the Next.js web
dashboard. Uses [ent](https://entgo.io/) over PostgreSQL.

## Run locally

```sh
cp .env.example .env
go run ./cmd/migrate      # apply schema migrations
go run ./cmd/seed         # (optional) seed demo data
go run ./cmd/api          # start the API (default 127.0.0.1:8080)
```

The service uses ent/PostgreSQL by default and currently falls back to seeded
in-memory data when the database cannot be reached. That fallback is useful for
prototype development and tests, but it is not durable production storage. The
default local URL is
`postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`.

Useful Makefile targets: `make dev` (air live-reload), `make lint`,
`make migrate`, `make seed`, and opt-in `make verify`. `make migrate-fresh` drops the
database schema and must never be run against shared or production data.

## Key local endpoints

- `GET  /healthz`
- `GET  /readyz`
- `POST /v1/auth/dev-login`
- `GET  /v1/client/dashboard-summary`
- `GET  /v1/client/protection-status`
- `GET  /v1/psychoeducation/modules`
- `GET  /v1/partners`
- `GET  /v1/approval-requests`
- `GET  /v1/portal/overview`

All responses use the envelope `{ "data", "error", "request_id" }` produced in
`internal/handler/handler.go` / `internal/middleware/middleware.go`.

## Layering

`cmd/*` (entrypoints) → `internal/api` (server) → `internal/routes` →
`internal/handler` (HTTP) → `internal/service` (business logic) →
`internal/repository` (data access) → `ent` (generated ORM).

See `AGENTS.md` for conventions and the privacy/AES/RBAC invariants.

## AI and contributor context

- `AGENTS.md` is the canonical repository instruction file.
- `docs/ai/README.md` records the product boundary and capability status that
  must survive a standalone clone.
- `make generate` regenerates ent output after schema changes.
- `make lint` (`go vet ./...`) is the default AI check.
- `make test` and `make verify` (build/vet/race-test) run only on explicit user
  request. CI may retain its automatic gates.
- `./scripts/verify-ai-context.sh` checks versioned agent context and adapters.

Target policy requires AES-256-GCM before reflection/journal persistence. The
current service still falls back to plaintext when its encryption key is
missing or encryption fails; this is a documented prototype gap, not a secure
production mode.
