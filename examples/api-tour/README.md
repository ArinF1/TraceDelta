# Public Go API tour

This example is a small, runnable tour of every exported function in `pkg/tracedelta`. It starts a temporary loopback-only HTTP server, fetches deterministic synthetic baseline and candidate OTLP JSON from that server, compares the snapshots both in memory and from files, and writes all three report formats.

Run it from the repository root with a directory that does not exist yet:

```bash
go run ./examples/api-tour --output-dir api-tour-output
```

The command creates:

- `baseline.json` and `candidate.json`, fetched from the temporary server;
- `report.txt`, the human-readable terminal format;
- `report.json`, the versioned machine-readable format; and
- `report.html`, the self-contained offline report.

Open `report.html` in a browser or inspect the other files in an editor. The example reports an added inventory span, a removed cache span, a checkout status change, an `error.type` change, and a payment latency increase. It also uses regenerated IDs, shifted timestamps, a duration bucket, built-in redaction, and one caller-supplied redacted key.

## Where each public function appears

The central workflow is in [`main.go`](main.go):

| API | What the example demonstrates |
| --- | --- |
| `tracedelta.DefaultOptions()` | Starts with the documented 20% and 10ms latency thresholds. |
| `tracedelta.Compare(...)` | Compares two in-memory `io.Reader` streams fetched from the server. |
| `tracedelta.CompareFiles(...)` | Compares the same snapshots after saving them locally. |
| `tracedelta.WriteText(...)` | Writes `report.txt`. |
| `tracedelta.WriteJSON(...)` | Writes `report.json`. |
| `tracedelta.WriteHTML(...)` | Writes `report.html`. |
| `Comparison.HasDifferences()` | Selects the differences result without treating it as a Go error. |
| `tracedelta.Change` | Iterates typed findings and prints their kind, field, and span name. |

The reader and file comparisons must be identical or the example fails. Output creation also fails if the requested directory already exists, so reruns cannot silently overwrite artifacts.

## Important boundary

The tiny server in [`server.go`](server.go) is an educational fixture server, not an OpenTelemetry collector and not automatic instrumentation. It hand-builds deterministic, synthetic OTLP/HTTP JSON so each TraceDelta behavior is easy to inspect. TraceDelta itself still performs no network calls, starts no applications, and collects no telemetry. In a real integration, your application and OpenTelemetry setup produce the sanitized baseline and candidate exports before you call this API.

The example server binds only to an ephemeral loopback address for the life of the command. Its client has a two-second timeout and a 1 MiB response limit. It uses only the Go standard library and contains no credentials, personal data, or production traces.
