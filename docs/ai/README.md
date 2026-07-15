# Backend AI Context

Context version: `2026-07-15.2`

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
- **Target invariant:** journal/reflection text must be AES-256-GCM encrypted
  before persistence. The current `ReflectionService` violates this invariant
  when the key is absent or encryption fails by falling back to plaintext;
  treat that path as a P0 gap and do not reproduce it.
- The browser extension is only a passive sensor; the client owns blocking.
- Anti-tamper never uses critical-process APIs.
- API responses use `{ data, error, request_id }` and stable error codes.

## Current capability truth

| Area | State | Evidence/limit |
|---|---|---|
| Auth, token rotation, RBAC | Implemented | handlers/services/tests exist |
| Organization and partner approvals | Implemented | route and service coverage exists |
| PrivacyGuard | Implemented | forbidden-key regression tests; values are not censored |
| Journal encryption | Prototype with P0 gap | AES helper/path exists, but missing key or encryption error currently stores plaintext; target must fail closed |
| PostgreSQL/ent persistence | Implemented with prototype fallback | API falls back to seeded memory on DB failure |
| WhatsApp approval batching | Implemented service logic | external provider behavior still needs environment integration evidence |
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
