package diff

import (
	"math"
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
	candidate := span("checkout.handle", model.StatusError, 100*time.Millisecond)
	candidate.ErrorType = "synthetic.Declined"
	candidate.ErrorTypePresent = true
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("checkout.handle", model.StatusOK, 100*time.Millisecond),
		Candidate: candidate,
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

func TestCompareDetectsErrorToOKAsOneStatusChange(t *testing.T) {
	baseline := span("checkout.handle", model.StatusError, 100*time.Millisecond)
	baseline.ErrorType = "synthetic.Declined"
	baseline.ErrorTypePresent = true
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  baseline,
		Candidate: span("checkout.handle", model.StatusOK, 100*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(got.Changes) != 1 || got.Changes[0].Field != FieldStatus || got.Changes[0].Before != "ERROR" || got.Changes[0].After != "OK" {
		t.Fatalf("Compare() changes = %#v, want one ERROR -> OK status change", got.Changes)
	}
}

func TestCompareErrorTypeEvidence(t *testing.T) {
	tests := []struct {
		name              string
		baselineStatus    model.StatusCode
		candidateStatus   model.StatusCode
		baselineType      string
		candidateType     string
		baselinePresent   bool
		candidatePresent  bool
		wantChange        bool
		wantBeforePresent bool
		wantAfterPresent  bool
	}{
		{name: "changed error type", baselineStatus: model.StatusError, candidateStatus: model.StatusError, baselineType: "synthetic.Timeout", candidateType: "synthetic.Declined", baselinePresent: true, candidatePresent: true, wantChange: true, wantBeforePresent: true, wantAfterPresent: true},
		{name: "new evidence", baselineStatus: model.StatusError, candidateStatus: model.StatusError, candidateType: "synthetic.Declined", candidatePresent: true, wantChange: true, wantAfterPresent: true},
		{name: "removed evidence", baselineStatus: model.StatusError, candidateStatus: model.StatusError, baselineType: "synthetic.Timeout", baselinePresent: true, wantChange: true, wantBeforePresent: true},
		{name: "unchanged error", baselineStatus: model.StatusError, candidateStatus: model.StatusError, baselineType: "synthetic.Timeout", candidateType: "synthetic.Timeout", baselinePresent: true, candidatePresent: true},
		{name: "both missing", baselineStatus: model.StatusError, candidateStatus: model.StatusError},
		{name: "non-error attributes ignored", baselineStatus: model.StatusOK, candidateStatus: model.StatusOK, baselineType: "synthetic.Timeout", candidateType: "synthetic.Declined", baselinePresent: true, candidatePresent: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseline := span("checkout.handle", test.baselineStatus, 100*time.Millisecond)
			baseline.ErrorType = test.baselineType
			baseline.ErrorTypePresent = test.baselinePresent
			candidate := span("checkout.handle", test.candidateStatus, 100*time.Millisecond)
			candidate.ErrorType = test.candidateType
			candidate.ErrorTypePresent = test.candidatePresent

			got, err := Compare(match.Result{Paired: []match.SpanPair{{Baseline: baseline, Candidate: candidate}}}, Options{DurationThreshold: 0.20})
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if !test.wantChange {
				if got.HasDifferences() {
					t.Fatalf("Compare() changes = %#v, want none", got.Changes)
				}
				return
			}
			if len(got.Changes) != 1 {
				t.Fatalf("Compare() changes = %#v, want one error.type change", got.Changes)
			}
			change := got.Changes[0]
			if change.Field != FieldErrorType || change.Before != test.baselineType || change.After != test.candidateType || change.BeforePresent != test.wantBeforePresent || change.AfterPresent != test.wantAfterPresent {
				t.Fatalf("error.type change = %#v, want before=%q/%t after=%q/%t", change, test.baselineType, test.wantBeforePresent, test.candidateType, test.wantAfterPresent)
			}
		})
	}
}

func TestCompareDetectsDurationAtThreshold(t *testing.T) {
	matches := match.Result{Paired: []match.SpanPair{{
		Baseline:  span("payment.charge", model.StatusOK, 100*time.Millisecond),
		Candidate: span("payment.charge", model.StatusOK, 120*time.Millisecond),
	}}}

	got, err := Compare(matches, Options{DurationThreshold: 0.20, DurationThresholdAbsolute: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(got.Changes) != 1 || got.Changes[0].Field != FieldDuration {
		t.Fatalf("Compare() changes = %#v, want one duration change", got.Changes)
	}
	if got.Changes[0].DurationThresholdRelative != 0.20 || got.Changes[0].DurationThresholdAbsolute != 20*time.Millisecond {
		t.Fatalf("duration policy evidence = %#v, want 20%% and 20ms", got.Changes[0])
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

func TestCompareRequiresRelativeAndAbsoluteDurationThresholds(t *testing.T) {
	tests := []struct {
		name      string
		baseline  time.Duration
		candidate time.Duration
		relative  float64
		absolute  time.Duration
		want      bool
	}{
		{name: "both boundaries equal", baseline: 100 * time.Millisecond, candidate: 120 * time.Millisecond, relative: 0.20, absolute: 20 * time.Millisecond, want: true},
		{name: "absolute boundary equal", baseline: 100 * time.Millisecond, candidate: 110 * time.Millisecond, relative: 0.10, absolute: 10 * time.Millisecond, want: true},
		{name: "relative passes absolute fails", baseline: time.Millisecond, candidate: 3 * time.Millisecond, relative: 1.0, absolute: 10 * time.Millisecond},
		{name: "absolute passes relative fails", baseline: time.Second, candidate: 1150 * time.Millisecond, relative: 0.20, absolute: 10 * time.Millisecond},
		{name: "zero baseline at absolute boundary", baseline: 0, candidate: 10 * time.Millisecond, relative: 0.20, absolute: 10 * time.Millisecond, want: true},
		{name: "zero baseline below absolute", baseline: 0, candidate: 9 * time.Millisecond, relative: 0.20, absolute: 10 * time.Millisecond},
		{name: "large regression", baseline: time.Second, candidate: 2 * time.Second, relative: 0.20, absolute: 10 * time.Millisecond, want: true},
		{name: "both thresholds disabled", baseline: time.Second, candidate: time.Second + time.Nanosecond, relative: 0, absolute: 0, want: true},
		{name: "decrease", baseline: time.Second, candidate: 500 * time.Millisecond, relative: 0, absolute: 0},
		{name: "equal", baseline: time.Second, candidate: time.Second, relative: 0, absolute: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matches := match.Result{Paired: []match.SpanPair{{
				Baseline:  span("operation", model.StatusOK, test.baseline),
				Candidate: span("operation", model.StatusOK, test.candidate),
			}}}
			got, err := Compare(matches, Options{DurationThreshold: test.relative, DurationThresholdAbsolute: test.absolute})
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if got.HasDifferences() != test.want {
				t.Fatalf("Compare() HasDifferences = %t, want %t; changes=%#v", got.HasDifferences(), test.want, got.Changes)
			}
		})
	}
}

func TestCompareRejectsInvalidDurationThresholds(t *testing.T) {
	for _, options := range []Options{
		{DurationThreshold: -0.1},
		{DurationThreshold: math.NaN()},
		{DurationThreshold: math.Inf(1)},
		{DurationThresholdAbsolute: -time.Nanosecond},
	} {
		if _, err := Compare(match.Result{}, options); err == nil {
			t.Fatalf("Compare(%#v) error = nil, want invalid-threshold error", options)
		}
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
