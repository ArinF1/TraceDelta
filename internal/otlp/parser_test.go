package otlp

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/model"
)

const validDocument = `{
  "resourceSpans": [{
    "resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "checkout-service"}}]},
    "scopeSpans": [{
      "scope": {"name": "example"},
      "spans": [{
        "traceId": "11111111111111111111111111111111",
        "spanId": "2222222222222222",
        "name": "checkout.handle",
        "kind": "SPAN_KIND_SERVER",
        "startTimeUnixNano": "1000000000",
        "endTimeUnixNano": "1120000000",
        "attributes": [{"key": "http.request.method", "value": {"stringValue": "POST"}}],
        "status": {"code": "STATUS_CODE_OK"}
      }]
    }]
  }]
}`

func TestParseLegacyCompatibilityDocument(t *testing.T) {
	snapshot, err := Parse(strings.NewReader(validDocument))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(snapshot.Traces) != 1 || len(snapshot.Traces[0].Spans) != 1 {
		t.Fatalf("Parse() spans = %#v, want one trace with one span", snapshot.Traces)
	}

	span := snapshot.Traces[0].Spans[0]
	if span.Name != "checkout.handle" {
		t.Errorf("span.Name = %q, want checkout.handle", span.Name)
	}
	if span.ServiceName != "checkout-service" {
		t.Errorf("span.ServiceName = %q, want checkout-service", span.ServiceName)
	}
	if span.Kind != "SPAN_KIND_SERVER" {
		t.Errorf("span.Kind = %q, want SPAN_KIND_SERVER", span.Kind)
	}
	if span.Status != model.StatusOK {
		t.Errorf("span.Status = %q, want OK", span.Status)
	}
	if span.Duration != 120*time.Millisecond {
		t.Errorf("span.Duration = %s, want 120ms", span.Duration)
	}
	method := span.Attributes["http.request.method"]
	if method.Type != model.AttributeValueString || method.StringValue != "POST" {
		t.Errorf("http.request.method = %#v, want string POST", method)
	}
}

func TestParseRepresentativeOTLPDocument(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "testdata", "otlp-representative.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	snapshot, err := Parse(bytes.NewReader(contents))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(snapshot.Traces) != 1 || len(snapshot.Traces[0].Spans) != 2 {
		t.Fatalf("Parse() spans = %#v, want one trace with two spans", snapshot.Traces)
	}

	root := snapshot.Traces[0].Spans[0]
	if root.TraceID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || root.SpanID != "1111111111111111" {
		t.Errorf("root IDs = %q/%q, want lowercase OTLP IDs", root.TraceID, root.SpanID)
	}
	if root.Kind != "SPAN_KIND_SERVER" || root.Status != model.StatusOK {
		t.Errorf("root kind/status = %q/%q, want SERVER/OK", root.Kind, root.Status)
	}
	if root.StartTime != 1700000000000000000 || root.Duration != 125*time.Millisecond {
		t.Errorf("root timing = %d/%s, want exact timestamp and 125ms", root.StartTime, root.Duration)
	}
	if root.TraceState != "vendor=synthetic" || root.Flags != 1 {
		t.Errorf("root trace context = %q/%d, want vendor=synthetic/1", root.TraceState, root.Flags)
	}
	if root.DroppedAttributesCount != 3 || root.DroppedEventsCount != 4 || root.DroppedLinksCount != 5 {
		t.Errorf("root dropped counts = %d/%d/%d, want 3/4/5", root.DroppedAttributesCount, root.DroppedEventsCount, root.DroppedLinksCount)
	}
	if root.ResourceSchemaURL != "https://opentelemetry.io/schemas/1.38.0" || root.Resource.DroppedAttributesCount != 1 {
		t.Errorf("root resource context = %#v/%q, want retained schema and count", root.Resource, root.ResourceSchemaURL)
	}
	if root.Scope.Name != "example.instrumentation" || root.Scope.Version != "1.2.3" || root.Scope.DroppedAttributesCount != 2 {
		t.Errorf("root scope = %#v, want representative instrumentation scope", root.Scope)
	}
	if root.ScopeSchemaURL != "https://opentelemetry.io/schemas/1.38.0" {
		t.Errorf("root.ScopeSchemaURL = %q, want retained schema URL", root.ScopeSchemaURL)
	}

	assertAttribute(t, root.Attributes["http.route"], model.AttributeValue{Type: model.AttributeValueString, StringValue: "/checkout"})
	assertAttribute(t, root.Attributes["feature.enabled"], model.AttributeValue{Type: model.AttributeValueBool, BoolValue: true})
	assertAttribute(t, root.Attributes["attempt.count"], model.AttributeValue{Type: model.AttributeValueInt, IntValue: 2})
	assertAttribute(t, root.Attributes["sample.ratio"], model.AttributeValue{Type: model.AttributeValueDouble, DoubleValue: 0.5})
	assertAttribute(t, root.Attributes["test.bytes"], model.AttributeValue{Type: model.AttributeValueBytes, BytesValue: []byte("tracedelta")})
	assertAttribute(t, root.Resource.Attributes["service.instance.id"], model.AttributeValue{Type: model.AttributeValueString, StringValue: "synthetic-instance"})
	assertAttribute(t, root.Scope.Attributes["scope.enabled"], model.AttributeValue{Type: model.AttributeValueBool, BoolValue: true})

	child := snapshot.Traces[0].Spans[1]
	if child.ParentSpanID != root.SpanID || child.Kind != "SPAN_KIND_CLIENT" {
		t.Errorf("child relationship/kind = %q/%q, want root parent and CLIENT", child.ParentSpanID, child.Kind)
	}
	if child.Status != model.StatusError || child.StatusMessage != "synthetic decline" {
		t.Errorf("child status = %q/%q, want ERROR with message", child.Status, child.StatusMessage)
	}
}

func TestParseAcceptsEmptyOTLPEnvelopes(t *testing.T) {
	for _, input := range []string{
		`{}`,
		`{"resourceSpans": []}`,
		`{"resourceSpans": null}`,
		`{"resourceSpans": [{}]}`,
	} {
		snapshot, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Errorf("Parse(%s) error = %v", input, err)
			continue
		}
		if len(snapshot.Traces) != 0 {
			t.Errorf("Parse(%s) traces = %#v, want none", input, snapshot.Traces)
		}
	}
}

func TestParseAllowsOmittedResourceAndScope(t *testing.T) {
	input := `{
  "resourceSpans": [{
    "scopeSpans": [{
      "spans": [{
        "traceId": "11111111111111111111111111111111",
        "spanId": "2222222222222222",
        "name": "worker.run",
        "kind": 1,
        "startTimeUnixNano": 1,
        "endTimeUnixNano": 2
      }]
    }]
  }]
}`

	snapshot, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	span := snapshot.Traces[0].Spans[0]
	if span.ServiceName != "" || span.Resource.Attributes != nil || span.Scope.Name != "" {
		t.Errorf("optional context = service %q, resource %#v, scope %#v; want empty values", span.ServiceName, span.Resource, span.Scope)
	}
}

func TestParseAcceptsProtoJSONIntegerSpellings(t *testing.T) {
	input := strings.Replace(validDocument, `"startTimeUnixNano": "1000000000"`, `"startTimeUnixNano": 1e9`, 1)
	input = strings.Replace(input, `{"stringValue": "POST"}`, `{"intValue": 2.0}`, 1)

	snapshot, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	span := snapshot.Traces[0].Spans[0]
	if span.StartTime != 1_000_000_000 {
		t.Errorf("span.StartTime = %d, want 1000000000", span.StartTime)
	}
	assertAttribute(t, span.Attributes["http.request.method"], model.AttributeValue{Type: model.AttributeValueInt, IntValue: 2})
}

func TestParseAcceptsProtoJSONPrimitiveVariants(t *testing.T) {
	doubleTests := []struct {
		encoded string
		check   func(float64) bool
	}{
		{encoded: `"NaN"`, check: math.IsNaN},
		{encoded: `"Infinity"`, check: func(value float64) bool { return math.IsInf(value, 1) }},
		{encoded: `"-Infinity"`, check: func(value float64) bool { return math.IsInf(value, -1) }},
	}
	for _, test := range doubleTests {
		input := strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"doubleValue": `+test.encoded+`}`, 1)
		snapshot, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Parse(doubleValue %s) error = %v", test.encoded, err)
		}
		value := snapshot.Traces[0].Spans[0].Attributes["http.request.method"]
		if value.Type != model.AttributeValueDouble || !test.check(value.DoubleValue) {
			t.Errorf("doubleValue %s parsed as %#v", test.encoded, value)
		}
	}

	input := strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"bytesValue": "-_8"}`, 1)
	snapshot, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse(URL-safe bytesValue) error = %v", err)
	}
	assertAttribute(t, snapshot.Traces[0].Spans[0].Attributes["http.request.method"], model.AttributeValue{
		Type:       model.AttributeValueBytes,
		BytesValue: []byte{0xfb, 0xff},
	})
}

func TestParseAcceptsMaximumSupportedDuration(t *testing.T) {
	input := strings.Replace(validDocument, `"startTimeUnixNano": "1000000000"`, `"startTimeUnixNano": "0"`, 1)
	input = strings.Replace(input, `"endTimeUnixNano": "1120000000"`, `"endTimeUnixNano": "9223372036854775807"`, 1)

	snapshot, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := snapshot.Traces[0].Spans[0].Duration; got != time.Duration(math.MaxInt64) {
		t.Errorf("span.Duration = %s, want %s", got, time.Duration(math.MaxInt64))
	}
}

func TestParseRejectsMalformedAndUnsupportedInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "invalid JSON",
			input:     `{"resourceSpans": [`,
			wantError: "decode OTLP JSON",
		},
		{
			name:      "null document",
			input:     `null`,
			wantError: "expected a JSON object",
		},
		{
			name:      "trailing document",
			input:     validDocument + ` {}`,
			wantError: "expected exactly one JSON document",
		},
		{
			name:      "missing span name",
			input:     strings.Replace(validDocument, `"name": "checkout.handle",`, `"name": "",`, 1),
			wantError: "name must not be empty",
		},
		{
			name:      "zero trace ID",
			input:     strings.Replace(validDocument, `"11111111111111111111111111111111"`, `"00000000000000000000000000000000"`, 1),
			wantError: "traceId must not be all zeros",
		},
		{
			name:      "zero span ID",
			input:     strings.Replace(validDocument, `"2222222222222222"`, `"0000000000000000"`, 1),
			wantError: "spanId must not be all zeros",
		},
		{
			name:      "missing start time",
			input:     strings.Replace(validDocument, `"startTimeUnixNano": "1000000000",`, ``, 1),
			wantError: "startTimeUnixNano is required",
		},
		{
			name:      "end before start",
			input:     strings.Replace(validDocument, `"endTimeUnixNano": "1120000000"`, `"endTimeUnixNano": "999999999"`, 1),
			wantError: "is before startTimeUnixNano",
		},
		{
			name: "duration above supported maximum",
			input: strings.Replace(
				strings.Replace(validDocument, `"startTimeUnixNano": "1000000000"`, `"startTimeUnixNano": "0"`, 1),
				`"endTimeUnixNano": "1120000000"`,
				`"endTimeUnixNano": "9223372036854775808"`,
				1,
			),
			wantError: "span duration exceeds the supported maximum",
		},
		{
			name:      "unsupported numeric kind",
			input:     strings.Replace(validDocument, `"kind": "SPAN_KIND_SERVER"`, `"kind": 6`, 1),
			wantError: "kind value 6 is not supported",
		},
		{
			name:      "unsupported numeric status",
			input:     strings.Replace(validDocument, `"code": "STATUS_CODE_OK"`, `"code": 3`, 1),
			wantError: "status.code value 3 is not supported",
		},
		{
			name:      "service name has wrong type",
			input:     strings.Replace(validDocument, `{"stringValue": "checkout-service"}`, `{"boolValue": true}`, 1),
			wantError: "resourceSpans[0].resource: attribute service.name must use stringValue",
		},
		{
			name:      "multiple primitive value forms",
			input:     strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"stringValue": "POST", "boolValue": true}`, 1),
			wantError: `attribute[0] "http.request.method": value must contain exactly one supported OTLP primitive value`,
		},
		{
			name:      "no primitive value form",
			input:     strings.Replace(validDocument, `{"stringValue": "POST"}`, `{}`, 1),
			wantError: `attribute[0] "http.request.method": value must contain exactly one supported OTLP primitive value`,
		},
		{
			name:      "empty attribute key",
			input:     strings.Replace(validDocument, `"key": "http.request.method"`, `"key": ""`, 1),
			wantError: "attribute[0].key must not be empty",
		},
		{
			name: "duplicate attribute key",
			input: strings.Replace(
				validDocument,
				`{"key": "http.request.method", "value": {"stringValue": "POST"}}`,
				`{"key": "http.request.method", "value": {"stringValue": "POST"}}, {"key": "http.request.method", "value": {"stringValue": "GET"}}`,
				1,
			),
			wantError: `attribute[1] duplicates key "http.request.method"`,
		},
		{
			name:      "array value",
			input:     strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"arrayValue": {"values": []}}`, 1),
			wantError: "arrayValue is not supported",
		},
		{
			name:      "key value list",
			input:     strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"kvlistValue": {"values": []}}`, 1),
			wantError: "kvlistValue is not supported",
		},
		{
			name:      "invalid bytes",
			input:     strings.Replace(validDocument, `{"stringValue": "POST"}`, `{"bytesValue": "%%%"}`, 1),
			wantError: "bytesValue must be a valid",
		},
		{
			name:      "events",
			input:     strings.Replace(validDocument, `"status": {`, `"events": [{"name": "synthetic"}], "status": {`, 1),
			wantError: "events are not supported",
		},
		{
			name:      "links",
			input:     strings.Replace(validDocument, `"status": {`, `"links": [{"traceId": "11111111111111111111111111111111"}], "status": {`, 1),
			wantError: "links are not supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(test.input))
			if err == nil {
				t.Fatal("Parse() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Parse() error = %q, want it to contain %q", err, test.wantError)
			}
		})
	}
}

func assertAttribute(t *testing.T, got, want model.AttributeValue) {
	t.Helper()
	if got.Type != want.Type ||
		got.StringValue != want.StringValue ||
		got.BoolValue != want.BoolValue ||
		got.IntValue != want.IntValue ||
		got.DoubleValue != want.DoubleValue ||
		!bytes.Equal(got.BytesValue, want.BytesValue) {
		t.Errorf("attribute = %#v, want %#v", got, want)
	}
}
