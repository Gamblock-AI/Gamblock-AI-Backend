# Backend AI Context

Context version: `2026-07-16.5`

## Product capsule

Gamblock-AI is an on-device gambling blocker and recovery platform for
Indonesian university students. The Android/Windows client performs all
classification and blocking locally. This backend manages accounts, groups,
approval workflows, aggregate protection status, psychoeducation, journal
workflows, release metadata, and support operations.

The PKM proposal is the product authority. Backend-relevant core requirements
include partner relationship/removal approval (`PKM-ACC-001`, `PKM-ACC-002`),
web recovery state/content (`PKM-WEB-001`, `PKM-WEB-002`, `PKM-WEB-003`,
`PKM-WEB-004`, `PKM-WEB-005`, `PKM-WEB-006`, `PKM-WEB-007`), and strict
local-detection privacy (`PKM-PRIV-001`, `PKM-PRIV-002`, `PKM-PRIV-003`).
WhatsApp, Group Codes, admin/operator flows, journals, and release management
are supporting/operational features.

## Hard boundaries

- Raw DOM, URLs, domains, screenshots, keystrokes, and browsing history never
  enter this API. Accept privacy-preserving aggregates only.
- Journal/reflection writes require AES-256-GCM and fail closed when the key or
  encryption operation is invalid. Decryption failure also fails closed.
- The browser extension is only a passive sensor; the client owns blocking.
- Anti-tamper never uses critical-process APIs.
- API responses use `{ data, error, request_id }` and stable error codes.

## Current capability truth

| Area | State | Evidence/limit |
|---|---|---|
| Auth, token rotation, RBAC | Implemented | Argon2id password verification, Google ID-token audience validation, refresh rotation, disabled-user checks, per-route roles, and current-password-protected password change with session revocation are wired; email reset remains planned |
| Device and aggregate client APIs | Implemented | stable client-instance upsert, owned-device enforcement, heartbeat/status, completed-day idempotent aggregate ingest, and 7/30-day aggregate analytics are wired; no browsing schema exists |
| Partner consent and approvals | Implemented supporting workflow | seven-day email-bound invitations, multiple relationships, scoped device/action requests, authoritative resolution, bounded one-time native apply grants, hashed quick tokens, and revoke/leave paths are wired; native device proof remains |
| PrivacyGuard | Implemented | forbidden-key regression tests; values are not censored |
| Journal encryption | Implemented server invariant | AES-256-GCM write/read paths fail closed; production validates a 32-byte hex key |
| PostgreSQL/ent persistence | Implemented production path | production fails closed on open/migration/load failure; development can use empty memory and explicitly enabled contextual demo data |
| Dashboard/profile/aggregate API | Implemented | user-scoped summaries derive from owned records; Flutter sends only bounded daily aggregate categories with idempotency |
| Emergency recovery | Implemented operational workflow | protected user requests for an owned device; one platform admin reviews and a distinct second admin issues within 30 minutes; hashed device-bound key is single-use for 24 hours and produces a ten-minute grant |
| Content/release gates | Prototype | modules are forced to draft and artifact creation validates storage path plus SHA-256; review/publish/rollback actions are not yet wired |
| WhatsApp delivery | Prototype adapter | immediate delivery can use configured partner phone/provider; demo logs omit tokens and the partner inbox remains authoritative |
| Model training/inference | Outside this repository | proposal-required training belongs to a governed model workstream; inference is client-side |

Do not infer production readiness from a handler or schema existing. Verify the
route wiring, persistence path, tests, and external integration separately.

## Default AI validation

Run `make lint`. When AI context changed, also run
`./scripts/verify-ai-context.sh` (use `--allow-untracked` while authoring new
context files). Tests, builds, race tests, and `make verify` run only when the
user explicitly requests them.

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
