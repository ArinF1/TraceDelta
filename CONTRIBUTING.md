# Contributing to TraceDelta

Thank you for helping make runtime behavior changes easier to review. TraceDelta is early-stage, so small, focused contributions with tests are especially valuable.

## Before you begin

Read these files to understand the current contract and implementation state:

- [`README.md`](README.md)
- [`docs/product-spec.md`](docs/product-spec.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/current-state.md`](docs/current-state.md)
- [`docs/tasks.md`](docs/tasks.md)
- relevant records in [`docs/decisions/`](docs/decisions/)

For substantial proposals, open a feature request before investing in an implementation. Security vulnerabilities must follow [`SECURITY.md`](SECURITY.md), not the public issue tracker.

## Development setup

You need Go 1.26 or a compatible newer stable release, GNU Make for the convenience targets, and Bash for the repository scripts.

```bash
git clone https://github.com/ArinF1/TraceDelta.git
cd TraceDelta
make check
```

Run the synthetic comparison example with:

```bash
make run-example
```

The example contains intentional differences. The Make target verifies that the CLI returns the expected difference exit code.

## Contribution workflow

1. Choose one task from `docs/tasks.md`, or create a focused issue before expanding scope.
2. Create a short-lived branch from the current default branch.
3. Make the smallest coherent change that satisfies the task.
4. Add or update tests for every behavioral change.
5. Update user-facing documentation when commands, formats, or guarantees change.
6. Run `make check` and, where practical, `make test-race`.
7. Update the project-state files required by `AGENTS.md`.
8. Open a pull request using the repository template.

Do not commit raw production traces, credentials, personal data, generated reports, or unrelated formatting changes.

## Code expectations

- Prefer the Go standard library and direct, testable code.
- Keep package responsibilities aligned with `docs/architecture.md`.
- Wrap errors with enough context to identify the operation that failed.
- Preserve deterministic output and ordering.
- Avoid global mutable state and premature interfaces.
- Add comments for packages and exported symbols when they clarify the public contract.
- Use `TODO(TD-###)` for deferred work, where the ID exists in `docs/tasks.md`.

## Tests and checks

`make check` verifies formatting, tests, vetting, and the CLI build. Useful individual targets are:

```bash
make fmt
make fmt-check
make test
make test-race
make vet
make build
```

Tests should use synthetic fixtures that are safe to publish. A passing test suite is required, but tests should demonstrate behavior rather than chase an arbitrary coverage percentage.

## Commits and pull requests

Write concise, imperative commit subjects. Conventional Commit prefixes such as `feat:`, `fix:`, `docs:`, and `chore:` are welcome but not required.

A pull request should explain the behavioral change, link the task or issue, list validation performed, call out compatibility or privacy effects, and remain focused enough to review in one sitting.

## Community standards

Participation is governed by the [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md). By contributing, you agree that your contributions are licensed under the repository's [MIT License](LICENSE).
