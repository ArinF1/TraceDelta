# GitHub Action

TraceDelta's v0.1 Action compares two caller-supplied local trace artifacts. It does not collect telemetry, download artifacts, comment on pull requests, use repository write access, or require a secret.

Pin the Action to a versioned release ref and install the matching Go toolchain before invoking it:

```yaml
- uses: actions/setup-go@v6
  with:
    go-version: "1.26.x"

- id: tracedelta
  continue-on-error: true
  uses: ArinF1/TraceDelta@v0.1.0
  with:
    baseline: artifacts/baseline.json
    candidate: artifacts/candidate.json
    report-directory: tracedelta-reports
```

The composite Action builds TraceDelta from `github.action_path`, which is the source snapshot selected by the caller's `uses: ...@REF`. It never runs a similarly named binary from the caller workspace or `PATH`. Go is the only runtime prerequisite for v0.1; release-binary consumption can be considered after the release workflow exists.

The complete [pull-request workflow example](../examples/github-action/compare.yml) uses `contents: read`, disables persisted checkout credentials, passes no secret, uploads reports after both pass and regression results, and enforces the final outcome in a separate step. `pull_request` workflows use the same unprivileged flow for fork contributions; do not replace it with `pull_request_target` or expose secrets to candidate-controlled code.

## Inputs

| Input | Required | Default | Meaning |
| --- | --- | --- | --- |
| `baseline` | Yes | none | Baseline OTLP JSON or trace-only File Exporter JSONL path. |
| `candidate` | Yes | none | Candidate OTLP JSON or trace-only File Exporter JSONL path. |
| `report-directory` | No | `tracedelta-reports` | New directory for selected reports; an existing path fails closed. |
| `duration-threshold` | No | `20%` | Minimum relative latency increase. |
| `duration-threshold-absolute` | No | `10ms` | Minimum absolute latency increase. |
| `redact-attributes` | No | empty | Newline-separated exact attribute keys added to built-in redaction. Blank lines are ignored. |

## Outputs and result handling

| Output | Values |
| --- | --- |
| `outcome` | `no_differences`, `differences`, or `error` |
| `exit-code` | `0`, `1`, or `2` |
| `report-directory` | Requested report-directory path. |
| `text-report` | Terminal text report path for a completed comparison. |
| `json-report` | JSON report path for completed comparisons. |
| `html-report` | Standalone HTML report path for completed comparisons. |

No differences and behavioral differences both complete the Action step so callers can upload the terminal text, JSON, and standalone HTML reports; inspect `exit-code` and fail the later policy gate on `1`. A tool/input failure writes `outcome=error` and `exit-code=2`, then fails the Action step. Use `continue-on-error: true` on the Action and an `if: always()` gate, as the full example does, to distinguish it from regression `1`. Upload reports only when the output code is `0` or `1`; exit `2` preserves no report.

Paths and attribute keys are passed through shell arrays rather than evaluated as shell source. Treat source traces and generated reports as sensitive despite built-in redaction: use synthetic or sanitized artifacts, short retention, and restricted artifact access, and never print their contents into the job log.
