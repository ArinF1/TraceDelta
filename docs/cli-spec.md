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
  [--format text] \
  [--output PATH]
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
| `--baseline PATH` | Yes | none | Reads the reference simplified OTLP-compatible JSON file. |
| `--candidate PATH` | Yes | none | Reads the proposed simplified OTLP-compatible JSON file. |
| `--duration-threshold PERCENT` | No | `20%` | Reports candidate duration increases that meet the relative threshold. |
| `--format FORMAT` | No | `text` | Supports `text` only. Any other value is an invalid invocation. |
| `--output PATH` | No | empty | Recognized but currently rejected as not implemented; standard output is the only destination. |

`--output PATH` is recognized but planned and not implemented in the initial slice. Passing a non-empty path returns a clear invalid-invocation error. Until output-file semantics are implemented, text is written to standard output and users may use normal shell redirection.

### Duration threshold

The threshold is a percentage, written with a trailing `%`, such as `20%` or `12.5%`. It applies to increases, not decreases. A duration change is meaningful when the candidate is slower and its relative increase meets the configured threshold. Invalid, negative, or non-percentage values are usage errors.

The current percentage-only policy is intentionally narrow. Absolute tolerances and combined policies remain planned because short spans can otherwise produce misleading percentages.

### Output and ordering

Successful comparison output is written to standard output. Usage, read, and parse diagnostics are written to standard error. The text report contains:

1. a heading and the two input paths;
2. summary counts;
3. an ordered change list; and
4. a result statement.

Changes are ordered deterministically by change category and then stable span key; multiple changes for one span use a stable rule order. Determinism is a compatibility requirement even if precise presentation evolves before v0.1.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Comparison completed and found no meaningful differences. |
| `1` | Comparison completed and found behavioral differences under the current policy. |
| `2` | Invalid command/flags, unsupported format, unreadable input, or parse/validation failure. |

Behavioral differences are a successful comparison result, not a parse error. Scripts must treat `1` separately from `2`.

When invoked through `go run`, the Go tool may print `exit status 1` and itself return a wrapper-specific nonzero code. CI should build/use the TraceDelta binary when it needs the exact documented process code.

## Planned v0.1 additions

- `--format json` and `--format html` backed by the same comparison result.
- `--output PATH` with explicit overwrite and standard-output behavior.
- Configuration-file support for normalization, thresholds, allowlists/denylists, and regression policy.
- Machine-readable diagnostics that do not confuse a regression (`1`) with a tool failure (`2`).

Planned syntax is not a compatibility promise until implemented, tested, and moved into the current section.
