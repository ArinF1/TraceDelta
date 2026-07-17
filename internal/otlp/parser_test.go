package otlp

import (
	"strings"
	"testing"
	"time"
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

func TestParseValidDocument(t *testing.T) {
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
	if span.Duration != 120*time.Millisecond {
		t.Errorf("span.Duration = %s, want 120ms", span.Duration)
	}
	if got := span.Attributes["http.request.method"]; got != "POST" {
		t.Errorf("http.request.method = %q, want POST", got)
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
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
			name:      "missing span name",
			input:     strings.Replace(validDocument, `"name": "checkout.handle",`, `"name": "",`, 1),
			wantError: "name must not be empty",
		},
		{
			name:      "unknown field",
			input:     strings.Replace(validDocument, `"name": "checkout.handle",`, `"name": "checkout.handle", "unsupported": true,`, 1),
			wantError: "unknown field",
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
