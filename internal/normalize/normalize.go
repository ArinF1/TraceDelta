// Package normalize removes nondeterministic identifiers and timestamps from
// parsed spans before matching.
package normalize

import (
	"sort"

	"github.com/ArinF1/TraceDelta/internal/model"
)

// Snapshot flattens a parsed snapshot in input order and assigns occurrence
// numbers to duplicate stable span keys. Trace IDs, span IDs, parent IDs, and
// absolute timestamps are deliberately omitted from the result.
func Snapshot(input model.Snapshot) model.NormalizedSnapshot {
	spans := make([]model.Span, 0)
	for _, trace := range input.Traces {
		spans = append(spans, trace.Spans...)
	}
	sort.SliceStable(spans, func(i, j int) bool {
		return spans[i].InputOrder < spans[j].InputOrder
	})

	occurrences := make(map[model.SpanKey]int)
	result := model.NormalizedSnapshot{Spans: make([]model.NormalizedSpan, 0, len(spans))}
	for _, span := range spans {
		key := model.SpanKey{
			ServiceName: span.ServiceName,
			Name:        span.Name,
			Kind:        span.Kind,
		}
		occurrence := occurrences[key]
		occurrences[key] = occurrence + 1

		result.Spans = append(result.Spans, model.NormalizedSpan{
			Key:           key,
			Occurrence:    occurrence,
			Duration:      span.Duration,
			Status:        span.Status,
			StatusMessage: span.StatusMessage,
			Attributes:    cloneAttributes(span.Attributes),
		})
	}
	return result
}

func cloneAttributes(attributes model.Attributes) model.Attributes {
	if attributes == nil {
		return nil
	}
	clone := make(model.Attributes, len(attributes))
	for key, value := range attributes {
		if value.BytesValue != nil {
			value.BytesValue = append([]byte(nil), value.BytesValue...)
		}
		clone[key] = value
	}
	return clone
}
