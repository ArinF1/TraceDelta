package match

import (
	"testing"

	"github.com/example/tracedelta/internal/model"
)

func TestSpansPairsDuplicateKeysByOccurrence(t *testing.T) {
	key := model.SpanKey{ServiceName: "api", Name: "db.query", Kind: "SPAN_KIND_CLIENT"}
	baseline := model.NormalizedSnapshot{Spans: []model.NormalizedSpan{
		{Key: key, Occurrence: 0},
		{Key: key, Occurrence: 1},
	}}
	candidate := model.NormalizedSnapshot{Spans: []model.NormalizedSpan{
		{Key: key, Occurrence: 0},
		{Key: key, Occurrence: 1},
		{Key: key, Occurrence: 2},
	}}

	got := Spans(baseline, candidate)
	if len(got.Paired) != 2 || len(got.Added) != 1 || len(got.Removed) != 0 {
		t.Fatalf("Spans() = %#v, want 2 paired, 1 added, 0 removed", got)
	}
	if got.Added[0].Occurrence != 2 {
		t.Fatalf("added occurrence = %d, want 2", got.Added[0].Occurrence)
	}
}
