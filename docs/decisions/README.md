# Architecture decision records

Architecture decision records (ADRs) preserve decisions that materially constrain TraceDelta's design, public behavior, security posture, or future work. They explain why the repository looks the way it does so a later session does not need chat history.

## Index

- [0001 — Initial architecture](0001-initial-architecture.md) — Accepted
- [0002 — Finite v0.1 release boundary](0002-v0.1-release-boundary.md) — Accepted

## Format

Use the next four-digit number and a short lowercase filename, for example `0002-json-report-schema.md`. Each ADR contains:

- Title
- Status
- Context
- Decision
- Consequences
- Alternatives considered

Allowed statuses are **Proposed**, **Accepted**, **Superseded by ADR NNNN**, and **Rejected**. Do not rewrite the substance of an accepted decision to make history look cleaner. Create a new ADR that supersedes it; small factual/link corrections are acceptable.

An ADR is appropriate when a choice affects several packages or stages, establishes a compatibility/security boundary, adds a consequential dependency, or rejects a likely recurring alternative. Routine implementation detail belongs in tests, code comments, or the task backlog.
