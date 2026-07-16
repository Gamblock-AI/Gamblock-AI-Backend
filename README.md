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

The service uses ent/PostgreSQL by default. Development may fall back to an
empty in-memory store when the database cannot be reached; contextual demo
records appear only when `ENABLE_DEMO_DATA=true` outside production.
Production validates its JWT/journal configuration and fails closed when
PostgreSQL is unavailable. The default local URL is
`postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`.

PostgreSQL seed data and the optional in-memory demo store use the shared dummy
password `password` for `gading@gmail.com`, `dery@gmail.com`,
`suci@gmail.com`, and `nasywa@gmail.com`. These credentials are development
fixtures only and demo data is forbidden in production.

Useful Makefile targets: `make dev` (air live-reload), `make lint`,
`make migrate`, `make seed`, and opt-in `make verify`. `make migrate-fresh` drops the
database schema and must never be run against shared or production data.

## Key local endpoints

- `GET  /healthz`
- `GET  /readyz`
- `POST /v1/auth/dev-login`
- `POST /v1/devices`
- `PATCH /v1/me/password`
- `GET  /v1/client/dashboard-summary`
- `GET  /v1/client/protection-status`
- `GET  /v1/client/protection-analytics`
- `POST /v1/client/aggregate-events`
- `GET/PATCH /v1/me`
- `GET  /v1/psychoeducation/modules`
- `GET  /v1/partners`
- `GET  /v1/approval-requests`
- `POST /v1/approval-requests/:id/apply`
- `GET/POST /v1/emergency-key-requests`
- `POST /v1/devices/unlock`
- `GET /v1/admin/emergency-key-requests`
- `POST /v1/admin/emergency-key-requests/:id/review`
- `POST /v1/admin/emergency-key-requests/:id/approve`
- `GET  /v1/portal/overview`

All responses use the envelope `{ "data", "error", "request_id" }` produced in
`internal/handler/handler.go` / `internal/middleware/middleware.go`.

### Client protection contract

- `POST /v1/devices` requires `client_instance_id` and upserts the authenticated
  user's installation; a new device starts `inactive`.
- `PATCH /v1/me/password` requires `current_password` and `new_password`, then
  revokes every refresh token so clients must reauthenticate.
- `GET /v1/client/protection-analytics?device_id=<id>&days=7|30` returns daily
  and total counters only.
- Approval responses keep stable `action`/`status` codes separate from
  localized labels. `POST /v1/approval-requests/:id/apply` is device-bound,
  idempotent after first use, and available for 30 minutes after resolution.
- Emergency recovery is device-bound: the user creates a request, one platform
  admin reviews it, a different platform admin approves/issues it, and
  `/v1/devices/unlock` consumes the 24-hour single-use key for a ten-minute
  grant.

These endpoints reject ownership mismatches and never accept URL, domain, DOM,
page title, browsing history, screenshot, feature-vector, or per-page score
fields.

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

Reflection/journal writes fail closed unless a valid AES-256-GCM key is
configured; decryption failures never expose ciphertext as user content.
Password login uses Argon2id. Google sign-in is enabled only when
`GOOGLE_CLIENT_ID` matches the website's public OAuth client ID. Development
login and contextual demo records are separately opt-in and forbidden in
production.
