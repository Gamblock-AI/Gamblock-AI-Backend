# Gamblock-AI Backend

Go/Gin API service for the Gamblock-AI Flutter client and the Next.js web
dashboard. Uses [ent](https://entgo.io/) over PostgreSQL.

## Run locally

```sh
cp .env.example .env
make key-generate  # creates and saves a valid JOURNAL_ENCRYPTION_KEY in .env
make migrate        # apply schema migrations with values loaded from .env
make seed           # (optional) seed demo data
make seed-education # upsert the six bilingual education modules/media
make run            # start the API (default 127.0.0.1:8080)
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
`suci@gmail.com`, `nasywa@gmail.com`, `student@gmail.com`,
and `partner@gmail.com`. These
credentials are development fixtures only and demo data is forbidden in
production.

Useful Makefile targets: `make dev` (air live-reload), `make key-generate`, `make lint`,
`make migrate`, `make seed`, `make seed-education`, and opt-in `make verify`. `make migrate-fresh` drops the
database schema and must never be run against shared or production data.
`make key-generate` refuses to replace a valid existing key; use
`make key-generate FORCE=1` only when no encrypted local journal, support, or
export data needs to be retained.

## Key local endpoints

- `GET  /healthz`
- `GET  /readyz`
- `POST /v1/auth/dev-login`
- `POST /v1/auth/google`
- `POST /v1/auth/password-reset/request`
- `POST /v1/auth/password-reset/confirm`
- `POST /v1/devices`
- `PATCH /v1/me/password`
- `POST /v1/me/google/link`
- `GET  /v1/client/dashboard-summary`
- `GET  /v1/client/protection-status`
- `GET  /v1/client/protection-analytics`
- `POST /v1/client/aggregate-events`
- `GET/POST /v1/check-ins`
- `GET/PATCH /v1/me`
- `POST/DELETE /v1/me/avatar`
- `GET  /v1/users/:id/avatar`
- `GET  /v1/psychoeducation/modules`
- `GET  /v1/psychoeducation/modules/:slug`
- `PUT  /v1/psychoeducation/modules/:id/revisions/:revision/progress`
- `POST /v1/psychoeducation/modules/:id/revisions/:revision/checks/:check_id/answer`
- `GET  /v1/education/media/:id`
- `GET/POST/PUT /v1/admin/content/modules[...]`
- `POST /v1/admin/content/media`
- `GET  /v1/accountability/workspace`
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
- `PATCH /v1/missions`
- `POST /v1/missions/claim`
- `POST /v1/missions/adjust`
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
- Accountability roles are backend-authoritative. Verified students preview
  and confirm one live group membership; verified email+WhatsApp partners own
  multiple groups with hashed, rate-limited, rotatable codes. Partner decisions,
  member removal, archive, and code rotation require a session authenticated
  within 15 minutes.
- Category-specific partner projections expose only protection health/activity,
  recovery engagement counts, and education progress bands. Unsafe student
  exit and partner removal stop sharing immediately.
- Emergency recovery is device-bound: the user creates a request, one platform
  admin reviews it, a different platform admin approves/issues it, and
  `/v1/devices/unlock` consumes the 24-hour single-use key for a ten-minute
  grant.
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
  weekly reviews retain for 12 months. Recovery-space unlock/placement state is
  deterministic, retained for the account lifetime, and included with practice
  history in account export and deletion. Active timers and task labels are not
  accepted by these endpoints.
- Support cases use encrypted message threads and explicit
  waiting-support/waiting-user/resolved/closed transitions. Only `user` and
  `partner` accounts can use owner-scoped requester endpoints; a verified
  `admin` handles reports through the admin queue and must claim an unassigned
  case before reading, replying, transitioning, or releasing it with an audited
  reason.
- `GET /v1/missions/today` returns a deterministic `Asia/Jakarta` daily set of
  one primary and two optional bonus tasks, plus the authenticated user's level
  and EXP progress. It derives claim eligibility from existing account records:
  active protection seen today, today's saved check-in, today's education
  section/module progress, or an active partner link. `POST
  /v1/missions/claim` rechecks eligibility and atomically grants the disclosed
  reward once. `POST /v1/missions/adjust` allows one primary replacement from
  the two non-assigned catalog tasks, followed by an optional skip; both require
  a bounded reason and never change EXP. Legacy `PATCH /v1/missions` is
  claim-only and rejects undo.
  Mission/EXP data is not projected to partners.
- Psychoeducation publication stores immutable bilingual document snapshots.
  Audience (`student`, `partner`, `all`) and experience type (`article`,
  `response_simulator`) are server-validated and enforced for both list and
  direct-slug reads. Progress is revision-scoped and counts required sections,
  media, and knowledge checks. Uploaded media is MIME-sniffed and size-bounded;
  external media is restricted to configured HTTPS hosts.
- Account roles are exactly `user`, `partner`, and `admin`. Admins directly
  provision immutable-role accounts with one-time temporary passwords, and
  manage content, releases, support queue, research, safe public social links,
  audit history, and dual-control emergency access. Refresh rotation preserves the original authentication time
  used by recent-auth gates, and disabled/changed operator identity is checked
  on every bearer request.
- Admins upload allowlisted artifacts to randomized managed storage;
  the backend computes SHA-256 before model/ruleset/network registration and
  supports manual staged cohort activate/pause/complete/rollback transitions.
- Account export is encrypted at rest and expires after seven days. Student or
  partner deletion requires a 30-minute email confirmation plus recent auth;
  account-scoped records are removed while retained audit/request rows are
  anonymized.

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
an ID-token audience is listed in `GOOGLE_CLIENT_IDS` (with
`GOOGLE_CLIENT_ID` as a single-ID fallback). Existing password accounts must
link the same verified Google email through `POST /v1/me/google/link` after
current-password authentication. `POST /v1/auth/password-reset/request` is
non-enumerating; `POST /v1/auth/password-reset/confirm` consumes the latest
hashed 12-character code within 30 minutes and revokes all refresh sessions.
Production requires working SMTP configuration. Development
login and contextual demo records are separately opt-in and forbidden in
production.
