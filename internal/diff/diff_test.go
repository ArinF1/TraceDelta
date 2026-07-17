package diff

import (
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/match"
	"github.com/ArinF1/TraceDelta/internal/model"
)

func TestCompareDetectsAddedAndRemovedSpans(t *testing.T) {
	matches := match.Result{
		Added:   []model.NormalizedSpan{span("inventory.reserve", model.StatusOK, 10*time.Millisecond)},
		Removed: []model.NormalizedSpan{span("cache.get", model.StatusOK, 10*time.Millisecond)},
	}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if got.AddedSpans != 1 || got.RemovedSpans != 1 {
		t.Fatalf("Compare() summary = %#v, want one added and one removed", got)
	}
	if len(got.Changes) != 2 || got.Changes[0].Kind != KindAdded || got.Changes[1].Kind != KindRemoved {
		t.Fatalf("Compare() changes = %#v, want added then removed", got.Changes)
	}
}

func TestCompareDetectsStatusChange(t *testing.T) {
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("checkout.handle", model.StatusOK, 100*time.Millisecond),
		Candidate: span("checkout.handle", model.StatusError, 100*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if got.ChangedSpans != 1 || len(got.Changes) != 1 {
		t.Fatalf("Compare() = %#v, want one changed span", got)
	}
	change := got.Changes[0]
	if change.Field != FieldStatus || change.Before != "OK" || change.After != "ERROR" {
		t.Fatalf("status change = %#v, want OK -> ERROR", change)
	}
}

func TestCompareDetectsDurationAtThreshold(t *testing.T) {
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("payment.charge", model.StatusOK, 100*time.Millisecond),
		Candidate: span("payment.charge", model.StatusOK, 120*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(got.Changes) != 1 || got.Changes[0].Field != FieldDuration {
		t.Fatalf("Compare() changes = %#v, want one duration change", got.Changes)
	}
}

func TestCompareIgnoresDurationBelowThreshold(t *testing.T) {
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("payment.charge", model.StatusOK, 100*time.Millisecond),
		Candidate: span("payment.charge", model.StatusOK, 119*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if got.HasDifferences() {
		t.Fatalf("Compare() changes = %#v, want none", got.Changes)
	}
}

func TestCompareIgnoresDurationDecrease(t *testing.T) {
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("payment.charge", model.StatusOK, 100*time.Millisecond),
		Candidate: span("payment.charge", model.StatusOK, 50*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if got.HasDifferences() {
		t.Fatalf("Compare() changes = %#v, want duration improvements ignored", got.Changes)
	}
}

func TestCompareOrdersChangesDeterministically(t *testing.T) {
	matches := match.Result{
		Added: []model.NormalizedSpan{
			span("z.added", model.StatusOK, time.Millisecond),
			span("a.added", model.StatusOK, time.Millisecond),
		},
		Removed: []model.NormalizedSpan{
			span("z.removed", model.StatusOK, time.Millisecond),
			span("a.removed", model.StatusOK, time.Millisecond),
		},
		Paired: []match.SpanPair{{
			Baseline:  span("changed", model.StatusOK, 100*time.Millisecond),
			Candidate: span("changed", model.StatusError, 200*time.Millisecond),
		}},
	}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	want := []struct {
		kind  Kind
		name  string
		field Field
	}{
		{KindAdded, "a.added", FieldNone},
		{KindAdded, "z.added", FieldNone},
		{KindRemoved, "a.removed", FieldNone},
		{KindRemoved, "z.removed", FieldNone},
		{KindChanged, "changed", FieldStatus},
		{KindChanged, "changed", FieldDuration},
	}
	if len(got.Changes) != len(want) {
		t.Fatalf("len(changes) = %d, want %d", len(got.Changes), len(want))
	}
	for index, expected := range want {
		change := got.Changes[index]
		if change.Kind != expected.kind || change.SpanName != expected.name || change.Field != expected.field {
			t.Errorf("change[%d] = %#v, want kind=%s name=%s field=%s", index, change, expected.kind, expected.name, expected.field)
		}
	}
}

func span(name string, status model.StatusCode, duration time.Duration) model.NormalizedSpan {
	return model.NormalizedSpan{
		Key:      model.SpanKey{ServiceName: "checkout-service", Name: name, Kind: "SPAN_KIND_INTERNAL"},
		Status:   status,
		Duration: duration,
	}
}
