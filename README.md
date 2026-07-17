# TraceDelta

> Runtime behavior diffs for pull requests.

[![Project status: experimental](https://img.shields.io/badge/status-experimental-orange.svg)](#project-status)
[![CI](https://github.com/OWNER/REPO/actions/workflows/ci.yml/badge.svg)](https://github.com/OWNER/REPO/actions/workflows/ci.yml)

**The CI badge uses the intentional placeholder `OWNER/REPO`; replace it with the publishing GitHub repository before making this project public.**

> [!WARNING]
> TraceDelta is early-stage software. The current implementation proves a small local comparison path; it is not yet a complete OTLP analyzer or a stable integration contract.

A source diff tells reviewers which lines changed. It does not tell them that checkout now writes an order before payment succeeds, calls inventory three times, drops a database predicate, or becomes materially slower. TraceDelta is being built to compare OpenTelemetry traces from a baseline and a candidate application version and report those runtime behavior changes directly.

## What it looks like

The included synthetic fixtures produce deterministic text like this:

```text
TraceDelta comparison

Baseline:  testdata/baseline.json
Candidate: testdata/candidate.json

Summary:
  Added spans:    1
  Removed spans:  1
  Changed spans:  2

Changes:
  ADDED    inventory.reserve
  REMOVED  cache.get
  CHANGED  checkout.handle status OK -> ERROR
  CHANGED  payment.charge duration 120ms -> 245ms

Result: behavioral differences detected
```

## Current capabilities

The initial vertical slice intentionally supports only a documented, simplified OTLP-compatible JSON shape. It can:

- read baseline and candidate files into strongly typed models;
- reject malformed input with contextual errors;
- remove generated trace/span IDs and absolute timestamps from the current flat comparison representation;
- compare spans deterministically by the current stable matching key;
- report added and removed spans, status changes, and duration increases meeting a percentage threshold;
- render a deterministic terminal report; and
- return exit code `0` for no meaningful differences, `1` for detected differences, and `2` for usage, file, or parse errors.

The fixture shape and its current restrictions are documented in [`testdata/README.md`](testdata/README.md). This format is a stepping stone, not a claim of complete OTLP JSON support.

## Planned capabilities

Version 0.1 is planned to add reliable OTLP JSON ingestion, stronger configurable normalization, trace and span matching, semantic service/database/error rules, JSON and standalone HTML reports, configurable regression thresholds, and a GitHub Actions-friendly workflow. See the [roadmap](ROADMAP.md) and [prioritized task backlog](docs/tasks.md) for the honest implementation state.

## Quick start

TraceDelta requires Go 1.26, the newest stable Go toolchain used by this repository at initialization.

```bash
go test ./...
go run ./cmd/tracedelta compare \
  --baseline testdata/baseline.json \
  --candidate testdata/candidate.json
```

The example intentionally contains behavioral differences, so the comparison exits with code `1`. `go run` may display `exit status 1`; that is expected. Build the binary when a script needs to inspect the exact program exit code:

```bash
go build -o ./bin/tracedelta ./cmd/tracedelta
./bin/tracedelta compare --baseline testdata/baseline.json --candidate testdata/candidate.json
```

The comparison command, flags, output contract, and exit codes are specified in [`docs/cli-spec.md`](docs/cli-spec.md).

## Development

Common commands are exposed through the Makefile:

```bash
make help
make check
make test-race
make run-example
```

`make check` formats/verifies the project as documented in [`docs/development.md`](docs/development.md). The equivalent shell entry point is `./scripts/check.sh`.

## Architecture

TraceDelta uses a staged local pipeline:

```text
OTLP JSON -> parse -> normalize -> match -> diff -> report
```

The current slice implements only the minimum of those stages needed for deterministic span comparison. Package boundaries preserve the complete pipeline without requiring a database, network service, or web application. Read [`docs/architecture.md`](docs/architecture.md) and [ADR 0001](docs/decisions/0001-initial-architecture.md) for the design and its tradeoffs.

## Project status

TraceDelta is experimental and has no stable release yet. `v0.1.0-dev` identifies the repository's current development line, not a published or compatibility-guaranteed release. Files under `docs/` distinguish current behavior from proposed behavior; incomplete features are tracked rather than implied.

The Go module path `github.com/example/tracedelta` is also an intentional pre-publication placeholder, not a real organization or project claim. Replace it together with `OWNER/REPO`, the security contact, and `CODEOWNERS` before publication.

- [Roadmap](ROADMAP.md)
- [Product specification](docs/product-spec.md)
- [Current state](docs/current-state.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## License

TraceDelta is available under the [MIT License](LICENSE).
