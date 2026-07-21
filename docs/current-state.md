# Current state

Last updated: 2026-07-20

## Current version

`v0.1.0-dev` — initial development foundation. No release has been published and no compatibility guarantee is implied yet.

The repository uses Go 1.26, the newest stable Go version available when the project was initialized. Its canonical module path is `github.com/ArinF1/TraceDelta`.

## What currently works

- A local `tracedelta compare` command accepts required baseline and candidate fixture paths.
- A strongly typed parser reads one OTLP/HTTP JSON request object or a trace-only OTLP File Exporter JSONL stream, accepts multiple records/resources/scopes/traces, canonical numeric enums, and exact 64-bit string/number forms, and retains the original symbolic-enum fixtures as compatibility cases.
- Primitive and recursive array/key-value-list attribute types plus resource/scope context are preserved with a 64-level depth limit. Events and links are validated, then their unused payloads are discarded before normalization or evidence construction.
- Safe unknown OTLP message fields are ignored; malformed required fields, zero IDs, duplicate spans across records, invalid values, recognized non-trace signals, and common vendor outer envelopes return contextual errors.
- A dedicated post-parse stage deep-copies snapshots and recursively removes conservative built-in credential/personal-data keys plus caller-supplied case-insensitive exact keys before normalization; deny rules override safe evidence keys and can clear derived service identity.
- Deterministic normalization preserves trace boundaries, resolves root/internal/missing-external parent state before removing raw IDs, replaces absolute timestamps with dense trace-local order, and canonically orders traces and sibling subtrees without using input-array order.
- A typed Go option can floor durations to a non-negative `time.Duration` bucket; zero is the default and preserves exact durations before the separate combined latency thresholds.
- The stable normalization projection sorts and type-tags `http.request.method`, `http.route`, `rpc.method`, and `rpc.service`; parsed input is not mutated and retains all supported attributes.
- Deterministic trace matching groups by normalized root operation/service/kind/safe attributes, pairs unique exact structures or mutual unique structural overlaps, records non-sensitive evidence, and returns a typed ambiguity error instead of guessing.
- Parent-aware span matching pairs top-level spans and descendants by service/name/kind, safe attributes, matched-parent context, and distinguishable sibling order without reusing candidates; unique relationship changes use an explicit fallback.
- Added and removed traces feed all of their spans into the existing findings; unmatched spans inside paired traces remain added/removed, and indistinguishable siblings return a typed non-sensitive ambiguity error.
- The comparison detects added spans, removed spans, status transitions, safe string `error.type` changes between matched error spans, and duration increases that meet configurable relative and absolute thresholds together.
- Text, `tracedelta.report/v1` JSON, and standalone offline HTML reporters produce the same deterministically ordered summaries, findings, evidence, and pass/differences policy. JSON uses standard encoding; HTML uses contextual escaping, inline CSS, and no scripts/external assets.
- Process outcomes distinguish no differences (`0`), behavioral differences (`1`), and invocation/input failures (`2`).
- A reusable composite GitHub Action builds TraceDelta from its pinned Action source, accepts caller-supplied artifacts plus finite latency/redaction inputs, exposes structured outcome/report outputs, and preserves reports for completed exit `0`/`1` comparisons while failing tool/input exit `2` without partial artifacts.
- A deterministic synthetic example application emits OTLP/HTTP JSON without services or secrets. Its pull-request workflow regenerates exact base/candidate artifacts, uses the trusted base Action, and public draft PR #7 demonstrates the four v0.1 finding categories while retaining text, JSON, and HTML reports.
- Focused fuzz targets cover arbitrary parser input, trace-order-independent matcher output, and terminal/JSON/HTML reporter escaping. CI runs bounded fuzz smoke plus actual composite-Action pass, regression, and tool-error paths with cross-format outcome checks.
- A provider-neutral CI wrapper creates JSON and HTML artifacts in a fresh directory without logging report contents, preserves completed comparison exits `0`/`1`, maps missing/invalid/tool/report failures to `2`, and removes partial output; its real-binary smoke runs in Linux CI.
- Synthetic fixtures demonstrate one unchanged span, one added span, one removed span, one status change, one meaningful duration increase, and equivalent reordered runs under an explicit duration bucket.
- Unit tests exercise parsing, normalization invariants and boundaries, supported diff behavior, ordering, and CLI exit mapping.
- Project documentation, local check entry points, and GitHub Actions CI configuration establish a maintainable repository baseline.
- A Windows PowerShell 5.1 check entry point runs formatting verification, tests, vet, and CLI build serially under a repository mutex; every native child is enrolled in a bounded kill-on-close Windows Job Object, and focused tests exercise success, failure, timeout cleanup, and lock contention.
- The security policy publishes the dedicated monitored contact `tracedelta.security@gmail.com` for private vulnerability reports.
- The canonical GitHub repository is public, its `main` CI workflow is passing, and an active `main-protection` branch ruleset is configured.

## Intentionally not implemented

- Universal OTLP JSON, protobuf-binary, mixed-signal, or arbitrary vendor-exporter compatibility beyond the documented trace-only profile.
- Changed service-call, error-attribute, database-shape, or relationship rules.
- Configuration files, CLI duration-bucket syntax, custom evidence allowlists, or per-rule regression policy beyond the two latency thresholds.
- GitHub pull-request comment/check APIs beyond normal Action outputs and artifacts.
- Remote storage, a database, a web application, telemetry collection, or release publishing.
- Versioned release binaries or the 45–60 second demonstration.

## Known limitations

- The parser supports a tested trace-only profile rather than arbitrary collector/file-exporter output: one OTLP/HTTP JSON object or a JSONL stream of `TracesData` records.
- Empty envelopes and omitted resource/scope context are accepted, so spans without `service.name` currently compare under an empty service key.
- Event/link payloads are validated but deliberately not retained because v0.1 has no event/link comparison rule. Nested values are retained in parsed memory but composite values are excluded from matching evidence.
- The trace matcher deliberately rejects indistinguishable repeated traces; configurable ambiguity policy and richer diagnostics are deferred to TD-017.
- Relationship changes are deliberately preserved as span pairs but do not produce a v0.1 finding; general relationship findings remain after v0.1.
- `error.type` is trusted as an explicitly safe semantic-convention value unless callers deny it; non-string values are treated as missing, and status/exception messages are never evidence.
- Every supported detected difference currently produces exit code `1`; severities and selective regression gates are not available.
- All three formats write to standard output by default or to a new `--output` file; existing files require `--force`, and input files are never valid output targets.
- Key-based redaction is not anonymization: custom sensitive values under unknown keys and sensitive non-attribute fields remain possible, and source trace files are unchanged on disk.
- Resource-exhaustion bounds for very large or adversarial inputs are not yet characterized.
- TD-033 release automation is implemented and its five-target local and Linux CI dry runs pass, but it is not publication-complete. A manual dispatch from the unmerged foundation branch returned `HTTP 404: workflow release.yml not found on the default branch`; merging PR #6 and pushing `v0.1.0` require explicit user approval, so no tag or GitHub Release exists yet.

## Important architecture facts

- Processing is local and deterministic; the CLI performs no network calls.
- The intended pipeline is parse → normalize → match traces → match spans → diff → report.
- Wire-format parsing, domain comparison, reporting, and process exit behavior have separate package responsibilities.
- OTLP numeric kind/status enums are normalized to stable domain names; attribute values retain scalar, array, or key/value-list types in the parsed model.
- Generated trace/span IDs are removed only after in-trace parent relationships are converted to canonical local references; they never serve as cross-run behavioral identity.
- Absolute start clocks and parser input order are absent from normalized values; dense relative order, structural subtree digests, and sorted typed stable attributes provide deterministic ordering.
- Comparison findings are domain results, not Go errors; the CLI maps them to policy exit codes.
- All reporters will consume one ordered comparison-result model.
- No database, web app, frontend framework, or external service is part of v0.1.
- ADR 0001 records the architectural rationale.
- ADR 0002 fixes a finite v0.1 boundary: realistic OTLP JSON, deterministic matching, added/removed/error/latency findings, early redaction, three report formats, a reusable Action and regressed example, useful unit/integration/race/fuzz coverage, versioned binaries, and a 45–60 second demonstration.

## Latest validation record

Validated on 2026-07-21 after TD-032. The bounded fuzz smoke passed 28,643 parser executions, 4,401 matcher executions, and 2,625 reporter executions in the recorded local run. The Action entry smoke passed pass/regression/tool-error outcomes and text/JSON/HTML consistency. The repository-native Windows entry point passed formatting, all ten packages, vet, and CLI build; `go test -race -count=1 ./...` passed all ten packages. Foundation PR #6 then passed Linux CI—including bounded fuzz plus real composite-Action `0`/`1`/`2` checks—and the Windows watchdog job.

The preceding comprehensive validation was recorded on 2026-07-19 after TD-029, using Go 1.26.0 and Windows PowerShell 5.1 on Windows:

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

The final focused Windows watchdog suite passed its duplicate-PATH application resolution, success, nonzero-exit, pre-assignment start-gate, timed-out parent/child-tree cleanup, path-with-spaces, and concurrent-lock probes. The pre-gate design passed an eight-run probe series and 800 rapid assignment attempts, but review correctly identified a theoretical start/assignment race; the final design prevents target execution until its launcher belongs to the Job Object and waits for the job's active-process count to reach zero. The bounded Windows entry point then passed from an unrelated working directory with the normal shared cache and removed its temporary CLI executable.

Final formatting covered all 15 Go files; all eight Go packages passed uncached normal and race-enabled suites; direct vet completed with no findings; the standard-library-only module graph verified; and the CLI built successfully. The compatibility comparison returned exactly `1` with the documented four findings, while representative-fixture self-comparison returned exactly `0`. The exact nested PowerShell vet command returned `0` in 1.6 seconds. Diff whitespace, merge-marker, TODO, high-confidence secret, temporary-artifact, and intended-status checks were clean.

The recurring `Script running with cell ID ...` status was the automation layer yielding after its observation window while the shell cell remained active. Resuming the same cell with the wait operation returned the command's real completion; the status was not a `go vet` hang. Repository automation must use the serial Windows entry point and treat its final mutex, timeout, cleanup, and process-exit messages as authoritative.

## Next recommended task

**TD-033 — Publish versioned v0.1 binaries.** Add least-privilege tag automation, reproducible five-target builds, SHA-256 checksums, and a verified dry run before any release tag is created.
