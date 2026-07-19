# Current state

Last updated: 2026-07-19

## Current version

`v0.1.0-dev` — initial development foundation. No release has been published and no compatibility guarantee is implied yet.

The repository uses Go 1.26, the newest stable Go version available when the project was initialized. Its canonical module path is `github.com/ArinF1/TraceDelta`.

## What currently works

- A local `tracedelta compare` command accepts required baseline and candidate fixture paths.
- A strongly typed parser reads one documented OTLP JSON resource/scope/span subset, accepts canonical numeric enums and exact 64-bit string/number forms, preserves primitive attribute types plus resource/scope context, and retains the original symbolic-enum fixtures as compatibility cases.
- Safe unknown OTLP message fields are ignored; malformed required span fields, zero IDs, invalid primitive values, and recognized unsupported events/links/nested attribute forms return contextual errors.
- Deterministic normalization preserves trace boundaries, resolves root/internal/missing-external parent state before removing raw IDs, replaces absolute timestamps with dense trace-local order, and canonically orders traces and sibling subtrees without using input-array order.
- A typed Go option can floor durations to a non-negative `time.Duration` bucket; zero is the default and preserves the CLI's existing exact-duration behavior before its separate percentage threshold.
- The stable normalization projection sorts and type-tags `http.request.method`, `http.route`, `rpc.method`, and `rpc.service`; parsed input is not mutated and retains all supported attributes.
- The initial comparison detects added spans, removed spans, status changes, and duration increases that meet a configurable percentage threshold.
- The text reporter produces deterministically ordered summaries and findings.
- Process outcomes distinguish no differences (`0`), behavioral differences (`1`), and invocation/input failures (`2`).
- Synthetic fixtures demonstrate one unchanged span, one added span, one removed span, one status change, one meaningful duration increase, and equivalent reordered runs under an explicit duration bucket.
- Unit tests exercise parsing, normalization invariants and boundaries, supported diff behavior, ordering, and CLI exit mapping.
- Project documentation, local check entry points, and GitHub Actions CI configuration establish a maintainable repository baseline.
- The security policy publishes the dedicated monitored contact `tracedelta.security@gmail.com` for private vulnerability reports.
- The canonical GitHub repository is public, its `main` CI workflow is passing, and an active `main-protection` branch ruleset is configured.

## Intentionally not implemented

- Complete OTLP JSON compatibility, including span events, links, nested array/key-value-list attributes, JSON Lines export streams, and a broader producer corpus.
- Cross-run trace matching and structural/semantic span matching.
- Changed service-call, error-attribute, database-shape, or relationship rules.
- JSON and standalone HTML reports.
- `--output`, configuration files, CLI duration-bucket syntax, absolute latency regression tolerances, or per-rule regression policy.
- GitHub pull-request comments/checks or reusable action packaging.
- Remote storage, a database, a web application, telemetry collection, or release publishing.

## Known limitations

- The parser supports a tested OTLP JSON subset rather than arbitrary collector/file-exporter output; each input is exactly one JSON object.
- Empty envelopes and omitted resource/scope context are accepted, so spans without `service.name` currently compare under an empty service key.
- Non-empty span events/links and nested or case-less `AnyValue` attributes are rejected rather than silently discarded.
- The current matcher flattens canonical normalized traces and uses exact span key plus global canonical occurrence; it does not yet use preserved trace/parent context or resolve repeated operations semantically.
- Percentage-only latency thresholds can overstate changes in very short spans.
- Every supported detected difference currently produces exit code `1`; severities and selective regression gates are not available.
- Text is the only report format, and output is standard output only.
- The fixed stable-attribute projection is not filtering or redaction; typed resource, scope, and span attributes remain in parsed memory, so users must provide sanitized input.
- Resource-exhaustion bounds for very large or adversarial inputs are not yet characterized.

## Important architecture facts

- Processing is local and deterministic; the CLI performs no network calls.
- The intended pipeline is parse → normalize → match traces → match spans → diff → report.
- Wire-format parsing, domain comparison, reporting, and process exit behavior have separate package responsibilities.
- OTLP numeric kind/status enums are normalized to stable domain names; primitive attribute values retain their string, Boolean, integer, double, or bytes type.
- Generated trace/span IDs are removed only after in-trace parent relationships are converted to canonical local references; they never serve as cross-run behavioral identity.
- Absolute start clocks and parser input order are absent from normalized values; dense relative order, structural subtree digests, and sorted typed stable attributes provide deterministic ordering.
- Comparison findings are domain results, not Go errors; the CLI maps them to policy exit codes.
- All reporters will consume one ordered comparison-result model.
- No database, web app, frontend framework, or external service is part of v0.1.
- ADR 0001 records the architectural rationale.

## Latest validation record

Revalidated on 2026-07-19 after TD-005, using Go 1.26.0 on Windows:

```bash
gofmt -l <all Go files>
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go mod verify
go build -trimpath -o <temporary binary> ./cmd/tracedelta
<temporary binary> compare --baseline testdata/baseline.json --candidate testdata/candidate.json
<temporary binary> compare --baseline testdata/otlp-representative.json --candidate testdata/otlp-representative.json
git diff --check
```

The pre-change `go test -count=1 ./...` baseline passed across all eight packages, and serial `go vet ./...` completed in 4 seconds with no findings.

After TD-005, formatting was clean; focused normalization/matching/orchestration tests passed; all eight Go packages passed both normal and race-enabled suites; vet completed in 4.6 seconds with no findings; the standard-library-only module graph verified; and the CLI built successfully. The compatibility comparison produced the documented four findings and returned exactly `1`; comparing the representative canonical OTLP fixture with itself produced no findings and returned exactly `0`. `git diff --check` was clean, and the temporary executable was removed. The repository's Bash wrapper was not rerun because its previously recorded Windows `find` incompatibility remains; every underlying required check was run directly.

Windows agent note: do not launch multiple elevated Go validation commands in one parallel tool batch. In this environment the command wrapper can remain waiting after a child Go command has exited. Running the commands serially with the normal Go cache avoided the issue in this session: both baseline and final `go vet ./...` completed normally. This is a command-runner concurrency interaction, not a repository test or vet failure.

## Next recommended task

**TD-006 — Match corresponding traces.** Pair normalized traces deterministically using root operation, service, kind, stable attributes, and explainable occurrence handling while surfacing added, removed, repeated, and ambiguous cases.
