// Package model defines TraceDelta's transport-independent domain types.
package model

import "time"

// StatusCode is the normalized OpenTelemetry status of a span.
type StatusCode string

const (
	// StatusUnset means the producer did not assign a status.
	StatusUnset StatusCode = "UNSET"
	// StatusOK means the operation explicitly completed successfully.
	StatusOK StatusCode = "OK"
	// StatusError means the operation completed with an error.
	StatusError StatusCode = "ERROR"
)

// Span is a parsed span before nondeterministic fields are removed.
type Span struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Name          string
	ServiceName   string
	Kind          string
	StartTime     uint64
	Duration      time.Duration
	Status        StatusCode
	StatusMessage string
	Attributes    map[string]string
	InputOrder    int
}

// Trace groups spans that shared a trace ID in the input document.
type Trace struct {
	ID    string
	Spans []Span
}

// Snapshot is one parsed trace export.
type Snapshot struct {
	Traces []Trace
}

// SpanKey contains the stable fields used by the initial span matcher.
type SpanKey struct {
	ServiceName string
	Name        string
	Kind        string
}

// NormalizedSpan contains the deterministic fields used for comparison.
// Occurrence disambiguates otherwise-identical span keys by input order.
type NormalizedSpan struct {
	Key           SpanKey
	Occurrence    int
	Duration      time.Duration
	Status        StatusCode
	StatusMessage string
	Attributes    map[string]string
}

// NormalizedSnapshot is a flattened, deterministic comparison input.
type NormalizedSnapshot struct {
	Spans []NormalizedSpan
}
