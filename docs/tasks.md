# Prioritized tasks

This is the execution backlog and status record. Work on one task at a time, normally the first unblocked task in **Now**. Reprioritize explicitly; do not silently expand a task. A task moves to **Completed** only after its acceptance criteria are demonstrated.

## Now

### TD-005 — Implement deterministic normalization

- **Description:** Canonicalize nondeterministic IDs, timestamps, ordering, durations, and selected attributes without mutating parsed input.
- **Acceptance criteria:** Equivalent traces with different generated IDs/timestamps normalize identically; parent relationships survive ID removal; duration tolerance is typed/configurable; repeated runs produce byte-equivalent normalized fixtures; boundary cases have tests.
- **Relevant files:** `internal/normalize/`, `internal/model/`, `testdata/`, `docs/architecture.md`
- **Dependencies:** TD-004

## Next

### TD-006 — Match corresponding traces

- **Description:** Pair baseline and candidate traces using normalized root operation, service, kind, route, stable attributes, and deterministic occurrence handling.
- **Acceptance criteria:** Unambiguous trace pairs, added/removed traces, repeated trace operations, reordered input, and ambiguous candidates are tested; raw generated IDs are not cross-run identity; every match records explainable evidence.
- **Relevant files:** `internal/match/`, `internal/model/`, `docs/trace-matching.md`, `testdata/`
- **Dependencies:** TD-005

### TD-007 — Match spans structurally and semantically

- **Description:** Pair spans inside matched traces using parent context, service/name/kind, semantic attributes, and deterministic sibling occurrence order.
- **Acceptance criteria:** Repeated span names under different parents pair correctly; no span is reused; reordering does not change results; ambiguity is surfaced; added/removed spans remain correct; tests cover relationship changes.
- **Relevant files:** `internal/match/`, `internal/model/`, `internal/diff/`, `docs/trace-matching.md`
- **Dependencies:** TD-006

### TD-008 — Add service, status, error, and relationship rules

- **Description:** Produce typed semantic findings for changed service calls, span status/error attributes, and parent-child relationships.
- **Acceptance criteria:** Each rule has finding and non-finding tests; evidence shows safe before/after values; findings have stable kinds/order; relationship changes do not become misleading add/remove pairs; current status behavior remains compatible or is documented.
- **Relevant files:** `internal/diff/`, `internal/model/`, `internal/report/`, `docs/product-spec.md`
- **Dependencies:** TD-007

### TD-009 — Compare database operation shapes safely

- **Description:** Detect meaningful database system/operation/statement-shape changes without reporting bound values or sensitive literals.
- **Acceptance criteria:** System, operation, table/collection, and predicate-shape cases are tested; literal-only changes do not produce leaked values; unsupported statement forms fail closed or emit an explicit limitation; privacy documentation matches behavior.
- **Relevant files:** `internal/diff/`, `internal/normalize/`, `internal/model/`, `docs/privacy-and-security.md`
- **Dependencies:** TD-005, TD-007, TD-008

### TD-010 — Strengthen latency comparison policy

- **Description:** Combine relative and absolute duration tolerances and define zero/short-span behavior.
- **Acceptance criteria:** Boundary equality, decreases, zero baselines, short spans, and large regressions are tested; finding evidence includes before/after durations and applicable threshold; defaults and CLI syntax are documented.
- **Relevant files:** `internal/diff/`, `cmd/tracedelta/`, `pkg/tracedelta/`, `docs/cli-spec.md`
- **Dependencies:** TD-005

### TD-011 — Add versioned JSON reporting

- **Description:** Render the shared comparison result as deterministic machine-readable JSON.
- **Acceptance criteria:** `--format json` works; schema version, input metadata, summary, ordered findings, evidence, and result policy are documented; golden/schema tests verify stable encoding and escaping; exit codes match text output.
- **Relevant files:** `internal/report/`, `cmd/tracedelta/`, `docs/cli-spec.md`, `docs/`
- **Dependencies:** TD-008, TD-010

### TD-012 — Add a standalone HTML report

- **Description:** Produce a basic self-contained HTML report from the same comparison result.
- **Acceptance criteria:** `--format html` works without external assets; trace-derived content is HTML-escaped; ordered findings/counts match text and JSON; the report opens offline; representative output is visually reviewed and tested.
- **Relevant files:** `internal/report/`, `cmd/tracedelta/`, `docs/cli-spec.md`, `examples/`
- **Dependencies:** TD-011

### TD-013 — Add typed configuration and output-file policy

- **Description:** Support explicit configuration for tolerances, filters, rules, and output while preserving safe defaults.
- **Acceptance criteria:** Precedence between defaults, config, and flags is documented/tested; unknown keys fail helpfully; configuration is passed without globals; `--output` has safe overwrite/stdout behavior; an effective configuration can be diagnosed without exposing secrets.
- **Relevant files:** `pkg/tracedelta/`, `cmd/tracedelta/`, `internal/`, `docs/cli-spec.md`, `examples/`
- **Dependencies:** TD-010, TD-011, TD-012

### TD-014 — Document and test generic CI usage

- **Description:** Define a headless artifact-to-comparison workflow that treats exit codes and reports correctly in generic CI.
- **Acceptance criteria:** A copyable example distinguishes regression code `1` from tool failure `2`; it preserves selected reports as artifacts without logging raw traces; a smoke test exercises the documented invocation; troubleshooting covers missing/invalid artifacts.
- **Relevant files:** `examples/`, `docs/cli-spec.md`, `docs/privacy-and-security.md`, `scripts/`
- **Dependencies:** TD-011, TD-013

### TD-015 — Provide GitHub Actions integration

- **Description:** Add a least-privilege reusable/example workflow for comparing baseline and candidate trace artifacts in pull requests.
- **Acceptance criteria:** The workflow runs from a documented sample, uploads or summarizes safe reports, distinguishes result/error exit codes, uses pinned major official actions, works safely for fork pull requests, and requests no unnecessary write permission or secret.
- **Relevant files:** `.github/workflows/`, `examples/`, `README.md`, `docs/privacy-and-security.md`
- **Dependencies:** TD-014

## Later

### TD-016 — Enforce attribute filtering and redaction

- **Description:** Implement allowlists, denylist precedence, and early deterministic redaction across every report/error path.
- **Acceptance criteria:** Sensitive nested values cannot reach text/JSON/HTML/error output; denylist precedence is tested; defaults cover common credential fields; documentation clearly states residual risk.
- **Relevant files:** `internal/normalize/`, `internal/report/`, `docs/privacy-and-security.md`
- **Dependencies:** TD-004, TD-013

### TD-017 — Improve ambiguity diagnostics

- **Description:** Make uncertain trace/span matches actionable without forcing misleading findings.
- **Acceptance criteria:** Diagnostics list non-sensitive competing signals; policy for ambiguous items is configurable and tested; deterministic behavior holds across reordered input; CLI and JSON distinguish ambiguity from parse failure.
- **Relevant files:** `internal/match/`, `internal/report/`, `docs/trace-matching.md`
- **Dependencies:** TD-006, TD-007, TD-011, TD-013

### TD-018 — Build an OTLP compatibility corpus

- **Description:** Add sanitized synthetic fixtures representing common exporter encodings and semantic-convention versions.
- **Acceptance criteria:** Each fixture names its producer shape/version without real telemetry; a compatibility document maps supported constructs to tests; unsupported constructs have explicit diagnostics; fixture licensing/provenance is clear.
- **Relevant files:** `testdata/`, `internal/otlp/`, `docs/`
- **Dependencies:** TD-004, TD-016

### TD-019 — Bound large-input resource use

- **Description:** Measure and constrain parser/matcher behavior for large or adversarial local trace sets.
- **Acceptance criteria:** Benchmarks use synthetic data; configurable size/depth/span limits fail with actionable errors; matching avoids accidental quadratic behavior for representative inputs; documented bounds are evidence-based.
- **Relevant files:** `internal/otlp/`, `internal/match/`, `pkg/tracedelta/`, `docs/privacy-and-security.md`
- **Dependencies:** TD-006, TD-007, TD-018

### TD-020 — Research evidence-based source attribution

- **Description:** Explore linking findings to commit/source metadata explicitly present in telemetry, without guessing causality.
- **Acceptance criteria:** A design note defines trusted inputs, uncertainty, privacy impact, and non-goals; a prototype remains opt-in/experimental; no public capability claim is made without tests and representative evidence.
- **Relevant files:** `docs/decisions/`, `docs/product-spec.md`, future experimental package
- **Dependencies:** TD-008, TD-016

## Completed

### TD-001 — Establish repository and project memory

- **Description:** Create the open-source repository structure, community documentation, architecture/product contracts, ADR process, and persistent agent instructions.
- **Acceptance criteria:** Required files are useful rather than empty; project state, task backlog, session log, roadmap, contribution/security policies, and publication placeholders are explicit; a new session can identify current behavior and next work from files alone.
- **Relevant files:** `AGENTS.md`, `README.md`, `docs/`, root community files, `.github/`
- **Dependencies:** none

### TD-002 — Implement the comparison vertical slice

- **Description:** Build a local CLI that parses two simplified OTLP-compatible JSON fixtures and reports added/removed spans, status changes, and thresholded duration increases.
- **Acceptance criteria:** Required arguments and threshold/format behavior work; malformed input is contextual; output ordering is deterministic; process results map to exit codes `0`, `1`, and `2`; the sample demonstrates every required initial change.
- **Relevant files:** `cmd/tracedelta/`, `internal/`, `pkg/tracedelta/`, `testdata/`
- **Dependencies:** TD-001

### TD-003 — Add baseline verification workflow

- **Description:** Cover the slice with tests and provide consistent formatting, test, vet, build, and example commands locally and in CI.
- **Acceptance criteria:** Parser/diff/order/exit behavior tests pass; `gofmt`, `go test ./...`, `go vet ./...`, and CLI build are checked; the example's expected difference exit is verified; CI has no release publishing.
- **Relevant files:** Go test files, `Makefile`, `scripts/check.sh`, `.github/workflows/ci.yml`
- **Dependencies:** TD-002

### TD-004 — Expand OTLP JSON parsing

- **Description:** Replace the simplified-only parser boundary with a strongly typed, documented subset of real OTLP JSON resource spans, scope spans, spans, statuses, and typed attributes while retaining the existing fixture as a compatibility case.
- **Acceptance criteria:** Representative synthetic OTLP JSON parses into domain values; unknown safe fields are tolerated; malformed required fields and unsupported value forms produce contextual errors; parser tests cover empty, malformed, and nested inputs; supported/unsupported schema details are documented.
- **Outcome:** Added canonical numeric enum support, exact string/number decoding for 64-bit values, primitive type-preserving attributes, resource/scope context, safe unknown-field tolerance, nonzero ID validation, a representative fixture, and contextual rejection of events, links, arrays, key-value lists, and invalid values. The original symbolic-enum fixtures remain compatible.
- **Relevant files:** `internal/otlp/`, `internal/model/`, `internal/normalize/`, `testdata/`, `README.md`, `docs/architecture.md`, `docs/cli-spec.md`, `docs/current-state.md`
- **Dependencies:** TD-002, TD-003

### TD-026 — Configure canonical GitHub repository metadata

- **Description:** Replace the temporary repository, Go module, and code-owner values with the canonical private GitHub repository metadata while tracking the then-unresolved security contact separately.
- **Acceptance criteria:** Repository links use `ArinF1/TraceDelta`; the Go module and imports use `github.com/ArinF1/TraceDelta`; `CODEOWNERS` assigns `@ArinF1`; the security contact is explicitly handed off to TD-027; tests, vet, build, commit, and push succeed.
- **Relevant files:** `go.mod`, Go imports, `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `.github/`, `docs/current-state.md`, `docs/session-log.md`
- **Dependencies:** TD-001, TD-003

### TD-027 — Configure the monitored security contact

- **Description:** Replace the pre-publication security-address placeholder with the dedicated monitored TraceDelta security mailbox.
- **Acceptance criteria:** Active security and privacy documentation consistently names `tracedelta.security@gmail.com`; no active publication warning references the old placeholder; historical append-only records remain accurate; repository checks, commit, and push succeed.
- **Relevant files:** `README.md`, `SECURITY.md`, `CHANGELOG.md`, `docs/privacy-and-security.md`, `docs/current-state.md`, `docs/session-log.md`
- **Dependencies:** TD-001, TD-026

### TD-028 — Publish and harden the GitHub repository

- **Description:** Publish the canonical repository and configure the minimum public-project security, CI, metadata, and default-branch controls.
- **Acceptance criteria:** GitHub publicly exposes `ArinF1/TraceDelta`; the repository description is accurate; CI succeeds on `main`; an active `main-protection` ruleset requires pull requests and CI while blocking deletion and force pushes; private vulnerability reporting, Dependabot security features, secret scanning, and push protection are enabled; project memory records the publication state without inventing a release.
- **Relevant files:** GitHub repository settings, `.github/workflows/ci.yml`, `CHANGELOG.md`, `docs/current-state.md`, `docs/session-log.md`
- **Dependencies:** TD-026, TD-027

## Explicitly out of scope for v0.1

### TD-021 — Hosted trace database or service

- **Description:** Operating a multi-user backend, database, or managed trace-storage service is deferred beyond v0.1.
- **Acceptance criteria:** No database/runtime service dependency enters v0.1; any future proposal requires a threat model, operating model, and ADR based on demonstrated local-tool need.
- **Relevant files:** `docs/product-spec.md`, `docs/architecture.md`, `ROADMAP.md`
- **Dependencies:** Reliable local v0.1 semantics and separate approval

### TD-022 — Web dashboard or frontend application

- **Description:** An interactive hosted/local web UI is deferred; v0.1 reports remain terminal, JSON, and standalone HTML artifacts.
- **Acceptance criteria:** No frontend framework/server is added for v0.1; a future proposal must identify a workflow that standalone reports cannot satisfy.
- **Relevant files:** `docs/product-spec.md`, `ROADMAP.md`
- **Dependencies:** Phase 3 report completion and separate approval

### TD-023 — Application execution and telemetry collection

- **Description:** Automatically building/running applications or operating an OpenTelemetry collector is outside the comparison tool's v0.1 boundary.
- **Acceptance criteria:** v0.1 accepts user-supplied exports only; integration docs state the boundary; any future runner design treats untrusted code execution as a separate security product surface.
- **Relevant files:** `docs/product-spec.md`, `docs/privacy-and-security.md`, `examples/`
- **Dependencies:** Separate product/security decision

### TD-024 — Probabilistic or AI-generated findings

- **Description:** Non-deterministic inference is deferred until deterministic semantic rules and evidence contracts are mature.
- **Acceptance criteria:** v0.1 findings remain deterministic and rule/evidence based; future experiments cannot affect default exit policy without an explicit ADR and validation plan.
- **Relevant files:** `docs/architecture.md`, `docs/product-spec.md`, future ADR
- **Dependencies:** Phase 2 completion and separate approval

### TD-025 — Metrics and log comparison

- **Description:** Comparing OpenTelemetry metrics or logs is outside the v0.1 trace-focused scope.
- **Acceptance criteria:** No metrics/log parser or diff rules are added in v0.1; future scope requires separate product requirements and domain models rather than overloading trace concepts.
- **Relevant files:** `docs/product-spec.md`, `ROADMAP.md`
- **Dependencies:** Trace v0.1 completion and separate approval
