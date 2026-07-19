# Current state

Last updated: 2026-07-18

## Current version

`v0.1.0-dev` — initial development foundation. No release has been published and no compatibility guarantee is implied yet.

The repository uses Go 1.26, the newest stable Go version available when the project was initialized. Its canonical module path is `github.com/ArinF1/TraceDelta`.

## What currently works

- A local `tracedelta compare` command accepts required baseline and candidate fixture paths.
- A strongly typed parser reads one documented OTLP JSON resource/scope/span subset, accepts canonical numeric enums and exact 64-bit string/number forms, preserves primitive attribute types plus resource/scope context, and retains the original symbolic-enum fixtures as compatibility cases.
- Safe unknown OTLP message fields are ignored; malformed required span fields, zero IDs, invalid primitive values, and recognized unsupported events/links/nested attribute forms return contextual errors.
- Minimal normalization removes generated IDs and absolute timestamps before the current flat span matcher runs.
- The initial comparison detects added spans, removed spans, status changes, and duration increases that meet a configurable percentage threshold.
- The text reporter produces deterministically ordered summaries and findings.
- Process outcomes distinguish no differences (`0`), behavioral differences (`1`), and invocation/input failures (`2`).
- Synthetic fixtures demonstrate one unchanged span, one added span, one removed span, one status change, and one meaningful duration increase.
- Unit tests exercise parsing, supported diff behavior, ordering, and CLI exit mapping.
- Project documentation, local check entry points, and GitHub Actions CI configuration establish a maintainable repository baseline.
- The security policy publishes the dedicated monitored contact `tracedelta.security@gmail.com` for private vulnerability reports.
- The canonical GitHub repository is public, its `main` CI workflow is passing, and an active `main-protection` branch ruleset is configured.

## Intentionally not implemented

- Complete OTLP JSON compatibility, including span events, links, nested array/key-value-list attributes, JSON Lines export streams, and a broader producer corpus.
- General normalization of trace IDs, span IDs, timestamps, attributes, or duration buckets.
- Cross-run trace matching and structural/semantic span matching.
- Changed service-call, error-attribute, database-shape, or relationship rules.
- JSON and standalone HTML reports.
- `--output`, configuration files, absolute latency tolerances, or per-rule regression policy.
- GitHub pull-request comments/checks or reusable action packaging.
- Remote storage, a database, a web application, telemetry collection, or release publishing.

## Known limitations

- The parser supports a tested OTLP JSON subset rather than arbitrary collector/file-exporter output; each input is exactly one JSON object.
- Empty envelopes and omitted resource/scope context are accepted, so spans without `service.name` currently compare under an empty service key.
- Non-empty span events/links and nested or case-less `AnyValue` attributes are rejected rather than silently discarded.
- The current exact span key assumes unambiguous fixtures and does not resolve repeated operations semantically.
- Percentage-only latency thresholds can overstate changes in very short spans.
- Every supported detected difference currently produces exit code `1`; severities and selective regression gates are not available.
- Text is the only report format, and output is standard output only.
- Attribute filtering and redaction are documented requirements but are not implemented; typed resource, scope, and span attributes remain in memory, so users must provide sanitized input.
- Resource-exhaustion bounds for very large or adversarial inputs are not yet characterized.

## Important architecture facts

- Processing is local and deterministic; the CLI performs no network calls.
- The intended pipeline is parse → normalize → match traces → match spans → diff → report.
- Wire-format parsing, domain comparison, reporting, and process exit behavior have separate package responsibilities.
- OTLP numeric kind/status enums are normalized to stable domain names; primitive attribute values retain their string, Boolean, integer, double, or bytes type.
- Generated trace/span IDs are never intended to serve as cross-run behavioral identity.
- Comparison findings are domain results, not Go errors; the CLI maps them to policy exit codes.
- All reporters will consume one ordered comparison-result model.
- No database, web app, frontend framework, or external service is part of v0.1.
- ADR 0001 records the architectural rationale.

## Latest validation record

Revalidated on 2026-07-18 after TD-004, using Go 1.26.0 on Windows:

```bash
gofmt -l <all Go files>
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build -trimpath -o <temporary binary> ./cmd/tracedelta
go mod verify
<temporary binary> compare --baseline testdata/baseline.json --candidate testdata/candidate.json
<temporary binary> compare --baseline testdata/otlp-representative.json --candidate testdata/otlp-representative.json
git diff --check
```

The pre-change `go test -count=1 ./...` baseline and `go vet ./...` passed. The first sandboxed test attempt could not access the Windows Go build cache; rerunning with that cache available passed and showed no repository failure.

After TD-004, formatting was clean; all eight Go packages loaded successfully; normal and race-enabled suites passed; vet was clean; the standard-library-only module graph verified; and the CLI built outside the repository. The compatibility comparison produced the documented four findings and returned exactly `1`; comparing the representative canonical OTLP fixture with itself produced no findings and returned exactly `0`. `git diff --check` was clean. The repository's Bash wrapper was not rerun because its previously recorded Windows `find` incompatibility remains; every underlying required check was run directly.

Windows agent note: do not launch multiple elevated Go validation commands in one parallel tool batch. In this environment the command wrapper can remain waiting after `go vet` has exited. A process-tree audit showed no surviving `go.exe`, `vet.exe`, compiler, or linker. Running `go vet ./...` and `go test -count=1 ./...` serially with a temporary workspace-local `GOCACHE` completed normally; the temporary cache was then removed. This is a command-runner/cache interaction, not a repository test or vet failure.

## Next recommended task

**TD-005 — Implement deterministic normalization.** Canonicalize IDs, timestamps, ordering, durations, and selected typed attributes without mutating parsed input, while preserving parent relationships and deterministic repeated-run output.
