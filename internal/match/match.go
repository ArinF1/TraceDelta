// Package match pairs normalized baseline and candidate traces and spans.
package match

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ArinF1/TraceDelta/internal/model"
)

const (
	signalRootOperation     = "root operation"
	signalRootService       = "root service"
	signalRootKind          = "root kind"
	signalStableAttributes  = "safe root attributes"
	signalExactStructure    = "exact span structure"
	signalStructuralOverlap = "unique structural overlap"
	ambiguitySignalSummary  = "root operation, service, kind, safe attributes, and structure"
)

// TraceMatchEvidence explains which non-sensitive signal categories established
// a trace pair. Attribute values are deliberately not copied into evidence.
type TraceMatchEvidence struct {
	Signals           []string
	StructuralOverlap int
}

// TracePair contains two traces matched one-to-one plus explainable evidence.
type TracePair struct {
	Baseline  model.NormalizedTrace
	Candidate model.NormalizedTrace
	Evidence  TraceMatchEvidence
}

// TraceResult partitions traces into matched, added, and removed groups.
type TraceResult struct {
	Paired  []TracePair
	Added   []model.NormalizedTrace
	Removed []model.NormalizedTrace
}

// AmbiguityError reports a trace group that cannot be paired without guessing.
// It intentionally omits trace-derived names and attribute values.
type AmbiguityError struct {
	BaselineCount  int
	CandidateCount int
}

func (err *AmbiguityError) Error() string {
	return fmt.Sprintf(
		"ambiguous trace match: %d baseline and %d candidate traces remain indistinguishable after %s",
		err.BaselineCount,
		err.CandidateCount,
		ambiguitySignalSummary,
	)
}

type traceDescriptor struct {
	trace        model.NormalizedTrace
	groupKey     string
	structureKey string
	orderKey     string
	shape        map[spanShape]int
	roots        []rootDescriptor
}

type rootDescriptor struct {
	Key        model.SpanKey
	Attributes string
}

type spanShape struct {
	Key              model.SpanKey
	Attributes       string
	ParentKind       model.ParentKind
	ParentKey        model.SpanKey
	ParentAttributes string
}

// Traces matches normalized traces deterministically without using raw IDs,
// wall-clock timestamps, status, or duration. An unresolved tie is returned as
// an AmbiguityError instead of being hidden behind input order.
func Traces(baseline, candidate model.NormalizedSnapshot) (TraceResult, error) {
	baselineTraces := describeTraces(baseline)
	candidateTraces := describeTraces(candidate)
	baselineGroups := groupTraces(baselineTraces)
	candidateGroups := groupTraces(candidateTraces)

	groupKeys := make([]string, 0, len(baselineGroups)+len(candidateGroups))
	seenKeys := make(map[string]struct{}, len(baselineGroups)+len(candidateGroups))
	for key := range baselineGroups {
		seenKeys[key] = struct{}{}
		groupKeys = append(groupKeys, key)
	}
	for key := range candidateGroups {
		if _, exists := seenKeys[key]; exists {
			continue
		}
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	result := TraceResult{}
	for _, key := range groupKeys {
		baselineGroup := baselineGroups[key]
		candidateGroup := candidateGroups[key]
		switch {
		case len(baselineGroup) == 0:
			result.Added = appendTraces(result.Added, candidateGroup)
		case len(candidateGroup) == 0:
			result.Removed = appendTraces(result.Removed, baselineGroup)
		default:
			groupResult, err := matchTraceGroup(baselineGroup, candidateGroup)
			if err != nil {
				return TraceResult{}, err
			}
			result.Paired = append(result.Paired, groupResult.Paired...)
			result.Added = append(result.Added, groupResult.Added...)
			result.Removed = append(result.Removed, groupResult.Removed...)
		}
	}
	return result, nil
}

func matchTraceGroup(baseline, candidate []traceDescriptor) (TraceResult, error) {
	usedBaseline := make([]bool, len(baseline))
	usedCandidate := make([]bool, len(candidate))
	result := TraceResult{}

	baselineStructures := indexesByStructure(baseline)
	candidateStructures := indexesByStructure(candidate)
	structureKeys := make([]string, 0, len(baselineStructures))
	for key := range baselineStructures {
		structureKeys = append(structureKeys, key)
	}
	sort.Strings(structureKeys)
	for _, key := range structureKeys {
		baselineIndexes := baselineStructures[key]
		candidateIndexes := candidateStructures[key]
		if len(baselineIndexes) != 1 || len(candidateIndexes) != 1 {
			continue
		}
		baselineIndex := baselineIndexes[0]
		candidateIndex := candidateIndexes[0]
		usedBaseline[baselineIndex] = true
		usedCandidate[candidateIndex] = true
		result.Paired = append(result.Paired, tracePair(
			baseline[baselineIndex],
			candidate[candidateIndex],
			true,
		))
	}

	for {
		remainingBaseline := unusedIndexes(usedBaseline)
		remainingCandidate := unusedIndexes(usedCandidate)
		if len(remainingBaseline) == 0 || len(remainingCandidate) == 0 {
			result.Removed = appendDescriptorIndexes(result.Removed, baseline, remainingBaseline)
			result.Added = appendDescriptorIndexes(result.Added, candidate, remainingCandidate)
			return result, nil
		}

		baselineBest := make(map[int]int, len(remainingBaseline))
		candidateBest := make(map[int]int, len(remainingCandidate))
		for _, baselineIndex := range remainingBaseline {
			bestIndex, unique := uniqueBestCandidate(baseline[baselineIndex], candidate, remainingCandidate)
			if unique {
				baselineBest[baselineIndex] = bestIndex
			}
		}
		for _, candidateIndex := range remainingCandidate {
			bestIndex, unique := uniqueBestCandidate(candidate[candidateIndex], baseline, remainingBaseline)
			if unique {
				candidateBest[candidateIndex] = bestIndex
			}
		}

		progress := false
		for _, baselineIndex := range remainingBaseline {
			candidateIndex, baselineUnique := baselineBest[baselineIndex]
			if !baselineUnique || usedCandidate[candidateIndex] {
				continue
			}
			candidateBaseline, candidateUnique := candidateBest[candidateIndex]
			if !candidateUnique || candidateBaseline != baselineIndex {
				continue
			}
			usedBaseline[baselineIndex] = true
			usedCandidate[candidateIndex] = true
			result.Paired = append(result.Paired, tracePair(
				baseline[baselineIndex],
				candidate[candidateIndex],
				false,
			))
			progress = true
		}
		if !progress {
			return TraceResult{}, &AmbiguityError{
				BaselineCount:  len(remainingBaseline),
				CandidateCount: len(remainingCandidate),
			}
		}
	}
}

func tracePair(baseline, candidate traceDescriptor, exactStructure bool) TracePair {
	signals := rootSignals(baseline.roots)
	if exactStructure {
		signals = append(signals, signalExactStructure)
	} else {
		signals = append(signals, signalStructuralOverlap)
	}
	return TracePair{
		Baseline:  baseline.trace,
		Candidate: candidate.trace,
		Evidence: TraceMatchEvidence{
			Signals:           signals,
			StructuralOverlap: structuralOverlap(baseline.shape, candidate.shape),
		},
	}
}

func rootSignals(roots []rootDescriptor) []string {
	signals := []string{signalRootOperation}
	hasService := false
	hasKind := false
	hasAttributes := false
	for _, root := range roots {
		hasService = hasService || root.Key.ServiceName != ""
		hasKind = hasKind || root.Key.Kind != ""
		hasAttributes = hasAttributes || root.Attributes != "[]"
	}
	if hasService {
		signals = append(signals, signalRootService)
	}
	if hasKind {
		signals = append(signals, signalRootKind)
	}
	if hasAttributes {
		signals = append(signals, signalStableAttributes)
	}
	return signals
}

func describeTraces(snapshot model.NormalizedSnapshot) []traceDescriptor {
	result := make([]traceDescriptor, 0, len(snapshot.Traces))
	for _, trace := range snapshot.Traces {
		if len(trace.Spans) == 0 {
			continue
		}
		roots := traceRoots(trace)
		shape := traceShape(trace)
		result = append(result, traceDescriptor{
			trace:        trace,
			groupKey:     encode(roots),
			structureKey: encodeShape(shape),
			orderKey:     encode(trace),
			shape:        shape,
			roots:        roots,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].orderKey < result[j].orderKey
	})
	return result
}

func traceRoots(trace model.NormalizedTrace) []rootDescriptor {
	indexes := make([]int, 0)
	for index, span := range trace.Spans {
		if span.Parent.Kind == model.ParentRoot {
			indexes = append(indexes, index)
		}
	}
	if len(indexes) == 0 {
		for index, span := range trace.Spans {
			if span.Parent.Kind == model.ParentExternal {
				indexes = append(indexes, index)
			}
		}
	}

	result := make([]rootDescriptor, 0, len(indexes))
	for _, index := range indexes {
		span := trace.Spans[index]
		result = append(result, rootDescriptor{
			Key:        span.Key,
			Attributes: encodeAttributes(span.Attributes),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return encode(result[i]) < encode(result[j])
	})
	return result
}

func traceShape(trace model.NormalizedTrace) map[spanShape]int {
	result := make(map[spanShape]int, len(trace.Spans))
	for _, span := range trace.Spans {
		shape := spanShape{
			Key:        span.Key,
			Attributes: encodeAttributes(span.Attributes),
			ParentKind: span.Parent.Kind,
		}
		if span.Parent.Kind == model.ParentSpan && span.Parent.SpanIndex >= 0 && span.Parent.SpanIndex < len(trace.Spans) {
			parent := trace.Spans[span.Parent.SpanIndex]
			shape.ParentKey = parent.Key
			shape.ParentAttributes = encodeAttributes(parent.Attributes)
		}
		result[shape]++
	}
	return result
}

func encodeShape(shape map[spanShape]int) string {
	entries := make([]string, 0, len(shape))
	for token, count := range shape {
		entries = append(entries, fmt.Sprintf("%s:%d", encode(token), count))
	}
	sort.Strings(entries)
	return encode(entries)
}

func encode(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode normalized match key: %v", err))
	}
	return string(encoded)
}

func encodeAttributes(attributes []model.NormalizedAttribute) string {
	if len(attributes) == 0 {
		return "[]"
	}
	return encode(attributes)
}

func groupTraces(traces []traceDescriptor) map[string][]traceDescriptor {
	result := make(map[string][]traceDescriptor)
	for _, trace := range traces {
		result[trace.groupKey] = append(result[trace.groupKey], trace)
	}
	return result
}

func indexesByStructure(traces []traceDescriptor) map[string][]int {
	result := make(map[string][]int)
	for index, trace := range traces {
		result[trace.structureKey] = append(result[trace.structureKey], index)
	}
	return result
}

func uniqueBestCandidate(source traceDescriptor, targets []traceDescriptor, indexes []int) (int, bool) {
	bestIndex := -1
	bestScore := -1
	bestCount := 0
	for _, index := range indexes {
		score := structuralOverlap(source.shape, targets[index].shape)
		switch {
		case score > bestScore:
			bestIndex = index
			bestScore = score
			bestCount = 1
		case score == bestScore:
			bestCount++
		}
	}
	return bestIndex, bestCount == 1
}

func structuralOverlap(left, right map[spanShape]int) int {
	score := 0
	for token, leftCount := range left {
		rightCount := right[token]
		if rightCount < leftCount {
			score += rightCount
		} else {
			score += leftCount
		}
	}
	return score
}

func unusedIndexes(used []bool) []int {
	result := make([]int, 0, len(used))
	for index, isUsed := range used {
		if !isUsed {
			result = append(result, index)
		}
	}
	return result
}

func appendTraces(destination []model.NormalizedTrace, descriptors []traceDescriptor) []model.NormalizedTrace {
	for _, descriptor := range descriptors {
		destination = append(destination, descriptor.trace)
	}
	return destination
}

func appendDescriptorIndexes(destination []model.NormalizedTrace, descriptors []traceDescriptor, indexes []int) []model.NormalizedTrace {
	for _, index := range indexes {
		destination = append(destination, descriptors[index].trace)
	}
	return destination
}

const (
	signalSpanOperation      = "span operation"
	signalSpanService        = "span service"
	signalSpanKind           = "span kind"
	signalSpanAttributes     = "safe span attributes"
	signalMatchedParent      = "matched parent"
	signalTopLevelRelation   = "same top-level relationship"
	signalSiblingOccurrence  = "sibling occurrence"
	signalRelationshipChange = "relationship-change fallback"
	spanAmbiguitySummary     = "operation, service, kind, safe attributes, parent context, and sibling order"
)

// SpanMatchEvidence explains the non-sensitive signal categories used for one
// span pair. It never contains attribute values.
type SpanMatchEvidence struct {
	Signals []string
}

// SpanPair contains two spans matched one-to-one inside a trace pair.
type SpanPair struct {
	Baseline  model.NormalizedSpan
	Candidate model.NormalizedSpan
	Evidence  SpanMatchEvidence
}

// Result partitions spans into matched, added, and removed groups.
type Result struct {
	Paired  []SpanPair
	Added   []model.NormalizedSpan
	Removed []model.NormalizedSpan
}

// SpanAmbiguityError reports a span group that cannot be paired without
// guessing. Trace-derived names and attribute values are intentionally omitted.
type SpanAmbiguityError struct {
	BaselineCount  int
	CandidateCount int
}

func (err *SpanAmbiguityError) Error() string {
	return fmt.Sprintf(
		"ambiguous span match: %d baseline and %d candidate spans remain indistinguishable after %s",
		err.BaselineCount,
		err.CandidateCount,
		spanAmbiguitySummary,
	)
}

type spanDescriptor struct {
	index       int
	span        model.NormalizedSpan
	semanticKey string
	orderKey    string
}

type spanGroupResult struct {
	pairs              [][2]int
	remainingBaseline  []int
	remainingCandidate []int
}

// Spans matches spans structurally and semantically inside matched traces.
// It uses matched-parent context first, then a relationship-change fallback for
// otherwise corresponding spans. Unresolved duplicate ties return an error.
func Spans(traces TraceResult) (Result, error) {
	result := Result{}
	for _, pair := range traces.Paired {
		if err := pairSpans(pair.Baseline.Spans, pair.Candidate.Spans, &result); err != nil {
			return Result{}, err
		}
	}
	for _, trace := range traces.Removed {
		result.Removed = append(result.Removed, trace.Spans...)
	}
	for _, trace := range traces.Added {
		result.Added = append(result.Added, trace.Spans...)
	}
	sortSpanResult(&result)
	return result, nil
}

func pairSpans(baseline, candidate []model.NormalizedSpan, result *Result) error {
	baselineDescriptors := describeSpans(baseline)
	candidateDescriptors := describeSpans(candidate)
	usedBaseline := make([]bool, len(baseline))
	usedCandidate := make([]bool, len(candidate))
	baselineParentPairs := make(map[int]int)
	candidateParentPairs := make(map[int]int)
	nextParentPair := 0

	for {
		baselineGroups := eligibleSpanGroups(baselineDescriptors, usedBaseline, baselineParentPairs, true)
		candidateGroups := eligibleSpanGroups(candidateDescriptors, usedCandidate, candidateParentPairs, true)
		keys := sharedSortedKeys(baselineGroups, candidateGroups)
		progress := false
		for _, key := range keys {
			group, err := pairSpanGroup(
				baselineDescriptors,
				candidateDescriptors,
				baselineGroups[key],
				candidateGroups[key],
			)
			if err != nil {
				return err
			}
			for _, indexes := range group.pairs {
				baselineIndex := indexes[0]
				candidateIndex := indexes[1]
				if usedBaseline[baselineIndex] || usedCandidate[candidateIndex] {
					continue
				}
				usedBaseline[baselineIndex] = true
				usedCandidate[candidateIndex] = true
				baselineParentPairs[baselineIndex] = nextParentPair
				candidateParentPairs[candidateIndex] = nextParentPair
				nextParentPair++
				result.Paired = append(result.Paired, SpanPair{
					Baseline:  baseline[baselineIndex],
					Candidate: candidate[candidateIndex],
					Evidence: SpanMatchEvidence{Signals: spanSignals(
						baseline[baselineIndex],
						true,
						baseline[baselineIndex].StartOrder != candidate[candidateIndex].StartOrder,
					)},
				})
				progress = true
			}
		}
		if !progress {
			break
		}
	}

	baselineGroups := eligibleSpanGroups(baselineDescriptors, usedBaseline, nil, false)
	candidateGroups := eligibleSpanGroups(candidateDescriptors, usedCandidate, nil, false)
	for _, key := range sharedSortedKeys(baselineGroups, candidateGroups) {
		group, err := pairSpanGroup(
			baselineDescriptors,
			candidateDescriptors,
			baselineGroups[key],
			candidateGroups[key],
		)
		if err != nil {
			return err
		}
		for _, indexes := range group.pairs {
			baselineIndex := indexes[0]
			candidateIndex := indexes[1]
			if usedBaseline[baselineIndex] || usedCandidate[candidateIndex] {
				continue
			}
			usedBaseline[baselineIndex] = true
			usedCandidate[candidateIndex] = true
			result.Paired = append(result.Paired, SpanPair{
				Baseline:  baseline[baselineIndex],
				Candidate: candidate[candidateIndex],
				Evidence: SpanMatchEvidence{Signals: spanSignals(
					baseline[baselineIndex],
					false,
					baseline[baselineIndex].StartOrder != candidate[candidateIndex].StartOrder,
				)},
			})
		}
	}

	for index, span := range baseline {
		if !usedBaseline[index] {
			result.Removed = append(result.Removed, span)
		}
	}
	for index, span := range candidate {
		if !usedCandidate[index] {
			result.Added = append(result.Added, span)
		}
	}
	return nil
}

func describeSpans(spans []model.NormalizedSpan) []spanDescriptor {
	result := make([]spanDescriptor, len(spans))
	for index, span := range spans {
		result[index] = spanDescriptor{
			index: index,
			span:  span,
			semanticKey: encode(struct {
				Key        model.SpanKey
				Attributes string
			}{Key: span.Key, Attributes: encodeAttributes(span.Attributes)}),
			orderKey: encode(span),
		}
	}
	return result
}

func eligibleSpanGroups(
	spans []spanDescriptor,
	used []bool,
	parentPairs map[int]int,
	withParentContext bool,
) map[string][]int {
	result := make(map[string][]int)
	for _, descriptor := range spans {
		if used[descriptor.index] {
			continue
		}
		context := ""
		if withParentContext {
			switch descriptor.span.Parent.Kind {
			case model.ParentRoot:
				context = "root"
			case model.ParentExternal:
				context = "external"
			case model.ParentSpan:
				pairID, matched := parentPairs[descriptor.span.Parent.SpanIndex]
				if !matched {
					continue
				}
				context = fmt.Sprintf("parent:%d", pairID)
			default:
				continue
			}
		}
		key := encode(struct {
			Context  string
			Semantic string
		}{Context: context, Semantic: descriptor.semanticKey})
		result[key] = append(result[key], descriptor.index)
	}
	return result
}

func sharedSortedKeys(left, right map[string][]int) []string {
	keys := make([]string, 0)
	for key := range left {
		if len(right[key]) > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func pairSpanGroup(
	baseline, candidate []spanDescriptor,
	baselineIndexes, candidateIndexes []int,
) (spanGroupResult, error) {
	result := spanGroupResult{}
	usedBaseline := make(map[int]bool, len(baselineIndexes))
	usedCandidate := make(map[int]bool, len(candidateIndexes))
	baselineStarts := indexesByStartOrder(baseline, baselineIndexes)
	candidateStarts := indexesByStartOrder(candidate, candidateIndexes)
	startOrders := make([]int, 0, len(baselineStarts))
	for startOrder := range baselineStarts {
		startOrders = append(startOrders, startOrder)
	}
	sort.Ints(startOrders)
	for _, startOrder := range startOrders {
		baselineAtStart := baselineStarts[startOrder]
		candidateAtStart := candidateStarts[startOrder]
		if len(baselineAtStart) != 1 || len(candidateAtStart) != 1 {
			continue
		}
		baselineIndex := baselineAtStart[0]
		candidateIndex := candidateAtStart[0]
		usedBaseline[baselineIndex] = true
		usedCandidate[candidateIndex] = true
		result.pairs = append(result.pairs, [2]int{baselineIndex, candidateIndex})
	}

	for _, index := range baselineIndexes {
		if !usedBaseline[index] {
			result.remainingBaseline = append(result.remainingBaseline, index)
		}
	}
	for _, index := range candidateIndexes {
		if !usedCandidate[index] {
			result.remainingCandidate = append(result.remainingCandidate, index)
		}
	}
	if len(result.remainingBaseline) == 0 || len(result.remainingCandidate) == 0 {
		return result, nil
	}
	if hasDuplicateStartOrder(baseline, result.remainingBaseline) || hasDuplicateStartOrder(candidate, result.remainingCandidate) {
		return spanGroupResult{}, &SpanAmbiguityError{
			BaselineCount:  len(result.remainingBaseline),
			CandidateCount: len(result.remainingCandidate),
		}
	}

	sortSpanIndexes(baseline, result.remainingBaseline)
	sortSpanIndexes(candidate, result.remainingCandidate)
	pairCount := len(result.remainingBaseline)
	if len(result.remainingCandidate) < pairCount {
		pairCount = len(result.remainingCandidate)
	}
	for index := 0; index < pairCount; index++ {
		result.pairs = append(result.pairs, [2]int{
			result.remainingBaseline[index],
			result.remainingCandidate[index],
		})
	}
	result.remainingBaseline = result.remainingBaseline[pairCount:]
	result.remainingCandidate = result.remainingCandidate[pairCount:]
	return result, nil
}

func indexesByStartOrder(spans []spanDescriptor, indexes []int) map[int][]int {
	result := make(map[int][]int)
	for _, index := range indexes {
		startOrder := spans[index].span.StartOrder
		result[startOrder] = append(result[startOrder], index)
	}
	return result
}

func hasDuplicateStartOrder(spans []spanDescriptor, indexes []int) bool {
	seen := make(map[int]struct{}, len(indexes))
	for _, index := range indexes {
		startOrder := spans[index].span.StartOrder
		if _, exists := seen[startOrder]; exists {
			return true
		}
		seen[startOrder] = struct{}{}
	}
	return false
}

func sortSpanIndexes(spans []spanDescriptor, indexes []int) {
	sort.Slice(indexes, func(i, j int) bool {
		left := spans[indexes[i]]
		right := spans[indexes[j]]
		if left.span.StartOrder != right.span.StartOrder {
			return left.span.StartOrder < right.span.StartOrder
		}
		return left.orderKey < right.orderKey
	})
}

func spanSignals(span model.NormalizedSpan, parentAware, occurrence bool) []string {
	signals := []string{signalSpanOperation}
	if span.Key.ServiceName != "" {
		signals = append(signals, signalSpanService)
	}
	if span.Key.Kind != "" {
		signals = append(signals, signalSpanKind)
	}
	if len(span.Attributes) > 0 {
		signals = append(signals, signalSpanAttributes)
	}
	if parentAware {
		if span.Parent.Kind == model.ParentSpan {
			signals = append(signals, signalMatchedParent)
		} else {
			signals = append(signals, signalTopLevelRelation)
		}
	} else {
		signals = append(signals, signalRelationshipChange)
	}
	if occurrence {
		signals = append(signals, signalSiblingOccurrence)
	}
	return signals
}

func sortSpanResult(result *Result) {
	sort.Slice(result.Paired, func(i, j int) bool {
		return encode(result.Paired[i].Baseline) < encode(result.Paired[j].Baseline)
	})
	sort.Slice(result.Added, func(i, j int) bool {
		return encode(result.Added[i]) < encode(result.Added[j])
	})
	sort.Slice(result.Removed, func(i, j int) bool {
		return encode(result.Removed[i]) < encode(result.Removed[j])
	})
}
