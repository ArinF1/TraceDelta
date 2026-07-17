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
