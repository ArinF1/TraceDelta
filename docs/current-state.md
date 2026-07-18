# Current state

Last updated: 2026-07-18

## Current version

`v0.1.0-dev` — initial development foundation. No release has been published and no compatibility guarantee is implied yet.

The repository uses Go 1.26, the newest stable Go version available when the project was initialized. Its canonical module path is `github.com/ArinF1/TraceDelta`.

## What currently works

- A local `tracedelta compare` command accepts required baseline and candidate fixture paths.
- A strongly typed parser reads the documented simplified OTLP-compatible JSON shape and returns contextual errors for malformed input.
- Minimal normalization removes generated IDs and absolute timestamps before the current flat span matcher runs.
- The initial comparison detects added spans, removed spans, status changes, and duration increases that meet a configurable percentage threshold.
- The text reporter produces deterministically ordered summaries and findings.
- Process outcomes distinguish no differences (`0`), behavioral differences (`1`), and invocation/input failures (`2`).
- Synthetic fixtures demonstrate one unchanged span, one added span, one removed span, one status change, and one meaningful duration increase.
- Unit tests exercise parsing, supported diff behavior, ordering, and CLI exit mapping.
- Project documentation, local check entry points, and GitHub Actions CI configuration establish a maintainable repository baseline.
- The security policy publishes the dedicated monitored contact `tracedelta.security@gmail.com` for private vulnerability reports.

## Intentionally not implemented

- Complete OTLP JSON resource/scope/span compatibility.
- General normalization of trace IDs, span IDs, timestamps, attributes, or duration buckets.
- Cross-run trace matching and structural/semantic span matching.
- Changed service-call, error-attribute, database-shape, or relationship rules.
- JSON and standalone HTML reports.
- `--output`, configuration files, absolute latency tolerances, or per-rule regression policy.
- GitHub pull-request comments/checks or reusable action packaging.
- Remote storage, a database, a web application, telemetry collection, or release publishing.

## Known limitations

- The current input schema is a simplified, synthetic compatibility shape rather than arbitrary collector output.
- The current exact span key assumes unambiguous fixtures and does not resolve repeated operations semantically.
- Percentage-only latency thresholds can overstate changes in very short spans.
- Every supported detected difference currently produces exit code `1`; severities and selective regression gates are not available.
- Text is the only report format, and output is standard output only.
- Attribute filtering and redaction are documented requirements but are not implemented; users must provide sanitized input.
- Resource-exhaustion bounds for very large or adversarial inputs are not yet characterized.

## Important architecture facts

- Processing is local and deterministic; the CLI performs no network calls.
- The intended pipeline is parse → normalize → match traces → match spans → diff → report.
- Wire-format parsing, domain comparison, reporting, and process exit behavior have separate package responsibilities.
- Generated trace/span IDs are never intended to serve as cross-run behavioral identity.
- Comparison findings are domain results, not Go errors; the CLI maps them to policy exit codes.
- All reporters will consume one ordered comparison-result model.
- No database, web app, frontend framework, or external service is part of v0.1.
- ADR 0001 records the architectural rationale.

## Latest validation record

Revalidated on 2026-07-18 after configuring the monitored security contact, using Go 1.26 on Windows:

```bash
gofmt -l <all Go files>
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build -trimpath -o <temporary binary> ./cmd/tracedelta
go mod verify
<temporary binary> compare --baseline testdata/baseline.json --candidate testdata/candidate.json
```

Formatting was clean; all eight Go packages loaded successfully; normal and race-enabled tests passed; vet was clean; the CLI built; the standard-library-only module graph verified; and the built example produced the documented four findings and returned exactly `1`.

`bash scripts/check.sh` was attempted, but this Windows Bash session resolved `find` incorrectly and stopped at `find: ‘gofmt’: No such file or directory`. The same formatting, test, vet, and build commands were run directly and passed. A private GitHub remote is configured; hosted CI status is not asserted by this local validation record.

## Next recommended task

**TD-004 — Expand OTLP JSON parsing beyond the simplified fixture shape.** It is the first dependency for trustworthy normalization and semantic matching. Keep the compatibility fixture small, add representative resource/scope/span forms, and preserve actionable validation errors.
