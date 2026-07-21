# Changelog

All notable changes to TraceDelta will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and releases follow [Semantic Versioning](https://semver.org/) within the documented pre-v1 compatibility policy.

## [Unreleased]

## [0.1.0] - 2026-07-21

### Added

- Initial repository structure, project memory, architecture documentation, and open-source governance files.
- A small local `tracedelta compare` vertical slice for simplified OTLP-compatible JSON fixtures.
- Deterministic text reporting for added spans, removed spans, status changes, and duration changes meeting relative and absolute thresholds.
- Synthetic baseline and candidate trace fixtures and unit tests for the current behavior.
- A representative canonical OTLP JSON fixture covering resource/scope context, numeric enums, exact 64-bit values, and primitive typed attributes.
- Paired synthetic normalization fixtures covering regenerated IDs, shifted clocks, reordered input, parent relationships, and explicit duration buckets.
- Local verification commands and GitHub Actions continuous integration configuration.
- A bounded Windows PowerShell validation entry point with serial Go checks, overlap prevention, process-tree cleanup, focused watchdog tests, and Windows CI coverage.
- A finite v0.1 release contract and delivery sequence covering realistic OTLP JSON, deterministic comparison, privacy-safe reports, reusable Action/example packaging, versioned binaries, and a 45–60 second demonstration.
- Deterministic one-to-one trace matching with safe evidence, structural disambiguation for repeated operations, explicit ambiguity errors, and trace-local span handoff.
- Parent-aware one-to-one span matching with safe semantic identity, deterministic sibling ordering, unique relationship-change fallback, no candidate reuse, and explicit non-sensitive duplicate ambiguity.
- A trace-only OTLP File Exporter JSONL fixture and parser support for multiple records, recursive `AnyValue` data, validated events/links, deterministic record merging, and actionable non-trace-envelope rejection.
- Early deterministic attribute redaction with conservative credential/personal-data defaults, recursive nested-key removal, caller-supplied case-insensitive deny keys, safe-key precedence, and derived service-name clearing.
- Typed safe `error.type` findings for matched error spans, with explicit missing/empty evidence, preserved status-transition behavior, terminal escaping, and no status-message or exception-content evidence.
- Configurable combined latency thresholds with `20%`/`10ms` defaults, explicit zero/short-span behavior, CLI duration parsing, and effective policy evidence in every duration finding.
- Deterministic `tracedelta.report/v1` JSON output with input metadata, summary, ordered findings/evidence, explicit result policy, standard-library escaping, a public writer, CLI format support, and golden/schema tests.
- Responsive standalone HTML output with contextual escaping, inline CSS, no scripts/external assets, shared finding order/policy, public and CLI support, structural tests, and desktop/responsive visual review.
- A finite CLI surface with repeatable additional redaction keys, buffered output-file creation, fail-closed overwrite behavior, explicit `--force`, input-target protection, and tests across all report formats and failure paths.
- A provider-neutral CI wrapper and reusable source-pinned composite GitHub Action with finite threshold/redaction inputs, structured `0`/`1`/`2` outputs, safe JSON/HTML artifact handling, fork-safe least-privilege workflow guidance, and real-binary smoke coverage.
- A deterministic synthetic OTLP example application and public deliberately regressed PR whose trusted-base Action workflow demonstrates added, removed, error, and latency findings while retaining terminal, JSON, and HTML reports.
- Focused parser-input, matcher-order, and reporter-escaping fuzz targets; bounded fuzz automation; and end-to-end CLI/composite-Action pass, regression, tool-error, and report-consistency gates.
- A least-privilege tag release workflow and fail-closed local builder for versioned Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 binaries with verified SHA-256 checksums and a non-publishing dry-run path.
- A published 52-second WebM demonstration, on-screen transcript, reproducible comparison/recording commands, and the public deliberately failing Action in PR #7.

### Changed

- Replaced pre-publication repository, Go module, and code-owner placeholders with the canonical GitHub metadata.
- Replaced the placeholder security contact with the monitored TraceDelta security mailbox.
- Published the source repository with successful CI, security reporting, dependency/security monitoring, and an active default-branch ruleset.
- Expanded parsing to a documented OTLP JSON subset with unknown-field tolerance, nonzero ID validation, canonical numeric enums, exact integer decoding, and type-preserving primitive attributes while retaining the original fixtures.
- Replaced input-order flattening with deterministic trace-preserving normalization that resolves parent relationships before dropping raw IDs, ranks relative start order, canonicalizes selected typed HTTP/RPC attributes, and supports an explicit typed duration bucket without changing the default CLI result.
- Deferred database-shape, general service/relationship, exhaustive exporter-compatibility, hosted, collection, and source-attribution work beyond v0.1 so the first release has a finite gate.

### Limitations

- Universal OTLP/vendor-exporter coverage, CLI normalization buckets, configuration files, arbitrary evidence/rule policy, and pull-request comment/check APIs are not implemented.
- v0.1.0 does not provide signing, package-manager publication, an SBOM, or a container image.

[Unreleased]: https://github.com/ArinF1/TraceDelta/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ArinF1/TraceDelta/releases/tag/v0.1.0
