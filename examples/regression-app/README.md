# Deliberately regressed example application

This tiny Go program writes a deterministic synthetic OTLP/HTTP JSON trace. It makes no network call, uses no environment secret, and contains no production data. The baseline scenario contains one checkout root span plus payment and cache child spans.

Generate an artifact from the current source:

```bash
mkdir -p artifacts
go run ./examples/regression-app --output artifacts/trace.json
```

The repository's `example-regression.yml` workflow checks out the pull request's base and candidate commits into separate directories, runs this command against each, and compares the two generated artifacts through the Action from the trusted base commit. Completed comparisons upload terminal text, JSON, and standalone HTML reports before the final policy gate.

The one-time foundation pull request is explicitly excluded because its base predates both the example application and trusted Action. Pull requests opened after that foundation exists, including the deliberate regression pull request, always compare two complete revisions.

Public draft [PR #7](https://github.com/ArinF1/TraceDelta/pull/7) intentionally changes only `scenario.go` and its contract test:

- `cache.get` is removed;
- `inventory.reserve` is added;
- `checkout.handle` changes from `OK` to `ERROR`; and
- `payment.charge` grows from 40ms to 70ms, exceeding both default thresholds.

The pull request is expected to fail with exactly those four behavioral findings. The base artifact is always regenerated from `github.event.pull_request.base.sha`, so maintainers can close/reopen or recreate the branch without rewriting history or storing generated trace files in Git.
