# Trace fixtures and supported OTLP JSON

Every file in this directory is synthetic and safe to publish. Do not replace
these fixtures with production, customer, or otherwise sensitive telemetry.

`baseline.json` and `candidate.json` are the original comparison fixtures.
They remain deliberately small compatibility cases and use symbolic enum names
accepted by TraceDelta's first parser. `otlp-representative.json` exercises
canonical numeric enums, resource and instrumentation-scope context, primitive
typed attributes, exact 64-bit values, and safe unknown fields.
`otlp-file.jsonl` is a two-record trace-only OTLP File Exporter stream covering
multiple resources, scopes, traces, nested values, events, and links.
`normalize-run-a.json` and `normalize-run-b.json` describe equivalent synthetic
operations with regenerated IDs, shifted timestamps, reordered traces/spans and
attributes, different unselected request IDs, and durations that share explicit
10-nanosecond buckets.

## Supported document shapes

TraceDelta accepts either:

- one lower-camel-case OTLP/HTTP JSON `ExportTraceServiceRequest` object; or
- a trace-only OTLP File Exporter JSON Lines stream whose records are
  lower-camel-case `TracesData` objects.

Both shapes use this trace envelope:

```text
resourceSpans[]
  resource
  scopeSpans[]
    scope
    spans[]
```

The envelope, `resource`, `scope`, and repeated arrays may be omitted, `null`,
or empty where ProtoJSON treats them as unset. Multiple JSONL records are
merged into one snapshot. A trace may span records, record order is not
identity, and a repeated span ID within the same trace is rejected even when
the duplicate occurs in another record.

For each non-empty span, the current profile supports:

- a required, nonzero, case-insensitive 32-character hexadecimal `traceId`;
- a required, nonzero, case-insensitive 16-character hexadecimal `spanId`;
- an optional nonzero `parentSpanId` in the same 16-character form;
- optional `traceState` and unsigned 32-bit `flags` context;
- a required non-empty `name`;
- `kind` as canonical OTLP integer `0` through `5`;
- required `startTimeUnixNano` and `endTimeUnixNano` unsigned 64-bit values,
  with the end not before the start and a derived duration no greater than
  `9223372036854775807` nanoseconds (Go's maximum `time.Duration`);
- attributes, events, links, and their dropped-item counts; and
- optional status message plus canonical OTLP integer status code `0`, `1`, or
  `2`.

The numeric span kinds map to `UNSPECIFIED`, `INTERNAL`, `SERVER`, `CLIENT`,
`PRODUCER`, and `CONSUMER`. Status codes map to `UNSET`, `OK`, and `ERROR`.
Omitted kind/status values use their zero value. The symbolic `SPAN_KIND_*` and
`STATUS_CODE_*` names in `baseline.json` and `candidate.json` remain accepted
only as a TraceDelta compatibility extension; canonical OTLP JSON encodes enums
as integers.

ProtoJSON writers normally emit 64-bit integers as quoted decimal strings.
TraceDelta accepts quoted or unquoted exact integer forms, including integral
decimal/exponent spellings, and range-checks them without converting through a
floating-point value.

## Resource, scope, attributes, events, and links

Resource and instrumentation-scope attributes are retained with each parsed
span. `service.name` is optional because OTLP permits a missing resource; when
present it must be a non-empty `stringValue`. The scope name, version,
attributes, dropped-attribute count, and resource/scope schema URLs are also
retained.

The supported recursive `AnyValue` forms are:

- `stringValue`;
- `boolValue`;
- signed 64-bit `intValue` as an exact quoted or unquoted integer;
- `doubleValue`, including the ProtoJSON strings `NaN`, `Infinity`, and
  `-Infinity`;
- `bytesValue` using standard or URL-safe base64, padded or unpadded;
- `arrayValue`; and
- `kvlistValue`.

Attribute keys must be non-empty and unique within a resource, scope, span, or
event attribute list. Each value must contain exactly one `AnyValue` case.
Nested values are limited to 64 levels. Types and list order are preserved in
the parsed domain model. Key/value-list entries retain their encoded order and
may repeat keys because the OTLP value is a list rather than a map.

Event timestamps, attributes, and dropped counts and link IDs, trace state,
attributes, dropped counts, and flags are validated. v0.1 does not compare
event or link payloads, so those payloads are deliberately discarded after
validation and cannot become match or report evidence. Span-level dropped
event/link counts remain in the parsed model but are not v0.1 findings.

Normalization projects only scalar values from the fixed safe matching keys.
If one of those keys contains an array or key/value list, it is omitted from
matching evidence rather than serialized or exposed.

## Compatibility and explicit exclusions

OTLP JSON requires receivers to ignore unknown message-field names. TraceDelta
therefore ignores safe unknown fields while still validating every field it
uses. A misspelled required span field remains an error because the required
value is then absent.

Metrics, logs, profiles, protobuf-binary input, exporter-specific outer
envelopes, and mixed-signal documents are not supported. Recognized non-trace
signals and common wrapper fields fail contextually instead of being mistaken
for an empty trace export. This is a tested v0.1 profile, not a claim of
complete OTLP or arbitrary vendor-exporter compatibility.

The encoding rules follow the official
[OTLP JSON specification](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding),
[OpenTelemetry Protocol File Exporter specification](https://opentelemetry.io/docs/specs/otel/protocol/file-exporter/),
and [ProtoJSON mapping](https://protobuf.dev/programming-guides/json/).

## Comparison fixtures

The candidate compatibility fixture intentionally changes trace/span IDs and
all absolute timestamps. Those fields are normalized away. Its comparison
contains:

- unchanged `auth.validate`;
- added `inventory.reserve`;
- removed `cache.get`;
- `checkout.handle` status changing from `OK` to `ERROR`; and
- `payment.charge` increasing from `120ms` to `245ms`.

The two normalization fixtures are expected to produce byte-equivalent
normalized snapshots only when the typed duration bucket is set to `10ns`.
They prove the normalization contract rather than define a new accepted wire
shape. Their `request.id` values deliberately differ and are not part of the
fixed stable matching projection; parsed snapshots still retain those values,
so this behavior must not be described as redaction or anonymization.
