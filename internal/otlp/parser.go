// Package otlp parses the documented OTLP JSON subset supported by TraceDelta.
package otlp

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ArinF1/TraceDelta/internal/model"
)

type exportDocument struct {
	ResourceSpans []resourceSpans `json:"resourceSpans"`
}

type resourceSpans struct {
	Resource   *resource    `json:"resource"`
	ScopeSpans []scopeSpans `json:"scopeSpans"`
	SchemaURL  string       `json:"schemaUrl,omitempty"`
}

type resource struct {
	Attributes             []keyValue      `json:"attributes"`
	DroppedAttributesCount json.RawMessage `json:"droppedAttributesCount,omitempty"`
}

type scopeSpans struct {
	Scope     *scope     `json:"scope,omitempty"`
	Spans     []jsonSpan `json:"spans"`
	SchemaURL string     `json:"schemaUrl,omitempty"`
}

type scope struct {
	Name                   string          `json:"name,omitempty"`
	Version                string          `json:"version,omitempty"`
	Attributes             []keyValue      `json:"attributes,omitempty"`
	DroppedAttributesCount json.RawMessage `json:"droppedAttributesCount,omitempty"`
}

type jsonSpan struct {
	TraceID                string          `json:"traceId"`
	SpanID                 string          `json:"spanId"`
	TraceState             string          `json:"traceState,omitempty"`
	ParentSpanID           string          `json:"parentSpanId,omitempty"`
	Flags                  json.RawMessage `json:"flags,omitempty"`
	Name                   string          `json:"name"`
	Kind                   json.RawMessage `json:"kind,omitempty"`
	StartTimeUnixNano      json.RawMessage `json:"startTimeUnixNano"`
	EndTimeUnixNano        json.RawMessage `json:"endTimeUnixNano"`
	Attributes             []keyValue      `json:"attributes,omitempty"`
	DroppedAttributesCount json.RawMessage `json:"droppedAttributesCount,omitempty"`
	Events                 []jsonEvent     `json:"events,omitempty"`
	DroppedEventsCount     json.RawMessage `json:"droppedEventsCount,omitempty"`
	Links                  []jsonLink      `json:"links,omitempty"`
	DroppedLinksCount      json.RawMessage `json:"droppedLinksCount,omitempty"`
	Status                 *jsonStatus     `json:"status,omitempty"`
}

type jsonEvent struct {
	TimeUnixNano           json.RawMessage `json:"timeUnixNano,omitempty"`
	Name                   string          `json:"name,omitempty"`
	Attributes             []keyValue      `json:"attributes,omitempty"`
	DroppedAttributesCount json.RawMessage `json:"droppedAttributesCount,omitempty"`
}

type jsonLink struct {
	TraceID                string          `json:"traceId,omitempty"`
	SpanID                 string          `json:"spanId,omitempty"`
	TraceState             string          `json:"traceState,omitempty"`
	Attributes             []keyValue      `json:"attributes,omitempty"`
	DroppedAttributesCount json.RawMessage `json:"droppedAttributesCount,omitempty"`
	Flags                  json.RawMessage `json:"flags,omitempty"`
}

type jsonStatus struct {
	Message string          `json:"message,omitempty"`
	Code    json.RawMessage `json:"code,omitempty"`
}

type keyValue struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type anyValue struct {
	StringValue *string          `json:"stringValue,omitempty"`
	BoolValue   *bool            `json:"boolValue,omitempty"`
	IntValue    json.RawMessage  `json:"intValue,omitempty"`
	DoubleValue json.RawMessage  `json:"doubleValue,omitempty"`
	ArrayValue  *jsonArrayValue  `json:"arrayValue,omitempty"`
	KVListValue *jsonKVListValue `json:"kvlistValue,omitempty"`
	BytesValue  *string          `json:"bytesValue,omitempty"`
}

type jsonArrayValue struct {
	Values []json.RawMessage `json:"values,omitempty"`
}

type jsonKVListValue struct {
	Values []keyValue `json:"values,omitempty"`
}

const maxAnyValueDepth = 64

// Parse decodes either one OTLP/HTTP JSON trace request object or a trace-only
// OTLP File Exporter JSON Lines stream. Unknown message fields are ignored as
// required by OTLP JSON; known non-trace envelopes fail contextually.
func Parse(r io.Reader) (model.Snapshot, error) {
	decoder := json.NewDecoder(r)
	documents := make([]exportDocument, 0, 1)
	for recordIndex := 0; ; recordIndex++ {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				if len(documents) == 0 {
					return model.Snapshot{}, errors.New("decode OTLP JSON: input is empty")
				}
				break
			}
			return model.Snapshot{}, fmt.Errorf("decode OTLP JSON record[%d]: %w", recordIndex, err)
		}
		if err := validateTopLevelObject(raw); err != nil {
			return model.Snapshot{}, fmt.Errorf("decode OTLP JSON record[%d]: %w", recordIndex, err)
		}
		var document exportDocument
		if err := json.Unmarshal(raw, &document); err != nil {
			return model.Snapshot{}, fmt.Errorf("decode OTLP JSON record[%d]: %w", recordIndex, err)
		}
		documents = append(documents, document)
	}

	return convertDocuments(documents)
}

func validateTopLevelObject(raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" || !strings.HasPrefix(trimmed, "{") {
		return errors.New("expected a JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, field := range []string{"resourceMetrics", "resourceLogs", "resourceProfiles"} {
		if _, exists := fields[field]; exists {
			return fmt.Errorf("field %s is not trace telemetry", field)
		}
	}
	for _, field := range []string{"traces", "spans", "data"} {
		if _, exists := fields[field]; exists {
			return fmt.Errorf("field %s is an unsupported outer envelope", field)
		}
	}
	return nil
}

func convertDocuments(documents []exportDocument) (model.Snapshot, error) {
	if len(documents) == 1 {
		return convert(documents[0])
	}

	result := model.Snapshot{}
	traceIndexes := make(map[string]int)
	seenSpans := make(map[string]struct{})
	inputOrder := 0
	for recordIndex, document := range documents {
		snapshot, err := convert(document)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON record[%d]: %w", recordIndex, err)
		}
		for _, trace := range snapshot.Traces {
			traceIndex, exists := traceIndexes[trace.ID]
			if !exists {
				traceIndex = len(result.Traces)
				traceIndexes[trace.ID] = traceIndex
				result.Traces = append(result.Traces, model.Trace{ID: trace.ID})
			}
			for _, span := range trace.Spans {
				identity := span.TraceID + "\x00" + span.SpanID
				if _, exists := seenSpans[identity]; exists {
					return model.Snapshot{}, fmt.Errorf("validate OTLP JSON record[%d]: duplicate spanId within its trace", recordIndex)
				}
				seenSpans[identity] = struct{}{}
				span.InputOrder = inputOrder
				inputOrder++
				result.Traces[traceIndex].Spans = append(result.Traces[traceIndex].Spans, span)
			}
		}
	}
	return result, nil
}

func convert(document exportDocument) (model.Snapshot, error) {
	snapshot := model.Snapshot{}
	traceIndexes := make(map[string]int)
	seenSpans := make(map[string]struct{})
	inputOrder := 0

	for resourceIndex, encodedResourceSpans := range document.ResourceSpans {
		resourceContext, serviceName, err := convertResource(encodedResourceSpans.Resource)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].resource: %w", resourceIndex, err)
		}

		for scopeIndex, encodedScopeSpans := range encodedResourceSpans.ScopeSpans {
			scopeContext, err := convertScope(encodedScopeSpans.Scope)
			if err != nil {
				return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].scopeSpans[%d].scope: %w", resourceIndex, scopeIndex, err)
			}

			for spanIndex, encodedSpan := range encodedScopeSpans.Spans {
				path := fmt.Sprintf("resourceSpans[%d].scopeSpans[%d].spans[%d]", resourceIndex, scopeIndex, spanIndex)
				span, err := convertSpan(
					encodedSpan,
					serviceName,
					resourceContext,
					encodedResourceSpans.SchemaURL,
					scopeContext,
					encodedScopeSpans.SchemaURL,
				)
				if err != nil {
					return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: %s: %w", path, err)
				}

				identity := span.TraceID + "\x00" + span.SpanID
				if _, exists := seenSpans[identity]; exists {
					return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: %s: duplicate spanId within its trace", path)
				}
				seenSpans[identity] = struct{}{}
				span.InputOrder = inputOrder
				inputOrder++

				traceIndex, exists := traceIndexes[span.TraceID]
				if !exists {
					traceIndex = len(snapshot.Traces)
					traceIndexes[span.TraceID] = traceIndex
					snapshot.Traces = append(snapshot.Traces, model.Trace{ID: span.TraceID})
				}
				snapshot.Traces[traceIndex].Spans = append(snapshot.Traces[traceIndex].Spans, span)
			}
		}
	}

	return snapshot, nil
}

func convertResource(encoded *resource) (model.Resource, string, error) {
	if encoded == nil {
		return model.Resource{}, "", nil
	}

	attributes, err := convertAttributes(encoded.Attributes)
	if err != nil {
		return model.Resource{}, "", fmt.Errorf("attributes: %w", err)
	}
	droppedAttributesCount, err := parseOptionalUint32("droppedAttributesCount", encoded.DroppedAttributesCount)
	if err != nil {
		return model.Resource{}, "", err
	}

	serviceName := ""
	if value, ok := attributes["service.name"]; ok {
		if value.Type != model.AttributeValueString {
			return model.Resource{}, "", errors.New("attribute service.name must use stringValue")
		}
		if strings.TrimSpace(value.StringValue) == "" {
			return model.Resource{}, "", errors.New("attribute service.name must not be empty")
		}
		serviceName = value.StringValue
	}

	return model.Resource{
		Attributes:             attributes,
		DroppedAttributesCount: droppedAttributesCount,
	}, serviceName, nil
}

func convertScope(encoded *scope) (model.InstrumentationScope, error) {
	if encoded == nil {
		return model.InstrumentationScope{}, nil
	}

	attributes, err := convertAttributes(encoded.Attributes)
	if err != nil {
		return model.InstrumentationScope{}, fmt.Errorf("attributes: %w", err)
	}
	droppedAttributesCount, err := parseOptionalUint32("droppedAttributesCount", encoded.DroppedAttributesCount)
	if err != nil {
		return model.InstrumentationScope{}, err
	}

	return model.InstrumentationScope{
		Name:                   encoded.Name,
		Version:                encoded.Version,
		Attributes:             attributes,
		DroppedAttributesCount: droppedAttributesCount,
	}, nil
}

func convertSpan(
	encoded jsonSpan,
	serviceName string,
	resourceContext model.Resource,
	resourceSchemaURL string,
	scopeContext model.InstrumentationScope,
	scopeSchemaURL string,
) (model.Span, error) {
	if err := validateHexID("traceId", encoded.TraceID, 16); err != nil {
		return model.Span{}, err
	}
	if err := validateHexID("spanId", encoded.SpanID, 8); err != nil {
		return model.Span{}, err
	}
	if encoded.ParentSpanID != "" {
		if err := validateHexID("parentSpanId", encoded.ParentSpanID, 8); err != nil {
			return model.Span{}, err
		}
	}
	if strings.TrimSpace(encoded.Name) == "" {
		return model.Span{}, errors.New("name must not be empty")
	}
	if err := validateEvents(encoded.Events); err != nil {
		return model.Span{}, err
	}
	if err := validateLinks(encoded.Links); err != nil {
		return model.Span{}, err
	}

	kind, err := convertKind(encoded.Kind)
	if err != nil {
		return model.Span{}, err
	}
	start, err := parseRequiredUint64("startTimeUnixNano", encoded.StartTimeUnixNano)
	if err != nil {
		return model.Span{}, err
	}
	end, err := parseRequiredUint64("endTimeUnixNano", encoded.EndTimeUnixNano)
	if err != nil {
		return model.Span{}, err
	}
	if end < start {
		return model.Span{}, fmt.Errorf("endTimeUnixNano %d is before startTimeUnixNano %d", end, start)
	}
	if end-start > math.MaxInt64 {
		return model.Span{}, errors.New("span duration exceeds the supported maximum")
	}

	attributes, err := convertAttributes(encoded.Attributes)
	if err != nil {
		return model.Span{}, fmt.Errorf("attributes: %w", err)
	}
	status, message, err := convertStatus(encoded.Status)
	if err != nil {
		return model.Span{}, err
	}
	flags, err := parseOptionalUint32("flags", encoded.Flags)
	if err != nil {
		return model.Span{}, err
	}
	droppedAttributesCount, err := parseOptionalUint32("droppedAttributesCount", encoded.DroppedAttributesCount)
	if err != nil {
		return model.Span{}, err
	}
	droppedEventsCount, err := parseOptionalUint32("droppedEventsCount", encoded.DroppedEventsCount)
	if err != nil {
		return model.Span{}, err
	}
	droppedLinksCount, err := parseOptionalUint32("droppedLinksCount", encoded.DroppedLinksCount)
	if err != nil {
		return model.Span{}, err
	}

	return model.Span{
		TraceID:                strings.ToLower(encoded.TraceID),
		SpanID:                 strings.ToLower(encoded.SpanID),
		TraceState:             encoded.TraceState,
		ParentSpanID:           strings.ToLower(encoded.ParentSpanID),
		Flags:                  flags,
		Name:                   encoded.Name,
		ServiceName:            serviceName,
		Kind:                   kind,
		StartTime:              start,
		Duration:               time.Duration(end - start),
		Status:                 status,
		StatusMessage:          message,
		Attributes:             attributes,
		DroppedAttributesCount: droppedAttributesCount,
		DroppedEventsCount:     droppedEventsCount,
		DroppedLinksCount:      droppedLinksCount,
		Resource:               resourceContext,
		ResourceSchemaURL:      resourceSchemaURL,
		Scope:                  scopeContext,
		ScopeSchemaURL:         scopeSchemaURL,
	}, nil
}

func validateEvents(events []jsonEvent) error {
	for index, event := range events {
		if jsonValuePresent(event.TimeUnixNano) {
			if _, err := parseRequiredUint64("timeUnixNano", event.TimeUnixNano); err != nil {
				return fmt.Errorf("events[%d]: %w", index, err)
			}
		}
		if _, err := convertAttributes(event.Attributes); err != nil {
			return fmt.Errorf("events[%d].attributes: %w", index, err)
		}
		if _, err := parseOptionalUint32("droppedAttributesCount", event.DroppedAttributesCount); err != nil {
			return fmt.Errorf("events[%d]: %w", index, err)
		}
	}
	return nil
}

func validateLinks(links []jsonLink) error {
	for index, link := range links {
		if link.TraceID != "" {
			if err := validateHexID("traceId", link.TraceID, 16); err != nil {
				return fmt.Errorf("links[%d]: %w", index, err)
			}
		}
		if link.SpanID != "" {
			if err := validateHexID("spanId", link.SpanID, 8); err != nil {
				return fmt.Errorf("links[%d]: %w", index, err)
			}
		}
		if _, err := convertAttributes(link.Attributes); err != nil {
			return fmt.Errorf("links[%d].attributes: %w", index, err)
		}
		if _, err := parseOptionalUint32("droppedAttributesCount", link.DroppedAttributesCount); err != nil {
			return fmt.Errorf("links[%d]: %w", index, err)
		}
		if _, err := parseOptionalUint32("flags", link.Flags); err != nil {
			return fmt.Errorf("links[%d]: %w", index, err)
		}
	}
	return nil
}

func validateHexID(field, value string, byteLength int) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != byteLength {
		return fmt.Errorf("%s must be %d hexadecimal characters", field, byteLength*2)
	}
	allZero := true
	for _, part := range decoded {
		if part != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return fmt.Errorf("%s must not be all zeros", field)
	}
	return nil
}

func convertKind(raw json.RawMessage) (string, error) {
	code, err := parseEnum(raw, map[string]uint64{
		"SPAN_KIND_UNSPECIFIED": 0,
		"SPAN_KIND_INTERNAL":    1,
		"SPAN_KIND_SERVER":      2,
		"SPAN_KIND_CLIENT":      3,
		"SPAN_KIND_PRODUCER":    4,
		"SPAN_KIND_CONSUMER":    5,
	})
	if err != nil {
		return "", fmt.Errorf("kind: %w", err)
	}
	kinds := [...]string{
		"SPAN_KIND_UNSPECIFIED",
		"SPAN_KIND_INTERNAL",
		"SPAN_KIND_SERVER",
		"SPAN_KIND_CLIENT",
		"SPAN_KIND_PRODUCER",
		"SPAN_KIND_CONSUMER",
	}
	if code >= uint64(len(kinds)) {
		return "", fmt.Errorf("kind value %d is not supported", code)
	}
	return kinds[code], nil
}

func convertStatus(status *jsonStatus) (model.StatusCode, string, error) {
	if status == nil {
		return model.StatusUnset, "", nil
	}
	code, err := parseEnum(status.Code, map[string]uint64{
		"STATUS_CODE_UNSET": 0,
		"STATUS_CODE_OK":    1,
		"STATUS_CODE_ERROR": 2,
	})
	if err != nil {
		return "", "", fmt.Errorf("status.code: %w", err)
	}
	switch code {
	case 0:
		return model.StatusUnset, status.Message, nil
	case 1:
		return model.StatusOK, status.Message, nil
	case 2:
		return model.StatusError, status.Message, nil
	default:
		return "", "", fmt.Errorf("status.code value %d is not supported", code)
	}
}

func parseEnum(raw json.RawMessage, compatibilityNames map[string]uint64) (uint64, error) {
	if !jsonValuePresent(raw) {
		return 0, nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "\"") {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil {
			return 0, fmt.Errorf("must be an integer: %w", err)
		}
		if value, ok := compatibilityNames[name]; ok {
			return value, nil
		}
		return 0, fmt.Errorf("symbolic value %q is not supported", name)
	}
	return parseUnsignedInteger(raw, 32)
}

func parseRequiredUint64(field string, raw json.RawMessage) (uint64, error) {
	if !jsonValuePresent(raw) {
		return 0, fmt.Errorf("%s is required", field)
	}
	value, err := parseUnsignedInteger(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned decimal nanosecond value: %w", field, err)
	}
	return value, nil
}

func parseOptionalUint32(field string, raw json.RawMessage) (uint32, error) {
	if !jsonValuePresent(raw) {
		return 0, nil
	}
	value, err := parseUnsignedInteger(raw, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned 32-bit integer: %w", field, err)
	}
	return uint32(value), nil
}

func parseUnsignedInteger(raw json.RawMessage, bitSize int) (uint64, error) {
	integer, err := parseInteger(raw)
	if err != nil {
		return 0, err
	}
	if integer.Sign() < 0 {
		return 0, errors.New("value must not be negative")
	}
	if integer.BitLen() > bitSize {
		return 0, fmt.Errorf("value exceeds %d-bit unsigned range", bitSize)
	}
	return integer.Uint64(), nil
}

func parseSignedInteger(raw json.RawMessage, bitSize int) (int64, error) {
	integer, err := parseInteger(raw)
	if err != nil {
		return 0, err
	}
	if bitSize != 64 || !integer.IsInt64() {
		return 0, fmt.Errorf("value exceeds %d-bit signed range", bitSize)
	}
	return integer.Int64(), nil
}

func parseInteger(raw json.RawMessage) (*big.Int, error) {
	if !jsonValuePresent(raw) {
		return nil, errors.New("value is required")
	}

	trimmed := strings.TrimSpace(string(raw))
	literal := trimmed
	if strings.HasPrefix(trimmed, "\"") {
		if err := json.Unmarshal(raw, &literal); err != nil {
			return nil, fmt.Errorf("value must be a decimal integer: %w", err)
		}
	}
	if literal == "" || strings.HasPrefix(literal, "+") || strings.ContainsAny(literal, "/pPxXoObB") {
		return nil, errors.New("value must be a decimal integer")
	}

	rational, ok := new(big.Rat).SetString(literal)
	if !ok || !rational.IsInt() {
		return nil, errors.New("value must be a decimal integer")
	}
	return new(big.Int).Set(rational.Num()), nil
}

func convertAttributes(attributes []keyValue) (model.Attributes, error) {
	converted := make(model.Attributes, len(attributes))
	for index, attribute := range attributes {
		if strings.TrimSpace(attribute.Key) == "" {
			return nil, fmt.Errorf("attribute[%d].key must not be empty", index)
		}
		if _, exists := converted[attribute.Key]; exists {
			return nil, fmt.Errorf("attribute[%d] duplicates key %q", index, attribute.Key)
		}
		value, err := convertAnyValue(attribute.Value)
		if err != nil {
			return nil, fmt.Errorf("attribute[%d] %q: %w", index, attribute.Key, err)
		}
		converted[attribute.Key] = value
	}
	return converted, nil
}

func convertAnyValue(raw json.RawMessage) (model.AttributeValue, error) {
	return convertAnyValueAtDepth(raw, 0)
}

func convertAnyValueAtDepth(raw json.RawMessage, depth int) (model.AttributeValue, error) {
	if depth >= maxAnyValueDepth {
		return model.AttributeValue{}, fmt.Errorf("nested AnyValue exceeds maximum depth %d", maxAnyValueDepth)
	}
	if !jsonValuePresent(raw) {
		return model.AttributeValue{}, errors.New("value must contain exactly one OTLP AnyValue case")
	}

	var encoded anyValue
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return model.AttributeValue{}, fmt.Errorf("decode value: %w", err)
	}
	present := 0
	if encoded.StringValue != nil {
		present++
	}
	if encoded.BoolValue != nil {
		present++
	}
	if jsonValuePresent(encoded.IntValue) {
		present++
	}
	if jsonValuePresent(encoded.DoubleValue) {
		present++
	}
	if encoded.BytesValue != nil {
		present++
	}
	if encoded.ArrayValue != nil {
		present++
	}
	if encoded.KVListValue != nil {
		present++
	}
	if present != 1 {
		return model.AttributeValue{}, errors.New("value must contain exactly one OTLP AnyValue case")
	}

	switch {
	case encoded.StringValue != nil:
		return model.AttributeValue{Type: model.AttributeValueString, StringValue: *encoded.StringValue}, nil
	case encoded.BoolValue != nil:
		return model.AttributeValue{Type: model.AttributeValueBool, BoolValue: *encoded.BoolValue}, nil
	case jsonValuePresent(encoded.IntValue):
		value, err := parseSignedInteger(encoded.IntValue, 64)
		if err != nil {
			return model.AttributeValue{}, fmt.Errorf("intValue must be a signed 64-bit decimal integer: %w", err)
		}
		return model.AttributeValue{Type: model.AttributeValueInt, IntValue: value}, nil
	case jsonValuePresent(encoded.DoubleValue):
		value, err := parseDouble(encoded.DoubleValue)
		if err != nil {
			return model.AttributeValue{}, fmt.Errorf("doubleValue: %w", err)
		}
		return model.AttributeValue{Type: model.AttributeValueDouble, DoubleValue: value}, nil
	case encoded.BytesValue != nil:
		value, err := decodeBytes(*encoded.BytesValue)
		if err != nil {
			return model.AttributeValue{}, err
		}
		return model.AttributeValue{Type: model.AttributeValueBytes, BytesValue: value}, nil
	case encoded.ArrayValue != nil:
		values := make([]model.AttributeValue, len(encoded.ArrayValue.Values))
		for index, rawValue := range encoded.ArrayValue.Values {
			value, err := convertAnyValueAtDepth(rawValue, depth+1)
			if err != nil {
				return model.AttributeValue{}, fmt.Errorf("arrayValue.values[%d]: %w", index, err)
			}
			values[index] = value
		}
		return model.AttributeValue{Type: model.AttributeValueArray, ArrayValue: values}, nil
	default:
		values := make([]model.AttributeKeyValue, len(encoded.KVListValue.Values))
		for index, encodedValue := range encoded.KVListValue.Values {
			if strings.TrimSpace(encodedValue.Key) == "" {
				return model.AttributeValue{}, fmt.Errorf("kvlistValue.values[%d].key must not be empty", index)
			}
			value, err := convertAnyValueAtDepth(encodedValue.Value, depth+1)
			if err != nil {
				return model.AttributeValue{}, fmt.Errorf("kvlistValue.values[%d] %q: %w", index, encodedValue.Key, err)
			}
			values[index] = model.AttributeKeyValue{Key: encodedValue.Key, Value: value}
		}
		return model.AttributeValue{Type: model.AttributeValueKVList, KVListValue: values}, nil
	}
}

func parseDouble(raw json.RawMessage) (float64, error) {
	trimmed := strings.TrimSpace(string(raw))
	literal := trimmed
	if strings.HasPrefix(trimmed, "\"") {
		if err := json.Unmarshal(raw, &literal); err != nil {
			return 0, fmt.Errorf("must be a JSON number or supported special string: %w", err)
		}
		switch literal {
		case "NaN":
			return math.NaN(), nil
		case "Infinity":
			return math.Inf(1), nil
		case "-Infinity":
			return math.Inf(-1), nil
		}
	}

	value, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		return 0, fmt.Errorf("must be a JSON number or one of NaN, Infinity, or -Infinity: %w", err)
	}
	return value, nil
}

func decodeBytes(value string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("bytesValue must be a valid standard or URL-safe base64 string")
}

func jsonValuePresent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}
