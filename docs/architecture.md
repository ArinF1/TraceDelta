# Architecture

## Purpose and current boundary

TraceDelta is a local command-line program that compares two OpenTelemetry trace exports and produces an ordered set of behavioral findings. The intended pipeline is parse, normalize, match, diff, and report. The current vertical slice implements the smallest useful form of that pipeline for a documented OTLP JSON subset and retains the original synthetic inputs as compatibility fixtures; sections below label behavior that is still proposed.

TraceDelta currently owns:

- reading two explicitly named local files;
- validating and translating supported JSON into domain values;
- comparing supported spans under a configured duration threshold;
- rendering results; and
- selecting a documented process exit code.

TraceDelta does not collect telemetry, run applications, call an OpenTelemetry backend, access source hosting, or send data over the network. A database and hosted control plane are outside the v0.1 boundary.

## Data flow

```mermaid
flowchart LR
    B["Baseline OTLP JSON"] --> PB["Parse and validate"]
    C["Candidate OTLP JSON"] --> PC["Parse and validate"]
    PB --> N["Normalize nondeterministic values"]
    PC --> N
    N --> T["Match traces"]
    T --> S["Match spans"]
    S --> D["Apply semantic diff rules"]
    D --> R["Ordered comparison result"]
    R --> TX["Text report"]
    R -. planned .-> JS["JSON report"]
    R -. planned .-> HT["Standalone HTML report"]
    R --> E["Exit policy"]
```

Normalization now produces a deterministic trace-preserving projection, while trace matching remains intentionally minimal and only text reporting is available. The package boundaries reserve the full flow so each stage can become more capable without coupling parsing to presentation.

## Package responsibilities

- `cmd/tracedelta` owns process concerns: command/flag parsing, standard streams, and exit codes. It should contain no comparison rules.
- `pkg/tracedelta` is the public orchestration boundary. It coordinates a comparison without exposing internal wire-format details.
- `internal/model` contains typed domain values such as traces, spans, primitive attribute values, resource/scope context, status, comparison results, and changes.
- `internal/otlp` decodes the supported OTLP JSON subset, validates required span fields, and translates wire values into domain values.
- `internal/normalize` removes raw identifiers and absolute clocks, resolves parent relationships, canonicalizes trace/span order and selected typed attributes, and applies an explicit duration bucket before matching.
- `internal/match` establishes one-to-one trace and span correspondence and explains ambiguity. Semantic matching is planned work.
- `internal/diff` produces typed findings from matched, added, and removed domain values. It owns threshold semantics, not formatting.
- `internal/report` turns a comparison result into deterministic output. Text is current; JSON and HTML are planned.

Internal packages are deliberate. The wire model and early matching rules will evolve during v0.1 and should not become accidental public APIs. The public package should grow only around demonstrated embedding use cases.

## Core domain types

The architecture revolves around a small set of concepts:

- **Trace**: a related span tree plus resource context, especially service identity.
- **Span**: a named operation with kind, status, duration, parent relationship, resource/scope context, and typed primitive attributes relevant to behavior.
- **Match**: an explicit baseline/candidate pair, or an unmatched item on either side, with the evidence used to decide it.
- **Change**: a stable kind such as added, removed, status changed, or duration increased, plus before/after evidence.
- **Comparison result**: ordered changes, summary counts, input labels, and enough policy information for every reporter and the exit decision.

Trace and span identifiers from an export are input correlation data, not stable behavioral identity. Reporters consume comparison results; they do not re-read spans or recompute rules.

## Parse stage

Parsing is strongly typed and fail-fast for unsupported or malformed input. The parser traverses resource spans, scope spans, and spans; preserves resource/scope context plus primitive string, Boolean, signed-integer, double, and bytes attribute types; accepts canonical numeric kind/status enums; and retains exact timestamp values. The original symbolic enum names remain a documented fixture-compatibility extension. Errors identify the input role (baseline or candidate), file operation, and failing field where practical.

OTLP JSON requires receivers to ignore unknown message fields, so safe unknown fields are tolerated at every decoded level. Required span IDs, names, and timestamps are validated rather than replaced with zero values, and trace/span IDs must have the correct nonzero hexadecimal form. Non-empty events and links, nested array/key-value-list attributes, and multiple JSON/JSONL records remain explicit subset errors. The exact supported and unsupported forms are documented under `testdata/`; broader producer compatibility remains planned.

## Normalization stage

Normalization makes supported semantically equivalent runs comparable without changing the parsed snapshot. The current policy:

- resolves each in-trace parent ID before removing all raw trace, span, and parent identifiers;
- preserves parent state as root, a canonical local parent index, or an external parent missing from a partial export;
- rejects self-parenting and parent cycles rather than constructing a misleading tree;
- replaces absolute start timestamps with dense trace-local order ranks, where equal timestamps share a rank;
- sorts sibling subtrees and traces by canonical structural encodings rather than JSON array order;
- floors non-negative durations to an explicitly supplied `time.Duration` bucket, with zero meaning exact duration; and
- projects `http.request.method`, `http.route`, `rpc.method`, and `rpc.service` into lexically sorted typed canonical scalar values.

The duration bucket is available through the typed Go orchestration options and defaults to zero, preserving the current CLI's exact-duration input to its separate relative regression threshold. Selected attributes form a stable matching projection only: parsed input still contains every supported attribute, and this is not filtering, redaction, or anonymization. User allowlists, denylist precedence, and early redaction remain TD-016. Repeated normalization is byte-equivalent for the tested normalized model, including non-finite doubles and bytes, and does not retain `InputOrder`, absolute clock values, or raw IDs.

## Matching stage

Matching is a separate stage because correspondence is uncertain, while a diff assumes correspondence is known. The current matcher temporarily flattens canonical normalized traces and uses the documented exact span key plus global canonical occurrence. It therefore still assumes unambiguous synthetic inputs and does not perform general trace matching or use parent structure as match evidence.

The proposed matcher uses ordered signals and deterministic tie-breaking, records ambiguity instead of guessing, and never uses raw trace/span IDs as cross-run identity. See [`trace-matching.md`](trace-matching.md) for the full proposal.

## Diff stage

Diff rules receive normalized matched or unmatched values and emit typed changes. The initial rules detect added spans, removed spans, status changes, and candidate duration increases that meet the configured relative threshold.

Rules for changed service calls, errors, database operation shapes, relationships, and richer latency policies are planned. New rules should be narrow functions until shared behavior demonstrates the need for an interface. Every rule must define stable identity, evidence, ordering, severity or policy effect, and tests for both findings and non-findings.

## Report and exit stages

Reporting is a pure projection of a comparison result. Stable ordering is part of the user-facing contract because CI output must not fluctuate between runs. Text output is implemented first. JSON will use an explicit schema version, and HTML will be a self-contained escaped document derived from the same result.

The process adapter maps a successful result with no meaningful differences to `0`, a successful result with differences to `1`, and invalid invocation/input/tool failure to `2`. Reporters do not terminate the process.

## Error handling

- Wrap errors at boundaries with action and input context.
- Preserve underlying errors for programmatic inspection where useful.
- Reject invalid flags and unsupported formats before reading files.
- Do not turn malformed telemetry into an empty successful comparison.
- Keep behavioral differences in the result path; they are not Go errors even though the CLI exits nonzero.
- Send diagnostics to standard error and comparison output to standard output.

## Extensibility approach

TraceDelta grows by adding data and rules to the existing stages, not by building a plugin framework in advance. Wire parsing remains isolated from domain models; matching remains isolated from diffing; all reporters share one result. Configuration will be a typed value passed explicitly through orchestration rather than global mutable state.

Compatibility-sensitive elements—CLI flags, exit codes, fixture/schema versions, finding kinds, and JSON fields—must be documented and tested before they are treated as stable.

## Test strategy

- Unit tests cover parsing validation, normalization invariants, matching ambiguity, each diff rule, ordering, and reporter escaping.
- Synthetic JSON fixtures cover end-to-end behavior without production or personal data.
- CLI-level tests cover arguments, streams, and exit-code mapping where practical.
- Golden files are appropriate only for deliberate report contracts and must remain reviewable.
- Fuzzing is a future addition for untrusted JSON and matcher edge cases.
- Repository validation runs formatting checks, `go test ./...`, `go vet ./...`, and a clean CLI build.

## Deliberately postponed

The v0.1 architecture deliberately avoids a database, web application, remote trace backend, telemetry collector, Kubernetes support, ClickHouse, Docker Compose, release automation, source-host API calls, AI-generated conclusions, and an extension/plugin runtime. These would add operational and security surface before local comparison semantics are reliable.
