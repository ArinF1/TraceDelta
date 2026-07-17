## Summary

<!-- Explain the user-visible or internal outcome, not only the files changed. -->

## Related work

<!-- Link an issue and include the TD-### task ID when applicable. -->

## Change type

- [ ] Behavioral change
- [ ] Bug fix
- [ ] Refactor with no intended behavior change
- [ ] Documentation, tests, or repository maintenance

## Validation

- [ ] Tests demonstrate each behavioral change.
- [ ] `make check` passes.
- [ ] `make test-race` passes, or the reason it was not run is stated below.
- [ ] The example comparison was run when CLI behavior changed.

Validation details:

<!-- Include exact commands and results. -->

## Compatibility, privacy, and security

- [ ] Backward compatibility is preserved, or the documented decision permitting a break is linked.
- [ ] No secrets, personal data, internal details, or raw production traces are included.
- [ ] Untrusted input and generated output were considered for privacy and security impact.

## Documentation and project memory

- [ ] Public behavior documentation is updated, or no update is needed.
- [ ] Important architecture decisions are recorded as an ADR, or no ADR is needed.
- [ ] `docs/current-state.md` and `docs/tasks.md` are current.
- [ ] A brief entry was appended to `docs/session-log.md`.
