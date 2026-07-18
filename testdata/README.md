# Trace fixtures and supported OTLP JSON

Every file in this directory is synthetic and safe to publish. Do not replace
these fixtures with production, customer, or otherwise sensitive telemetry.

`baseline.json` and `candidate.json` are the original comparison fixtures.
They remain deliberately small compatibility cases and use symbolic enum names
accepted by TraceDelta's first parser. `otlp-representative.json` exercises the
canonical numeric enum encoding, resource and instrumentation-scope context,
primitive typed attributes, exact 64-bit values, and safe unknown fields.

## Supported document shape

TraceDelta accepts exactly one JSON object with this lower-camel-case OTLP
trace envelope:

```text
resourceSpans[]
  resource
  scopeSpans[]
    scope
    spans[]
```

The envelope, `resource`, `scope`, and repeated arrays may be omitted, `null`,
or empty where ProtoJSON treats them as unset. Such an envelope parses as zero
or more traces. Multiple top-level JSON objects and JSON Lines export streams
are not supported yet.

For each non-empty span, the current subset supports:

- a required, nonzero, case-insensitive 32-character hexadecimal `traceId`;
- a required, nonzero, case-insensitive 16-character hexadecimal `spanId`;
- an optional nonzero `parentSpanId` in the same 16-character form;
- optional `traceState` and unsigned 32-bit `flags` context;
- a required non-empty `name`;
- `kind` as canonical OTLP integer `0` through `5`;
- required `startTimeUnixNano` and `endTimeUnixNano` unsigned 64-bit values,
  with the end not before the start and a derived duration no greater than
  `9223372036854775807` nanoseconds (Go's maximum `time.Duration`);
- primitive attributes and dropped-item counts; and
- optional status message plus canonical OTLP integer status code `0`, `1`, or
  `2`.

The numeric span kinds map to `UNSPECIFIED`, `INTERNAL`, `SERVER`, `CLIENT`,
`PRODUCER`, and `CONSUMER`. Status codes map to `UNSET`, `OK`, and `ERROR`.
Omitted kind/status values use their zero value. The symbolic
`SPAN_KIND_*` and `STATUS_CODE_*` names in `baseline.json` and
`candidate.json` remain accepted only as a TraceDelta compatibility extension;
canonical OTLP JSON encodes enums as integers.

ProtoJSON writers normally emit 64-bit integers as quoted decimal strings.
TraceDelta accepts quoted or unquoted exact integer forms, including integral
decimal/exponent spellings, and range-checks them without converting through a
floating-point value.

## Resource, scope, and attributes

Resource and instrumentation-scope attributes are retained with each parsed
span. `service.name` is optional because OTLP permits a missing resource; when
present it must be a non-empty `stringValue`. The scope name, version,
attributes, dropped-attribute count, and resource/scope schema URLs are also
retained.

The supported primitive `AnyValue` forms are:

- `stringValue`;
- `boolValue`;
- signed 64-bit `intValue` as an exact quoted or unquoted integer;
- `doubleValue`, including the ProtoJSON strings `NaN`, `Infinity`, and
  `-Infinity`;
- `bytesValue` using standard or URL-safe base64, padded or unpadded.

Attribute keys must be non-empty and unique within their attribute list. Each
attribute must contain exactly one supported primitive value; types are
preserved in the domain model rather than converted to text.

## Compatibility and explicit exclusions

OTLP JSON requires receivers to ignore unknown message-field names. TraceDelta
therefore ignores safe unknown fields while still validating every field it
uses. A misspelled required span field remains an error because the required
value is then absent.

The following valid OTLP constructs are outside this task's intentionally
bounded subset and produce contextual errors when non-empty:

- span events;
- span links;
- `arrayValue` attributes;
- `kvlistValue` attributes; and
- attribute `AnyValue` objects with no supported populated case or with multiple populated cases.

Other signals, protobuf-binary input, multiple JSON/JSONL export records, and
arbitrary exporter-specific file wrappers are not supported. This is a tested
subset, not a claim of complete OTLP compatibility. The encoding rules follow
the [OTLP JSON specification](https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding)
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
