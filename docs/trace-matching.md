# Trace and span matching

Matching answers which baseline operation corresponds to which candidate operation. It is deliberately separate from diffing: a diff rule should not have to guess whether two spans describe the same work.

## Implemented now

Trace matching now runs before span matching. It groups normalized traces by their root operation set: root span name, service, kind, and the fixed safe HTTP/RPC attribute projection. Within a repeated-operation group it first pairs unique exact structures, then accepts only mutual unique-best structural-overlap pairs. Status, duration, generated IDs, wall-clock timestamps, and input order never establish trace identity.

Each trace pair records the non-sensitive signal categories that justified it plus the structural-overlap count; trace-derived attribute values are not copied into evidence. Groups present on only one side become added or removed traces. If multiple candidates remain indistinguishable, matching returns a typed ambiguity error with counts and signal categories rather than guessing or exposing trace values.

Span matching runs separately inside each matched trace. It pairs roots and external-parent spans by semantic identity and top-level relationship, then walks descendants using the already matched parent as context. Semantic identity is service/name/kind plus the fixed safe attribute projection; status and duration remain diff evidence rather than identity. Repeated siblings with distinct normalized start ranks pair by exact rank or deterministic sibling order. Candidate spans are marked used and cannot appear in two pairs.

After parent-aware matching stops, a unique semantic span may pair through an explicit relationship-change fallback. This avoids fabricating an added/removed pair when only its parent changed; relationship changes themselves are outside the v0.1 finding set. Every span left in an unmatched trace or unmatched inside a paired trace becomes added or removed.

Indistinguishable duplicate siblings return a typed error containing counts and signal categories only. Generated IDs, trace-derived names, and attribute values do not appear in ambiguity diagnostics.

### Trace signals

Trace candidates are grouped and ranked using stable signals in this order:

1. root span name;
2. root service name;
3. root span kind;
4. normalized HTTP method and route, when present;
5. exact span/parent shape when unique; and
6. mutual unique-best span/parent shape overlap for remaining repeated traces.

Occurrence order is not used to break an otherwise unresolved trace tie. Exact generated IDs and wall-clock timestamps are deliberately excluded from cross-run identity.

### Span signals

Within a matched trace, span pairing uses:

1. parent-child position relative to already matched spans;
2. service name and span name;
3. span kind;
4. HTTP method and normalized route;
5. the fixed safe HTTP/RPC attribute projection; and
6. normalized sibling start/occurrence order only when otherwise identical siblings remain distinguishable.

Parent-child structure prevents two identically named spans under different parents from cross-pairing. Conversely, the relationship fallback keeps an otherwise unique strong span match from becoming a misleading added/removed pair.

## Stable attributes

Only attributes in the fixed, documented safe evidence set influence v0.1 matching. The set is limited to normalized HTTP route/method and RPC service/method. A string `error.type` is retained separately as comparison evidence and never participates in trace/span identity. Database, messaging, and application-defined matching attributes are deferred until their privacy and stability contracts are justified.

Values such as request IDs, user/customer IDs, tokens, timestamps, random message IDs, raw SQL parameters, and high-cardinality URLs are not matching defaults. Built-in and caller-supplied deny rules run before normalization and take precedence over the fixed safe evidence set; denied values therefore cannot enter match keys or evidence.

## Candidate scoring and determinism

The trace matcher produces candidate pairs from exact root grouping keys, scores only the documented normalized span/parent tokens, and accepts mutual unique-best pairs. Deterministic behavior requires:

- canonical input ordering before candidate generation;
- fixed signal weights or lexicographic precedence;
- stable ordering independent of map and input iteration;
- no reuse of a span or trace in two pairs; and
- the same output for the same normalized input and configuration.

A score is an implementation detail unless it can be explained. Reports should present human-readable evidence, such as “same root service, route, and child shape,” rather than an unexplained number.

## Ambiguity and unmatched items

When trace or span candidates remain indistinguishable after supported signals, TraceDelta returns a typed comparison error. The current diagnostic reports only baseline/candidate counts and the safe signal categories considered. Configurable ambiguity policy and richer diagnostics remain after v0.1 under TD-017.

Items with no credible candidate become added or removed at the appropriate trace/span level. A low-confidence forced match is worse than an honest unmatched result because it can fabricate status, latency, or relationship changes.

## Matching sequence

1. Parse and validate both exports.
2. Resolve each trace's internal parent relationships while IDs are available.
3. Normalize unstable values and canonicalize supported attributes.
4. Build trace fingerprints and candidate groups.
5. Pair unambiguous traces using exact structure and mutual unique structural overlap.
6. Fail with a non-sensitive typed diagnostic when a trace group remains ambiguous.
7. Within each pair, match top-level spans and descendants by semantic identity plus matched-parent context.
8. Add or remove every span from an unmatched trace.
9. Pair otherwise unique reparented spans through the documented relationship fallback, then mark remaining spans added or removed before semantic diff rules run.

## Test strategy

Tests should cover different generated IDs/timestamps, repeated root operations, reordered input arrays, duplicate sibling operations, missing attributes, the fixed safe evidence set, denylist precedence, ambiguity, and deterministic output across repeated runs. Fixtures must remain synthetic and safe to publish.
