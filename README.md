# Gamblock-AI Backend

Go/Gin API service for the Flutter multi-surface app.

## Run locally

```sh
cp .env.example .env
HTTP_ADDR=127.0.0.1:8080 go run ./cmd/api
```

The service now uses ent/PostgreSQL by default and falls back to privacy-safe in-memory seed data only when the database cannot be reached. The default local URL is `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`.

Useful local commands:

```sh
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/api
```

## Key local endpoints

- `GET /healthz`
- `GET /readyz`
- `POST /v1/auth/dev-login`
- `GET /v1/client/dashboard-summary`
- `GET /v1/client/protection-status`
- `GET /v1/psychoeducation/modules`
- `GET /v1/partners`
- `GET /v1/approval-requests`
- `GET /v1/portal/overview`

All responses use the envelope documented in `docs/10-mvp/API_AND_DATA_MODEL_MVP.md`.
