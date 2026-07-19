// Package model defines TraceDelta's transport-independent domain types.
package model

import "time"

// AttributeValueType identifies the OTLP primitive stored in an attribute.
type AttributeValueType string

const (
	// AttributeValueString is an OTLP stringValue.
	AttributeValueString AttributeValueType = "string"
	// AttributeValueBool is an OTLP boolValue.
	AttributeValueBool AttributeValueType = "bool"
	// AttributeValueInt is an OTLP intValue.
	AttributeValueInt AttributeValueType = "int"
	// AttributeValueDouble is an OTLP doubleValue.
	AttributeValueDouble AttributeValueType = "double"
	// AttributeValueBytes is an OTLP bytesValue.
	AttributeValueBytes AttributeValueType = "bytes"
)

// AttributeValue preserves the type of one supported OTLP primitive value.
// Exactly one value field is meaningful according to Type.
type AttributeValue struct {
	Type        AttributeValueType
	StringValue string
	BoolValue   bool
	IntValue    int64
	DoubleValue float64
	BytesValue  []byte
}

// Attributes is a validated set of unique OTLP key/value attributes.
type Attributes map[string]AttributeValue

// Resource describes the resource context attached to a group of spans.
type Resource struct {
	Attributes             Attributes
	DroppedAttributesCount uint32
}

// InstrumentationScope describes the scope that produced a group of spans.
type InstrumentationScope struct {
	Name                   string
	Version                string
	Attributes             Attributes
	DroppedAttributesCount uint32
}

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
	TraceID                string
	SpanID                 string
	TraceState             string
	ParentSpanID           string
	Flags                  uint32
	Name                   string
	ServiceName            string
	Kind                   string
	StartTime              uint64
	Duration               time.Duration
	Status                 StatusCode
	StatusMessage          string
	Attributes             Attributes
	DroppedAttributesCount uint32
	DroppedEventsCount     uint32
	DroppedLinksCount      uint32
	Resource               Resource
	ResourceSchemaURL      string
	Scope                  InstrumentationScope
	ScopeSchemaURL         string
	InputOrder             int
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
	Attributes    Attributes
}

// NormalizedSnapshot is a flattened, deterministic comparison input.
type NormalizedSnapshot struct {
	Spans []NormalizedSpan
}
