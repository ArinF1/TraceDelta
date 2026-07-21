package match

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ArinF1/TraceDelta/internal/model"
)

func FuzzTraceAndSpanMatchingIgnoresTraceOrder(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5})
	f.Add([]byte{255, 17, 42})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, order []byte) {
		traceCount := 1
		if len(order) > 0 {
			traceCount += int(order[0] % 8)
		}

		baselineTraces := make([]model.NormalizedTrace, traceCount)
		candidateTraces := make([]model.NormalizedTrace, traceCount)
		for index := 0; index < traceCount; index++ {
			baselineTraces[index] = trace(
				fmt.Sprintf("operation-%d", index),
				"fuzz-service",
				"SPAN_KIND_SERVER",
				fmt.Sprintf("/fuzz/%d", index),
				fmt.Sprintf("child-%d", index),
			)
			candidateTraces[index] = cloneTrace(baselineTraces[index])
		}

		wantTraces, err := Traces(
			model.NormalizedSnapshot{Traces: baselineTraces},
			model.NormalizedSnapshot{Traces: candidateTraces},
		)
		if err != nil {
			t.Fatalf("canonical Traces() error = %v", err)
		}
		wantSpans, err := Spans(wantTraces)
		if err != nil {
			t.Fatalf("canonical Spans() error = %v", err)
		}

		gotTraces, err := Traces(
			model.NormalizedSnapshot{Traces: permuteTraces(baselineTraces, order, 0)},
			model.NormalizedSnapshot{Traces: permuteTraces(candidateTraces, order, 1)},
		)
		if err != nil {
			t.Fatalf("reordered Traces() error = %v", err)
		}
		gotSpans, err := Spans(gotTraces)
		if err != nil {
			t.Fatalf("reordered Spans() error = %v", err)
		}

		if !reflect.DeepEqual(gotTraces, wantTraces) {
			t.Fatalf("reordered trace result differs\ngot:  %#v\nwant: %#v", gotTraces, wantTraces)
		}
		if !reflect.DeepEqual(gotSpans, wantSpans) {
			t.Fatalf("reordered span result differs\ngot:  %#v\nwant: %#v", gotSpans, wantSpans)
		}
	})
}

func permuteTraces(traces []model.NormalizedTrace, order []byte, offset int) []model.NormalizedTrace {
	result := cloneTraces(traces)
	if len(order) == 0 {
		return result
	}
	for index := len(result) - 1; index > 0; index-- {
		key := order[(index+offset)%len(order)]
		swapWith := int(key) % (index + 1)
		result[index], result[swapWith] = result[swapWith], result[index]
	}
	return result
}
