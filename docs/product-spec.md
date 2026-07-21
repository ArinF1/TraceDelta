# Product specification

## Problem statement

Code review shows textual source changes, but many regressions are visible only while software runs. A small code change can add an outbound call, reorder a write before a payment succeeds, alter a database operation, change an error path, or increase latency without making that effect obvious in the Git diff.

TraceDelta compares baseline and candidate OpenTelemetry traces and reports meaningful runtime behavior differences. It is intended to complement source review, tests, and observability—not replace them.

## Target users

- Application developers reviewing behavior-sensitive pull requests.
- Maintainers of service-oriented applications who need a local, reproducible comparison.
- CI engineers who want a machine-detectable regression signal from captured test traces.
- OpenTelemetry users who can produce representative, sanitized trace exports.

The first release assumes users are comfortable running a command-line tool and deciding how representative traces are collected.

## Primary use cases

1. Compare a known-good test run with a candidate run before merging a pull request.
2. Identify added or removed operations and changed status/error behavior.
3. Detect meaningful latency changes while ignoring configured run-to-run noise.
4. Produce deterministic reports for a human reviewer and CI automation.
5. Investigate a behavioral change locally without uploading traces to a hosted service.

## Non-goals

Version 0.1 does not aim to:

- collect or store production telemetry;
- replace an OpenTelemetry collector or observability backend;
- prove that a candidate is correct or safe;
- infer source-code causality without evidence;
- compare metrics or logs;
- provide a hosted dashboard, database, or web application;
- automatically execute the baseline and candidate applications;
- support every exporter and semantic-convention variant immediately; or
- use probabilistic or AI analysis as a substitute for deterministic rules.

## Terminology

- **Baseline**: the reference trace export, normally produced by the target branch or known-good build.
- **Candidate**: the trace export produced by the proposed build.
- **Trace**: a tree or graph of related spans representing one distributed operation.
- **Span**: one timed operation with a name, kind, status, attributes, and relationship context.
- **Normalization**: removal or canonicalization of nondeterministic values so equivalent behavior can compare equal.
- **Matching**: deciding which baseline and candidate traces/spans represent the same logical operation.
- **Change/finding**: an evidence-backed behavioral difference emitted by a diff rule.
- **Meaningful difference**: a supported change that survives configured tolerance and policy.
- **Regression threshold**: a configured condition that controls whether a result should fail CI.

## v0.1 release contract

Version 0.1 is complete only when all of the following are demonstrated:

1. The CLI accepts baseline and candidate files containing either one OTLP/HTTP JSON `ExportTraceServiceRequest` object or a trace-only OTLP File Exporter JSON Lines stream of `TracesData` objects. The documented profile supports multiple resource/scope groups and traces, nested OTLP attribute values, events, and links without claiming universal vendor-exporter compatibility.
2. Generated trace/span IDs, absolute timestamps, input order, and configured duration noise do not make equivalent behavior compare differently.
3. Trace and span correspondence is deterministic, one-to-one, structure-aware, and explicit about unresolved ambiguity.
4. The supported regression set is deliberately small: added spans, removed spans, error-state changes derived from span status and the safe `error.type` attribute, and candidate latency increases that meet configurable relative and absolute thresholds. Error messages and exception stack traces are not rendered as evidence.
5. Terminal, schema-versioned JSON, and self-contained offline HTML reports contain the same ordered findings and result policy.
6. Common credential and personal-data attributes are redacted before match evidence or findings are created; callers can add denylisted attribute keys; report evidence comes only from an explicit safe set.
7. Process exit codes remain `0` for a passing comparison, `1` for a completed comparison with policy-detected regressions, and `2` for invalid invocation, unsupported input, or tool failure.
8. A reusable, least-privilege GitHub Action compares caller-supplied trace artifacts without collecting telemetry or requiring a hosted TraceDelta service.
9. A synthetic example application and deliberately regressed public pull request demonstrate the complete workflow and every v0.1 finding category.
10. Unit, integration, race-enabled, and focused fuzz tests cover the release contract where each test type provides distinct evidence.
11. Tagged releases publish versioned binaries for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 with SHA-256 checksums.
12. The README links a reproducible 45–60 second demonstration from trace inputs through the failing comparison and reports.

The current repository slice implements the bounded v0.1 OTLP/HTTP JSON and trace-only File Exporter JSONL profile, early built-in/caller-configured attribute redaction, deterministic trace-preserving normalization and semantic trace/span matching, the complete finding set, equivalent terminal/schema-versioned JSON/standalone HTML reports, and a reusable least-privilege composite GitHub Action. The deliberately regressed example, release artifacts, and demonstration remain open in [`tasks.md`](tasks.md).

The exact boundary and its rationale are recorded in [ADR 0002](decisions/0002-v0.1-release-boundary.md).

The input profile follows the official [OTLP JSON encoding](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding) and [OpenTelemetry Protocol File Exporter](https://opentelemetry.io/docs/specs/otel/protocol/file-exporter/) specifications. File-exporter support is trace-only in v0.1; metrics and logs remain out of scope.

## Future possibilities

After v0.1 semantics are dependable, possible work includes database statement-shape analysis, service-call and relationship rules, broader semantic-convention and exporter coverage, source attribution from explicit telemetry metadata, comparison across repeated runs, richer confidence explanations, additional CI providers, and a carefully constrained rule-extension mechanism. These are possibilities, not current commitments or capabilities.

## Success criteria

TraceDelta v0.1 succeeds when:

- a developer can compare representative baseline/candidate exports locally with one documented command;
- identical behavior with different generated IDs and timestamps produces no finding;
- each supported finding is deterministic, explains its before/after evidence, and is backed by tests;
- unsupported or ambiguous input is visible rather than silently misclassified;
- sensitive trace data never needs to leave the user's machine;
- automation can distinguish detected regressions from tool/input errors;
- users can install a tagged binary or pin the Action to a versioned ref;
- the deliberately regressed example pull request reproduces the documented reports without secrets or production traces; and
- repository documentation accurately separates current behavior from planned behavior.

Success is not measured by invented adoption, benchmark, or detection-rate claims. Those require real evidence that the project does not yet have.

## Example v0.1 user journey

This journey describes the intended completed v0.1 workflow; the current implementation supports only the subset listed in [`current-state.md`](current-state.md).

1. A developer runs the same representative integration scenario against the target branch and the pull-request branch.
2. Their OpenTelemetry setup writes sanitized OTLP JSON exports to local files.
3. They run `tracedelta compare --baseline baseline.json --candidate candidate.json`.
4. TraceDelta validates and normalizes both exports, matches corresponding operations, and applies configured rules.
5. The terminal report shows, for example, one added inventory call and a checkout status regression with before/after evidence.
6. The command exits `1`, so the developer investigates before opening or updating the pull request.
7. If the behavior is intentional, the team updates its explicit policy or fixtures in a reviewed change; TraceDelta does not silently learn or waive it.
