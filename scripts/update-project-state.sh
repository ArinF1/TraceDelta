#!/usr/bin/env bash
set -euo pipefail

# Usage: bash scripts/update-project-state.sh
#
# This read-only helper validates the persistent project-memory files and prints
# a concise working-tree summary. It intentionally does not rewrite narrative
# state: the contributor who made a change must review and update those files.

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

required_files=(
  "AGENTS.md"
  "README.md"
  "docs/product-spec.md"
  "docs/architecture.md"
  "docs/current-state.md"
  "docs/tasks.md"
  "docs/session-log.md"
)

missing=0
for path in "${required_files[@]}"; do
  if [[ ! -s "${path}" ]]; then
    printf 'Missing or empty required project-state file: %s\n' "${path}" >&2
    missing=1
  fi
done

if [[ "${missing}" -ne 0 ]]; then
  exit 1
fi

required_task_sections=("Now" "Next" "Later" "Completed" "Explicitly out of scope for v0.1")
for section in "${required_task_sections[@]}"; do
  if ! grep -Fqx "## ${section}" docs/tasks.md; then
    printf 'docs/tasks.md is missing required section: %s\n' "${section}" >&2
    exit 1
  fi
done

printf 'Project-state files and backlog sections are present.\n\n'
printf 'Working tree:\n'
git status --short
printf '\nReview before ending the session:\n'
printf '  1. Update docs/current-state.md for meaningful implementation changes.\n'
printf '  2. Update docs/tasks.md for completed, added, or reprioritized work.\n'
printf '  3. Append a brief entry to docs/session-log.md.\n'
printf '  4. Complete the checklist in AGENTS.md.\n'
