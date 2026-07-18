package normalize

import (
	"reflect"
	"testing"

	"github.com/ArinF1/TraceDelta/internal/model"
)

func TestSnapshotDropsNondeterministicFieldsAndNumbersOccurrences(t *testing.T) {
	input := model.Snapshot{Traces: []model.Trace{{
		ID: "trace-id",
		Spans: []model.Span{
			{TraceID: "trace-id", SpanID: "span-a", Name: "db.query", ServiceName: "api", Kind: "SPAN_KIND_CLIENT", InputOrder: 1},
			{TraceID: "trace-id", SpanID: "span-b", Name: "db.query", ServiceName: "api", Kind: "SPAN_KIND_CLIENT", InputOrder: 2},
		},
	}}}

	got := Snapshot(input)
	want := model.NormalizedSnapshot{Spans: []model.NormalizedSpan{
		{Key: model.SpanKey{ServiceName: "api", Name: "db.query", Kind: "SPAN_KIND_CLIENT"}, Occurrence: 0},
		{Key: model.SpanKey{ServiceName: "api", Name: "db.query", Kind: "SPAN_KIND_CLIENT"}, Occurrence: 1},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Snapshot() = %#v, want %#v", got, want)
	}
}

func TestSnapshotPreservesTypedAttributesWithoutAliasingBytes(t *testing.T) {
	bytesValue := []byte("synthetic")
	input := model.Snapshot{Traces: []model.Trace{{
		ID: "trace-id",
		Spans: []model.Span{{
			TraceID:     "trace-id",
			SpanID:      "span-id",
			Name:        "operation",
			ServiceName: "service",
			Kind:        "SPAN_KIND_INTERNAL",
			Attributes: model.Attributes{
				"attempt.count": {Type: model.AttributeValueInt, IntValue: 2},
				"test.bytes":    {Type: model.AttributeValueBytes, BytesValue: bytesValue},
			},
		}},
	}}}

	got := Snapshot(input)
	bytesValue[0] = 'X'

	if len(got.Spans) != 1 {
		t.Fatalf("len(got.Spans) = %d, want 1", len(got.Spans))
	}
	if value := got.Spans[0].Attributes["attempt.count"]; value.Type != model.AttributeValueInt || value.IntValue != 2 {
		t.Errorf("attempt.count = %#v, want typed integer 2", value)
	}
	if value := got.Spans[0].Attributes["test.bytes"]; string(value.BytesValue) != "synthetic" {
		t.Errorf("test.bytes = %#v, want an independent copy of synthetic", value)
	}
}
