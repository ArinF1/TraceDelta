// Package model defines TraceDelta's transport-independent domain types.
package model

import "time"

// AttributeValueType identifies the OTLP value form stored in an attribute.
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
	// AttributeValueArray is an OTLP arrayValue.
	AttributeValueArray AttributeValueType = "array"
	// AttributeValueKVList is an OTLP kvlistValue.
	AttributeValueKVList AttributeValueType = "kvlist"
)

// AttributeKeyValue preserves one entry in a nested OTLP key-value list.
type AttributeKeyValue struct {
	Key   string
	Value AttributeValue
}

// AttributeValue preserves the type of one supported OTLP AnyValue.
// Exactly one value field is meaningful according to Type.
type AttributeValue struct {
	Type        AttributeValueType
	StringValue string
	BoolValue   bool
	IntValue    int64
	DoubleValue float64
	BytesValue  []byte
	ArrayValue  []AttributeValue
	KVListValue []AttributeKeyValue
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

// ParentKind identifies how a normalized span relates to its parent after raw
// span identifiers have been removed.
type ParentKind string

const (
	// ParentRoot identifies a span with no parent in the source trace.
	ParentRoot ParentKind = "root"
	// ParentSpan identifies a parent present in the same normalized trace.
	ParentSpan ParentKind = "span"
	// ParentExternal identifies a parent omitted from a partial export.
	ParentExternal ParentKind = "external"
)

// ParentReference preserves a span's relationship without retaining its raw
// parent span identifier. SpanIndex is meaningful only when Kind is ParentSpan.
type ParentReference struct {
	Kind      ParentKind
	SpanIndex int
}

// NormalizedAttribute is a sorted, typed attribute projection with a canonical
// scalar representation.
type NormalizedAttribute struct {
	Key   string
	Type  AttributeValueType
	Value string
}

// NormalizedSpan contains deterministic fields used for comparison.
// Occurrence remains global to a snapshot for compatibility with the initial
// flat span matcher. StartOrder is a dense rank within one trace; equal source
// timestamps share a rank.
type NormalizedSpan struct {
	Key        SpanKey
	Occurrence int
	Parent     ParentReference
	StartOrder int
	Duration   time.Duration
	Status     StatusCode
	// ErrorType is safe diff evidence only and never participates in matching.
	// ErrorTypePresent distinguishes an absent attribute from an empty string.
	ErrorType        string
	ErrorTypePresent bool
	Attributes       []NormalizedAttribute
}

// NormalizedTrace is one trace after raw identifiers and absolute timestamps
// have been replaced with deterministic local structure.
type NormalizedTrace struct {
	Spans []NormalizedSpan
}

// NormalizedSnapshot is a trace-preserving deterministic comparison input.
type NormalizedSnapshot struct {
	Traces []NormalizedTrace
}
