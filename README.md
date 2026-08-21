# Gamblock-AI Backend

Go/Gin API service for the Gamblock-AI Flutter client and the Next.js web
dashboard. Uses [ent](https://entgo.io/) over PostgreSQL.

## Run locally

```sh
cp .env.example .env
make key-generate  # creates and saves a valid JOURNAL_ENCRYPTION_KEY in .env
make migrate-up     # apply schema migrations with values loaded from .env
make seeder         # install missing production-safe public baseline content
make seed           # (optional) seed demo data plus the Learning Hub catalog
make seed-education # upsert the six bilingual education modules/media
make seed-learning-hub # install or verify the UTY Learning Hub catalog

> `make seed` fills demo users and demo content AND installs the Learning Hub
> catalog (skills page "Pilih arah belajar"). The production-safe `make seeder`
> path installs the same Learning Hub baseline without creating demo users;
> `make seed-learning-hub` installs or verifies only the Learning Hub catalog.
make run            # start the API with `go run` (default 127.0.0.1:8080)
make start          # build then run ./bin/api, loading .env into the process
```

The service uses ent/PostgreSQL by default. Development may fall back to an
empty in-memory store when the database cannot be reached; contextual demo
records appear only when `ENABLE_DEMO_DATA=true` outside production.
Every environment validates the required 32-byte AES key at startup so journal,
support-message, and export encryption cannot fail later during a user action.
Production additionally validates its JWT configuration and fails closed when
PostgreSQL is unavailable. The default local URL is
`postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`.

PostgreSQL seed data and the optional in-memory demo store use the shared dummy
password `password` for `gading@gmail.com`, `dery@gmail.com`,
`suci@gmail.com`, and `nasywa@gmail.com`. These
credentials are public fixtures. Automatic production deploys install only the
four accounts through the separately guarded `/app/seed-accounts` with
`CONFIRM_SEED_ACCOUNTS=CREATE_FOUR_DEMO_ACCOUNTS`; that path never seeds
education, Learning Hub, social, activity, support, or operational fixtures, so
production holds exactly the four accounts with no fixture content. A
staging-only owner-authorized demo environment may instead invoke
`/app/demo-seeder` with
`CONFIRM_DEMO_SEED=CREATE_FOUR_DEMO_ACCOUNTS`, which seeds the accounts plus
their activity fixtures (two active accountability groups
and privacy-safe daily aggregate fixtures for partner-dashboard testing; it
contains no URL, domain, title, timestamped visit, or browsing-history fixture).
The production-safe `make seeder` path never creates demo users or activity.

Useful Makefile targets: `make dev` (air live-reload), `make start` (build +
run `./bin/api` with `.env`), `make key-generate`,
`make lint`, `make migrate-up`, `make seeder`, `make seed`,
`make seed-education`, `make seed-learning-hub`, `make seed-scale` (local 500-2000 rows/table),
`make seed-local-accounts` (clean zero-data local accounts), and opt-in `make verify`. The
Docker image exposes the same operational commands as `/app/migrate-up`,
`/app/migrate-down`, `/app/reset-storage`, `/app/seeder`, `/app/demo-seeder`,
`/app/seed-accounts`, and `/app/seed-learning-hub`; the automatic production
seeder installs the four accounts only, and the automatic seeder logs only
aggregate Learning Hub inserted/skipped counts.
Outside production, CORS accepts any `http://localhost:*`/`http://127.0.0.1:*`
origin, so a local Flutter web run (`flutter run -d chrome --web-port 45051`)
works without editing `CORS_ALLOWED_ORIGINS` or restarting the server.
`migrate-down` and `make migrate-fresh` drop the database schema
and must never be run against shared or production data; `migrate-down`
additionally refuses to run without `CONFIRM_MIGRATE_DOWN=DROP_ALL_DATA`.
Both commands also empty the runtime storage directories (media/avatars and
encrypted exports) through the separately guarded reset-storage command, since
every dynamic file is orphaned once the schema is dropped; run `make seed`
afterwards to regenerate the bundled seed media.
`make key-generate` refuses to replace a valid existing key; use
`make key-generate FORCE=1` only when no encrypted local journal, support, or
export data needs to be retained.

## Key local endpoints

- `GET  /healthz`
- `GET  /readyz`
- `POST /v1/auth/dev-login`
- `POST /v1/auth/password-reset/request`
- `POST /v1/auth/password-reset/confirm`
- `POST /v1/devices`
- `POST /v1/devices/:device_id/grant-key/challenge`
- `PUT  /v1/devices/:device_id/grant-key`
- `PATCH /v1/me/password`
- `GET  /v1/client/dashboard-summary`
- `GET  /v1/client/protection-status`
- `GET  /v1/client/protection-analytics`
- `POST /v1/client/aggregate-events` (may include `blocked_event_times`)
- `GET  /v1/client/spk-recommendation`
- `POST /v1/client/spk-interventions/:id/complete`
- `GET/PUT /v1/client/spk-preference`
- `GET/POST /v1/check-ins`
- `GET/PATCH /v1/me`
- `GET/PUT /v1/me/reminder-preference`
- `POST/DELETE /v1/me/push-subscription`
- `POST/DELETE /v1/me/avatar`
- `GET  /v1/users/:id/avatar`
- `GET  /v1/psychoeducation/modules`
- `GET  /v1/psychoeducation/modules/:slug`
- `PUT  /v1/psychoeducation/modules/:id/revisions/:revision/progress`
- `POST /v1/psychoeducation/modules/:id/revisions/:revision/checks/:check_id/answer`
- `GET  /v1/education/media/:id`
- `GET/POST/PUT /v1/admin/content/modules[...]`
- `GET/POST/PUT /v1/admin/content/learning-hub/items[...]`
- `GET/POST/PUT/DELETE /v1/admin/content/learning-hub/taxonomy[...]`
- `POST /v1/admin/content/media`
- `GET  /v1/accountability/workspace`
- `GET  /v1/accountability/analytics?days=14|30&group_id=`
- `GET  /v1/admin/analytics?days=14|30`
- `POST /v1/accountability/groups[...]`
- `PATCH /v1/accountability/memberships/:membership_id/sharing`
- `POST /v1/accountability/memberships/:membership_id/leave`
- `POST /v1/accountability/exit-requests/:request_id/cancel`
- `GET  /v1/approval-requests`
- `POST /v1/approval-requests/:id/apply`
- `GET/POST /v1/emergency-key-requests`
- `POST /v1/devices/unlock`
- `GET /v1/admin/emergency-key-requests`
- `POST /v1/admin/emergency-key-requests/:id/review`
- `POST /v1/admin/emergency-key-requests/:id/approve`
- `GET  /v1/portal/overview`
- `GET  /v1/missions/today`
- `POST /v1/missions/claim`
- `POST /v1/missions/custom`
- `PATCH/DELETE /v1/missions/custom/:id`
- `GET  /v1/client/progress?days=7|30|90`
- `GET/POST/PATCH /v1/reflections[...]`
- `GET/POST /v1/recovery-practices`
- `GET/PATCH /v1/recovery-space`
- `GET/PUT /v1/weekly-reviews/current`
- `GET/PUT /v1/recovery-records`
- `GET/POST /v1/support-cases[...]`
- `GET/POST /v1/data-requests`
- `GET /v1/data-requests/:id/download`

All responses use the envelope `{ "data", "error", "request_id" }` produced in
`internal/handler/handler.go` / `internal/middleware/middleware.go`.

`POST /v1/accountability/exit-requests/:request_id/cancel` is student-scoped.
It changes only the requesting student's pending normal exit to `cancelled`
and restores that membership to `active`; unsafe exits and already-resolved
requests cannot be cancelled. Success returns `{ "cancelled": true }` inside
the standard envelope.

### Client protection contract

- `POST /v1/devices` requires `client_instance_id` and upserts the authenticated
  user's installation; a new device starts `inactive`.
- `PATCH /v1/me/password` requires `current_password` and `new_password`, then
  revokes every refresh token so clients must reauthenticate.
- `GET /v1/me` includes `password_enabled` so clients can distinguish
  password-backed accounts from provider-only accounts without exposing a hash.
- `POST /v1/me/avatar` accepts an authenticated user's cropped WebP avatar up
  to 2 MiB. `GET /v1/users/:id/avatar` is authenticated, returns only the
  managed image with a private cache directive, and never exposes a storage
  path. `DELETE /v1/me/avatar` restores the initials fallback.
- `GET /v1/client/protection-analytics?device_id=<id>&days=7|30` returns daily
  and total counters only.
- Approval responses keep stable `action`/`status` codes separate from
  localized labels. `POST /v1/approval-requests/:id/apply` is device-bound,
  idempotent after first use, and available for 30 minutes after resolution.
  Applying requires the device grant key described below and returns a compact
  ES256 JWS in `data.grant_token`; the legacy request/action/time fields remain
  during client rollout.
- Accountability roles are backend-authoritative. Verified students preview
  and confirm one live group membership; verified email+WhatsApp partners own
  multiple groups with hashed, rate-limited, rotatable codes. Partner decisions,
  member removal, archive, and code rotation require a session authenticated
  within 15 minutes. Authenticated group projections include `owner_name` and
  the optional auth-protected `owner_avatar_url`; storage paths are never
  exposed.
- Category-specific partner projections expose only protection health/activity,
  recovery engagement counts, and education progress bands. Unsafe student
  exit and partner removal stop sharing immediately. Protection activity is
  summed over the latest seven `Asia/Jakarta` calendar dates. The accompanying
  sharing preference lets clients render an omitted numeric zero as a shared
  zero while keeping an unshared category hidden. No event timestamp, URL,
  domain, or browsing-history detail enters the projection.
- Emergency recovery is device-bound: the user creates a request, one platform
  admin reviews it, a different platform admin approves/issues it, and
  `/v1/devices/unlock` consumes the 24-hour single-use key for a ten-minute
  `emergency_access` grant. A retry while that grant remains valid returns the
  same persisted `jti`, `iat`, and `exp` claims.
- `POST /v1/check-ins` persists the authenticated user's structured mood score
  and optional urge score (`0` means not disclosed); it accepts no browsing
  context. Partner visibility is not exposed by this endpoint.
- Account export is created synchronously as an AES-256-GCM encrypted ZIP at
  rest. A completed result is downloadable with recent authentication for seven
  days; expired or legacy records without a valid managed file are retained as
  history but explicitly marked unavailable so clients can offer regeneration.
- `GET/PUT /v1/recovery-records` stores only explicitly submitted student
  records; sensitive text is AES-256-GCM encrypted, reminders default off, and
  records older than 12 months are removed. `GET /v1/client/progress` supports
  7/30/90 days and withholds trend claims until three check-ins exist.
- Reflection payload v2 encrypts journal content, optional mood/next-step, and
  current-focus state with AES-256-GCM. Completed recovery practices and typed
  weekly reviews retain for 12 months. The current-week review endpoint upserts
  one encrypted structured review and returns its idempotent EXP/cap result.
  Recovery-space unlock/placement state is
  deterministic, retained for the account lifetime, and included with practice
  history in account export and deletion. Active timers and task labels are not
  accepted by these endpoints.
- Support cases use encrypted message threads and explicit
  waiting-support/waiting-user/resolved/closed transitions. Only `user` and
  `partner` accounts can use owner-scoped requester endpoints; a verified
  `admin` handles reports through the admin queue and must claim an unassigned
  case before reading, replying, transitioning, or releasing it with an audited
  reason.
- `GET /v1/missions/today` returns exactly five `Asia/Jakarta` slots and the
  authenticated student's private level/EXP progress. Each system or custom
  mission is worth 10 EXP. The system catalog covers active protection, a
  daily check-in, education section/module progress, and a recovery practice;
  system claims are rechecked against existing account evidence. The student
  may create up to five custom missions through `/v1/missions/custom`; a custom
  mission replaces one system slot, can be edited/deleted while pending, and is
  completed as a private self-attestation. Both sources use the same claim
  contract; no skip endpoint exists. Custom titles are AES-256-GCM encrypted;
  custom self-attestations never appear in partner/admin aggregates.
- Psychoeducation publication stores immutable bilingual document snapshots.
  Audience (`student`, `partner`, `all`) and experience type (`article`,
  `response_simulator`) are server-validated and enforced for both list and
  direct-slug reads. Progress is revision-scoped and counts required sections,
  media, and knowledge checks. Uploaded media is MIME-sniffed and size-bounded;
  external media is restricted to configured HTTPS hosts.
- Account roles are exactly `user`, `partner`, and `admin`. Admins directly
  provision immutable-role accounts with one-time temporary passwords, and
  manage content, support queue, safe public social links,
  audit history, and dual-control emergency access. Refresh rotation preserves the original authentication time
  used by recent-auth gates, and disabled/changed operator identity is checked
  on every bearer request.
- Account export is encrypted at rest and expires after seven days. Student or
  partner deletion requires a 30-minute email confirmation plus recent auth;
  account-scoped records are removed while retained audit/request rows are
  anonymized.

These endpoints reject ownership mismatches and never accept URL, domain, DOM,
page title, browsing history, screenshot, feature-vector, or per-page score
fields.

### Signed native protection grants

Before a native Android or Windows installation can apply an approved pause,
uninstall, or emergency action, it enrolls a non-exportable device P-256 key:

1. Authenticated owner calls
   `POST /v1/devices/:device_id/grant-key/challenge`. The response data contains
   `{"challenge_token":"<JWS>","expires_at":"<RFC3339>"}`. The challenge is
   valid for five minutes and has JWS `typ=gamblock-device-key-challenge+jwt`.
2. The client calls `PUT /v1/devices/:device_id/grant-key` with
   `{"challenge_token":"<JWS>","public_jwk":<P-256-public-JWK>,"proof":"<base64url>"}`.
   `proof` is the raw 64-byte ES256 `R || S` signature over the SHA-256 digest
   of the exact UTF-8 bytes
   `gamblock-device-key-v1\n<device_id>\n<challenge_token>`.
3. The backend stores the canonical public JWK and its RFC 7638 thumbprint.
   Enrollment is set-once: resubmitting the same key is idempotent and replacing
   it with a different key is rejected.

Both `POST /v1/approval-requests/:id/apply` and
`POST /v1/devices/unlock` return `data.grant_token`. The compact JWS header is
`alg=ES256`, `typ=gamblock-grant+jwt`, plus the configured `kid`. Its claims are
`iss=gamblock-ai-backend`, `aud=gamblock-protection-native`,
`grant_version=1`, `request_id`, `device_id`, `action`, `iat`, `nbf`, `exp`,
`jti`, and `cnf.jkt`. Pause grants are exactly 15, 30, 60, or 120 minutes;
uninstall and emergency grants are at most ten minutes. Native clients must
verify the signature and every binding/time claim offline, compare `cnf.jkt`
with their enrolled local key, and reject tokens whose `jti` is missing or
malformed or whose `exp` has passed. Backend retries preserve the same `jti`,
`iat`, and `exp` so a grant is a bounded maintenance window rather than a
one-shot command.

Production reads only the active private signing key and ID:

```sh
openssl ecparam -name prime256v1 -genkey -noout -out protection-grant-private.pem
openssl pkcs8 -topk8 -nocrypt -in protection-grant-private.pem -outform DER -out protection-grant-private.der
openssl pkey -in protection-grant-private.pem -pubout -outform DER -out protection-grant-public.der

# Backend secret/config:
base64 < protection-grant-private.der | tr -d '\n'
GRANT_KEY_ID=grant-2026-01
# Set PROTECTION_GRANT_SIGNING_KEY_ID to GRANT_KEY_ID.

# Client public trust-store value:
GRANT_PUBLIC_SPKI_BASE64="$(base64 < protection-grant-public.der | tr -d '\n')"
printf '{"%s":"%s"}' "$GRANT_KEY_ID" "$GRANT_PUBLIC_SPKI_BASE64" | base64 | tr -d '\n'
```

The final command produces the public client environment value
`PROTECTION_GRANT_TRUST_STORE_BASE64`: base64 of a JSON map
`{kid: base64-DER-SPKI-P256}`. It is not a backend variable or a secret. Keep
the PEM/DER private-key files out of source control. During rotation, publish
both old and new public keys to clients before switching the backend's active
`kid`, and retain the old public key until every previously issued grant and
challenge has expired.

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
Password login uses Argon2id. `POST /v1/auth/password-reset/request` is
non-enumerating; `POST /v1/auth/password-reset/confirm` consumes the latest
hashed 12-character code within 30 minutes and revokes all refresh sessions.
Fonnte is the production transactional delivery adapter. Production requires
`FONNTE_TOKEN`; WhatsApp verification/reset/export notifications are unavailable
without it rather than exposing demo codes. Email remains the login identity.
Development login and
runtime contextual demo records remain separately opt-in and forbidden in
production; only the explicitly confirmed one-shot demo seeder is exempt.

Production CI builds the private GHCR image on `main`, including the API,
migrate-up, guarded migrate-down/reset-storage, production-safe seeder,
owner-confirmed demo seeder, and Learning Hub seeder binaries. Its
deploy step is disabled until `ENABLE_VPS_DEPLOY=true`. The current production
configuration keeps that gate false because GitHub-hosted runners cannot
reliably reach the password-authenticated SSH endpoint; use the authorized
infrastructure `make deploy` path instead. If the gate is enabled, the
workflow connects to the pinned VPS as root with password authentication on
port 22 and runs the Ansible-installed `update.sh`, which backs up PostgreSQL,
runs migrate-up and the safe seeder, then replaces the API container.
Infrastructure rejects
application deployment until the private GHCR pull PAT and core
database/JWT/AES secrets exist; delivery-provider credentials are optional.
