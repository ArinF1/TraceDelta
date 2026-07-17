# TraceDelta roadmap

This roadmap describes capability phases, not commitments to dates. A phase exits only when its criteria are demonstrated by tests and documentation. The prioritized execution order lives in [`docs/tasks.md`](docs/tasks.md).

## Phase 0: repository foundation

### Goals

Establish a small, credible open-source Go project and prove one complete local comparison path.

### Deliverables

- Documented product, architecture, CLI, security, and contribution contracts.
- Typed parsing of the simplified synthetic fixture format.
- Deterministic detection and text reporting for added/removed spans, status changes, and duration changes.
- Tests, local checks, and continuous integration scaffolding.

### Exit criteria

- The repository builds and all documented checks pass.
- The example comparison deterministically exits `1` and explains why.
- A new contributor can identify current behavior and the next task from repository files alone.

## Phase 1: reliable local comparison

### Goals

Accept representative OTLP JSON exports and remove nondeterministic telemetry details before comparison.

### Deliverables

- Stronger OTLP JSON parsing across resource, scope, trace, and span data.
- Configurable normalization of identifiers, timestamps, durations, and selected attributes.
- Deterministic trace and span matching with explainable fallback behavior.
- A larger synthetic compatibility fixture corpus.

### Exit criteria

- Equivalent exports with different IDs and timestamps compare as equivalent.
- Ambiguous and unmatched telemetry is reported explicitly rather than guessed silently.
- Malformed and unsupported data produces actionable errors.

## Phase 2: semantic runtime diffing

### Goals

Turn structural differences into behavior-oriented findings with stable severity and evidence.

### Deliverables

- Rules for service calls, span status, error attributes, and parent-child changes.
- Database operation-shape comparison that avoids exposing query values.
- Meaningful absolute and relative latency regression rules.
- Configurable thresholds and regression policy.

### Exit criteria

- Every supported rule has positive, negative, and deterministic-ordering tests.
- Findings identify the matched context, before/after evidence, and rule that produced them.
- Policy violations reliably control the process exit code.

## Phase 3: report formats

### Goals

Make a single comparison result useful to humans and automation without duplicating diff logic.

### Deliverables

- A versioned machine-readable JSON report.
- A self-contained HTML report with no external runtime dependency.
- Output-file handling with safe overwrite behavior.
- Consistency tests across text, JSON, and HTML renderers.

### Exit criteria

- All formats render the same ordered findings and summary counts.
- The JSON schema is documented and compatibility policy is explicit.
- The HTML report opens offline and safely escapes trace-derived content.

## Phase 4: CI and GitHub integration

### Goals

Make TraceDelta predictable in generic CI and useful in GitHub pull-request workflows.

### Deliverables

- Documented headless invocation, artifact, and exit-code patterns.
- A reusable GitHub Actions example that compares supplied trace artifacts.
- Optional pull-request summary/check integration designed with least-privilege permissions.
- Troubleshooting guidance for fixture collection and failed comparisons.

### Exit criteria

- A sample repository can run TraceDelta from clean baseline and candidate artifacts.
- Forked pull requests do not receive unnecessary write permissions or secrets.
- The integration distinguishes behavioral regressions from tool/input failures.

## Phase 5: broader OpenTelemetry support

### Goals

Handle a wider set of valid telemetry producers without weakening deterministic behavior.

### Deliverables

- Compatibility coverage for common OTLP JSON exporter variants.
- Richer HTTP, RPC, messaging, and database semantic-convention handling.
- Multi-trace and repeated-operation matching improvements.
- Bounded-resource behavior for larger trace sets.

### Exit criteria

- A documented compatibility matrix is backed by sanitized fixtures.
- Unsupported constructs degrade with explicit diagnostics.
- Representative large local inputs complete within documented resource bounds.

## Phase 6: source attribution and advanced analysis

### Goals

Connect runtime findings to useful development context after the local semantic engine is trustworthy.

### Deliverables

- Optional source/commit attribution based on explicit telemetry metadata.
- Cross-run aggregation and confidence explanations.
- An extension mechanism for narrowly scoped analysis rules.
- Research prototypes for advanced analysis, kept outside the stable contract until validated.

### Exit criteria

- Attribution is evidence-based and communicates uncertainty.
- Extensions cannot bypass redaction or deterministic ordering contracts.
- Advanced findings demonstrably improve review usefulness without requiring a hosted service.
