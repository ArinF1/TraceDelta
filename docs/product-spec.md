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
4. Produce a deterministic report for a human reviewer and, later in v0.1, CI automation.
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

## v0.1 requirements

The v0.1 line should eventually:

1. Accept baseline and candidate OTLP JSON trace files.
2. Parse a documented and useful subset of OTLP JSON with actionable validation errors.
3. Normalize trace IDs, span IDs, timestamps, and duration noise using explicit policy.
4. Match corresponding traces and spans deterministically and report ambiguity.
5. Detect added and removed spans, changed service calls, status/error changes, database operation-shape changes, relationship changes, and meaningful latency increases.
6. Produce deterministic terminal, versioned JSON, and basic standalone HTML reports from one result model.
7. Return `0` for a passing comparison, `1` for policy-detected behavioral differences, and `2` for invalid invocation or input/tool failure.
8. Support typed configuration for tolerances and regression policy.
9. Operate entirely locally and be straightforward to invoke from generic CI.
10. Include a documented GitHub Actions integration path without requiring a hosted TraceDelta service.

The current repository slice implements a documented OTLP JSON subset with primitive typed attributes, deterministic trace-preserving normalization, and a subset of structural/status/duration text findings. Broader OTLP constructs, semantic matching, user-configurable filtering/redaction, and the remaining requirements stay open in [`tasks.md`](tasks.md).

## Future possibilities

After v0.1 semantics are dependable, possible work includes broader semantic-convention coverage, source attribution from explicit telemetry metadata, comparison across repeated runs, richer confidence explanations, additional CI providers, and a carefully constrained rule-extension mechanism. These are possibilities, not current commitments or capabilities.

## Success criteria

TraceDelta v0.1 succeeds when:

- a developer can compare representative baseline/candidate exports locally with one documented command;
- identical behavior with different generated IDs and timestamps produces no finding;
- each supported finding is deterministic, explains its before/after evidence, and is backed by tests;
- unsupported or ambiguous input is visible rather than silently misclassified;
- sensitive trace data never needs to leave the user's machine;
- automation can distinguish detected regressions from tool/input errors; and
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
