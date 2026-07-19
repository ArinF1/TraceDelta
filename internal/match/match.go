// Package match pairs normalized baseline and candidate spans.
package match

import "github.com/ArinF1/TraceDelta/internal/model"

// SpanPair contains two spans that share a stable key and occurrence number.
type SpanPair struct {
	Baseline  model.NormalizedSpan
	Candidate model.NormalizedSpan
}

// Result partitions spans into matched, added, and removed groups.
type Result struct {
	Paired  []SpanPair
	Added   []model.NormalizedSpan
	Removed []model.NormalizedSpan
}

type identity struct {
	key        model.SpanKey
	occurrence int
}

// Spans matches on service name, span name, kind, and occurrence order. It
// intentionally does not use trace IDs, span IDs, or timestamps.
func Spans(baseline, candidate model.NormalizedSnapshot) Result {
	baselineSpans := flattenSpans(baseline)
	candidateSpans := flattenSpans(candidate)
	candidateIndexes := make(map[identity]int, len(candidateSpans))
	for index, span := range candidateSpans {
		candidateIndexes[spanIdentity(span)] = index
	}

	usedCandidates := make([]bool, len(candidateSpans))
	result := Result{}
	for _, baselineSpan := range baselineSpans {
		candidateIndex, ok := candidateIndexes[spanIdentity(baselineSpan)]
		if !ok {
			result.Removed = append(result.Removed, baselineSpan)
			continue
		}
		usedCandidates[candidateIndex] = true
		result.Paired = append(result.Paired, SpanPair{
			Baseline:  baselineSpan,
			Candidate: candidateSpans[candidateIndex],
		})
	}

	for index, candidateSpan := range candidateSpans {
		if !usedCandidates[index] {
			result.Added = append(result.Added, candidateSpan)
		}
	}
	return result
}

func spanIdentity(span model.NormalizedSpan) identity {
	return identity{key: span.Key, occurrence: span.Occurrence}
}

func flattenSpans(snapshot model.NormalizedSnapshot) []model.NormalizedSpan {
	count := 0
	for _, trace := range snapshot.Traces {
		count += len(trace.Spans)
	}
	spans := make([]model.NormalizedSpan, 0, count)
	for _, trace := range snapshot.Traces {
		spans = append(spans, trace.Spans...)
	}
	return spans
}
