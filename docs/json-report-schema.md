# JSON report schema

TraceDelta's machine-readable report uses schema identifier
`tracedelta.report/v1`. The CLI emits it with `--format json`; Go callers can
use `tracedelta.WriteJSON`. Output is UTF-8 JSON followed by one newline and is
deterministic for a deterministic comparison result.

## Top-level object

```json
{
  "schemaVersion": "tracedelta.report/v1",
  "inputs": {
    "baseline": { "path": "baseline.json" },
    "candidate": { "path": "candidate.json" }
  },
  "summary": {
    "addedSpans": 0,
    "removedSpans": 0,
    "changedSpans": 0,
    "findings": 0
  },
  "result": {
    "status": "no_differences",
    "exitCode": 0
  },
  "findings": []
}
```

- `inputs` contains the path labels supplied by the caller. These can expose
  local directory names and should be reviewed before publishing a report.
- summary span counts match the terminal report. `findings` is the number of
  entries in the findings array; one changed span may have multiple findings.
- `result.status` is `no_differences` with `exitCode` `0`, or `differences`
  with `exitCode` `1`. Tool/input failures use process exit `2` and do not
  produce a successful report object.
- `findings` is always an array, including when empty, and preserves the shared
  comparison model's deterministic order.

## Finding object

Every finding has the same shape:

```json
{
  "kind": "CHANGED",
  "field": "duration",
  "span": {
    "name": "payment.charge",
    "serviceName": "payments",
    "occurrence": 0
  },
  "evidence": {
    "before": { "present": true, "value": "120ms" },
    "after": { "present": true, "value": "245ms" },
    "durationThreshold": {
      "present": true,
      "relativeRatio": 0.2,
      "absolute": "10ms"
    }
  }
}
```

- `kind` is `ADDED`, `REMOVED`, or `CHANGED`.
- `field` is empty for added/removed spans, otherwise `status`, `error.type`,
  or `duration`.
- `span` contains only the safe normalized name, service name, and
  deterministic occurrence.
- `before.present` and `after.present` distinguish missing evidence from a
  present empty string. Added/removed findings have both set to false.
- `durationThreshold.present` is true only for duration findings. Its relative
  value is a ratio (`0.2` means 20 percent) and its absolute value is a Go
  duration string. When false, the other two fields are zero/empty.

## Safety and compatibility

The JSON reporter consumes only the post-redaction comparison model. It never
re-reads parsed traces, status messages, exception content, or arbitrary
attributes. Encoding uses Go's JSON encoder, including escaping control and
HTML-significant characters. This reduces accidental disclosure but does not
make span/service names or input paths anonymous.

The complete document is buffered before it is written, so an encoding error
does not leave a partial report. Consumers should require the schema identifier
they understand and may ignore additional fields within v1. Removing or
changing the meaning/type of an existing v1 field requires a new schema major
identifier and release documentation.
