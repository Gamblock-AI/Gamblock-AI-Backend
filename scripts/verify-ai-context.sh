#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ALLOW_UNTRACKED=false
if [[ "${1:-}" == "--allow-untracked" ]]; then
  ALLOW_UNTRACKED=true
elif [[ $# -gt 0 ]]; then
  echo "Usage: $0 [--allow-untracked]" >&2
  exit 2
fi

required=(
  .gitattributes AGENTS.md README.md CLAUDE.md GEMINI.md
  .github/copilot-instructions.md
  .cursor/rules/gamblock-ai.mdc
  docs/ai/README.md docs/ai/manifest.yaml
  .agents/skills/verify-gamblock-change/SKILL.md
  .agents/skills/verify-gamblock-change/agents/openai.yaml
)

for file in "${required[@]}"; do
  [[ -f "$ROOT_DIR/$file" ]] || { echo "missing AI context file: $file" >&2; exit 1; }
  if [[ "$ALLOW_UNTRACKED" == false ]]; then
    git -C "$ROOT_DIR" ls-files --error-unmatch "$file" >/dev/null 2>&1 || {
      echo "AI context file is not tracked: $file" >&2
      exit 1
    }
  fi
done

grep -Fq '@./AGENTS.md' "$ROOT_DIR/CLAUDE.md"
grep -Fq '@./AGENTS.md' "$ROOT_DIR/GEMINI.md"
grep -Fq 'alwaysApply: true' "$ROOT_DIR/.cursor/rules/gamblock-ai.mdc"
grep -Fq 'context_version: 2026-07-18.4' "$ROOT_DIR/docs/ai/manifest.yaml"

if git -C "$ROOT_DIR" ls-files --error-unmatch .env >/dev/null 2>&1; then
  echo '.env must not be tracked' >&2
  exit 1
fi

echo 'backend AI context verification passed'
