# Changelog

All notable changes to TraceDelta will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project intends to follow [Semantic Versioning](https://semver.org/) once releases begin.

## [Unreleased]

### Added

- Initial repository structure, project memory, architecture documentation, and open-source governance files.
- A small local `tracedelta compare` vertical slice for simplified OTLP-compatible JSON fixtures.
- Deterministic text reporting for added spans, removed spans, status changes, and percentage-threshold duration changes.
- Synthetic baseline and candidate trace fixtures and unit tests for the current behavior.
- A representative canonical OTLP JSON fixture covering resource/scope context, numeric enums, exact 64-bit values, and primitive typed attributes.
- Paired synthetic normalization fixtures covering regenerated IDs, shifted clocks, reordered input, parent relationships, and explicit duration buckets.
- Local verification commands and GitHub Actions continuous integration configuration.
- A bounded Windows PowerShell validation entry point with serial Go checks, overlap prevention, process-tree cleanup, focused watchdog tests, and Windows CI coverage.

### Changed

- Replaced pre-publication repository, Go module, and code-owner placeholders with the canonical GitHub metadata.
- Replaced the placeholder security contact with the monitored TraceDelta security mailbox.
- Published the source repository with successful CI, security reporting, dependency/security monitoring, and an active default-branch ruleset.
- Expanded parsing to a documented OTLP JSON subset with unknown-field tolerance, nonzero ID validation, canonical numeric enums, exact integer decoding, and type-preserving primitive attributes while retaining the original fixtures.
- Replaced input-order flattening with deterministic trace-preserving normalization that resolves parent relationships before dropping raw IDs, ranks relative start order, canonicalizes selected typed HTTP/RPC attributes, and supports an explicit typed duration bucket without changing the default CLI result.

### Limitations

- No release has been published.
- Complete OTLP JSON coverage (including events, links, nested attributes, and JSON Lines exports), semantic trace/span matching, configurable filtering/redaction and CLI normalization policy, JSON reports, HTML reports, configuration files, and GitHub pull-request integration are not implemented yet.

[Unreleased]: https://github.com/ArinF1/TraceDelta/commits/main
