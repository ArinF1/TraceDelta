// Package match pairs normalized baseline and candidate spans.
package match

import "github.com/example/tracedelta/internal/model"

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
	candidateIndexes := make(map[identity]int, len(candidate.Spans))
	for index, span := range candidate.Spans {
		candidateIndexes[spanIdentity(span)] = index
	}

	usedCandidates := make([]bool, len(candidate.Spans))
	result := Result{}
	for _, baselineSpan := range baseline.Spans {
		candidateIndex, ok := candidateIndexes[spanIdentity(baselineSpan)]
		if !ok {
			result.Removed = append(result.Removed, baselineSpan)
			continue
		}
		usedCandidates[candidateIndex] = true
		result.Paired = append(result.Paired, SpanPair{
			Baseline:  baselineSpan,
			Candidate: candidate.Spans[candidateIndex],
		})
	}

	for index, candidateSpan := range candidate.Spans {
		if !usedCandidates[index] {
			result.Added = append(result.Added, candidateSpan)
		}
	}
	return result
}

func spanIdentity(span model.NormalizedSpan) identity {
	return identity{key: span.Key, occurrence: span.Occurrence}
}
