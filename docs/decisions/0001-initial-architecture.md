# ADR 0001: Initial TraceDelta architecture

## Status

Accepted

## Context

TraceDelta aims to compare baseline and candidate OpenTelemetry traces and identify behavior changes useful during pull-request review. The project needs to prove that comparison semantics are valuable before taking on collection infrastructure, storage, hosted operations, or a broad public API.

Telemetry exports contain nondeterministic identifiers/timing and potentially sensitive data. Parsing wire data, deciding correspondence, classifying changes, and presenting results have different reasons to evolve and different failure modes. The initial repository also needs to be approachable for contributors and deterministic in local tests and CI.

## Decision

TraceDelta will begin as a Go command-line tool with a staged parse → normalize → match → diff → report architecture.

Specifically:

- **Go:** Use Go because it produces a portable single binary, has strong standard-library support for JSON, CLI/process handling, testing, HTML/template safety, and deterministic explicit data modeling, and fits common CI environments. The repository begins on Go 1.26, the newest stable local toolchain at initialization.
- **Local CLI:** Make local file-to-report comparison the product boundary. This keeps behavior reproducible, scriptable, useful in generic CI, and local-first for sensitive traces.
- **OTLP JSON fixtures:** Start with synthetic JSON fixtures and a strongly typed simplified OTLP-compatible shape. JSON is inspectable in reviews and easy to vary in tests; the parser boundary can grow toward representative OTLP JSON without committing to a generated protocol stack in the first slice.
- **Staged pipeline:** Keep parsing, normalization, trace/span matching, semantic diffing, and reporting as distinct responsibilities with typed domain values between them. Process/exit behavior remains at the CLI edge.
- **No database:** Read two files and compare in memory. Persistence, querying, retention, tenancy, migration, and operational complexity are not justified by the initial use case.
- **No web application:** Provide terminal output first, then machine-readable JSON and self-contained HTML artifacts. A frontend/server would add security and maintenance surface before comparison rules are trustworthy.
- **Minimal dependencies:** Prefer the Go standard library and add third-party packages only when a concrete requirement outweighs supply-chain and maintenance cost. The initial CLI is small enough not to require a framework such as Cobra.

Generated trace IDs, span IDs, and wall-clock timestamps will not be treated as cross-run behavioral identity. Configuration will be passed explicitly rather than stored in global mutable state, and all reporters will consume the same ordered comparison result.

## Consequences

Positive consequences:

- Contributors can build, test, and understand the core locally with a small toolchain.
- The project can validate domain semantics before committing to operations or storage.
- Stage boundaries make parser growth, matching improvements, new rules, and report formats independently testable.
- A local-first workflow reduces accidental trace-data transmission.
- Deterministic fixtures and output are suitable for CI and regression testing.
- Minimal dependencies reduce startup complexity and supply-chain surface.

Costs and limitations:

- The simplified initial format cannot accept arbitrary OTLP JSON exports.
- In-memory comparison will need explicit resource bounds for large inputs.
- Users must collect and sanitize baseline/candidate traces themselves.
- JSON's OTLP encodings and semantic-convention variations require careful compatibility work.
- Deferring a web interface means early reports are artifacts rather than interactive exploration.
- Avoiding a CLI framework is efficient now but may require reevaluation if command structure becomes genuinely complex.

These limitations must remain visible in `README.md`, `docs/current-state.md`, and the backlog rather than being presented as completed capability.

## Alternatives considered

- **Protocol Buffers as the only initial fixture format:** More canonical on the wire, but binary fixtures are harder to review and hand-author. OTLP protobuf support may be added later without replacing the domain pipeline.
- **A hosted service and trace database first:** Could enable history and collaboration, but introduces tenancy, security, retention, cost, migrations, and operations before core matching/diff value is established. Rejected for the initial product.
- **A web UI first:** Could make exploration richer, but couples the project to a server/frontend and expands its attack surface. Standalone reports are a smaller later step.
- **Reuse an observability backend as mandatory storage:** Offers powerful querying but makes TraceDelta deployment-specific and less local/reproducible. Future adapters can be considered after file comparison is reliable.
- **A monolithic parse-and-print implementation:** Faster for a throwaway prototype but makes nondeterministic normalization, uncertain matching, semantic rules, and multiple reporters difficult to test independently.
- **A plugin/rule framework from the start:** Provides theoretical extensibility at the cost of an unproven public contract. Concrete rules and functions will establish the right extension shape first.
- **A larger CLI/configuration framework:** Useful for many commands and nested configuration, but unnecessary for one command and a few flags. Reconsider only when complexity is demonstrated.
