# Security Policy

TraceDelta processes observability data, which may contain credentials, tokens, personal information, internal hostnames, SQL statements, or other sensitive values. Treat trace inputs and generated reports as sensitive unless you have verified otherwise.

## Supported versions

TraceDelta has no published stable release yet. Security fixes are currently made on the default branch. Once releases exist, this section will identify supported release lines explicitly.

## Reporting a vulnerability

Do not open a public issue or include sensitive trace data in a pull request.

Send a private report to **tracedelta.security@gmail.com**. If the repository host enables private vulnerability reporting, that channel may also be used.

Include, when available:

- a concise description of the issue and its impact;
- affected versions, commits, or configurations;
- minimal reproduction steps using synthetic data;
- any known mitigations; and
- whether details have been shared elsewhere.

Never send credentials, personal data, or raw production traces. Create the smallest sanitized reproduction that demonstrates the problem.

Maintainers will acknowledge reports, coordinate investigation and remediation privately, and credit reporters when requested and safe. Response times are best effort until the project establishes a staffed security process. TraceDelta does not currently operate a bug bounty program.

## Security expectations for contributions

Contributions should preserve local-first processing, avoid unnecessary network access, validate untrusted input, bound resource use where practical, avoid logging trace contents by default, and keep CI permissions at the minimum required level. See [`docs/privacy-and-security.md`](docs/privacy-and-security.md) for the project threat model and data-handling direction.
