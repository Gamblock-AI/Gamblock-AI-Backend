# Gamblock-AI Backend Agent Rules

Context version: `2026-07-15.2`

This repository is the Go/Gin API for Gamblock-AI. It must remain safe and
understandable as a standalone clone; no parent workspace files are required.
Read `docs/ai/README.md` for the product capsule, capability status, and related
repository contracts before changing behavior.

## Start and finish

1. Inspect `git status` and preserve unrelated user changes.
2. Read the implementation, adjacent tests, and relevant README/context before
   editing.
3. Keep one API behavior or contract per change unit.
4. Run `make lint` before handoff. Do not run tests, builds, race tests, or
   `make verify` unless the user explicitly requests them in the current
   conversation.
5. Update `README.md`, this file, and `docs/ai/` when commands, paths,
   architecture, privacy boundaries, or capability status change.

## Layering (do not skip layers)

`cmd/*` -> `internal/api` -> `internal/routes` -> `internal/handler` ->
`internal/service` -> `internal/repository` -> `ent`

- Handlers parse HTTP input, call services, and return the response envelope.
  They contain no business logic or ent queries.
- Services own business rules and transactions. Repositories own persistence.
- Domain types live only in `internal/model/`. The in-memory store uses aliases
  to those types; do not introduce parallel store structs.
- Register every endpoint in `internal/routes/routes.go` and add handler/service
  tests for new behavior.

## ent generation

After changing `ent/schema/schema.go`, run `make generate`. Never hand-edit
generated ent files. Review generated diffs and run `make lint`; tests/builds
remain explicit opt-in checks.

## Response envelope and errors

Every response uses `{ "data", "error", "request_id" }`. Use the helpers in
`internal/handler/handler.go`; do not return raw ad-hoc shapes or leak
`err.Error()` in production.

- `respondErrorErr` logs technical detail and returns catalog-safe output.
- `respondCode` resolves validation errors without an underlying Go error.
- `internal/i18n/messages.go` owns stable backend error codes.
- Every new stable code must also be added to the website and Flutter catalogs.
  If sibling repositories are not checked out, name both follow-up paths in the
  handoff rather than silently leaving the contract incomplete.

## Privacy boundary

All classification runs on-device. The API must never receive or store raw DOM,
URLs, domains, screenshots, keystrokes, or browsing history. Only aggregate
protection events are permitted.

`PrivacyGuard` currently enforces this by rejecting forbidden JSON/query field
names on non-GET, non-OPTIONS, non-auth requests. It intentionally does not
censor string values: journal text may legitimately mention a URL. Quick
approval token routes are explicitly exempt. Keep these regression behaviors
covered in `internal/middleware/middleware_test.go`; do not reintroduce the old
URL/length value censorship.

## Auth and role model

`AuthRequired` validates access tokens and `RequireRoles(...)` gates actions.
The product-facing roles include `user`, `partner`, `platform_admin`,
`support_operator`, `model_release_operator`, and `content_admin`. The current
ent user enum also contains `organization_owner` and `organization_admin`.
Treat that difference as existing implementation state, not permission to
expand access; route authorization must remain explicit.

Quick approval verify/resolve routes are intentionally unauthenticated and use
single-use tokens. Do not put them behind session auth or expose their tokens.

## Proposal-derived backend role

The PKM core requires on-device Hybrid Analysis, local blocking, Pattern
Interrupt, web self-regulation, and partner-controlled removal. This backend
supports `PKM-ACC-001`, `PKM-ACC-002`, `PKM-WEB-001`, `PKM-WEB-002`,
`PKM-WEB-003`, `PKM-WEB-004`, `PKM-WEB-005`, `PKM-WEB-006`, `PKM-WEB-007`,
and privacy-safe aggregate/recovery state. It must never become the classifier
or blocking authority. Group Codes, WhatsApp, admin/operator portals, journals,
and release management are supporting/operational, not substitutes for core.

## Sensitive data and storage

- Target invariant: encrypt journal/reflection text with AES-256-GCM before
  persistence via `internal/crypto/aes.go`; never log/store plaintext. Current
  `ReflectionService` falls back to plaintext when the key is absent or
  encryption fails. Treat this as a documented P0 gap: do not broaden it or
  claim encryption is fail-closed, and fix it only when implementation is in
  scope.
- `.env` and credentials are local only. Update `.env.example` for config
  shape changes.
- `cmd/api` can fall back to seeded in-memory data when PostgreSQL is missing or
  fails. This is current prototype behavior, not a production durability
  guarantee; do not describe it as persistent storage.
- WhatsApp approval notifications are batched, not sent per event in real time.

## Validation policy

```sh
make generate          # only after ent schema changes
make lint              # default AI check: go vet ./...
./scripts/verify-ai-context.sh  # additionally when AI context changed

# Explicit user request only:
make test
make verify            # build all packages, vet, and race-test
```

Tests live beside code as `*_test.go`, but the AI does not run them by default.
The seeded in-memory store supports integration tests without PostgreSQL. Do
not hit production services from tests.

## Protected and external actions

- Never hand-edit generated `ent/` output.
- Never commit binaries, coverage, `.env`, keys, or runtime databases.
- Do not migrate/drop a real database, deploy, push, release, or change secrets
  without explicit user authorization. `migrate-fresh` is destructive.
