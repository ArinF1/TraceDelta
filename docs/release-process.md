# Release process

TraceDelta v0.1.0 was published on 2026-07-21. The release workflow performs a read-only five-target dry run when manually dispatched and publishes only for pushed `v*` tags. Its build job has `contents: read`; only the tag-gated publish job receives `contents: write`.

## Release artifacts

For `v0.1.0`, the workflow produces exactly:

```text
tracedelta_0.1.0_linux_amd64
tracedelta_0.1.0_linux_arm64
tracedelta_0.1.0_darwin_amd64
tracedelta_0.1.0_darwin_arm64
tracedelta_0.1.0_windows_amd64.exe
tracedelta_0.1.0_checksums.txt
```

All binaries are `CGO_ENABLED=0`, trimmed-path Go builds from the tagged source. The checksum file uses SHA-256 and relative artifact names. The workflow verifies checksums before upload and again after the tag job downloads the build artifact. v0.1 does not promise signing, package-manager publication, an SBOM, or a container image.

The release also carries `tracedelta-v0.1-demo.webm`, a separately generated 52-second demonstration. Its transcript, SHA-256 digest, provenance, and reproduction steps are recorded in [`demo.md`](demo.md). The binary checksum manifest deliberately covers only the five executable artifacts.

Run the exact five-target build locally with `make test-release-smoke`, or manually dispatch `.github/workflows/release.yml` with a non-release label such as `v0.1.0-test`. A manual dispatch uploads a short-lived workflow artifact and never enters the publish job.

## Versioning

The repository uses Semantic Versioning tags. The current published line is `v0.1.0`:

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
bash ./scripts/fuzz-smoke.sh

build_dir="$(mktemp -d)"
go build -trimpath -o "${build_dir}/tracedelta" ./cmd/tracedelta
bash ./scripts/ci-compare.smoke.sh "${build_dir}/tracedelta"
bash ./scripts/run-action.smoke.sh "${build_dir}/tracedelta"
```

Expected outcomes:

| Gate | Required result |
| --- | --- |
| Essential suite | Formatting, all packages, vet, and CLI build pass. |
| Race suite | Every package passes under `-race`. |
| Fuzz smoke | Parser, matcher determinism, and reporter escaping each complete their bounded fuzz interval without a crash or invariant failure. |
| Generic CLI smoke | Pass returns `0` with JSON/HTML reports, regression returns `1` with reports, invalid/missing input returns `2` without reports. |
| Action entry smoke | Pass and regression complete with mutually consistent text/JSON/HTML reports and structured outputs; missing input returns/fails with `2` and no report. |
| Composite Action CI | Local Action invocations prove pass `0`, regression `1`, and failing tool/input `2` behavior through GitHub's actual composite-step/output semantics. |
| Deliberate example | Normal repository jobs pass; `compare-example` fails on the four documented findings after uploading all three reports. |

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

Do not publish binaries from an unreviewed developer workstation workflow and do not add credentials to the repository to automate these steps. The tag workflow uses only GitHub's scoped job token and immutable commit pins for official actions.

## Rollback and corrections

Git tags and published artifacts should be treated as immutable. If a release is defective, document the problem, mark the release appropriately on the hosting platform, and issue a corrected version. Do not silently replace an artifact under an existing version. If a credential or personal data is exposed, follow the incident and history-removal guidance in [`privacy-and-security.md`](privacy-and-security.md) and [`../SECURITY.md`](../SECURITY.md).
