# AGENTS.md

This file is the persistent working agreement for coding agents contributing to TraceDelta. The repository, not prior chat history, is the source of truth.

## Before making changes

1. Read, in order:
   - `README.md`
   - `docs/product-spec.md`
   - `docs/architecture.md`
   - `docs/current-state.md`
   - `docs/tasks.md`
   - any relevant ADRs in `docs/decisions/`
2. Run the existing tests before changing code. Record any pre-existing failure rather than hiding it in later results.
3. Select one clearly defined task. If it is not already in `docs/tasks.md`, add it before expanding the implementation.

## Working rules

- Work on one clearly defined task at a time.
- Do not silently expand project scope. Record useful follow-up work in `docs/tasks.md` instead.
- Add or update tests for every behavioral change.
- Update documentation whenever public behavior, commands, formats, or guarantees change.
- Record important architectural decisions as ADRs in `docs/decisions/`.
- Update `docs/current-state.md` after meaningful work.
- Update `docs/tasks.md` whenever tasks are completed, added, removed, or reprioritized.
- Append a brief entry to `docs/session-log.md` at the end of every work session. Do not rewrite old entries except to correct an objective factual error.
- Never commit secrets, credentials, personal data, or raw production traces. Test traces must be synthetic and safe to publish.
- Preserve backward compatibility unless a documented decision explicitly permits a break.
- Prefer simple, testable code and standard-library solutions over premature abstractions.
- Do not claim a feature is complete unless tests demonstrate its required behavior.
- Do not add a `TODO` comment without a corresponding TraceDelta task ID, for example `TODO(TD-014)`.
- Leave the repository buildable and testable.

## Windows validation

- Use `.\scripts\check.ps1` as the repository-native Windows formatting, test, vet, and build entry point. Run `scripts/check.tests.ps1` after changing its watchdog behavior.
- Let the script run Go checks serially with the normal shared Go cache; do not place required Go validation commands in a parallel command batch or replace `GOCACHE` with a disposable cache.
- The script's repository mutex and per-command watchdog are validation failures when they trigger; report their final messages exactly.
- If an automation client returns a still-running command-cell identifier after its polling window, resume that same cell with the client's wait operation. The yield itself is not evidence that the displayed child command is hung.

## Required end-of-session checklist

Before ending a session, complete every applicable item:

- [ ] The selected task's acceptance criteria have been checked.
- [ ] Tests were added or updated for behavioral changes.
- [ ] `gofmt` was run on changed Go files.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] The CLI builds when executable code changed.
- [ ] Relevant user-facing documentation is accurate.
- [ ] Architectural decisions were recorded as ADRs when warranted.
- [ ] `docs/current-state.md` reflects the repository now.
- [ ] `docs/tasks.md` reflects completed and newly discovered work.
- [ ] A brief entry was appended to `docs/session-log.md`.
- [ ] The diff contains no secrets, personal data, production traces, or accidental generated files.
- [ ] `git status` contains only intended changes.

If a check cannot be completed, document the exact reason in `docs/current-state.md` and the session log. Never represent an unrun check as passing.
