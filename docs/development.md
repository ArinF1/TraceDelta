# Development guide

## Prerequisites

- Go 1.26, which was the newest stable toolchain used when the repository was initialized.
- Git.
- GNU Make for the documented convenience targets.
- Bash for `scripts/check.sh` (the underlying Go commands also work directly on Windows).

TraceDelta deliberately has no database, container stack, frontend toolchain, or external service dependency.

## First checkout

From the repository root:

```bash
go version
go mod download
go test ./...
go vet ./...
go build ./cmd/tracedelta
```

Run existing tests before changing code. If the starting state fails, record the exact failure before attempting a task so it is not confused with a regression.

## Common commands

The default Make target prints help.

| Command | Purpose |
| --- | --- |
| `make help` | List supported development targets. |
| `make build` | Build the CLI. |
| `make test` | Run all Go tests. |
| `make test-race` | Run tests with the race detector. |
| `make fmt` | Apply `gofmt` to Go files. |
| `make fmt-check` | Fail if a Go file is not formatted. |
| `make vet` | Run `go vet ./...`. |
| `make check` | Run the essential local verification suite. |
| `make run-example` | Run the intentionally different sample comparison. |
| `make clean` | Remove known local build/report outputs. |

`./scripts/check.sh` is the shell equivalent used to keep essential local checks aligned with CI.

## Repository map

- `cmd/tracedelta/`: process entry point and CLI adaptation.
- `internal/`: non-public domain, parser, normalization, matching, diff, and reporting code.
- `pkg/tracedelta/`: deliberately small public orchestration package.
- `testdata/`: synthetic, publishable fixture data.
- `docs/`: product contracts, architecture, state, tasks, and ADRs.
- `examples/`: documented user workflows.
- `.github/`: contribution templates and continuous integration.

Read [`../AGENTS.md`](../AGENTS.md) before making changes. It defines the required project-memory and end-of-session workflow.

## Development workflow

1. Read the project memory and relevant ADRs.
2. Run the current test suite.
3. Choose one task from [`tasks.md`](tasks.md), normally the first unblocked item in **Now**.
4. Make the smallest coherent change that meets its acceptance criteria.
5. Add tests that fail without the behavioral change.
6. Run focused tests while iterating, then the full repository checks.
7. Update public documentation and project-state files.
8. Review `git diff` and `git status` for accidental files or sensitive data.
9. Append a session-log entry.

Do not bundle opportunistic refactors with a behavior change. Add a separately prioritized task when the refactor is genuinely useful.

## Go conventions

- Use standard-library packages where practical and justify new dependencies.
- Keep process exits in `cmd/`; library packages return typed results and wrapped errors.
- Avoid global mutable state. Pass typed configuration explicitly.
- Prefer concrete functions and structs until multiple real implementations justify an interface.
- Add package comments and comments for exported identifiers.
- Wrap boundary errors with the action and input role/path, without dumping sensitive input.
- Sort output explicitly; never depend on Go map iteration.
- Use `TODO(TD-NNN)` only when the referenced task exists.
- Preserve backward-compatible CLI and data contracts unless an ADR approves a break.

## Tests and fixtures

Unit tests should live next to the package they exercise. End-to-end behavior may use repository fixtures. Tests should cover both findings and intentional non-findings, malformed data, ordering, and error context.

All fixtures must be synthetic. Never commit production traces, real credentials, personal data, internal hostnames, or proprietary query text. When a test needs a sensitive-looking field, use an unmistakably fictional value and assert that it is redacted or excluded.

Prefer small inline values for one-field unit cases and JSON fixtures when the wire shape itself matters. Keep golden output reviewable and regenerate it only as part of an intentional contract change.

## Running the example and checking its exit

The included comparison contains differences and should exit `1`:

```bash
go build -o ./bin/tracedelta ./cmd/tracedelta
set +e
./bin/tracedelta compare --baseline testdata/baseline.json --candidate testdata/candidate.json
status=$?
set -e
test "$status" -eq 1
```

On PowerShell:

```powershell
go build -o .\bin\tracedelta.exe .\cmd\tracedelta
.\bin\tracedelta.exe compare --baseline testdata\baseline.json --candidate testdata\candidate.json
if ($LASTEXITCODE -ne 1) { throw "expected exit code 1, got $LASTEXITCODE" }
```

`go run` is convenient for humans but may wrap program exit codes, so use a built binary for exit-code assertions.

## Adding public behavior

When adding a flag, format, finding kind, configuration key, or public Go symbol:

- define its error and compatibility behavior;
- add positive, negative, and deterministic-ordering tests as applicable;
- update `README.md`, `docs/cli-spec.md`, and/or format documentation;
- update `CHANGELOG.md` under **Unreleased**;
- update project state/tasks; and
- create an ADR if the decision constrains multiple stages or future compatibility.
