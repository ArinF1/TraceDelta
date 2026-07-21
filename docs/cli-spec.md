# Command-line specification

This document separates the current vertical slice from planned v0.1 behavior. It is the public CLI contract unless a newer documented decision supersedes it.

## Command overview

Current:

```text
tracedelta compare
```

No collection, server, upload, or GitHub command exists. Future commands should be added only when a concrete workflow cannot remain a flag or a separate integration script.

## `tracedelta compare`

Compare one baseline trace file with one candidate trace file.

```text
tracedelta compare \
  --baseline PATH \
  --candidate PATH \
  [--duration-threshold PERCENT] \
  [--duration-threshold-absolute DURATION] \
  [--redact-attribute KEY ...] \
  [--format text|json|html] \
  [--output PATH [--force]]
```

Example:

```bash
go run ./cmd/tracedelta compare \
  --baseline testdata/baseline.json \
  --candidate testdata/candidate.json
```

The example fixtures contain differences, so this invocation is expected to exit nonzero.

### Current flags

| Flag | Required | Default | Current behavior |
| --- | --- | --- | --- |
| `--baseline PATH` | Yes | none | Reads one reference OTLP/HTTP JSON object or trace-only OTLP File Exporter JSONL stream. |
| `--candidate PATH` | Yes | none | Reads one proposed OTLP/HTTP JSON object or trace-only OTLP File Exporter JSONL stream. |
| `--duration-threshold PERCENT` | No | `20%` | Minimum relative candidate duration increase. |
| `--duration-threshold-absolute DURATION` | No | `10ms` | Minimum absolute candidate duration increase. |
| `--redact-attribute KEY` | No, repeatable | none | Adds an exact case-insensitive key to built-in redaction. |
| `--format FORMAT` | No | `text` | Supports `text`, schema-versioned `json`, and standalone `html`. |
| `--output PATH` | No | empty | Writes the complete report to a newly created file instead of standard output. |
| `--force` | No | false | Allows replacing an existing `--output`; invalid without `--output`. |

No configuration-file, arbitrary rule, plugin, or evidence-allowlist flag is accepted in v0.1.

Each input may contain one OTLP/HTTP JSON object or a trace-only OTLP File Exporter JSON Lines stream. The accepted resource/scope/span fields, recursive attribute forms, event/link validation, compatibility extensions, and explicit exclusions are documented in [`testdata/README.md`](../testdata/README.md).

### Duration threshold

The relative threshold is a percentage with a trailing `%`, such as `20%` or `12.5%`. The absolute threshold is a non-negative Go duration such as `10ms`, `250us`, or `1s`; unitless nonzero values are invalid. Both thresholds must be met or exceeded, and only candidate increases are findings. Negative, non-finite, malformed, or overflowing values are usage errors.

The defaults are `20%` and `10ms`. A zero baseline satisfies the relative side for any positive candidate, but must still meet the absolute threshold. Setting either threshold to zero disables only that side; setting both to zero reports every positive increase. Decreases and equal durations never produce latency findings. Each duration finding includes the effective thresholds.

The normalization stage also has a distinct typed duration bucket for Go embedding callers. It floors each duration to a non-negative `time.Duration` width before matching; zero disables bucketing and is the current default. The CLI does not yet expose this option, and `--duration-threshold` remains solely the relative finding threshold rather than a normalization setting.

### Output and ordering

Without `--output`, successful comparison output is written to standard output. With `--output`, TraceDelta renders the complete report in memory before creating the destination with owner-only requested permissions. An existing path fails closed and remains unchanged unless `--force` is present. Output may never refer to either input path, including an existing filesystem alias. Parent directories are not created automatically. A comparison or rendering failure creates no new output file; file errors use exit `2`. A successfully written regression report still uses exit `1`.

Usage, read, parse, render, and output diagnostics are written to standard error. The text report contains:

1. a heading and the two input paths;
2. summary counts;
3. an ordered change list; and
4. a result statement.

Changes are ordered deterministically by change category and then stable span key; multiple changes for one span use a stable rule order. Determinism is a compatibility requirement even if precise presentation evolves before v0.1.

Current error findings preserve status transitions as `status BEFORE -> AFTER`. When both matched spans have `ERROR` status, a changed string `error.type` produces a separate `error.type BEFORE -> AFTER` finding. Missing and present-empty evidence render as `(missing)` and `(empty)`. Status messages, exception messages, and stack traces are never finding evidence.

`--format json` writes one `tracedelta.report/v1` object with input path metadata, summary counts, the shared ordered findings/evidence, and a `no_differences`/`differences` result whose embedded exit code matches the process. Tool/input errors produce exit `2` and no successful report object. The exact fields and compatibility policy are documented in [`json-report-schema.md`](json-report-schema.md).

`--format html` writes one complete UTF-8 HTML document with inline CSS, no scripts, and no external resources. It contains the same inputs, summary counts, ordered findings, evidence, and result outcome as text/JSON. Dynamic values are control-normalized and escaped with Go's contextual HTML templating. The layout is responsive and works offline; use `--output report.html` to save it without sending report content to the terminal.

### Additional redacted keys

Each `--redact-attribute KEY` adds one organization-specific exact key to the conservative built-in rules. The flag can be repeated; surrounding whitespace is removed and comparison is case-insensitive. Empty keys are usage errors. Additional rules always win over the fixed safe evidence set, so `--redact-attribute http.route`, `--redact-attribute error.type`, or `--redact-attribute service.name` removes that evidence before matching/diffing. The option narrows evidence only; no CLI option can add a new evidence key. See [`privacy-and-security.md`](privacy-and-security.md) for built-in coverage and residual risk.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Comparison completed and found no meaningful differences. |
| `1` | Comparison completed and found behavioral differences under the current policy. |
| `2` | Invalid command/flags, unsupported format, unreadable input, or parse/validation failure. |

Behavioral differences are a successful comparison result, not a parse error. Scripts must treat `1` separately from `2`.

When invoked through `go run`, the Go tool may print `exit status 1` and itself return a wrapper-specific nonzero code. CI should build/use the TraceDelta binary when it needs the exact documented process code.

Configuration files and arbitrary rule/plugin configuration are outside v0.1. Typed Go options remain explicit values passed through orchestration rather than globals.

## Generic CI invocation

Use the repository's provider-neutral [`scripts/ci-compare.sh`](../scripts/ci-compare.sh) wrapper when a CI job needs both JSON and HTML artifacts without writing report content to its log. It accepts four positional arguments—a built TraceDelta binary, a baseline file, a candidate file, and a new report-directory path—followed only by value pairs for `--duration-threshold`, `--duration-threshold-absolute`, or repeatable `--redact-attribute`. It performs the same comparison once per selected format and verifies that both runs agree.

The wrapper preserves `tracedelta-report.json` and `tracedelta-report.html` for completed comparisons, returning `0` for no findings and `1` for findings. It returns `2` for missing files, invalid inputs, tool/output failure, absent reports, or inconsistent results and removes any partial reports. The report directory must not already exist. A complete copyable CI gate and artifact-upload guidance are in [`examples/ci/README.md`](../examples/ci/README.md).
