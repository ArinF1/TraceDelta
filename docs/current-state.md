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
- A Windows PowerShell 5.1 check entry point runs formatting verification, tests, vet, and CLI build serially under a repository mutex; every native child is enrolled in a bounded kill-on-close Windows Job Object, and focused tests exercise success, failure, timeout cleanup, and lock contention.
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

Revalidated on 2026-07-19 after TD-029, using Go 1.26.0 and Windows PowerShell 5.1 on Windows:

```text
powershell.exe -Command 'go vet ./...'
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\scripts\check.tests.ps1
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File .\scripts\check.ps1
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

The pre-change `go test -count=1 ./...` baseline passed across all eight packages. Direct, nested PowerShell, and five concurrent warm-cache vet probes all exited `0` in approximately 0.4–4.0 seconds, with no surviving Go tool processes. A deliberately fresh disposable cache exceeded a separate 30-second diagnostic bound, confirming that repository validation should retain the normal shared Go cache.

The final focused Windows watchdog suite passed its success, nonzero-exit, pre-assignment start-gate, timed-out parent/child-tree cleanup, path-with-spaces, and concurrent-lock probes. The pre-gate design passed an eight-run probe series and 800 rapid assignment attempts, but review correctly identified a theoretical start/assignment race; the final design prevents target execution until its launcher belongs to the Job Object and waits for the job's active-process count to reach zero. The bounded Windows entry point then passed from an unrelated working directory with the normal shared cache and removed its temporary CLI executable.

Final formatting covered all 15 Go files; all eight Go packages passed uncached normal and race-enabled suites; direct vet completed with no findings; the standard-library-only module graph verified; and the CLI built successfully. The compatibility comparison returned exactly `1` with the documented four findings, while representative-fixture self-comparison returned exactly `0`. The exact nested PowerShell vet command returned `0` in 1.6 seconds. Diff whitespace, merge-marker, TODO, high-confidence secret, temporary-artifact, and intended-status checks were clean.

The recurring `Script running with cell ID ...` status was the automation layer yielding after its observation window while the shell cell remained active. Resuming the same cell with the wait operation returned the command's real completion; the status was not a `go vet` hang. Repository automation must use the serial Windows entry point and treat its final mutex, timeout, cleanup, and process-exit messages as authoritative.

## Next recommended task

**TD-006 — Match corresponding traces.** Pair normalized traces deterministically using root operation, service, kind, stable attributes, and explainable occurrence handling while surfacing added, removed, repeated, and ambiguous cases.
