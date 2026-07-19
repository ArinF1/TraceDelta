# Trace and span matching

Matching answers which baseline operation corresponds to which candidate operation. It is deliberately separate from diffing: a diff rule should not have to guess whether two spans describe the same work.

## Implemented now

The current vertical slice does **not** implement general trace matching. Normalization now preserves canonical trace boundaries and parent references, but the temporary matcher flattens those traces into one comparison set. A span's current exact identity is the tuple of `service.name`, span name, normalized span kind, and its zero-based occurrence among otherwise identical keys in canonical normalized order. This is sufficient to demonstrate added, removed, status, and duration findings, but it has important limits:

- repeated spans with the same key cannot be matched semantically;
- generated trace/span IDs do not establish cross-run identity;
- preserved parent-child and trace context are not yet scored; and
- similar traces from repeated scenarios are not paired.

These assumptions must not be described as complete OTLP matching.

## Proposed strategy

The future matcher will work in two deterministic one-to-one stages: pair traces, then pair spans inside each trace. It will normalize supported inputs first and attach evidence to each decision.

### Trace signals

Trace candidates will be grouped and ranked using stable signals in roughly this order:

1. root span name;
2. root service name;
3. root span kind;
4. normalized HTTP method and route, when present;
5. the shape of child service calls and operations;
6. selected stable resource/span attributes; and
7. occurrence order only as a final deterministic tie-breaker among otherwise equivalent repeated traces.

Exact generated IDs and wall-clock timestamps are deliberately excluded from cross-run identity.

### Span signals

Within a matched trace, span pairing will consider:

1. parent-child position relative to already matched spans;
2. service name and span name;
3. span kind;
4. HTTP method and normalized route;
5. database system and operation, without using bound values;
6. messaging system and destination;
7. an allowlisted set of stable semantic attributes; and
8. sibling occurrence order as a last, explicitly recorded tie-breaker.

Parent-child structure matters because two identically named spans under different parents may represent different behavior. Conversely, relationship changes should be detectable without forcing an otherwise strong span match to become an added/removed pair.

## Stable attributes

Only attributes classified as stable and safe should influence matching. Likely examples include normalized HTTP route, RPC service/method, database system/operation, messaging destination name, and application-defined operation identifiers that the user explicitly allowlists.

Values such as request IDs, user/customer IDs, tokens, timestamps, random message IDs, raw SQL parameters, and high-cardinality URLs must not be matching defaults. A denylist must take precedence over an allowlist, and future redaction must occur before evidence can reach a report.

## Candidate scoring and determinism

The matcher should produce candidate pairs from exact/grouping keys, score only documented signals, and choose a one-to-one assignment. Deterministic behavior requires:

- canonical input ordering before candidate generation;
- fixed signal weights or lexicographic precedence;
- stable tie-breaking independent of map iteration;
- no reuse of a span or trace in two pairs; and
- the same output for the same normalized input and configuration.

A score is an implementation detail unless it can be explained. Reports should present human-readable evidence, such as “same root service, route, and child shape,” rather than an unexplained number.

## Ambiguity and unmatched items

When two candidates remain indistinguishable after supported signals, TraceDelta should report or diagnose ambiguity instead of silently selecting an arbitrary match. Depending on configured policy, ambiguous items may be treated as unmatched or as a comparison error; that decision must be explicit and tested.

Items with no credible candidate become added or removed at the appropriate trace/span level. A low-confidence forced match is worse than an honest unmatched result because it can fabricate status, latency, or relationship changes.

## Proposed matching sequence

1. Parse and validate both exports.
2. Resolve each trace's internal parent relationships while IDs are available.
3. Normalize unstable values and canonicalize supported attributes.
4. Build trace fingerprints and candidate groups.
5. Pair unambiguous traces using ordered signals.
6. Within each pair, match root spans, then descendants using matched-parent context.
7. Run a deterministic global or sibling-level assignment where greedy pairing would reuse candidates.
8. Record matched, unmatched, and ambiguous items with evidence.
9. Pass only those results to semantic diff rules.

## Test strategy

Tests should cover different generated IDs/timestamps, repeated root operations, reordered input arrays, duplicate sibling operations, relationship changes, missing attributes, stable-attribute allowlists, denylist precedence, ambiguity, and deterministic output across repeated runs. Fixtures must remain synthetic and safe to publish.
