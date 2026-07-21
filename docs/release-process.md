# Release process

TraceDelta has no published release and no automated publishing workflow. Versioned binaries and checksum-producing tag automation are required for v0.1 under TD-033; until that task is complete, this document describes the conservative release controls the workflow must preserve.

## Versioning

The repository currently uses `v0.1.0-dev` to describe its development line. When releases begin, TraceDelta intends to use Semantic Versioning:

- before `v1.0.0`, minor versions may contain documented compatibility changes;
- patch versions contain backward-compatible fixes; and
- prerelease identifiers may be used while CLI/report contracts are still being validated.

Even before v1, avoid gratuitous breaks. Any deliberate CLI, configuration, JSON schema, or public Go API break requires changelog documentation and normally an ADR.

## Release readiness

A release candidate must satisfy all of the following:

1. The target scope and acceptance criteria are complete in `docs/tasks.md`.
2. `docs/current-state.md` accurately describes the candidate, including limitations.
3. `CHANGELOG.md` contains all user-visible changes and no unreleased claims are missing.
4. CLI flags, output formats, exit codes, schemas, and compatibility notes are documented.
5. Test fixtures are synthetic and the diff has been reviewed for secrets/personal data.
6. Repository URLs, the Go module path, the security contact, and `CODEOWNERS` are canonical; no publication placeholders remain.
7. Dependencies and their licenses/security posture have been reviewed.
8. The complete verification suite passes from a clean checkout.

## Verification commands

Run at minimum:

```bash
make clean
make check
make test-race
go build ./cmd/tracedelta
```

Then run the documented example and confirm its output and exit code with a built binary. For a report-format release, inspect every generated artifact and run format-specific escaping/schema tests.

Record exact commands, toolchain version, and outcomes in `docs/current-state.md` and append a session-log entry. An unrun check must never be recorded as passing.

## Release steps

1. Choose the version from demonstrated compatibility and scope.
2. Move relevant `CHANGELOG.md` entries from **Unreleased** into a dated version section.
3. Update version references and project state.
4. Re-run readiness and verification from the exact release commit.
5. Have another maintainer review the changelog, security contact, generated artifacts, and tag target when a second maintainer is available; until then, record the single-maintainer review limitation.
6. Create an annotated Git tag named `vX.Y.Z` from the reviewed commit.
7. Push the tag and let the reviewed release workflow create a GitHub release whose notes are derived from the changelog.
8. Download every published target artifact and verify it against the published SHA-256 checksums.
9. Restore an empty **Unreleased** section and update the development version on the next change.

Do not publish binaries from an unreviewed developer workstation workflow and do not add credentials to the repository to automate these steps. v0.1 does not promise artifact signing, package-manager distribution, an SBOM, or container images.

## Rollback and corrections

Git tags and published artifacts should be treated as immutable. If a release is defective, document the problem, mark the release appropriately on the hosting platform, and issue a corrected version. Do not silently replace an artifact under an existing version. If a credential or personal data is exposed, follow the incident and history-removal guidance in [`privacy-and-security.md`](privacy-and-security.md) and [`../SECURITY.md`](../SECURITY.md).
