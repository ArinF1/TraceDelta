# Basic comparison

Run this example from the repository root:

```bash
go run ./cmd/tracedelta compare \
  --baseline testdata/baseline.json \
  --candidate testdata/candidate.json
```

The report shows one added span, one removed span, one status change, and one
duration regression. Because meaningful differences are present, the
TraceDelta program exits with status `1`. The `go run` wrapper may print
`exit status 1`; this is an expected comparison result, not a parsing failure.

The built TraceDelta binary exits `0` when comparing `baseline.json` with
itself. Invalid arguments, unreadable files, malformed input, and unsupported
output paths produce binary exit code `2`. Build the CLI before asserting
exact exit codes in automation; `go run` maps child-process failures through
the Go tool.

To preserve a standalone report without writing it into the terminal log:

```bash
tracedelta compare \
  --baseline testdata/baseline.json \
  --candidate testdata/candidate.json \
  --format html \
  --output tracedelta-report.html
```

TraceDelta refuses to replace that file unless `--force` is also supplied.
Repeat `--redact-attribute KEY` for organization-specific sensitive attributes;
these entries can only remove evidence and cannot broaden the fixed safe set.
