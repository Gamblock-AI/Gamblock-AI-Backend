# Backend AI Context


Jika ada pertentangan dengan `pkm_proposal.md`, proposal PKM adalah sumber mutlak.

## Product capsule

Gamblock-AI is an on-device gambling blocker and recovery platform for
Indonesian university students. The Android/Windows client performs all
classification and blocking locally. This backend manages accounts, groups,
approval workflows, aggregate protection status, psychoeducation, journal
workflows, release metadata, and support operations.

The PKM proposal is the product authority. Core requirements include partner
relationship/removal approval, web recovery state/content, and strict
local-detection privacy. WhatsApp, Group Codes, admin/operator flows, journals,
and release management are supporting/operational features.

## Hard boundaries

- Raw DOM, URLs, domains, screenshots, keystrokes, and browsing history never
  enter this API. Accept privacy-preserving aggregates only.
- Journal/reflection writes require AES-256-GCM and fail closed when the key or
  encryption operation is invalid. Decryption failure also fails closed.
- The browser extension is only a passive sensor; the client owns blocking.
- Anti-tamper never uses critical-process APIs.
- API responses use `{ data, error, request_id }` and stable error codes.
- Client-visible error text is catalog-safe in every environment; expected 4xx
  rejections log metadata only, 5xx details stay in server logs, and the root
  context validator rejects literal handler/middleware codes missing from the
  synchronized catalogs.

## Current capability truth

| Area | State | Evidence/limit |
|---|---|---|
| Auth, contact verification, token rotation, RBAC | Implemented code-complete prototype with Fonnte WhatsApp delivery | Argon2id password verification; authoritative roles; phone verification and single-use hashed password-reset codes with latest-code/attempt limits; refresh rotation; per-request disabled/role checks; and password/reset session revocation are wired. Production requires FONNTE_TOKEN. |
| Daily reminder & web push | Implemented supporting feature | opt-in reminder preference synced via `GET/PUT /v1/me/reminder-preference`; Web Push subscriptions via `POST/DELETE /v1/me/push-subscription`; a once-per-minute scheduler delivers Duolingo-style nudges at each user's local time using VAPID keys and prunes 404/410 endpoints; supporting feature, not a proposal requirement |
| Device and aggregate client APIs | Implemented | stable client-instance upsert, owned-device enforcement, heartbeat/status, completed-day idempotent aggregate ingest, and 7/30-day aggregate analytics are wired; no browsing schema exists |
| Accountability groups and approvals | Implemented supporting workflow | verified partners own multiple groups with hashed/rotatable codes; a verified student has one live membership with category-specific aggregate sharing; the workspace projection exposes consented per-student `block_count_sync` totals over the latest seven `Asia/Jakarta` calendar dates, lets clients distinguish a shared zero through the accompanying sharing preference, and has privacy-safe demo fixtures for two groups; pending normal exits can be cancelled by the requesting student, while unsafe exit remains immediate and support-reviewed; removal, archive, scoped pause/uninstall requests, recent-auth partner decisions, bounded one-time native grants, and hashed quick tokens are wired; native device proof remains |
| PrivacyGuard | Implemented | forbidden-key regression tests; values are not censored; narrow credential routes are exempt and CORS wraps guard rejections so localhost browser clients receive readable envelopes |
| Journal encryption | Implemented server invariant | AES-256-GCM write/read paths fail closed; every environment validates a required 32-byte hex key at startup |
| PostgreSQL/ent persistence | Implemented production path | production fails closed on open/migration/load failure; development can use empty memory and explicitly enabled contextual demo data |
| Structured check-ins | Implemented for account persistence | authenticated users save a 1-5 mood and optional 1-5 urge (`0` means not disclosed); no browsing data is accepted and partner visibility remains planned pending explicit consent design |
| Recovery room, journal, and progress | Implemented supporting workflow | student-only completed practices and typed weekly reviews retain a rolling 12 months; deterministic room unlock/placement state retains for account lifetime (rule version 2: a 20-item tiered catalog over practices/active days/reviews/missions/levels computed as the union of stored and derived unlocks so earlier unlocks are always preserved, per-item placement slots validated server-side with one item per slot, and a second `sunrise_study` theme at level 18; locked placements/themes reject with the stable code `recovery_space_item_locked`); AES-256-GCM reflection payload v3 stores one rich-text daily journal per `Asia/Jakarta` day (headings, emphasis, lists, quotations, and up to five private inline images), while legacy v1/v2 reflections remain readable; check-ins update the current `Asia/Jakarta` day without backfill; private progress exposes category-tagged 7/30/90-day activity and suppresses trends below three check-ins; the full intention focus-period/reminder lifecycle remains incomplete core work |
| Threaded support | Implemented operational workflow | only user/partner requesters access their own cases; encrypted messages transition between waiting-support/waiting-user/resolved/closed; verified admins work exclusively from the queue and atomically claim/release ownership before reading or replying |
| Daily mission EXP | implemented | exactly five `Asia/Jakarta` slots per day, each worth 10 EXP: the default system catalog has active protection, check-in, education section, education module, and recovery practice. A student may create up to five private custom missions; each replaces one system slot, is editable/deletable while pending, and completes by private self-attestation. System claims remain server-verified, custom self-attestations use the same claim contract, and all claims are idempotent. The skip endpoint and mutation are retired; historical skipped records are read only for compatibility. Custom titles are AES-256-GCM encrypted at rest, and custom self-attestations are excluded from partner/admin aggregates. The old primary/bonus rotation, replacement flow, and separate practice EXP award are retired |
| Learning Hub catalog, progress, and CMS | implemented | idempotent UTY institution/program/cluster/content seed; published bilingual catalog API; user-scoped saved/started/completed progress; AES-256-GCM checkpoint reflections/outcomes; once-per-item 10 EXP grants with a shared Jakarta daily cap of 50; verified-admin item/taxonomy CRUD, review/publish/archive lifecycle, immutable revisions, and reasoned rollback; catalog reads never use an unpublished draft and CMS reads never expose student progress/reflections; learning items expose a provider logo + course thumbnail (self-hosted via the education media pipeline, served by `GET /v1/education/media/:id`); the seeder bundles dummy logos/thumbnails and retrofits missing media onto existing items |

Weekly review is implemented for
the structured weekly-review slice: `/v1/weekly-reviews/current` restores and
upserts one encrypted review per Jakarta week, preserves legacy fields, and
grants one idempotent 10-EXP reward under the shared daily cap. The response
reports the reward/cap state and current experience; no separate journal entry,
URL, DOM, or browsing data is accepted.
| Dashboard/profile/aggregate API | Implemented | user-scoped summaries derive from owned records; Flutter sends only bounded daily aggregate categories with idempotency; authenticated avatar upload/delete and session-gated avatar retrieval use managed 2 MiB WebP files rather than provider-hosted image URLs; own-profile responses expose only a derived password-enabled boolean for account security UI |
| Emergency recovery | Implemented operational workflow | protected user requests for an owned device; one platform admin reviews and a distinct second admin issues within 30 minutes; hashed device-bound key is single-use for 24 hours and produces a ten-minute grant |
| Psychoeducation authoring and progress | implemented | bilingual revisioned rich-text documents, immutable draft/publish/rollback snapshots, role-enforced student/partner/all audience and article/response-simulator experience metadata, 1–8 thumbnails, allowlisted image/video/PDF media, reviewer/source metadata, review/publish/archive lifecycle, and revision-scoped section/media/check progress are wired; editorial and clinical governance remain operational responsibilities |
| Data export/deletion | Implemented operational workflow | export creates an AES-256-GCM encrypted ZIP at rest with a seven-day recent-auth download; missing configuration fails at startup, expired/legacy results are marked unavailable rather than advertised as downloads, and failed processing remains visible for recovery; student/partner deletion requires a hashed 30-minute email token and recent auth, deletes account-scoped records, and anonymizes retained audit/request rows; external lifecycle cleanup remains operational |
| Three-role admin control plane | Implemented operational v1 | authoritative roles are `user`, `partner`, and `admin`; legacy roles migrate transactionally; admins directly create immutable-role accounts with a one-time temporary password and forced first-login change, enable/disable other accounts, and manage all operational work behind verified-email/recent-auth/audit gates |
| Release gates and rollout | Implemented operational v1 | allowlisted artifact upload uses randomized managed storage and server-computed SHA-256; model/ruleset/network releases can be staged to manual platform/percentage/app-version cohorts and activated, paused, completed, or rolled back; contextual demo releases explicitly say `evaluated: false` and carry no fabricated Phase 4 metrics; signing and automated health decisions remain planned |
| WhatsApp delivery | Implemented Fonnte adapter | immediate delivery uses configured phone numbers and Fonnte; logs omit tokens, recipients, and message bodies |
| Model training/inference | Outside this repository | proposal-required training belongs to a governed model workstream; inference is client-side |

Do not infer production readiness from a handler or schema existing. Verify the
route wiring, persistence path, tests, and external integration separately.

Operational deployment status: the Docker image creates persistent
artifact/export/media/avatar paths and contains the API plus migrate-up,
guarded migrate-down, production-safe seeder, and Learning Hub seeder
binaries. The safe seeder logs only aggregate Learning Hub inserted/skipped
counts and
installs public education/social defaults only when their collections are
empty; it creates no demo users or activity. CI can deploy the private GHCR
image to the pinned VPS as root over password SSH, where `update.sh` takes a
pre-migration PostgreSQL backup and runs migrate-up plus the safe seeder before
replacing the API container. The deploy step remains disabled by repository
variable until intentionally enabled; Ansible rejects an application deploy
while the private-GHCR PAT or core database/JWT/AES/Fonnte secrets are missing.

## Default AI validation

Run `make lint`. When AI context changed, also run
`./scripts/verify-ai-context.sh` (use `--allow-untracked` while authoring new
context files). Tests, builds, race tests, and `make verify` run only when the
user explicitly requests them.

## Local encryption-key bootstrap

After copying `.env.example` to `.env`, run `make key-generate`. It writes a
cryptographically random 32-byte hex `JOURNAL_ENCRYPTION_KEY` and refuses to
replace a valid existing key unless `FORCE=1` is explicitly supplied. Key
replacement makes encrypted local journal, support, and export data
unreadable.

## Related repositories and contracts

- Website: `https://github.com/Gamblock-AI/Gamblock-AI-Website`
- Flutter client: `https://github.com/Gamblock-AI/Gamblock-AI-Apps`
- Browser extension: `https://github.com/Gamblock-AI/Gamblock-AI-Browser-Extention`
- Infrastructure: `https://github.com/Gamblock-AI/Gamblock-AI-Infrastructure`

Error-code changes require matching website and Flutter catalog updates.
Payload changes must preserve the privacy boundary. Client-facing endpoint
changes require consumer updates, but website page-route policy is changed only
when an actual web page/access route changes.

## Context maintenance

Update this file when backend architecture, commands, privacy enforcement, or
capability state changes. Shared invariant changes require a context-version
bump coordinated through the umbrella workspace.
