# TraceDelta

> Runtime behavior diffs for pull requests.

[![Project status: experimental](https://img.shields.io/badge/status-experimental-orange.svg)](#project-status)
[![CI](https://github.com/ArinF1/TraceDelta/actions/workflows/ci.yml/badge.svg)](https://github.com/ArinF1/TraceDelta/actions/workflows/ci.yml)

> [!WARNING]
> TraceDelta v0.1 is experimental software. It implements a finite local comparison contract over a documented OTLP JSON profile; it is not a universal OTLP analyzer or a v1-stable integration contract.

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
  CHANGED  payment.charge duration 120ms -> 245ms (thresholds: >=20% and >=10ms)

Result: behavioral differences detected
```

[Watch the 52-second v0.1 demonstration](https://github.com/ArinF1/TraceDelta/releases/download/v0.1.0/tracedelta-v0.1-demo.webm), then use the accompanying [on-screen transcript and reproducible commands](docs/demo.md). It walks through the synthetic inputs, terminal result, JSON/HTML artifacts, and deliberately failing Action in public [PR #7](https://github.com/ArinF1/TraceDelta/pull/7).

## Current capabilities

The current vertical slice supports one documented, strongly typed subset of OTLP JSON while retaining the original synthetic fixtures as compatibility cases. It can:

- read baseline and candidate files into strongly typed models;
- parse one OTLP/HTTP JSON request or a trace-only OTLP File Exporter JSONL stream, including multiple records, resources, scopes, and traces;
- preserve recursive typed attributes and validate events and links while keeping unused payloads out of comparison evidence;
- accept numeric OTLP kind/status enums and exact 64-bit timestamp forms, ignore safe unknown fields, and reject malformed required fields or recognized non-trace envelopes contextually;
- replace generated IDs, absolute timestamps, and input-array order with a deterministic trace-preserving representation that retains root, internal-parent, and missing-external-parent relationships;
- floor durations through an explicit typed embedding option (disabled by default) and canonically project selected HTTP/RPC attributes without mutating parsed input;
- remove built-in credential/personal-data attributes plus caller-denied keys recursively before normalization or evidence construction;
- pair traces and spans deterministically using normalized root identity, safe semantic attributes, matched-parent context, and distinguishable sibling order while failing explicitly on unresolved ambiguity;
- report added and removed spans, status changes, safe `error.type` changes between error spans, and duration increases meeting relative and absolute thresholds;
- render deterministic terminal, schema-versioned JSON, and self-contained offline HTML reports; and
- return exit code `0` for no meaningful differences, `1` for detected differences, and `2` for usage, file, or parse errors.

The supported input profile, compatibility extensions, and current exclusions are documented in [`testdata/README.md`](testdata/README.md); the machine report contract is in [`docs/json-report-schema.md`](docs/json-report-schema.md). This is a bounded trace-only profile, not universal OTLP or arbitrary vendor-exporter compatibility.

## Finite v0.1 scope

Version 0.1 has a fixed release boundary:

- accept either one OTLP/HTTP JSON request object or a trace-only OTLP File Exporter JSONL stream;
- compare traces and spans deterministically;
- detect added spans, removed spans, error-state changes, and latency regressions that meet configurable relative and absolute thresholds;
- redact common secret/personal-data attributes plus caller-supplied denylisted keys before evidence is built;
- render equivalent terminal, versioned JSON, and standalone HTML reports;
- ship a reusable least-privilege GitHub Action, a synthetic example application, and a deliberately regressed public pull request;
- gate the release with useful unit, integration, race, and fuzz tests; and
- publish versioned cross-platform binaries with checksums and a reproducible 45–60 second demonstration.

Database-shape analysis, general service/relationship rules, exhaustive OTLP compatibility, hosted services, telemetry collection, and source attribution do not block v0.1. The rationale is in [ADR 0002](docs/decisions/0002-v0.1-release-boundary.md); the [roadmap](ROADMAP.md) and [prioritized task backlog](docs/tasks.md) show the finite delivery sequence and honest implementation state.

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

## Versioned binaries

The published [v0.1.0 release](https://github.com/ArinF1/TraceDelta/releases/tag/v0.1.0) provides Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 binaries plus one SHA-256 checksum file. Download the binary for your platform and `tracedelta_0.1.0_checksums.txt`, then verify it before execution:

```bash
sha256sum --check tracedelta_0.1.0_checksums.txt --ignore-missing
```

Exact names, dry-run instructions, permissions, and the conservative release procedure are documented in [`docs/release-process.md`](docs/release-process.md). v0.1.0 is experimental and deliberately makes no signing, package-manager, SBOM, or container-image promise.

## Development

For provider-neutral CI, use [`scripts/ci-compare.sh`](scripts/ci-compare.sh) to create JSON and HTML artifacts while preserving the distinction between regression exit `1` and tool/input exit `2`. The [copyable generic CI example](examples/ci/README.md) includes safe artifact-upload and troubleshooting guidance.

The reusable composite [GitHub Action](docs/github-action.md) builds the source selected by its pinned Action ref and exposes finite threshold/redaction inputs plus structured outcome/report outputs. Its [least-privilege pull-request example](examples/github-action/compare.yml) preserves reports before enforcing regressions and uses no secret or write permission.

The deterministic [example application](examples/regression-app/README.md) is exercised by public draft [PR #7](https://github.com/ArinF1/TraceDelta/pull/7): normal repository checks pass, the behavior comparison fails intentionally with the four documented findings, and all three reports are retained as one short-lived workflow artifact.

For embedding TraceDelta in Go, the [public API tour](examples/api-tour/README.md) runs a tiny loopback fixture server and demonstrates `DefaultOptions`, reader and file comparison, typed findings, and all three report writers in one inspectable standard-library example.

Common commands are exposed through the Makefile:

```bash
make help
make check
make test-race
make run-example
```

`make check` formats/verifies the project as documented in [`docs/development.md`](docs/development.md). The equivalent shell entry point is `./scripts/check.sh`.

On Windows PowerShell 5.1, use the native bounded entry point. It runs formatting verification, tests, vet, and the CLI build serially, prevents overlapping repository checks, and applies a per-command timeout while retaining the normal shared Go cache:

```powershell
.\scripts\check.ps1
```

## Architecture

TraceDelta uses a staged local pipeline:

```text
OTLP JSON -> parse -> normalize -> match -> diff -> report
```

The current slice implements only the minimum of those stages needed for deterministic span comparison. Package boundaries preserve the complete pipeline without requiring a database, network service, or web application. Read [`docs/architecture.md`](docs/architecture.md) and [ADR 0001](docs/decisions/0001-initial-architecture.md) for the design and its tradeoffs.

## Project status

TraceDelta v0.1.0 is published but remains experimental. Its documented v0.1 CLI, report, Action, and input-profile contracts are the supported release boundary; compatibility is not promised beyond that boundary or as a v1-stable API. Files under `docs/` distinguish current behavior from future work, which remains tracked rather than implied.

The canonical Go module path is `github.com/ArinF1/TraceDelta`. Security reports should be sent privately to `tracedelta.security@gmail.com` as described in the [security policy](SECURITY.md).

- [Roadmap](ROADMAP.md)
- [Product specification](docs/product-spec.md)
- [Current state](docs/current-state.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## License

TraceDelta is available under the [MIT License](LICENSE).
