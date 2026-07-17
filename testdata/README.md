# Trace fixtures

`baseline.json` and `candidate.json` are synthetic, safe-to-publish fixtures for
the initial comparison example. They contain no production or personal data.

The files use a strict subset of the OTLP JSON trace export shape:

- a top-level `resourceSpans` array;
- `resource.attributes` with a non-empty `service.name` string;
- a `scopeSpans` array containing `spans`;
- hexadecimal `traceId`, `spanId`, and optional `parentSpanId` values;
- decimal-string `startTimeUnixNano` and `endTimeUnixNano` timestamps;
- standard OTLP span-kind and status-code names; and
- primitive string, Boolean, integer, double, or bytes attribute values.

The parser rejects unknown fields and unsupported OTLP structures rather than
silently dropping data. Events, links, array values, key-value-list values, and
other portions of the full OTLP schema are not supported yet.

The candidate fixture intentionally changes the trace and span IDs and all
absolute timestamps. Those fields are normalized away. The comparison contains:

- unchanged `auth.validate`;
- added `inventory.reserve`;
- removed `cache.get`;
- `checkout.handle` status changing from `OK` to `ERROR`; and
- `payment.charge` increasing from `120ms` to `245ms`.
