# v0.1 demonstration

[Watch the 52-second TraceDelta v0.1 demonstration](https://github.com/ArinF1/TraceDelta/releases/download/v0.1.0/tracedelta-v0.1-demo.webm).

The silent 1280×720 WebM uses only deterministic synthetic data. It shows the exact four findings and check outcomes from public draft [PR #7](https://github.com/ArinF1/TraceDelta/pull/7) and its retained `example-tracedelta-reports` workflow artifact. It contains no credentials, production traces, personal data, narration, external media, or analytics.

Published asset:

```text
name:     tracedelta-v0.1-demo.webm
duration: 52 seconds
bytes:    4,309,214
sha256:   e7cbb9841b58df4da9ae99c5763996a71078ca75071050041a0697307961af1b
```

## On-screen transcript

- **00:00–00:06 — Scope:** TraceDelta v0.1 compares OTLP JSON deterministically, builds redacted evidence, and produces terminal, JSON, and standalone HTML reports.
- **00:06–00:14 — Inputs:** The synthetic baseline contains `checkout.handle` in `OK`, a 40 ms `payment.charge`, and `cache.get`. The candidate changes checkout to `ERROR`, raises payment to 70 ms, removes the cache call, and adds `inventory.reserve`.
- **00:14–00:25 — Terminal:** The command reports one added span, one removed span, and two changed spans. Exit `1` means the comparison completed and found behavioral differences.
- **00:25–00:33 — Reports:** The same four findings are preserved as terminal text, `tracedelta.report/v1` JSON, and a standalone offline HTML document.
- **00:33–00:41 — HTML:** The HTML artifact shows the summary and all four escaped findings without scripts, external assets, or a server.
- **00:41–00:49 — Action:** Normal Linux and Windows checks pass on PR #7. `compare-example` fails intentionally after uploading the `.txt`, `.json`, and `.html` reports.
- **00:49–00:52 — Release:** v0.1.0 provides five platform binaries, SHA-256 checksums, a reusable Action, and redacted reports.

## Reproduce the comparison

The public demonstration branches regenerate both trace exports rather than storing generated traces in Git. From a clone with Go 1.26 and Bash:

```bash
git fetch origin codex/v0.1-foundation codex/example-regression

demo_root="$(mktemp -d)"
git worktree add --detach "${demo_root}/baseline-source" origin/codex/v0.1-foundation
git worktree add --detach "${demo_root}/candidate-source" origin/codex/example-regression

(cd "${demo_root}/baseline-source" && \
  go run ./examples/regression-app --output "${demo_root}/baseline.json")
(cd "${demo_root}/candidate-source" && \
  go run ./examples/regression-app --output "${demo_root}/candidate.json")
(cd "${demo_root}/baseline-source" && \
  go build -trimpath -o "${demo_root}/tracedelta" ./cmd/tracedelta)

set +e
bash "${demo_root}/baseline-source/scripts/ci-compare.sh" \
  "${demo_root}/tracedelta" \
  "${demo_root}/baseline.json" \
  "${demo_root}/candidate.json" \
  "${demo_root}/reports" >"${demo_root}/tracedelta-report.txt"
status=$?
set -e
test "${status}" -eq 1

ls -l "${demo_root}/tracedelta-report.txt" "${demo_root}/reports/"*
```

The expected findings are `ADDED inventory.reserve`, `REMOVED cache.get`, `checkout.handle OK -> ERROR`, and `payment.charge 40ms -> 70ms` under the default `20%` and `10ms` thresholds.

## Reproduce the recording

On Windows with Microsoft Edge installed, render the self-contained recording source and verify its media duration:

```powershell
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass `
  -File .\scripts\record-demo.ps1 `
  -OutputDirectory C:\tmp\tracedelta-demo
```

The script refuses to overwrite an existing output, records the local canvas in real time, repairs streamed WebM duration metadata, and rejects any recording outside 45–60 seconds. The source is [`demo/recording.html`](demo/recording.html). Its local duration repair helper is vendored from MIT-licensed [`fix-webm-duration` 1.0.6](https://github.com/yusitnikov/fix-webm-duration); the upstream license is retained beside the source.
