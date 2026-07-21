# Privacy and security

OpenTelemetry traces can contain credentials, session tokens, personal information, customer identifiers, internal hostnames, raw URLs, query text, request/response content, and other sensitive operational data. Treat every trace file as sensitive unless it was deliberately synthesized or reviewed for publication.

## Local-first processing

TraceDelta's v0.1 design reads local files and produces local/stdout reports. It does not require a hosted service, remote database, analytics endpoint, or network upload. Local-first reduces exposure; it does not make unsafe files harmless. Users remain responsible for filesystem permissions, CI logs, retained artifacts, and who can read generated reports.

No future network behavior should be introduced silently. Any integration that transmits trace-derived data requires an explicit architecture/security decision, opt-in behavior, destination documentation, and tests.

## Safe fixtures and development data

- Commit only synthetic trace fixtures created for the repository.
- Never copy raw production traces into issues, pull requests, testdata, snapshots, or session logs.
- Do not use real names, emails, account/customer IDs, tokens, internal domains, or proprietary query data in examples.
- Review generated reports before attaching them to CI artifacts or public issues.
- If sensitive data is committed, stop sharing it, follow the host's secret-removal procedure, rotate exposed credentials, and report the incident through the security process.

The repository's included fixtures are intended to be obviously fictional and safe to publish.

## Attribute evidence and denylists

TraceDelta applies deterministic key-based redaction immediately after parsing and before normalization, matching, diffing, or reporting. It deep-copies both parsed snapshots and removes denied entries from span, resource, scope, and recursively nested key/value-list attributes. Arrays are traversed so nested key/value lists are covered. Redaction does not mutate caller-owned parsed values.

Built-in rules are case-insensitive and conservatively remove:

- all `http.request.header.*` and `http.response.header.*` attributes;
- `user.*`, `enduser.*`, `person.*`, `session.*`, `account.*`, `contact.*`, `customer.*`, and `device.*` namespaces;
- raw address, URL, query/statement, connection-string, user-agent, error-message, and exception-message/stack keys documented in the implementation; and
- keys whose normalized segments contain common credential or personal-data markers such as API key, authorization, cookie, credential, email, password, phone, private key, secret, SSN, or token.

Go embedding callers can add exact denylisted keys through `tracedelta.Options.RedactedAttributeKeys`. Matching is case-insensitive and surrounding whitespace in configured keys is ignored. An empty configured key fails without rendering trace values. Caller rules always override the fixed safe evidence set: denying `http.route`, an RPC key, `error.type`, or `service.name` removes it before evidence construction. Denying `service.name` also clears the derived service field.

CLI callers can supply the same narrowing policy with repeatable `--redact-attribute KEY` flags. There is intentionally no option to add evidence keys or load an arbitrary policy/configuration file.

The fixed matching set remains `http.request.method`, `http.route`, `rpc.method`, and `rpc.service`. A string `error.type` is safe error evidence but is excluded from identity; non-string forms are treated as missing and callers may deny the key. Composite values under safe keys are not serialized into evidence. General user-defined evidence allowlists are deferred because they can silently broaden disclosure.

Removal rather than a visible placeholder is intentional: a constant replacement could still make a caller-denied key influence correspondence. Text, JSON, and HTML consume the sanitized comparison-result model rather than parsed spans. JSON escapes control and HTML-significant characters; HTML uses contextual templating, inline CSS, and no active/external content. Input paths and safe span/service metadata can still be sensitive.

## Residual risk

Key-based redaction reduces accidental disclosure; it is not anonymization. Custom attributes can carry sensitive values under harmless-looking keys, and span names, service names, route templates, RPC names, file paths, schema URLs, trace state, status text, or other non-attribute fields can themselves be sensitive or malformed. The current comparison model drops most of those fields, and status messages are not evidence, but users must still sanitize source telemetry and use `RedactedAttributeKeys` for organization-specific attributes.

Parser validation occurs before redaction and may identify a failing field or attribute key, though it does not dump attribute values. Input trace files remain sensitive on disk. Replacing a value only in a renderer would be insufficient, which is why the implemented stage sits before every evidence-producing component.

## Threat model

TraceDelta should assume:

- input files are untrusted and may be malformed, deeply nested, oversized, or crafted to exhaust CPU/memory;
- attribute strings may contain terminal controls, newlines, markup, spreadsheet formulas, or HTML/script content;
- file paths and parser errors may expose environment details in shared CI logs;
- generated reports may be published more broadly than the source trace;
- configuration can accidentally broaden data exposure; and
- pull requests, especially from forks, may control fixtures or command arguments without being trusted with repository secrets.

Mitigations include strict typed parsing, bounded resource use, contextual but non-dumping errors, terminal-safe rendering, HTML escaping, JSON encoding through standard libraries, fail-closed filtering, safe output-file behavior, least-privilege CI permissions, and comprehensive adversarial tests. Some mitigations are planned and must not be represented as implemented until tested.

TraceDelta does not protect a compromised machine, malicious Go toolchain, or already-exposed trace artifact. It also cannot determine whether a custom attribute is personal data without user policy.

## CI guidance

- Generate the minimum telemetry necessary for the comparison.
- Avoid production traces and production credentials.
- Use short artifact retention and restricted access where the CI platform permits it.
- Do not echo entire input files on failure.
- Treat exit code `1` as a behavioral result and `2` as a tool/input failure; neither justifies dumping raw traces.
- Give the reusable GitHub Action read-only repository permissions and no secrets by default. v0.1 uses normal job summaries and artifacts rather than calling pull-request comment/check APIs.
- Prefer `--output` for report artifacts so trace-derived content is not echoed into logs. Existing files are not replaced without `--force`, and neither input trace can be selected as the output target.
- The generic CI wrapper writes only to a newly created report directory, logs only status and artifact paths, preserves reports only for completed exit `0`/`1` comparisons, and removes partial reports on exit `2`. Configure artifact upload to run after exit `1`, but never print the input or report contents as a debugging shortcut.
- Pin the reusable Action to a versioned ref, grant only `contents: read`, disable persisted checkout credentials, and use `pull_request` rather than privileged `pull_request_target` for untrusted forks. The Action builds only its pinned `github.action_path`, passes values through shell arrays, and requires no secret; caller-controlled trace generation remains untrusted code.

## Responsible disclosure

Report suspected vulnerabilities privately to `tracedelta.security@gmail.com` according to [`SECURITY.md`](../SECURITY.md). Do not open a public issue containing exploit details, secrets, personal data, or sensitive traces.
