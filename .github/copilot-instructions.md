# Gamblock-AI Backend Copilot Instructions

Read and follow `AGENTS.md` and `docs/ai/README.md` before changing code.
Preserve handler -> service -> repository -> ent layering, the response
envelope, encrypted journals, and the on-device privacy boundary. Never send or
persist browsing data, bypass stable error catalogs, or hand-edit generated ent
files. Run `make lint` by default; tests/builds/`make verify` require explicit
user request.
