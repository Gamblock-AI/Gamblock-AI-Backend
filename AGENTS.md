# Gamblock-AI Backend Agent Rules

Go/Gin + ent + PostgreSQL API. See the root `AGENTS.md` for the full
architecture and PRD alignment.

## Layering (do not skip layers)

`cmd/*` → `internal/api` → `internal/routes` → `internal/handler` →
`internal/service` → `internal/repository` → `ent`.

- Handlers do HTTP only: parse, call a service, return the envelope. No business
  logic or ent calls in handlers.
- Services own business logic and transactions. Repositories own ent queries.
- After changing `ent/schema/schema.go`, regenerate with `go generate ./ent`
  (or the project's codegen command). Do not hand-edit generated `ent/` files.

## Response envelope

Every JSON response uses `{ "data", "error", "request_id" }`. Success sets
`data` and `error: null`; errors set `error: { code, message }` and
`data: null`. Use `Handler.respond` / `Handler.respondError`
(`internal/handler/handler.go`). Do not return raw shapes.

## Auth & RBAC (PRD §2)

- `middleware.AuthRequired()` validates the Bearer access token and sets
  `user_id` / `email` / `role`.
- `middleware.RequireRoles(...)` gates an action by role. Roles:
  `user`, `partner` (Kepala), `platform_admin`, `support_operator`,
  `model_release_operator`, `content_admin`.
- Quick-approval (`/v1/approval-requests/verify/:token`,
  `/resolve-by-token`) is intentionally unauthenticated — it is validated by a
  single-use token instead (PRD §5.2).

## Privacy by design (PRD §6.1) — the PrivacyGuard

`middleware.PrivacyGuard()` rejects any non-auth request body/query whose keys
or values look like browsing data or secrets (`url`, `domain`, `dom`,
`raw_url`, `history`, `screenshot`, `password`, `otp`, `token`, …) or any
string value containing `http://`/`https://` or longer than 4000 chars. When
adding an endpoint or field, ensure you never send raw URLs/DOM to the server —
only aggregate events (timestamp + platform type) are allowed (PRD §4).

## Testing

- `go test ./...` (or `make test`). Tests live beside code as `*_test.go`.
- testify + enttest available. In-memory store path (`store.NewSeeded()` +
  `repository.New(nil, store)`) powers integration tests without a DB.
- `internal/middleware/middleware_test.go` guards the PrivacyGuard invariants
  (exempt quick-approval + auth; forbidden KEYS rejected; values not censored).
  Do not reintroduce value-based URL/length censorship — it breaks jurnal text
  and quick-approval tokens (PRD §5.2).
- `internal/handler/handler_test.go` asserts the env gate (production = friendly
  message, dev = `[code] detail`) and envelope shape.

## Error messages & env gate (PRD §6.1 — privacy by design)

- `internal/i18n/messages.go` is the **single source of truth** for end-user
  error text, keyed by stable error `code`. Handlers MUST NOT leak `err.Error()`
  to clients.
- Use `Handler.respondErrorErr(c, status, code, err)` for errors with an
  underlying Go error: production returns the friendly catalog message; dev
  returns `[code] err.Error()`; the technical error is always logged (zap) with
  the request id — never lost, never leaked.
- Use `Handler.respondCode(c, status, code)` for validation/hint errors with no
  underlying error (resolved from the catalog, env-gated).
- Adding a new error: add the `code` to the catalog AND mirror it in the FE
  catalogs (`gamblock-ai-website/lib/messages.ts`,
  `gamblock_ai_apps/lib/core/messaging/app_messages.dart`). Keep codes in sync.
- `config.IsProduction()` gates messages (default safe = production when
  `APP_ENV` is unset).

## Encryption (PRD §4 / §7.1)

Reflection/journal text is encrypted with AES-256-GCM before storage, using
`internal/crypto/aes.go` (`Encrypt`/`Decrypt`) keyed by a hex env secret. Never
store journal plaintext. The nonce is prepended to the ciphertext and stored as
hex.

## WhatsApp batching (PRD §5.1)

Uninstall approval requests are batched (e.g. every 4–12h) into a single
WhatsApp message per Kepala, not sent one-by-one in real time. Keep batching in
`internal/service/whatsapp_service.go`.

## Domain types (single source of truth)

- `internal/model/*` is the ONLY domain type set. `internal/store` holds in-memory
  backing data using `model.*` via type aliases (`store.User = model.User`, etc.).
  Do NOT re-introduce parallel struct definitions in `store` — that caused a
  broken build. Adding a field means adding it to the `model` type only.
- Presentation helpers (`humanExpiry`, `humanApprovalStatus`, `humanApprovalAction`,
  `humanDataRequestTitle`, `moduleProgress`, `humanPublished`) live in
  `internal/repository/repository.go` (and a duplicate set in `internal/db/db.go`
  for the loader). Keep them package-local; do not cross-call between `db` and
  `repository`.

## Code hygiene

- Do not commit generated binaries (`/bin`, `api`, `migrate`, `seed`) — they are
  in `.gitignore`. Build with `make build`.
- Keep dependencies tidy: run `go mod tidy` after adding deps.
