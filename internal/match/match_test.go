package match

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/model"
)

func TestTracesPairsUnambiguousTraceWithEvidence(t *testing.T) {
	baseline := model.NormalizedSnapshot{Traces: []model.NormalizedTrace{
		trace("checkout.handle", "api", "SPAN_KIND_SERVER", "/checkout", "payment.charge"),
	}}
	candidate := baseline
	candidate.Traces = cloneTraces(baseline.Traces)
	candidate.Traces[0].Spans[0].Status = model.StatusError
	candidate.Traces[0].Spans[0].Duration = 250 * time.Millisecond

	got, err := Traces(baseline, candidate)
	if err != nil {
		t.Fatalf("Traces() error = %v", err)
	}
	if len(got.Paired) != 1 || len(got.Added) != 0 || len(got.Removed) != 0 {
		t.Fatalf("Traces() = %#v, want one pair", got)
	}
	wantSignals := []string{
		signalRootOperation,
		signalRootService,
		signalRootKind,
		signalStableAttributes,
		signalExactStructure,
	}
	if !reflect.DeepEqual(got.Paired[0].Evidence.Signals, wantSignals) {
		t.Fatalf("match signals = %v, want %v", got.Paired[0].Evidence.Signals, wantSignals)
	}
	if got.Paired[0].Evidence.StructuralOverlap != 2 {
		t.Fatalf("structural overlap = %d, want 2", got.Paired[0].Evidence.StructuralOverlap)
	}
}

func TestTracesReportsAddedAndRemovedTraceGroups(t *testing.T) {
	baselineTrace := trace("checkout.handle", "api", "SPAN_KIND_SERVER", "/checkout", "payment.charge")
	candidateTrace := trace("catalog.list", "api", "SPAN_KIND_SERVER", "/catalog", "cache.get")

	got, err := Traces(
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{baselineTrace}},
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{candidateTrace}},
	)
	if err != nil {
		t.Fatalf("Traces() error = %v", err)
	}
	if len(got.Paired) != 0 || len(got.Added) != 1 || len(got.Removed) != 1 {
		t.Fatalf("Traces() = %#v, want one added and one removed trace", got)
	}

	spans, err := Spans(got)
	if err != nil {
		t.Fatalf("Spans() error = %v", err)
	}
	if len(spans.Added) != 2 || len(spans.Removed) != 2 || len(spans.Paired) != 0 {
		t.Fatalf("Spans() = %#v, want every unmatched trace span added or removed", spans)
	}
}

func TestTracesPairsRepeatedOperationsByUniqueStructureIndependentOfOrder(t *testing.T) {
	database := trace("checkout.handle", "api", "SPAN_KIND_SERVER", "/checkout", "database.write")
	cache := trace("checkout.handle", "api", "SPAN_KIND_SERVER", "/checkout", "cache.get")
	databaseWithAudit := trace("checkout.handle", "api", "SPAN_KIND_SERVER", "/checkout", "audit.write", "database.write")

	baseline := model.NormalizedSnapshot{Traces: []model.NormalizedTrace{database, cache}}
	candidate := model.NormalizedSnapshot{Traces: []model.NormalizedTrace{cache, databaseWithAudit}}
	got, err := Traces(baseline, candidate)
	if err != nil {
		t.Fatalf("Traces() error = %v", err)
	}
	if len(got.Paired) != 2 || len(got.Added) != 0 || len(got.Removed) != 0 {
		t.Fatalf("Traces() = %#v, want two repeated-operation pairs", got)
	}

	pairs := make(map[string]string)
	for _, pair := range got.Paired {
		pairs[childNames(pair.Baseline)] = childNames(pair.Candidate)
	}
	if pairs["cache.get"] != "cache.get" || pairs["database.write"] != "audit.write,database.write" {
		t.Fatalf("paired child shapes = %v, want cache-to-cache and database-to-database-plus-audit", pairs)
	}

	reordered, err := Traces(
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{cache, database}},
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{databaseWithAudit, cache}},
	)
	if err != nil {
		t.Fatalf("Traces(reordered) error = %v", err)
	}
	if !reflect.DeepEqual(got, reordered) {
		gotJSON, _ := json.Marshal(got)
		reorderedJSON, _ := json.Marshal(reordered)
		t.Fatalf("reordered trace match changed result:\nfirst: %s\nsecond: %s", gotJSON, reorderedJSON)
	}
}

func TestTracesReturnsNonSensitiveAmbiguityError(t *testing.T) {
	privateName := "customer-secret-operation"
	baselineTrace := trace(privateName, "private-service", "SPAN_KIND_SERVER", "/private", "database.read")
	candidateTraceA := cloneTrace(baselineTrace)
	candidateTraceB := cloneTrace(baselineTrace)
	candidateTraceA.Spans[0].Duration = 10 * time.Millisecond
	candidateTraceB.Spans[0].Duration = 20 * time.Millisecond

	_, err := Traces(
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{cloneTrace(baselineTrace), cloneTrace(baselineTrace)}},
		model.NormalizedSnapshot{Traces: []model.NormalizedTrace{candidateTraceB, candidateTraceA}},
	)
	if err == nil {
		t.Fatal("Traces() error = nil, want ambiguity")
	}
	var ambiguity *AmbiguityError
	if !errors.As(err, &ambiguity) {
		t.Fatalf("Traces() error = %T %v, want *AmbiguityError", err, err)
	}
	if ambiguity.BaselineCount != 2 || ambiguity.CandidateCount != 2 {
		t.Fatalf("ambiguity counts = %d/%d, want 2/2", ambiguity.BaselineCount, ambiguity.CandidateCount)
	}
	if strings.Contains(err.Error(), privateName) || strings.Contains(err.Error(), "/private") {
		t.Fatalf("ambiguity error leaked trace-derived values: %v", err)
	}
	want := "ambiguous trace match: 2 baseline and 2 candidate traces remain indistinguishable after " + ambiguitySignalSummary
	if err.Error() != want {
		t.Fatalf("ambiguity error = %q, want %q", err, want)
	}
}

func TestSpansPairsDuplicateSiblingsByTraceLocalOrder(t *testing.T) {
	key := model.SpanKey{ServiceName: "api", Name: "db.query", Kind: "SPAN_KIND_CLIENT"}
	traceMatches := TraceResult{Paired: []TracePair{{
		Baseline: model.NormalizedTrace{Spans: []model.NormalizedSpan{
			{Key: key, Occurrence: 7, Parent: model.ParentReference{Kind: model.ParentRoot}, StartOrder: 1},
			{Key: key, Occurrence: 9, Parent: model.ParentReference{Kind: model.ParentRoot}, StartOrder: 3},
		}},
		Candidate: model.NormalizedTrace{Spans: []model.NormalizedSpan{
			{Key: key, Occurrence: 1, Parent: model.ParentReference{Kind: model.ParentRoot}, StartOrder: 2},
			{Key: key, Occurrence: 2, Parent: model.ParentReference{Kind: model.ParentRoot}, StartOrder: 4},
			{Key: key, Occurrence: 3, Parent: model.ParentReference{Kind: model.ParentRoot}, StartOrder: 5},
		}},
	}}}

	got, err := Spans(traceMatches)
	if err != nil {
		t.Fatalf("Spans() error = %v", err)
	}
	if len(got.Paired) != 2 || len(got.Added) != 1 || len(got.Removed) != 0 {
		t.Fatalf("Spans() = %#v, want 2 paired, 1 added, 0 removed", got)
	}
	if got.Added[0].Occurrence != 3 {
		t.Fatalf("added occurrence = %d, want original normalized occurrence 3", got.Added[0].Occurrence)
	}
	for _, pair := range got.Paired {
		if !contains(pair.Evidence.Signals, signalSiblingOccurrence) {
			t.Fatalf("pair evidence = %v, want sibling occurrence", pair.Evidence.Signals)
		}
	}
}

func TestSpansPairsRepeatedNamesUnderMatchedParentsWithoutReuse(t *testing.T) {
	root := span("request", "SERVER", model.ParentReference{Kind: model.ParentRoot}, 0)
	branchA := span("branch.a", "INTERNAL", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 1)
	branchB := span("branch.b", "INTERNAL", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 2)
	childA := span("repeat", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 1}, 3)
	childB := span("repeat", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 3}, 4)
	childB.Duration = 10 * time.Millisecond

	candidateChildB := childB
	candidateChildB.Parent.SpanIndex = 1
	candidateChildB.Status = model.StatusError
	candidateChildA := childA
	candidateChildA.Parent.SpanIndex = 3
	candidate := model.NormalizedTrace{Spans: []model.NormalizedSpan{
		root,
		branchB,
		candidateChildB,
		branchA,
		candidateChildA,
	}}
	baseline := model.NormalizedTrace{Spans: []model.NormalizedSpan{root, branchA, childA, branchB, childB}}

	got, err := Spans(TraceResult{Paired: []TracePair{{Baseline: baseline, Candidate: candidate}}})
	if err != nil {
		t.Fatalf("Spans() error = %v", err)
	}
	if len(got.Paired) != 5 || len(got.Added) != 0 || len(got.Removed) != 0 {
		t.Fatalf("Spans() = %#v, want five one-to-one pairs", got)
	}
	foundBranchBChild := false
	for _, pair := range got.Paired {
		if pair.Baseline.Key.Name != "repeat" || pair.Baseline.Parent.SpanIndex != 3 {
			continue
		}
		foundBranchBChild = true
		if pair.Candidate.Parent.SpanIndex != 1 || pair.Candidate.Status != model.StatusError {
			t.Fatalf("branch B child pair = %#v, want candidate child under reordered branch B", pair)
		}
		if !contains(pair.Evidence.Signals, signalMatchedParent) {
			t.Fatalf("branch B child evidence = %v, want matched parent", pair.Evidence.Signals)
		}
	}
	if !foundBranchBChild {
		t.Fatal("branch B repeated child was not paired")
	}
}

func TestSpansPairsUniqueRelationshipChangeAsFallback(t *testing.T) {
	root := span("request", "SERVER", model.ParentReference{Kind: model.ParentRoot}, 0)
	branch := span("branch", "INTERNAL", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 1)
	baselineChild := span("work", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 1}, 2)
	candidateChild := baselineChild
	candidateChild.Parent = model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}

	got, err := Spans(TraceResult{Paired: []TracePair{{
		Baseline:  model.NormalizedTrace{Spans: []model.NormalizedSpan{root, branch, baselineChild}},
		Candidate: model.NormalizedTrace{Spans: []model.NormalizedSpan{root, branch, candidateChild}},
	}}})
	if err != nil {
		t.Fatalf("Spans() error = %v", err)
	}
	if len(got.Paired) != 3 || len(got.Added) != 0 || len(got.Removed) != 0 {
		t.Fatalf("Spans() = %#v, want relationship change paired without additions/removals", got)
	}
	foundFallback := false
	for _, pair := range got.Paired {
		if pair.Baseline.Key.Name == "work" {
			foundFallback = contains(pair.Evidence.Signals, signalRelationshipChange)
		}
	}
	if !foundFallback {
		t.Fatal("relationship-changed span did not record fallback evidence")
	}
}

func TestSpansReturnsNonSensitiveAmbiguityForIndistinguishableSiblings(t *testing.T) {
	root := span("private-root", "SERVER", model.ParentReference{Kind: model.ParentRoot}, 0)
	privateChild := span("private-child", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 1)
	baseline := model.NormalizedTrace{Spans: []model.NormalizedSpan{root, privateChild, privateChild}}
	candidate := cloneTrace(baseline)

	_, err := Spans(TraceResult{Paired: []TracePair{{Baseline: baseline, Candidate: candidate}}})
	if err == nil {
		t.Fatal("Spans() error = nil, want ambiguity")
	}
	var ambiguity *SpanAmbiguityError
	if !errors.As(err, &ambiguity) {
		t.Fatalf("Spans() error = %T %v, want *SpanAmbiguityError", err, err)
	}
	if ambiguity.BaselineCount != 2 || ambiguity.CandidateCount != 2 {
		t.Fatalf("ambiguity counts = %d/%d, want 2/2", ambiguity.BaselineCount, ambiguity.CandidateCount)
	}
	if strings.Contains(err.Error(), "private-root") || strings.Contains(err.Error(), "private-child") {
		t.Fatalf("span ambiguity leaked trace-derived names: %v", err)
	}
}

func TestSpansPreservesAddedAndRemovedInsideMatchedTrace(t *testing.T) {
	root := span("request", "SERVER", model.ParentReference{Kind: model.ParentRoot}, 0)
	removed := span("cache.get", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 1)
	added := span("inventory.reserve", "CLIENT", model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0}, 1)

	got, err := Spans(TraceResult{Paired: []TracePair{{
		Baseline:  model.NormalizedTrace{Spans: []model.NormalizedSpan{root, removed}},
		Candidate: model.NormalizedTrace{Spans: []model.NormalizedSpan{root, added}},
	}}})
	if err != nil {
		t.Fatalf("Spans() error = %v", err)
	}
	if len(got.Paired) != 1 || len(got.Added) != 1 || len(got.Removed) != 1 {
		t.Fatalf("Spans() = %#v, want root paired plus one added and removed", got)
	}
	if got.Added[0].Key.Name != "inventory.reserve" || got.Removed[0].Key.Name != "cache.get" {
		t.Fatalf("added/removed = %q/%q, want inventory.reserve/cache.get", got.Added[0].Key.Name, got.Removed[0].Key.Name)
	}
}

func trace(rootName, service, kind, route string, children ...string) model.NormalizedTrace {
	attributes := []model.NormalizedAttribute{}
	if route != "" {
		attributes = append(attributes, model.NormalizedAttribute{
			Key:   "http.route",
			Type:  model.AttributeValueString,
			Value: route,
		})
	}
	spans := []model.NormalizedSpan{{
		Key: model.SpanKey{
			ServiceName: service,
			Name:        rootName,
			Kind:        kind,
		},
		Parent:     model.ParentReference{Kind: model.ParentRoot},
		Attributes: attributes,
		Status:     model.StatusOK,
	}}
	for _, name := range children {
		spans = append(spans, model.NormalizedSpan{
			Key: model.SpanKey{
				ServiceName: service,
				Name:        name,
				Kind:        "SPAN_KIND_INTERNAL",
			},
			Parent: model.ParentReference{Kind: model.ParentSpan, SpanIndex: 0},
			Status: model.StatusOK,
		})
	}
	return model.NormalizedTrace{Spans: spans}
}

func childNames(trace model.NormalizedTrace) string {
	names := make([]string, 0, len(trace.Spans)-1)
	for _, span := range trace.Spans[1:] {
		names = append(names, span.Key.Name)
	}
	return strings.Join(names, ",")
}

func cloneTraces(traces []model.NormalizedTrace) []model.NormalizedTrace {
	result := make([]model.NormalizedTrace, len(traces))
	for index, trace := range traces {
		result[index] = cloneTrace(trace)
	}
	return result
}

func cloneTrace(trace model.NormalizedTrace) model.NormalizedTrace {
	result := model.NormalizedTrace{Spans: make([]model.NormalizedSpan, len(trace.Spans))}
	copy(result.Spans, trace.Spans)
	for index := range result.Spans {
		result.Spans[index].Attributes = append([]model.NormalizedAttribute(nil), trace.Spans[index].Attributes...)
	}
	return result
}

func span(name, kind string, parent model.ParentReference, startOrder int) model.NormalizedSpan {
	return model.NormalizedSpan{
		Key: model.SpanKey{
			ServiceName: "api",
			Name:        name,
			Kind:        kind,
		},
		Parent:     parent,
		StartOrder: startOrder,
		Status:     model.StatusOK,
		Attributes: []model.NormalizedAttribute{},
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
