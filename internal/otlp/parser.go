// Package otlp parses the deliberately small OTLP JSON subset supported by
// TraceDelta's first vertical slice.
package otlp

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/example/tracedelta/internal/model"
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
	Attributes             []keyValue `json:"attributes"`
	DroppedAttributesCount uint32     `json:"droppedAttributesCount,omitempty"`
}

type scopeSpans struct {
	Scope     *scope     `json:"scope,omitempty"`
	Spans     []jsonSpan `json:"spans"`
	SchemaURL string     `json:"schemaUrl,omitempty"`
}

type scope struct {
	Name                   string     `json:"name,omitempty"`
	Version                string     `json:"version,omitempty"`
	Attributes             []keyValue `json:"attributes,omitempty"`
	DroppedAttributesCount uint32     `json:"droppedAttributesCount,omitempty"`
}

type jsonSpan struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId,omitempty"`
	Name              string      `json:"name"`
	Kind              string      `json:"kind,omitempty"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []keyValue  `json:"attributes,omitempty"`
	Status            *jsonStatus `json:"status,omitempty"`
}

type jsonStatus struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

type keyValue struct {
	Key   string   `json:"key"`
	Value anyValue `json:"value"`
}

type anyValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	IntValue    *string  `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BytesValue  *string  `json:"bytesValue,omitempty"`
}

// Parse decodes one simplified OTLP JSON trace export from r. Unknown fields
// are rejected so unsupported input cannot be mistaken for fully parsed data.
func Parse(r io.Reader) (model.Snapshot, error) {
	var document exportDocument
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return model.Snapshot{}, fmt.Errorf("decode OTLP JSON: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return model.Snapshot{}, errors.New("decode OTLP JSON: expected exactly one JSON document")
		}
		return model.Snapshot{}, fmt.Errorf("decode OTLP JSON trailing data: %w", err)
	}

	if document.ResourceSpans == nil {
		return model.Snapshot{}, errors.New("validate OTLP JSON: resourceSpans is required and must be an array")
	}

	return convert(document)
}

func convert(document exportDocument) (model.Snapshot, error) {
	snapshot := model.Snapshot{}
	traceIndexes := make(map[string]int)
	seenSpans := make(map[string]struct{})
	inputOrder := 0

	for resourceIndex, resourceSpans := range document.ResourceSpans {
		if resourceSpans.Resource == nil {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].resource is required", resourceIndex)
		}
		if resourceSpans.ScopeSpans == nil {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].scopeSpans is required and must be an array", resourceIndex)
		}

		resourceAttributes, err := convertAttributes(resourceSpans.Resource.Attributes)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].resource.attributes: %w", resourceIndex, err)
		}
		serviceName, ok := resourceAttributes["service.name"]
		if !ok || strings.TrimSpace(serviceName) == "" {
			return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].resource.attributes must contain a non-empty string service.name", resourceIndex)
		}
		for _, attribute := range resourceSpans.Resource.Attributes {
			if attribute.Key == "service.name" && attribute.Value.StringValue == nil {
				return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].resource attribute service.name must use stringValue", resourceIndex)
			}
		}

		for scopeIndex, scopeSpans := range resourceSpans.ScopeSpans {
			if scopeSpans.Spans == nil {
				return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: resourceSpans[%d].scopeSpans[%d].spans is required and must be an array", resourceIndex, scopeIndex)
			}
			for spanIndex, encodedSpan := range scopeSpans.Spans {
				path := fmt.Sprintf("resourceSpans[%d].scopeSpans[%d].spans[%d]", resourceIndex, scopeIndex, spanIndex)
				span, err := convertSpan(encodedSpan, serviceName)
				if err != nil {
					return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: %s: %w", path, err)
				}

				identity := span.TraceID + "\x00" + span.SpanID
				if _, exists := seenSpans[identity]; exists {
					return model.Snapshot{}, fmt.Errorf("validate OTLP JSON: %s: duplicate spanId %q in trace %q", path, span.SpanID, span.TraceID)
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

func convertSpan(encoded jsonSpan, serviceName string) (model.Span, error) {
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

	kind, err := normalizeKind(encoded.Kind)
	if err != nil {
		return model.Span{}, err
	}
	start, err := parseTimestamp("startTimeUnixNano", encoded.StartTimeUnixNano)
	if err != nil {
		return model.Span{}, err
	}
	end, err := parseTimestamp("endTimeUnixNano", encoded.EndTimeUnixNano)
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

	return model.Span{
		TraceID:       strings.ToLower(encoded.TraceID),
		SpanID:        strings.ToLower(encoded.SpanID),
		ParentSpanID:  strings.ToLower(encoded.ParentSpanID),
		Name:          encoded.Name,
		ServiceName:   serviceName,
		Kind:          kind,
		StartTime:     start,
		Duration:      time.Duration(end - start),
		Status:        status,
		StatusMessage: message,
		Attributes:    attributes,
	}, nil
}

func validateHexID(field, value string, byteLength int) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != byteLength {
		return fmt.Errorf("%s must be %d hexadecimal characters", field, byteLength*2)
	}
	return nil
}

func parseTimestamp(field, value string) (uint64, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned decimal nanosecond value: %w", field, err)
	}
	return parsed, nil
}

func normalizeKind(kind string) (string, error) {
	if kind == "" {
		return "SPAN_KIND_UNSPECIFIED", nil
	}
	switch kind {
	case "SPAN_KIND_UNSPECIFIED", "SPAN_KIND_INTERNAL", "SPAN_KIND_SERVER", "SPAN_KIND_CLIENT", "SPAN_KIND_PRODUCER", "SPAN_KIND_CONSUMER":
		return kind, nil
	default:
		return "", fmt.Errorf("kind %q is not a supported OTLP span kind", kind)
	}
}

func convertStatus(status *jsonStatus) (model.StatusCode, string, error) {
	if status == nil || status.Code == "" || status.Code == "STATUS_CODE_UNSET" {
		if status == nil {
			return model.StatusUnset, "", nil
		}
		return model.StatusUnset, status.Message, nil
	}
	switch status.Code {
	case "STATUS_CODE_OK":
		return model.StatusOK, status.Message, nil
	case "STATUS_CODE_ERROR":
		return model.StatusError, status.Message, nil
	default:
		return "", "", fmt.Errorf("status.code %q is not supported", status.Code)
	}
}

func convertAttributes(attributes []keyValue) (map[string]string, error) {
	converted := make(map[string]string, len(attributes))
	for index, attribute := range attributes {
		if strings.TrimSpace(attribute.Key) == "" {
			return nil, fmt.Errorf("attribute[%d].key must not be empty", index)
		}
		if _, exists := converted[attribute.Key]; exists {
			return nil, fmt.Errorf("attribute[%d] duplicates key %q", index, attribute.Key)
		}
		value, err := attribute.Value.text()
		if err != nil {
			return nil, fmt.Errorf("attribute[%d] %q: %w", index, attribute.Key, err)
		}
		converted[attribute.Key] = value
	}
	return converted, nil
}

func (value anyValue) text() (string, error) {
	count := 0
	result := ""
	if value.StringValue != nil {
		count++
		result = *value.StringValue
	}
	if value.BoolValue != nil {
		count++
		result = strconv.FormatBool(*value.BoolValue)
	}
	if value.IntValue != nil {
		count++
		parsed, err := strconv.ParseInt(*value.IntValue, 10, 64)
		if err != nil {
			return "", fmt.Errorf("intValue must be a signed decimal integer: %w", err)
		}
		result = strconv.FormatInt(parsed, 10)
	}
	if value.DoubleValue != nil {
		count++
		result = strconv.FormatFloat(*value.DoubleValue, 'g', -1, 64)
	}
	if value.BytesValue != nil {
		count++
		result = *value.BytesValue
	}
	if count != 1 {
		return "", errors.New("value must contain exactly one supported OTLP primitive value")
	}
	return result, nil
}
