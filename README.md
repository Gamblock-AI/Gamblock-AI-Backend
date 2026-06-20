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

The service uses ent/PostgreSQL by default and falls back to privacy-safe
in-memory seed data only when the database cannot be reached. The default local
URL is `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`.

Useful Makefile targets: `make dev` (air live-reload), `make build`,
`make migrate`, `make migrate-fresh`, `make seed`.

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
