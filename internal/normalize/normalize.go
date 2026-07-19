// Package normalize replaces nondeterministic trace data with a canonical,
// trace-preserving representation before matching.
package normalize

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/ArinF1/TraceDelta/internal/model"
)

// Options controls deterministic normalization.
type Options struct {
	// DurationBucket is the width used to floor span durations. Zero preserves
	// exact durations; negative values are invalid.
	DurationBucket time.Duration
}

type canonicalNode struct {
	sourceIndex int
	parentIndex int
	children    []int
	span        model.NormalizedSpan
	subtreeKey  string
}

type canonicalTrace struct {
	trace model.NormalizedTrace
	key   string
}

type spanFingerprint struct {
	Key        model.SpanKey
	ParentKind model.ParentKind
	StartOrder int
	Duration   int64
	Status     model.StatusCode
	Attributes []model.NormalizedAttribute
	Children   []string
}

// Snapshot canonicalizes one parsed snapshot without mutating it. Raw trace,
// span, and parent identifiers plus absolute timestamps and input-array order
// are omitted from the result.
func Snapshot(input model.Snapshot, options Options) (model.NormalizedSnapshot, error) {
	if options.DurationBucket < 0 {
		return model.NormalizedSnapshot{}, fmt.Errorf("duration bucket must be non-negative")
	}

	canonical := make([]canonicalTrace, 0, len(input.Traces))
	for traceIndex, trace := range input.Traces {
		if len(trace.Spans) == 0 {
			continue
		}
		normalized, err := normalizeTrace(trace, options)
		if err != nil {
			return model.NormalizedSnapshot{}, fmt.Errorf("trace[%d]: %w", traceIndex, err)
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return model.NormalizedSnapshot{}, fmt.Errorf("trace[%d]: encode canonical trace key: %w", traceIndex, err)
		}
		canonical = append(canonical, canonicalTrace{trace: normalized, key: string(encoded)})
	}

	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].key < canonical[j].key
	})

	result := model.NormalizedSnapshot{Traces: make([]model.NormalizedTrace, len(canonical))}
	occurrences := make(map[model.SpanKey]int)
	for traceIndex := range canonical {
		result.Traces[traceIndex] = canonical[traceIndex].trace
		for spanIndex := range result.Traces[traceIndex].Spans {
			span := &result.Traces[traceIndex].Spans[spanIndex]
			span.Occurrence = occurrences[span.Key]
			occurrences[span.Key]++
		}
	}

	return result, nil
}

func normalizeTrace(trace model.Trace, options Options) (model.NormalizedTrace, error) {
	spanIndexes := make(map[string]int, len(trace.Spans))
	for index, span := range trace.Spans {
		if span.SpanID == "" {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: span ID is required for relationship normalization", index)
		}
		if _, exists := spanIndexes[span.SpanID]; exists {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: duplicate span ID", index)
		}
		spanIndexes[span.SpanID] = index
	}

	startOrders := denseStartOrders(trace.Spans)
	nodes := make([]canonicalNode, len(trace.Spans))
	for index, span := range trace.Spans {
		duration, err := bucketDuration(span.Duration, options.DurationBucket)
		if err != nil {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: %w", index, err)
		}
		attributes, err := canonicalAttributes(span.Attributes)
		if err != nil {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: attributes: %w", index, err)
		}

		nodes[index] = canonicalNode{
			sourceIndex: index,
			parentIndex: -1,
			children:    make([]int, 0),
			span: model.NormalizedSpan{
				Key: model.SpanKey{
					ServiceName: span.ServiceName,
					Name:        span.Name,
					Kind:        span.Kind,
				},
				Parent:     model.ParentReference{Kind: model.ParentRoot},
				StartOrder: startOrders[index],
				Duration:   duration,
				Status:     span.Status,
				Attributes: attributes,
			},
		}
	}

	for index, span := range trace.Spans {
		if span.ParentSpanID == "" {
			continue
		}
		parentIndex, exists := spanIndexes[span.ParentSpanID]
		if !exists {
			nodes[index].span.Parent.Kind = model.ParentExternal
			continue
		}
		if parentIndex == index {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: span cannot be its own parent", index)
		}
		nodes[index].parentIndex = parentIndex
		nodes[index].span.Parent.Kind = model.ParentSpan
		nodes[parentIndex].children = append(nodes[parentIndex].children, index)
	}

	if err := validateAcyclicParents(nodes); err != nil {
		return model.NormalizedTrace{}, err
	}
	for index := range nodes {
		if _, err := computeSubtreeKey(nodes, index); err != nil {
			return model.NormalizedTrace{}, fmt.Errorf("span[%d]: %w", nodes[index].sourceIndex, err)
		}
	}

	topLevel := make([]int, 0)
	for index := range nodes {
		if nodes[index].parentIndex < 0 {
			topLevel = append(topLevel, index)
		}
	}
	sort.Slice(topLevel, func(i, j int) bool {
		return lessNode(nodes, topLevel[i], topLevel[j])
	})

	order := make([]int, 0, len(nodes))
	var appendPreorder func(int)
	appendPreorder = func(index int) {
		order = append(order, index)
		for _, childIndex := range nodes[index].children {
			appendPreorder(childIndex)
		}
	}
	for _, index := range topLevel {
		appendPreorder(index)
	}

	canonicalIndexes := make(map[int]int, len(order))
	for canonicalIndex, sourceIndex := range order {
		canonicalIndexes[sourceIndex] = canonicalIndex
	}

	result := model.NormalizedTrace{Spans: make([]model.NormalizedSpan, 0, len(order))}
	for _, sourceIndex := range order {
		span := nodes[sourceIndex].span
		if nodes[sourceIndex].parentIndex >= 0 {
			span.Parent.SpanIndex = canonicalIndexes[nodes[sourceIndex].parentIndex]
		}
		result.Spans = append(result.Spans, span)
	}
	return result, nil
}

func denseStartOrders(spans []model.Span) []int {
	starts := make([]uint64, len(spans))
	for index, span := range spans {
		starts[index] = span.StartTime
	}
	sort.Slice(starts, func(i, j int) bool {
		return starts[i] < starts[j]
	})

	ranks := make(map[uint64]int, len(starts))
	rank := -1
	var previous uint64
	for index, start := range starts {
		if index == 0 || start != previous {
			rank++
			previous = start
		}
		ranks[start] = rank
	}

	result := make([]int, len(spans))
	for index, span := range spans {
		result[index] = ranks[span.StartTime]
	}
	return result
}

func validateAcyclicParents(nodes []canonicalNode) error {
	const (
		unvisited = iota
		visiting
		visited
	)
	states := make([]int, len(nodes))
	var visit func(int) error
	visit = func(index int) error {
		switch states[index] {
		case visiting:
			return fmt.Errorf("span[%d]: parent relationships contain a cycle", nodes[index].sourceIndex)
		case visited:
			return nil
		}

		states[index] = visiting
		if nodes[index].parentIndex >= 0 {
			if err := visit(nodes[index].parentIndex); err != nil {
				return err
			}
		}
		states[index] = visited
		return nil
	}

	for index := range nodes {
		if err := visit(index); err != nil {
			return err
		}
	}
	return nil
}

func computeSubtreeKey(nodes []canonicalNode, index int) (string, error) {
	if nodes[index].subtreeKey != "" {
		return nodes[index].subtreeKey, nil
	}
	for _, childIndex := range nodes[index].children {
		if _, err := computeSubtreeKey(nodes, childIndex); err != nil {
			return "", err
		}
	}
	sort.Slice(nodes[index].children, func(i, j int) bool {
		return lessNode(nodes, nodes[index].children[i], nodes[index].children[j])
	})

	childKeys := make([]string, len(nodes[index].children))
	for childIndex, nodeIndex := range nodes[index].children {
		childKeys[childIndex] = nodes[nodeIndex].subtreeKey
	}
	fingerprint := spanFingerprint{
		Key:        nodes[index].span.Key,
		ParentKind: nodes[index].span.Parent.Kind,
		StartOrder: nodes[index].span.StartOrder,
		Duration:   int64(nodes[index].span.Duration),
		Status:     nodes[index].span.Status,
		Attributes: nodes[index].span.Attributes,
		Children:   childKeys,
	}
	encoded, err := json.Marshal(fingerprint)
	if err != nil {
		return "", fmt.Errorf("encode canonical subtree key: %w", err)
	}
	digest := sha256.Sum256(encoded)
	nodes[index].subtreeKey = hex.EncodeToString(digest[:])
	return nodes[index].subtreeKey, nil
}

func lessNode(nodes []canonicalNode, leftIndex, rightIndex int) bool {
	left := nodes[leftIndex]
	right := nodes[rightIndex]
	if left.span.StartOrder != right.span.StartOrder {
		return left.span.StartOrder < right.span.StartOrder
	}
	return left.subtreeKey < right.subtreeKey
}

func bucketDuration(duration, bucket time.Duration) (time.Duration, error) {
	if duration < 0 {
		return 0, fmt.Errorf("duration must be non-negative")
	}
	if bucket == 0 {
		return duration, nil
	}
	return duration - duration%bucket, nil
}

func canonicalAttributes(attributes model.Attributes) ([]model.NormalizedAttribute, error) {
	keys := make([]string, 0)
	for key := range attributes {
		if isSelectedAttribute(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	result := make([]model.NormalizedAttribute, 0, len(keys))
	for _, key := range keys {
		value := attributes[key]
		canonical, err := canonicalAttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		result = append(result, model.NormalizedAttribute{
			Key:   key,
			Type:  value.Type,
			Value: canonical,
		})
	}
	return result, nil
}

func isSelectedAttribute(key string) bool {
	switch key {
	case "http.request.method", "http.route", "rpc.method", "rpc.service":
		return true
	default:
		return false
	}
}

func canonicalAttributeValue(value model.AttributeValue) (string, error) {
	switch value.Type {
	case model.AttributeValueString:
		return value.StringValue, nil
	case model.AttributeValueBool:
		return strconv.FormatBool(value.BoolValue), nil
	case model.AttributeValueInt:
		return strconv.FormatInt(value.IntValue, 10), nil
	case model.AttributeValueDouble:
		switch {
		case math.IsNaN(value.DoubleValue):
			return "NaN", nil
		case math.IsInf(value.DoubleValue, 1):
			return "Infinity", nil
		case math.IsInf(value.DoubleValue, -1):
			return "-Infinity", nil
		case value.DoubleValue == 0:
			return "0", nil
		default:
			return strconv.FormatFloat(value.DoubleValue, 'g', -1, 64), nil
		}
	case model.AttributeValueBytes:
		return base64.StdEncoding.EncodeToString(value.BytesValue), nil
	default:
		return "", fmt.Errorf("unsupported attribute type %q", value.Type)
	}
}
