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

## Attribute allowlists and denylists

The initial vertical slice does not yet implement configurable attribute filtering or redaction. Before broad OTLP support is considered complete, TraceDelta should provide:

- conservative built-in handling for known credential and personal-data attribute names;
- user-defined allowlists for attributes permitted to influence matching and appear as evidence;
- user-defined denylists that always take precedence over allowlists;
- separation between an attribute being useful internally and being safe to render; and
- tests proving excluded values do not leak through text, JSON, HTML, errors, or debug output.

Safe defaults should favor semantic keys such as HTTP route templates and database operation names over raw URLs, statements, or bound values.

## Future redaction

Redaction should occur immediately after parsing and before matching evidence or diff findings are constructed. Replacing a value only in the final renderer is insufficient because it can still leak through errors, intermediate objects, alternate formats, or logs.

Planned redaction needs deterministic replacement, documented precedence, protection across nested OTLP attribute values, and explicit behavior for malformed values. TraceDelta should not claim that an export is anonymized merely because common fields were removed.

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
- Give GitHub Actions only the permissions needed for checkout and, when eventually implemented, an explicit pull-request summary.

## Responsible disclosure

Report suspected vulnerabilities privately to `tracedelta.security@gmail.com` according to [`SECURITY.md`](../SECURITY.md). Do not open a public issue containing exploit details, secrets, personal data, or sensitive traces.
