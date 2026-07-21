# Generic CI comparison

Build TraceDelta, then run the provider-neutral wrapper with two already-collected trace artifacts and a new report-directory path:

```bash
go build -trimpath -o ./tracedelta ./cmd/tracedelta

set +e
bash ./scripts/ci-compare.sh \
  ./tracedelta \
  ./artifacts/baseline.json \
  ./artifacts/candidate.json \
  ./tracedelta-reports \
  --duration-threshold 20% \
  --duration-threshold-absolute 10ms \
  --redact-attribute organization.internal_id
tracedelta_status=$?
set -e

case "${tracedelta_status}" in
  0)
    echo "TraceDelta passed"
    ;;
  1)
    echo "TraceDelta found behavioral differences"
    exit 1
    ;;
  *)
    echo "TraceDelta could not complete" >&2
    exit 2
    ;;
esac
```

Configure the CI system's artifact-upload step to run even when this step exits `1`, and upload only `tracedelta-reports/tracedelta-report.json` and `tracedelta-reports/tracedelta-report.html`. Do not print either input or report file into the job log. The wrapper itself logs only status and artifact paths.

The report directory must not already exist, which prevents an old report from being mistaken for the current run. Exit `0` and `1` both preserve complete JSON and HTML reports. Exit `2` removes any partial report, because it means the binary, an input, parsing, rendering, or output handling failed.

## Troubleshooting

- `binary is missing or not executable`: build the platform-appropriate TraceDelta binary and pass its path as the first argument.
- `baseline artifact is missing` or `candidate artifact is missing`: verify that the trace-producing/download step completed and that the wrapper receives a regular file, not an artifact directory.
- A TraceDelta parse diagnostic followed by `exit 2`: the artifact exists but is not valid supported OTLP JSON/JSONL. Fix the producer; do not dump the trace into CI logs.
- `report directory must be a new path`: remove or rename the directory in a controlled cleanup step before comparison. The wrapper will not overwrite an unknown artifact directory.
- Exit `1`: this is a completed comparison with findings, not a tool crash. Preserve both reports, then fail the policy gate.
- Exit `2`: do not publish reports from that attempt. Preserve the concise standard-error diagnostic as the job failure and investigate the producer or tool invocation.

Run the real-binary smoke test with:

```bash
go build -trimpath -o ./tracedelta ./cmd/tracedelta
bash ./scripts/ci-compare.smoke.sh ./tracedelta
```
