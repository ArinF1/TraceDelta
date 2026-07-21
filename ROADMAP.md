# TraceDelta roadmap

This roadmap defines a finite v0.1 release, not an expanding sequence of desirable features. The prioritized task details live in [`docs/tasks.md`](docs/tasks.md), and the release boundary is fixed by [ADR 0002](docs/decisions/0002-v0.1-release-boundary.md).

## Already established

The finite v0.1 sequence below is complete. TraceDelta v0.1.0 is published with the documented OTLP profile, deterministic matching and bounded findings, redaction, three report formats, reusable Action, synthetic regression PR, verification matrix, five checksum-verified binaries, and a 52-second demonstration.

## Completed v0.1 release sequence

### 1. Trustworthy input and correspondence

- Accept one OTLP/HTTP JSON request object or a trace-only OTLP File Exporter JSONL stream, including multiple resource/scope groups and traces, nested attribute values, events, and links.
- Match corresponding traces and spans deterministically using normalized structure and safe semantic evidence.
- Surface unresolved ambiguity instead of using raw generated IDs or arbitrary input order.

Exit gate: reordered equivalent inputs compare identically; repeated operations are one-to-one; unsupported envelopes fail with actionable diagnostics.

### 2. Bounded regression rules and privacy

- Detect added and removed spans.
- Detect error-state changes using span status and the safe `error.type` attribute, never error messages or stack traces.
- Detect candidate latency regressions only when configurable relative and absolute thresholds are both met; default to `20%` and `10ms`.
- Redact built-in credential/personal-data keys and caller-supplied denylisted keys before matching evidence or findings are created.

Exit gate: each rule has finding, non-finding, boundary, and deterministic-order tests; redacted values cannot reach terminal, JSON, HTML, or diagnostics covered by the comparison result.

### 3. Consistent reports and CLI policy

- Render terminal, schema-versioned JSON, and standalone offline HTML from one result.
- Provide bounded flags/configuration for latency thresholds, additional redacted keys, output destination, and safe overwrite behavior.
- Preserve exit codes `0` (pass), `1` (regression), and `2` (invocation/input/tool failure).

Exit gate: all formats contain the same ordered findings and policy outcome; JSON encoding and HTML escaping have contract tests.

### 4. Reusable pull-request workflow

- Package a least-privilege reusable GitHub Action for caller-supplied baseline and candidate artifacts.
- Add a synthetic example application and a deliberately regressed public pull request that exercises every v0.1 finding category.
- Add end-to-end integration coverage and the useful race/fuzz gates around the completed workflow.

Exit gate: the example pull request distinguishes regression exit `1` from tool failure `2`, exposes no secret or production trace data, and preserves reports as reviewable artifacts.

### 5. Publish v0.1

- Publish tagged binaries for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 with SHA-256 checksums.
- Pin the reusable Action through the release tag and document installation/upgrade behavior.
- Publish a reproducible 45–60 second demonstration linked from the README.
- Complete the release checklist and update version/support documentation.

Exit gate: a clean machine can download and verify a binary, reproduce the example comparison, and follow the one-minute demonstration using the published v0.1 tag.

## Explicitly after v0.1

The following do not block v0.1:

- database statement-shape comparison;
- general service-call and parent/relationship change rules;
- exhaustive OTLP/vendor/file-export compatibility;
- hosted storage, dashboards, collectors, or automatic application execution;
- metrics or log comparison;
- pull-request comment/check APIs beyond normal Action summary/artifact behavior;
- source attribution, statistical aggregation across repeated runs, or AI-generated findings;
- a plugin/rule runtime; and
- broad performance engineering beyond safe release input bounds.

Post-v0.1 priorities will be selected from observed user needs rather than added to the first-release gate in advance.
