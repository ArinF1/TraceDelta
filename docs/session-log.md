# Session log

This is a lightweight, append-only record for context that does not belong in commits or ADRs. Add the newest entry at the end. Keep entries brief and factual; never include secrets, personal data, raw production traces, or unreviewed trace excerpts.

Use this shape:

```text
## YYYY-MM-DD — Short session title

- Scope: task IDs or concise goal.
- Outcome: what changed and what remains.
- Decisions: ADR links or small implementation choices.
- Validation: exact commands run and their outcomes; say “not run” when applicable.
- Next: one recommended task or explicit blocker.
```

## 2026-07-17 — Initial repository foundation

- **Scope:** TD-001, TD-002, and TD-003: establish the repository, persistent project memory, open-source files, and a small comparison vertical slice.
- **Outcome:** Created the staged Go project structure, simplified synthetic OTLP-compatible fixtures, deterministic text comparison path, tests, local/CI checks, and honest product/architecture/current-state documentation. Full OTLP parsing and semantic matching remain open.
- **Decisions:** Accepted [ADR 0001](decisions/0001-initial-architecture.md): Go, local CLI, JSON fixtures, staged pipeline, no database/web app, and minimal dependencies.
- **Validation:** Go 1.26.0 on Windows: `gofmt` clean; normal and race-enabled tests passed across all eight packages (16 top-level test functions, with `internal/model` intentionally having no direct test file); `go vet` clean; CLI build and `scripts/check.sh` passed; built CLI exit codes `1`, `0`, and `2` were verified for differences, equality, and invalid format. GNU Make and a YAML parser were unavailable; underlying commands and workflow structure were checked directly.
- **Next:** TD-004 — expand the OTLP JSON parser while preserving contextual errors and the compatibility fixture.

## 2026-07-17 — Configure private GitHub metadata

- **Scope:** TD-026: replace temporary repository, Go module, and code-owner values with the canonical private GitHub metadata.
- **Outcome:** Updated repository links to `ArinF1/TraceDelta`, changed the Go module and imports to `github.com/ArinF1/TraceDelta`, and assigned `@ArinF1` as the default code owner. The security email remains an explicit placeholder until a monitored address is available.
- **Decisions:** Keep `security@example.com` while the repository is private; it remains a documented blocker for public release.
- **Validation:** `gofmt` clean; `go mod tidy` and `go mod verify` passed; normal and race-enabled tests passed across all eight packages; `go vet` clean; CLI build and `scripts/check.sh` passed; the example produced the expected four findings and exit code `1`.
- **Next:** TD-004 — expand the OTLP JSON parser; replace the security contact before changing repository visibility to public.

## 2026-07-18 — Configure monitored security contact

- **Scope:** TD-027: replace the pre-publication security-address placeholder with the dedicated TraceDelta security mailbox.
- **Outcome:** Updated the active README, security policy, privacy guidance, changelog, project state, and backlog to use `tracedelta.security@gmail.com`. Preserved the preceding append-only session entry because it accurately records the earlier placeholder state.
- **Decisions:** Use a dedicated project mailbox rather than a maintainer's everyday personal address; keep private vulnerability reporting as an additional GitHub-hosted channel when the repository becomes public.
- **Validation:** The pre-change `go test -count=1 ./...` baseline passed. After the documentation changes, `gofmt` verification, normal and race-enabled tests across all eight packages, `go vet ./...`, `go mod verify`, CLI build, and the example's expected exit code `1` all passed. `bash scripts/check.sh` was attempted but stopped because this Windows Bash session resolved `find` incorrectly (`find: ‘gofmt’: No such file or directory`); its underlying checks were run directly and passed.
- **Next:** TD-004 — expand the OTLP JSON parser while preserving contextual errors and the compatibility fixture.

## 2026-07-18 — Publish and harden the GitHub repository

- **Scope:** TD-028: record the public repository and its initial GitHub security and branch-governance configuration.
- **Outcome:** Published `ArinF1/TraceDelta`, retained the experimental `v0.1.0-dev` status without creating a release, enabled the planned public-project safeguards, and updated the existing project-memory files rather than adding a redundant memory document.
- **Decisions:** Keep `AGENTS.md`, current state, backlog, ADRs, and the append-only session log as the complete handoff system. Require pull requests and the existing CI check on `main`, but require zero human approvals while there is only one maintainer.
- **Validation:** Pre-change and post-change `go test -count=1 ./...` passed across all eight packages; post-change `go vet ./...`, `gofmt` verification, and `git diff --check` also passed. GitHub's public API confirmed public visibility, default branch `main`, the intended description, a successful completed `CI` run on `main`, and the active `main-protection` branch ruleset. The maintainer confirmed the administrative security toggles and detailed ruleset options; those access-controlled settings were not independently inspected.
- **Next:** TD-004 — expand the OTLP JSON parser while preserving contextual errors and the compatibility fixture.

## 2026-07-18 — Expand OTLP JSON parsing

- **Scope:** TD-004: expand the parser to a strongly typed, documented subset of canonical OTLP JSON without changing comparison semantics.
- **Outcome:** Added numeric kind/status enums, exact quoted/unquoted integer handling, nonzero ID validation, primitive type-preserving attributes, resource/scope context, safe unknown-field tolerance, and a representative synthetic fixture. Empty envelopes and omitted resource/scope context parse; non-empty events/links and nested, case-less, or otherwise invalid attribute values fail contextually. The original symbolic-enum fixtures remain compatible.
- **Decisions:** Keep one JSON document and primitive attributes as the bounded subset; document JSON Lines, events, links, arrays, and key-value lists as unsupported. Retain symbolic enums only as a compatibility extension. No new ADR was needed because the implementation follows ADR 0001's existing staged parser boundary and standard-library constraint.
- **Validation:** The pre-change normal suite and vet passed. After the change, `gofmt -l` was clean; `go test -count=1 ./...` and `go test -race -count=1 ./...` passed across all eight packages; `go vet ./...`, `go mod verify`, CLI build, and `git diff --check` passed. The built CLI returned `1` with the documented four compatibility-fixture findings and `0` when the representative OTLP fixture was compared with itself. A later parallel elevated check batch left its wrapper waiting after vet had exited; no Go/vet process remained, and serial vet plus normal tests passed with a temporary workspace-local `GOCACHE`, which was removed. The Bash wrapper was not rerun because its previously recorded Windows `find` incompatibility remains; all underlying required checks ran directly.
- **Next:** TD-005 — implement deterministic normalization over the expanded typed parse model.

## 2026-07-19 — Implement deterministic normalization

- **Scope:** TD-005: canonicalize generated identifiers, clocks, input ordering, durations, and selected typed attributes while preserving trace/parent structure and parsed-input immutability.
- **Outcome:** Replaced input-order flattening with canonical trace forests, root/internal/external parent references, dense start-order ranks, fixed-size structural subtree digests, global canonical occurrences for the temporary matcher, a typed duration bucket, and sorted HTTP/RPC attribute projections. Added paired equivalent-run fixtures plus relationship, malformed graph, duration boundary, typed value, non-mutation, deterministic byte, deep-chain, and public orchestration tests. General trace matching, filtering/redaction, latency regression policy, and resource limits remain assigned to TD-006, TD-016, TD-010, and TD-019.
- **Decisions:** Default duration bucketing to zero so existing CLI comparisons remain exact; expose the bucket only through typed Go options until configuration/CLI policy work; treat a missing parent as external while rejecting self-parenting and cycles; keep only `http.request.method`, `http.route`, `rpc.method`, and `rpc.service` in the stable normalization projection without claiming redaction. No new ADR was needed because these choices implement ADR 0001's existing deterministic staged pipeline.
- **Validation:** The pre-change normal suite passed across all eight packages and serial vet completed in 4 seconds. After the change, `gofmt` verification, focused tests, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go mod verify`, CLI build, and `git diff --check` passed. The built CLI returned `1` with the documented four example findings and `0` for representative-fixture self-comparison. Serial final vet completed in 4.6 seconds; avoiding parallel elevated Go commands prevented the previously observed Windows command-wrapper wait. The temporary executable was removed.
- **Next:** TD-006 — match corresponding normalized traces deterministically and expose ambiguity rather than guessing.

## 2026-07-19 — Bound native Windows validation

- **Scope:** TD-029: diagnose the recurring Windows vet wait and provide a repository-native bounded validation path without changing TraceDelta product behavior.
- **Outcome:** Added a PowerShell 5.1 entry point that serializes formatting, tests, vet, and CLI build under a repository mutex with configurable per-command timeouts and normal shared-cache use. Native commands start only after a gated launcher belongs to a kill-on-close Windows Job Object; timeout success requires zero active job processes. Added focused success, failure, gate-order, timeout-tree, path-with-spaces, and lock tests plus a least-privilege Windows CI job. This supersedes the earlier session diagnosis: the observed `Script running with cell ID ...` state was an automation polling yield that required waiting on the same cell, not a hung `go vet` process.
- **Decisions:** Use Windows Job Objects instead of `taskkill` or WMI because both were access-denied in the restricted runner; gate command startup to eliminate the process-assignment race; retain the normal shared `GOCACHE` because a deliberately disposable cache exceeded a 30-second diagnostic bound. No ADR was needed because this changes development validation only, not TraceDelta architecture or public comparison behavior.
- **Validation:** The pre-change uncached normal suite passed all eight Go packages. Direct, nested, and concurrent warm-cache vet probes all exited `0` in approximately 0.4–4.0 seconds with no surviving tool process; the user's exact nested command returned `0` in 1.6 seconds. The final focused watchdog suite passed; earlier stress covered eight full probe runs, 800 rapid process assignments, and ten detached child trees without a survivor. The final Windows entry point passed from `C:\tmp`; `gofmt` was clean across 15 files; uncached normal and race-enabled suites passed all eight packages; direct vet, `go mod verify`, CLI build, expected comparison exits `1` and `0`, temporary-output cleanup, diff whitespace, merge-marker, TODO, high-confidence secret, and intended-status checks passed. The first push-triggered Windows CI run exposed three `git.exe` PATH matches; explicit first-match resolution and a duplicate-PATH regression probe were added, and focused plus full local Windows checks passed again before the replacement CI run.
- **Next:** TD-006 — match corresponding normalized traces deterministically and expose ambiguity rather than guessing.

## 2026-07-20 — Lock the finite v0.1 scope

- **Scope:** TD-030: replace the open-ended first-release roadmap with a finite, demonstrable product boundary.
- **Outcome:** Defined v0.1 as official OTLP/HTTP JSON plus trace-only OTLP File Exporter JSONL input; deterministic trace/span matching; added/removed/error/latency findings; early denylist redaction; equivalent terminal/JSON/HTML reports; a reusable Action and synthetic regressed pull request; useful unit/integration/race/fuzz gates; five binary targets with checksums; and a 45–60 second demonstration. Moved database/service/relationship analysis, exhaustive compatibility, hosted/collection work, and source attribution beyond v0.1. No executable behavior changed.
- **Decisions:** Accepted [ADR 0002](decisions/0002-v0.1-release-boundary.md). v0.1 error evidence is span status plus `error.type`; latency defaults will be `20%` and `10ms` with both required; CLI configuration is flags-only; release binaries target Linux amd64/arm64, macOS amd64/arm64, and Windows amd64. The official OTLP and File Exporter specifications justify the two accepted input shapes.
- **Validation:** The pre-change repository-native Windows check passed formatting, tests across all eight packages, vet, and CLI build. A sandbox-only baseline attempt failed because the restricted runner could not read the normal shared Go cache; the approved rerun in the intended environment passed. The final repository-native Windows check passed the same formatting, eight-package test, vet, and CLI build stages. `git diff --check` passed before task closure; final documentation/link/status audits followed after this entry.
- **Next:** TD-006 — match corresponding normalized traces deterministically and expose ambiguity without leaking unsafe attributes.

## 2026-07-20 — Match corresponding traces

- **Scope:** TD-006: pair normalized baseline and candidate traces deterministically before span matching.
- **Outcome:** Added root-identity trace groups, unique exact-structure pairing, mutual unique-best structural-overlap pairing for repeated operations, non-sensitive match evidence, added/removed trace propagation, and a typed ambiguity error that fails rather than guessing. Changed the temporary span handoff from snapshot-global to trace-local occurrence. TD-007 parent-aware semantic span matching remains next.
- **Decisions:** Exclude status, duration, raw IDs, clocks, and input order from trace identity. Do not use occurrence order to break an otherwise indistinguishable trace tie. Ambiguity diagnostics expose counts and signal categories only; richer/configurable policy remains TD-017. No new ADR was needed because this implements ADR 0001's staged matcher and ADR 0002's deterministic v0.1 boundary.
- **Validation:** The pre-change Windows repository check passed. The first focused build failed before tests because the helper function and identity type were both named `spanIdentity`; the helper was renamed and the corrected focused suites passed. Final Windows formatting, all-eight-package tests, vet, and CLI build passed; `go test -race -count=1 ./...` passed all eight packages. A temporary built CLI produced the documented four findings with exit `1` and a representative self-comparison with exit `0`, then was removed. Final diff, secret, task-state, and status audits followed.
- **Next:** TD-007 — match spans structurally and semantically inside matched traces.

## 2026-07-20 — Match spans structurally and semantically

- **Scope:** TD-007: replace the temporary trace-local occurrence matcher with deterministic parent-aware span correspondence.
- **Outcome:** Added semantic span groups, matched-parent context, deterministic sibling order for distinguishable repeats, unique relationship-change fallback, candidate-use tracking, stable result ordering, and typed non-sensitive sibling ambiguity. Added focused and public orchestration tests for repeated names under reordered parents, relationship changes, additions/removals, sibling occurrence, ambiguity, and error context.
- **Decisions:** Status and duration remain diff evidence, not identity. A unique reparented span stays paired so a relationship-only change does not become a false add/remove; v0.1 does not emit a relationship finding. Equal-rank duplicate siblings fail instead of using input order. Nil and empty normalized attribute slices canonicalize identically at the matcher boundary. No new ADR was needed because this implements the accepted staged and finite matching contracts.
- **Validation:** The first focused behavior run expected ambiguity but returned none because the test clone changed empty attributes from `[]` to `null`; matcher key canonicalization was corrected and the focused suites then passed. The final repository-native Windows check passed formatting, tests across all eight packages, vet, and CLI build. `go test -race -count=1 ./...` passed all eight packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-018 — complete the realistic OTLP/HTTP JSON and trace-only File Exporter JSONL profile.

## 2026-07-20 — Complete the realistic v0.1 OTLP JSON profile

- **Scope:** TD-018: accept the two official v0.1 trace input shapes and realistic nested trace data without expanding into universal exporter compatibility.
- **Outcome:** Added trace-only OTLP File Exporter JSONL decoding and a synthetic two-record fixture; deterministic record merging; cross-record duplicate detection; recursive `arrayValue`/`kvlistValue` preservation with a 64-level limit; event/link validation with deliberately discarded payloads; recognized non-trace/wrapper rejection; and evidence-exclusion tests for unused nested, event, and link data.
- **Decisions:** Treat record order as transport order rather than behavioral identity. Preserve recursive attribute values in the parsed model, but omit composite safe-key values from normalization rather than serializing them into evidence. Validate events and links so malformed accepted input is visible, then discard their payloads because v0.1 defines no event/link rule. Keep metrics, logs, profiles, protobuf binary, mixed signals, vendor envelopes, and exhaustive producer compatibility outside the bounded profile. No new ADR was needed because this implements ADR 0002's accepted input boundary.
- **Validation:** Focused parser, normalization, and public orchestration tests passed. The repository-native Windows check passed formatting, all-eight-package tests, vet, and CLI build; `go test -race -count=1 ./...` passed all eight packages. A temporary built CLI self-compared `testdata/otlp-file.jsonl` with exit `0` and was removed. The first direct script launch was blocked before execution by PowerShell policy; the documented bypass invocation passed. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-016 — enforce early built-in and caller-configured attribute redaction before evidence construction.

## 2026-07-20 — Enforce early attribute redaction

- **Scope:** TD-016: remove built-in and caller-denied credential/personal-data attribute keys before matching evidence or findings are constructed.
- **Outcome:** Added `internal/redact` as an immutable post-parse stage for both inputs. It recursively removes denied span/resource/scope and nested key/value-list entries, covers conservative credential/personal-data names and namespaces case-insensitively, accepts caller exact keys through the public Go options, lets denial override the fixed safe set, and clears derived service identity when `service.name` is denied. Public and unit tests prove sensitive synthetic values do not reach sanitized snapshots, comparison values, terminal reports, or covered errors.
- **Decisions:** Remove denied entries instead of inserting a placeholder so a caller-denied key cannot still influence correspondence. Keep the built-in rules conservative and case-insensitive; custom organization keys remain caller responsibility. Treat redaction as risk reduction rather than anonymization, and require future JSON/HTML reporters to consume only the sanitized comparison model. No new ADR was needed because this implements ADR 0002's privacy boundary.
- **Validation:** Focused redaction, public orchestration, and reporter tests passed. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build. `go test -race -count=1 ./...` passed all nine packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-008 — finalize safe error findings using status plus `error.type` only.

## 2026-07-20 — Finalize v0.1 error findings

- **Scope:** TD-008: derive deterministic safe error changes from span status and string `error.type` evidence only.
- **Outcome:** Retained existing status-transition findings and added a typed `error.type` change for matched spans that both remain in `ERROR`. Normalization keeps string error types separate from match attributes, non-string values become missing evidence, comparison values distinguish absent/empty/present forms, and terminal output escapes the safe value. Public tests prove status/exception messages do not reach results and caller denial suppresses error-type evidence.
- **Decisions:** Do not emit a second error-type finding alongside OK/error status transitions; the status finding already captures that state change. Compare `error.type` only when both sides are errors so stray attributes on successful spans do not become findings. Preserve empty presence separately from missing evidence for future structured reports. No new ADR was needed because this implements ADR 0002's fixed error evidence.
- **Validation:** Focused normalization, diff, public orchestration, and terminal-report tests passed. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build; `go test -race -count=1 ./...` passed all nine packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-010 — require both relative and absolute latency thresholds with documented boundary behavior.

## 2026-07-20 — Strengthen latency comparison policy

- **Scope:** TD-010: require candidate latency increases to meet configurable relative and absolute thresholds together and define boundary/zero/short-span behavior.
- **Outcome:** Added `DurationThresholdAbsolute` through diff and public options with `10ms` default beside the existing `20%` relative default; exposed `--duration-threshold-absolute` using bounded Go-duration parsing; required both thresholds; and attached before/after durations plus effective thresholds to each duration finding and terminal line.
- **Decisions:** Treat a positive candidate over a zero baseline as satisfying the relative side but still require the absolute side. Let zero disable either threshold independently while equal/decreased durations remain non-findings. Preserve `--duration-threshold` as the relative flag and use the already specified explicit absolute flag rather than adding configuration-file policy. No new ADR was needed because this implements ADR 0002's fixed latency defaults.
- **Validation:** Focused diff, public, CLI, and reporter tests passed across equality boundaries, zero/short/large spans, disabled and invalid thresholds, defaults, override behavior, and evidence rendering. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build; `go test -race -count=1 ./...` passed all nine packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-011 — add deterministic schema-versioned JSON reporting from the shared sanitized result.

## 2026-07-20 — Add versioned JSON reporting

- **Scope:** TD-011: expose deterministic machine-readable output from the shared sanitized comparison model without adding a parallel rules path.
- **Outcome:** Added buffered `tracedelta.report/v1` JSON generation, a public `WriteJSON` wrapper, and `--format json`. The documented uniform schema includes input paths, summary/finding counts, explicit result status/process code, stable finding order, safe span identity, before/after presence and values, and duration thresholds. Empty findings remain `[]`; standard JSON escaping covers controls and HTML-significant characters.
- **Decisions:** Keep all finding objects structurally uniform rather than making consumers infer many optional shapes. Represent missing versus empty evidence with an explicit presence Boolean. Embed only successful comparison policy (`0`/`1`); tool/input failure `2` produces no successful report. Buffer before writing so encoding failure cannot leave a partial document. No new ADR was needed because the v1 schema implements ADR 0002's shared-reporter requirement.
- **Validation:** Golden/schema, escaping, buffering, redaction, public API, deterministic CLI order, and pass/regression exit-policy tests passed. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build; `go test -race -count=1 ./...` passed all nine packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-012 — render the same result as a self-contained escaped offline HTML report.

## 2026-07-20 — Add a standalone HTML report

- **Scope:** TD-012: render the same sanitized ordered comparison result as one offline, self-contained, safely escaped HTML document.
- **Outcome:** Added buffered contextual HTML templating, inline responsive CSS, semantic summary/outcome/findings structure, fixed kind badges, explicit empty state, public `WriteHTML`, and `--format html`. All dynamic paths, span/service names, and evidence are control-normalized and contextually escaped; no script, link, source, or network resource is emitted.
- **Decisions:** Keep the report script-free and dependency-free rather than adding a frontend bundle. Use a table inside an explicit horizontal-overflow container and hide the service column at the responsive breakpoint while preserving evidence. Buffer the document before destination writes, matching JSON failure behavior. No new ADR was needed because this implements ADR 0002's standalone report requirement.
- **Validation:** Structural, ordering, counts, evidence, escaping, empty-state, redaction, public API, CLI, and exit-policy tests passed. The browser integration could not start because its local Node process received `EPERM` on the Windows AppData path, so the same actual CLI-generated report was rendered offline with local headless Edge and visually inspected at 1440px desktop and 720px responsive widths; both were clean and legible. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build; the race suite passed all nine packages. All temporary HTML, screenshots, browser profiles, generator source, and binary were removed.
- **Next:** TD-013 — finalize the bounded CLI, repeatable redaction option, and safe output-file overwrite policy.

## 2026-07-20 — Finalize bounded CLI and output-file policy

- **Scope:** TD-013: expose only the agreed v0.1 flags and make file output fail closed unless replacement is explicit.
- **Outcome:** Added repeatable `--redact-attribute`, functional `--output`, and `--force` around the existing input, threshold, and format flags. The CLI renders to memory before touching a destination, creates new output exclusively with owner-only requested permissions, refuses existing files without force, rejects input/self and hard-link aliases, keeps parse failures artifact-free, detects short writes, and preserves result exit `0`/`1`; all diagnostics remain exit `2` on standard error.
- **Decisions:** Keep custom redaction narrowing-only and reject configuration/rule/plugin flags. Require parent directories to exist rather than creating filesystem structure implicitly. Forbid input paths as outputs even with force. Preserve stdout as the default while making `--force` invalid without a path. No new ADR was needed because this implements ADR 0002's finite flag-only contract.
- **Validation:** Focused CLI and public tests passed across defaults, repeated/empty redaction keys, unknown flags, new/existing/forced outputs, input and filesystem-alias protection, parse failure, short writes, formats, and exit policy. The repository-native Windows check passed formatting, all-nine-package tests, vet, and CLI build; the race suite passed all nine packages. Final diff, task-state, secret, and status audits followed.
- **Next:** TD-014 — document and smoke-test generic headless CI artifact handling without logging trace contents.

## 2026-07-20 — Document and test generic CI usage

- **Scope:** TD-014: provide one copyable headless comparison workflow with safe report artifact handling and exact exit semantics.
- **Outcome:** Added `scripts/ci-compare.sh`, which accepts a built TraceDelta binary, baseline/candidate artifacts, and a new report-directory path; emits only concise statuses and report paths; produces JSON and standalone HTML; preserves exit `0`/`1`; maps missing, invalid, tool, report, or consistency failures to `2`; and removes partial reports. Added a real-binary smoke test for pass, regression, malformed, and missing inputs; wired it into Linux CI; and documented copyable gating, artifact upload, and troubleshooting without printing traces or reports.
- **Decisions:** Require a fresh report directory rather than silently overwriting an unknown artifact set. Run each format independently and require identical comparison outcomes before preserving either report. Keep this wrapper provider-neutral and positional instead of adding configuration, collection, or provider-specific behavior before TD-015. No new ADR was needed because this is the generic orchestration layer already required by ADR 0002.
- **Validation:** Bash syntax validation passed. The real built CLI smoke passed all four documented outcomes and verified non-empty schema-marked JSON plus standalone HTML for exits `0`/`1`, with no reports for invalid/missing exits `2`. The first Windows-check launch was interrupted by the command client with the exact message `command timed out after 5051 milliseconds`; the fresh full invocation passed formatting, all-nine-package tests, vet, and CLI build. `go test -race -count=1 ./...` passed all nine packages. Final diff, task-state, secret, artifact, and status audits followed.
- **Next:** TD-015 — package the verified comparison workflow as a reusable least-privilege GitHub Action.

## 2026-07-20 — Start reusable GitHub Action packaging

- **Scope:** TD-015: package the verified generic comparison path as a source-pinned composite Action with finite inputs, structured outputs, and an unprivileged pull-request example.
- **Outcome:** Added `action.yml`, a bounded Action entry point, a smoke harness, a local composite-Action CI exercise, and a copyable `pull_request` workflow. The Action builds from its pinned `github.action_path`, accepts only the two latency thresholds and newline-separated additional redaction keys, preserves JSON/HTML reports for completed outcomes, exposes `0`/`1`/`2` plus named outcomes, fails its step only for tool/input error `2`, and uses an explicit later gate so reports upload before regression failure. The sample requests only `contents: read`, persists no checkout credential, passes no secret, and avoids `pull_request_target`.
- **Validation:** Bash syntax passed for all four CI/Action scripts. PyYAML parsed `action.yml`, the repository CI workflow, and the sample workflow; `git diff --check` passed. A real current-binary smoke and final repository-native/race validation remain unrun: the required build escalation was rejected with `Automatic approval review failed: You've hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Jul 25th, 2026 2:35 PM.` The task remains in **Now** and is not represented as complete.
- **Next:** When execution capacity is available, build the current CLI, run `scripts/ci-compare.smoke.sh` and `scripts/run-action.smoke.sh`, run the repository-native Windows check and race suite, then close TD-015 only if all results pass.

## 2026-07-21 — Complete reusable GitHub Action packaging

- **Scope:** Resume TD-015 at its validation boundary and close the reusable source-pinned composite Action only after real-binary and repository-wide evidence passed.
- **Outcome:** Confirmed the composite Action, finite inputs, structured outcomes, safe artifact policy, fork-safe sample workflow, and local-Action CI exercise. Updated the implementation-state, architecture, product, changelog, task, and security-facing documentation so Action packaging is current while pull-request APIs remain explicitly out of scope.
- **Decisions:** Keep regression exit `1` as a completed Action step so callers can upload reports before an explicit policy gate; keep tool/input exit `2` as an Action failure with structured outputs. Build from a `mktemp` directory under `RUNNER_TEMP` and the pinned `github.action_path`; do not execute a caller-workspace or `PATH` binary. No new ADR was needed because these choices implement ADR 0002's accepted Action boundary.
- **Validation:** The current-source CLI build passed. The generic CI real-binary smoke passed `0`, `1`, malformed-input `2`, and missing-input `2` paths. The Action entry-point real-binary smoke passed regression `1` output/report preservation and missing-input `2` failure/output behavior. Bash syntax and YAML parsing passed. The repository-native Windows check passed formatting, tests across all nine packages, vet, and CLI build; `go test -race -count=1 ./...` passed all nine packages.
- **Next:** TD-031 — add the deterministic synthetic example application and replayable deliberately regressed pull-request workflow.
