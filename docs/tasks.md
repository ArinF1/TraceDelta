# Prioritized tasks

This is the execution backlog and status record. Work on one task at a time, normally the first unblocked task in **Now**. Reprioritize explicitly; do not silently expand a task. A task moves to **Completed** only after its acceptance criteria are demonstrated.

## Now

### TD-031 — Add the deliberately regressed example pull request

- **Description:** Add a small synthetic example application and a public pull request whose intentional regression exercises added/removed spans, an error change, and a latency regression through the reusable Action.
- **Acceptance criteria:** The application emits deterministic synthetic OTLP JSON without external services or secrets; baseline and candidate collection is reproducible; the linked pull request fails for the documented reasons and preserves all three report formats; maintainers can reset/replay the demonstration without rewriting history.
- **Relevant files:** `examples/`, `.github/workflows/`, `README.md`
- **Dependencies:** TD-015

## Next

The following sequence is the complete remaining v0.1 gate after the current task. Execute one task at a time in this order unless a dependency or verified finding requires an explicit reprioritization.

### TD-032 — Complete the v0.1 verification matrix

- **Description:** Add the integration, race, and focused fuzz coverage needed to make the completed release workflow trustworthy without duplicating unit-test coverage.
- **Acceptance criteria:** End-to-end CLI and Action smoke tests cover pass/regression/tool-error outcomes and report consistency; `go test -race ./...` passes; bounded fuzz smoke runs cover parser input, matcher determinism, and reporter escaping; the release checklist names exact commands and expected outcomes.
- **Relevant files:** Go tests, `.github/workflows/`, `scripts/`, `docs/development.md`, `docs/release-process.md`
- **Dependencies:** TD-013, TD-015, TD-031

### TD-033 — Publish versioned v0.1 binaries

- **Description:** Add tag-driven release automation for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 binaries plus SHA-256 checksums.
- **Acceptance criteria:** Artifact naming is documented; a dry run builds all five targets; tags matching `v*` create a GitHub Release with binaries and checksums; the workflow uses least privilege and pinned official actions; no signing, package manager, SBOM, or container promise is implied for v0.1.
- **Relevant files:** `.github/workflows/`, `docs/release-process.md`, `README.md`, `CHANGELOG.md`
- **Dependencies:** TD-032

### TD-034 — Publish the 45–60 second demonstration

- **Description:** Record and publish a concise demonstration of the versioned binary and deliberately regressed pull-request workflow.
- **Acceptance criteria:** The recording is 45–60 seconds, shows the inputs, terminal regression result, JSON/HTML artifacts, and failing Action without exposing personal data; a short transcript and reproducible command sequence accompany it; the README links the final asset.
- **Relevant files:** `README.md`, `docs/`, `examples/`
- **Dependencies:** TD-031, TD-033

## After v0.1

These tasks remain useful but do not block the first release.

### TD-009 — Compare database operation shapes safely

- **Description:** Detect meaningful database system/operation/statement-shape changes without reporting bound values or sensitive literals.
- **Acceptance criteria:** System, operation, table/collection, and predicate-shape cases are tested; literal-only changes do not produce leaked values; unsupported statement forms fail closed or emit an explicit limitation; privacy documentation matches behavior.
- **Relevant files:** `internal/diff/`, `internal/normalize/`, `internal/model/`, `docs/privacy-and-security.md`
- **Dependencies:** v0.1, separate prioritization

### TD-017 — Improve ambiguity diagnostics beyond the v0.1 minimum

- **Description:** Add configurable policy and richer diagnostics for uncertain trace/span matches without forcing misleading findings.
- **Acceptance criteria:** Diagnostics list non-sensitive competing signals; policy is configurable and tested; deterministic behavior holds across reordered input; CLI and JSON distinguish ambiguity from parse failure.
- **Relevant files:** `internal/match/`, `internal/report/`, `docs/trace-matching.md`
- **Dependencies:** v0.1, separate prioritization

### TD-019 — Characterize larger-input resource use

- **Description:** Measure and further constrain parser/matcher behavior beyond the safe input bounds required for v0.1.
- **Acceptance criteria:** Benchmarks use synthetic data; configurable size/depth/span limits fail with actionable errors; matching avoids accidental quadratic behavior for representative inputs; documented bounds are evidence-based.
- **Relevant files:** `internal/otlp/`, `internal/match/`, `pkg/tracedelta/`, `docs/privacy-and-security.md`
- **Dependencies:** v0.1, separate prioritization

### TD-020 — Research evidence-based source attribution

- **Description:** Explore linking findings to commit/source metadata explicitly present in telemetry, without guessing causality.
- **Acceptance criteria:** A design note defines trusted inputs, uncertainty, privacy impact, and non-goals; a prototype remains opt-in/experimental; no public capability claim is made without tests and representative evidence.
- **Relevant files:** `docs/decisions/`, `docs/product-spec.md`, future experimental package
- **Dependencies:** v0.1, separate prioritization

## Completed

### TD-015 — Provide GitHub Actions integration

- **Description:** Package TraceDelta as a reusable GitHub Action for comparing caller-supplied baseline and candidate trace artifacts.
- **Acceptance criteria:** `action.yml` has documented inputs/outputs and executes the version of TraceDelta at the pinned Action ref; a sample workflow uploads or summarizes safe reports, distinguishes exit `1` from `2`, works for fork pull requests, and requests no write permission or secret.
- **Outcome:** Added a source-pinned composite Action with required baseline/candidate inputs, finite latency/redaction inputs, structured outcome/exit/report outputs, a bounded shell entry point, and local composite-Action CI coverage. Completed comparisons preserve JSON/HTML artifacts while tool/input failures return `2` without reports. The copyable `pull_request` example uses `contents: read`, no secret, no persisted checkout credential, and an explicit post-upload gate that distinguishes regression `1` from failure `2` for untrusted forks.
- **Relevant files:** `action.yml`, `scripts/run-action.sh`, `scripts/run-action.smoke.sh`, `.github/workflows/ci.yml`, `examples/github-action/`, `docs/github-action.md`, `README.md`, `docs/privacy-and-security.md`
- **Dependencies:** TD-014

### TD-014 — Document and test generic CI usage

- **Description:** Define a headless artifact-to-comparison workflow that treats exit codes and reports correctly in generic CI.
- **Acceptance criteria:** A copyable example distinguishes regression code `1` from tool failure `2`; it preserves selected reports as artifacts without logging raw traces; a smoke test exercises the documented invocation; troubleshooting covers missing/invalid artifacts.
- **Outcome:** Added a provider-neutral shell wrapper that accepts a built binary and two existing artifacts, requires a fresh report directory, creates equivalent JSON/HTML reports without printing their contents, preserves comparison exits `0`/`1`, maps input/tool/report failures to `2`, checks report presence and result agreement, and removes partial output. Added copyable gate/upload guidance, missing/invalid-artifact troubleshooting, a real-binary smoke covering pass/regression/invalid/missing outcomes, and a Linux CI step that runs it.
- **Relevant files:** `scripts/ci-compare.sh`, `scripts/ci-compare.smoke.sh`, `examples/ci/README.md`, `.github/workflows/ci.yml`, `docs/cli-spec.md`, `docs/privacy-and-security.md`, `README.md`
- **Dependencies:** TD-011, TD-013

### TD-013 — Finalize bounded CLI and output-file policy

- **Description:** Support only the v0.1 flags for latency thresholds, additional redacted attribute keys, report format, output destination, and explicit overwrite.
- **Acceptance criteria:** Defaults and repeated-key behavior are documented/tested; unknown flags fail helpfully; options are passed without globals; `--output` refuses accidental overwrite unless `--force` is supplied; no configuration-file or arbitrary rule/plugin syntax is added.
- **Outcome:** Finalized the finite CLI around baseline/candidate, `20%`/`10ms` thresholds, repeatable narrowing-only `--redact-attribute`, text/JSON/HTML, `--output`, and explicit `--force`. Reports render fully before destination creation; new files use exclusive creation, existing files remain unchanged without force, input paths and existing aliases are forbidden output targets, parse/render failures create no new artifact, short writes fail, and output success preserves comparison exit `0`/`1`. Unknown/config-style flags and empty redaction keys fail helpfully.
- **Relevant files:** `cmd/tracedelta/`, `pkg/tracedelta/`, `docs/cli-spec.md`, `docs/privacy-and-security.md`, `examples/basic/README.md`
- **Dependencies:** TD-010, TD-012, TD-016

### TD-012 — Add a standalone HTML report

- **Description:** Produce a basic self-contained HTML report from the same comparison result.
- **Acceptance criteria:** `--format html` works without external assets; trace-derived content is HTML-escaped; ordered findings/counts match text and JSON; the report opens offline; representative output is visually reviewed and tested.
- **Outcome:** Added buffered `html/template` reporting through the report package, public API, and `--format html`. One responsive semantic document contains inline CSS, no scripts or external resources, shared input/count/finding/evidence order, explicit pass/differences presentation, contextual escaping, and a useful empty state. Structural, escaping, redaction, public, CLI, and exit-policy tests pass; actual generated output was visually reviewed in offline desktop and responsive Edge renders.
- **Relevant files:** `internal/report/html.go`, `internal/report/html_test.go`, `pkg/tracedelta/`, `cmd/tracedelta/`, `README.md`, `docs/cli-spec.md`, `docs/privacy-and-security.md`
- **Dependencies:** TD-011

### TD-011 — Add versioned JSON reporting

- **Description:** Render the shared comparison result as deterministic machine-readable JSON.
- **Acceptance criteria:** `--format json` works; schema version, input metadata, summary, ordered findings, evidence, and result policy are documented; golden/schema tests verify stable encoding and escaping; exit codes match text output.
- **Outcome:** Added buffered `tracedelta.report/v1` JSON output through the report package, public Go API, and `--format json`. The uniform schema contains input paths, summary/finding counts, explicit no-differences/differences status and exit code, ordered safe span identity, evidence presence/value, and duration policy. Golden, escaping, no-partial-write, redaction, CLI order, and pass/regression tests lock the contract; tool errors remain exit `2` without a successful report object.
- **Relevant files:** `internal/report/json.go`, `internal/report/json_test.go`, `pkg/tracedelta/`, `cmd/tracedelta/`, `docs/json-report-schema.md`, `docs/cli-spec.md`
- **Dependencies:** TD-008, TD-010, TD-016

### TD-010 — Strengthen latency comparison policy

- **Description:** Require candidate latency increases to meet configurable relative and absolute tolerances and define zero/short-span behavior.
- **Acceptance criteria:** Boundary equality, decreases, zero baselines, short spans, and large regressions are tested; both thresholds must be met; finding evidence includes safe before/after durations and effective thresholds; defaults are `20%` and `10ms`; CLI syntax is documented.
- **Outcome:** Added a non-negative absolute duration threshold alongside the existing finite relative ratio, with `20%`/`10ms` public and CLI defaults and a new `--duration-threshold-absolute` Go-duration flag. Both boundaries must be met; equality passes; zero baselines still require the absolute minimum; decreases/equality never report; either side can be disabled with zero. Duration findings carry safe before/after durations and both effective thresholds, and terminal output renders that policy deterministically.
- **Relevant files:** `internal/diff/`, `internal/report/`, `cmd/tracedelta/`, `pkg/tracedelta/`, `README.md`, `docs/cli-spec.md`
- **Dependencies:** TD-007

### TD-008 — Finalize v0.1 error findings

- **Description:** Produce typed findings for error-state changes using span status and the safe `error.type` attribute; never use error messages or exception stack traces as evidence.
- **Acceptance criteria:** OK-to-error, error-to-OK, changed `error.type`, unchanged error, and missing evidence are tested; findings have stable kinds/order and safe before/after evidence; existing status behavior remains compatible or is documented.
- **Outcome:** Preserved status transitions as one compatible `status` finding and added a separate typed `error.type` finding only when both matched spans remain in `ERROR`. String error types are normalized as safe diff evidence but excluded from match identity; non-string types are missing evidence; caller redaction takes precedence. Comparison values distinguish missing, empty, and present evidence; the terminal report escapes values and never receives status messages or exception content.
- **Relevant files:** `internal/diff/`, `internal/model/`, `internal/normalize/`, `internal/report/`, `pkg/tracedelta/`, `docs/`
- **Dependencies:** TD-007, TD-016

### TD-016 — Enforce early attribute redaction

- **Description:** Redact built-in credential/personal-data attribute keys and caller-supplied denylisted keys before matching evidence or findings are constructed.
- **Acceptance criteria:** Sensitive primitive and nested values cannot reach comparison results, text/JSON/HTML output, or covered diagnostics; deny rules override the fixed safe evidence set; replacement is deterministic; defaults cover common credential and personal-data keys; documentation states residual risk without claiming anonymization.
- **Outcome:** Added an immutable post-parse redaction stage that removes conservative case-insensitive credential/personal-data keys from span, resource, scope, and recursive key/value-list attributes before normalization. Go callers can add exact denied keys; caller rules override safe evidence keys and clear derived `service.name` when applicable. Tests cover common defaults, nested arrays/lists, deterministic removal, non-mutation, safe-key precedence, sanitized comparison/text results, and value-free option diagnostics. Future JSON/HTML reporters remain constrained to the sanitized comparison model.
- **Relevant files:** `internal/redact/`, `internal/model/`, `pkg/tracedelta/`, `internal/report/`, `docs/privacy-and-security.md`, `docs/architecture.md`
- **Dependencies:** TD-018

### TD-018 — Complete the realistic v0.1 OTLP JSON profile

- **Description:** Accept either one OTLP/HTTP JSON `ExportTraceServiceRequest` object or a trace-only OTLP File Exporter JSON Lines stream rather than expanding toward universal vendor compatibility.
- **Acceptance criteria:** Synthetic fixtures cover multiple JSONL records, multiple resource/scope groups and traces, canonical enum/integer encodings, nested `AnyValue` values, events, and links; record order is irrelevant; accepted unused data cannot become report evidence; unknown safe fields remain tolerated; mixed telemetry/vendor envelopes fail actionably; fixture provenance is documented.
- **Outcome:** Added multi-record trace-only JSONL parsing and a synthetic File Exporter fixture; recursive array/key/value-list attributes with a 64-level depth bound; event and link field validation with deliberately discarded payloads; deterministic cross-record merging independent of record order; cross-record duplicate detection; actionable rejection of recognized non-trace signals and common outer envelopes; and tests proving composite values cannot enter matching evidence.
- **Relevant files:** `testdata/`, `internal/otlp/`, `internal/model/`, `internal/normalize/`, `pkg/tracedelta/`, `docs/`
- **Dependencies:** TD-004, TD-030

### TD-007 — Match spans structurally and semantically

- **Description:** Pair spans inside matched traces using parent context, service/name/kind, semantic attributes, and deterministic sibling occurrence order.
- **Acceptance criteria:** Repeated span names under different parents pair correctly; no span is reused; reordering does not change results; ambiguity is surfaced without leaking unsafe attributes; added/removed spans remain correct.
- **Outcome:** Replaced trace-local exact occurrence matching with parent-aware one-to-one span matching. Top-level and descendant groups use service/name/kind plus fixed safe attributes and matched-parent context; distinguishable repeated siblings use normalized start/order evidence; unique reparented spans pair through an explicit relationship fallback; candidates cannot be reused; unmatched spans remain added/removed; and indistinguishable duplicates return typed value-free errors.
- **Relevant files:** `internal/match/`, `pkg/tracedelta/`, `docs/trace-matching.md`, `docs/architecture.md`, `docs/current-state.md`
- **Dependencies:** TD-006

### TD-006 — Match corresponding traces

- **Description:** Pair baseline and candidate traces using normalized root operation, service, kind, route, safe stable attributes, and deterministic occurrence handling.
- **Acceptance criteria:** Unambiguous trace pairs, added/removed traces, repeated trace operations, reordered input, and ambiguous candidates are tested; raw generated IDs are not cross-run identity; every match records non-sensitive explainable evidence.
- **Outcome:** Added a deterministic trace-match stage that groups by normalized root identity, pairs unique exact structures before mutual unique-best structural overlaps, records only signal categories and overlap counts as evidence, treats unmatched trace spans as added/removed, and returns a typed non-sensitive error for unresolved repeated-trace ambiguity. The temporary span handoff now matches by trace-local rather than snapshot-global occurrence. Regenerated IDs, duration/status changes, and input order do not establish trace identity.
- **Relevant files:** `internal/match/`, `pkg/tracedelta/`, `docs/trace-matching.md`, `docs/architecture.md`, `docs/current-state.md`
- **Dependencies:** TD-005, TD-030

### TD-030 — Lock the v0.1 release scope

- **Description:** Replace the open-ended v0.1 capability list with a finite release contract covering realistic OTLP JSON input, deterministic span comparison, safe reports, CI/release packaging, a reproducible regressed example, and a 45–60 second demonstration.
- **Acceptance criteria:** The product specification, architecture, roadmap, and backlog agree on the required v0.1 capabilities and explicit non-goals; each remaining release deliverable has a bounded task and dependency order; current behavior is not presented as complete; an accepted ADR records the compatibility and security boundary.
- **Outcome:** Accepted ADR 0002 and aligned the product, architecture, roadmap, CLI/privacy/release guidance, current state, and backlog around a finite contract: official OTLP/HTTP JSON plus trace-only File Exporter JSONL; deterministic trace/span matching; added/removed/error/latency findings; early denylist redaction; terminal/JSON/HTML reports; a reusable Action and regressed example PR; useful unit/integration/race/fuzz gates; five binary targets with checksums; and a 45–60 second demonstration. Deferred broader semantic and hosted capabilities beyond v0.1.
- **Relevant files:** `README.md`, `ROADMAP.md`, `docs/product-spec.md`, `docs/architecture.md`, `docs/current-state.md`, `docs/tasks.md`, `docs/decisions/`
- **Dependencies:** TD-005

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

### TD-005 — Implement deterministic normalization

- **Description:** Canonicalize nondeterministic IDs, timestamps, ordering, durations, and selected attributes without mutating parsed input.
- **Acceptance criteria:** Equivalent traces with different generated IDs/timestamps normalize identically; parent relationships survive ID removal; duration tolerance is typed/configurable; repeated runs produce byte-equivalent normalized fixtures; boundary cases have tests.
- **Outcome:** Added a trace-preserving canonical model with root/internal/external parent references, dense relative start ranks, structural ordering independent of input arrays and raw IDs, a non-negative typed duration bucket, and a sorted typed projection of selected HTTP/RPC attributes. Paired fixtures and unit/integration tests demonstrate byte-equivalent repeated runs, non-mutation, malformed-parent handling, boundary behavior, and unchanged comparison compatibility.
- **Relevant files:** `internal/normalize/`, `internal/model/`, `internal/match/`, `pkg/tracedelta/`, `testdata/`, `README.md`, `docs/architecture.md`, `docs/trace-matching.md`, `docs/cli-spec.md`, `docs/current-state.md`
- **Dependencies:** TD-004

### TD-029 — Add bounded native Windows validation

- **Description:** Provide a repository-native Windows verification entry point that runs required checks serially with bounded child-process waits, while documenting how automation must resume yielded command cells instead of misreporting them as hung Go commands.
- **Acceptance criteria:** A PowerShell check script runs formatting, tests, vet, and build serially from any working directory; it uses the normal shared Go cache, applies a configurable positive timeout to each Go command, reports child failures/timeouts contextually, cleans temporary build output, and leaves no spawned child running after a timeout; tests exercise successful, failing, and timed-out child processes; Windows development and agent guidance distinguish an orchestration yield from a process hang; the exact nested `go vet ./...` invocation and the new check entry point complete successfully; existing repository validation remains green.
- **Outcome:** Added a PowerShell 5.1 check entry point with serial formatting, test, vet, and build steps; deterministic first-match application resolution when PATH contains duplicates; a repository mutex; configurable per-command bounds; gated kill-on-close Windows Job Objects that prove the full process tree is empty before timeout cleanup succeeds; normal shared-cache behavior; contextual failures; temporary-build cleanup; focused resolution, success, failure, start-gate, timeout-tree, path, and lock tests; and a least-privilege Windows CI job. Corrected the prior diagnosis: yielded automation cells require their wait operation and do not by themselves show a hung Go process.
- **Relevant files:** `scripts/`, `.github/workflows/ci.yml`, `AGENTS.md`, `README.md`, `CHANGELOG.md`, `docs/development.md`, `docs/current-state.md`, `docs/session-log.md`
- **Dependencies:** TD-003

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
