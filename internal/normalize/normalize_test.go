package normalize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/model"
	"github.com/ArinF1/TraceDelta/internal/otlp"
)

func TestSnapshotCanonicalizesEquivalentFixtures(t *testing.T) {
	runA := parseFixture(t, "normalize-run-a.json")
	runB := parseFixture(t, "normalize-run-b.json")
	options := Options{DurationBucket: 10 * time.Nanosecond}

	normalizedA, err := Snapshot(runA, options)
	if err != nil {
		t.Fatalf("Snapshot(run A) error = %v", err)
	}
	normalizedB, err := Snapshot(runB, options)
	if err != nil {
		t.Fatalf("Snapshot(run B) error = %v", err)
	}
	bytesA := marshalSnapshot(t, normalizedA)
	bytesB := marshalSnapshot(t, normalizedB)
	if !bytes.Equal(bytesA, bytesB) {
		t.Fatalf("normalized fixtures differ:\nrun A: %s\nrun B: %s", bytesA, bytesB)
	}

	repeated, err := Snapshot(runA, options)
	if err != nil {
		t.Fatalf("repeated Snapshot(run A) error = %v", err)
	}
	if repeatedBytes := marshalSnapshot(t, repeated); !bytes.Equal(bytesA, repeatedBytes) {
		t.Fatalf("repeated normalization differs:\nfirst:  %s\nsecond: %s", bytesA, repeatedBytes)
	}

	checkout := findTraceByRootName(t, normalizedA, "checkout.handle")
	if len(checkout.Spans) != 2 {
		t.Fatalf("checkout trace span count = %d, want 2", len(checkout.Spans))
	}
	root, child := checkout.Spans[0], checkout.Spans[1]
	if root.Parent.Kind != model.ParentRoot || root.StartOrder != 0 || root.Duration != 100*time.Nanosecond {
		t.Fatalf("normalized root = %#v, want root at order 0 with 100ns duration", root)
	}
	if child.Parent.Kind != model.ParentSpan || child.Parent.SpanIndex != 0 || child.StartOrder != 1 || child.Duration != 60*time.Nanosecond {
		t.Fatalf("normalized child = %#v, want parent index 0 at order 1 with 60ns duration", child)
	}
	wantRootAttributes := []model.NormalizedAttribute{
		{Key: "http.request.method", Type: model.AttributeValueString, Value: "POST"},
		{Key: "http.route", Type: model.AttributeValueString, Value: "/checkout"},
	}
	if !reflect.DeepEqual(root.Attributes, wantRootAttributes) {
		t.Fatalf("normalized root attributes = %#v, want %#v", root.Attributes, wantRootAttributes)
	}
	wantChildAttributes := []model.NormalizedAttribute{
		{Key: "rpc.method", Type: model.AttributeValueString, Value: "Charge"},
		{Key: "rpc.service", Type: model.AttributeValueString, Value: "payments.v1.PaymentService"},
	}
	if !reflect.DeepEqual(child.Attributes, wantChildAttributes) {
		t.Fatalf("normalized child attributes = %#v, want %#v", child.Attributes, wantChildAttributes)
	}
}

func TestSnapshotPreservesInternalAndExternalRelationships(t *testing.T) {
	input := model.Snapshot{Traces: []model.Trace{{
		ID: "trace",
		Spans: []model.Span{
			{SpanID: "grandchild", ParentSpanID: "child", Name: "grandchild", StartTime: 30},
			{SpanID: "external", ParentSpanID: "missing", Name: "external", StartTime: 40},
			{SpanID: "child", ParentSpanID: "root", Name: "child", StartTime: 20},
			{SpanID: "root", Name: "root", StartTime: 10},
		},
	}}}

	got, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	spans := got.Traces[0].Spans
	if names := []string{spans[0].Key.Name, spans[1].Key.Name, spans[2].Key.Name, spans[3].Key.Name}; !reflect.DeepEqual(names, []string{"root", "child", "grandchild", "external"}) {
		t.Fatalf("canonical span order = %v, want parent preorder then external", names)
	}
	wantParents := []model.ParentReference{
		{Kind: model.ParentRoot},
		{Kind: model.ParentSpan, SpanIndex: 0},
		{Kind: model.ParentSpan, SpanIndex: 1},
		{Kind: model.ParentExternal},
	}
	for index, want := range wantParents {
		if spans[index].Parent != want {
			t.Errorf("span[%d] parent = %#v, want %#v", index, spans[index].Parent, want)
		}
	}

	reparented := input
	reparented.Traces = append([]model.Trace(nil), input.Traces...)
	reparented.Traces[0].Spans = append([]model.Span(nil), input.Traces[0].Spans...)
	reparented.Traces[0].Spans[0].ParentSpanID = "root"
	reparentedNormalized, err := Snapshot(reparented, Options{})
	if err != nil {
		t.Fatalf("Snapshot(reparented) error = %v", err)
	}
	if bytes.Equal(marshalSnapshot(t, got), marshalSnapshot(t, reparentedNormalized)) {
		t.Fatal("reparenting did not change the normalized snapshot")
	}

	externalRunA := model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{{
		SpanID: "span-a", ParentSpanID: "missing-a", Name: "external", StartTime: 10,
	}}}}}
	externalRunB := model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{{
		SpanID: "span-b", ParentSpanID: "missing-b", Name: "external", StartTime: 100,
	}}}}}
	normalizedExternalA, err := Snapshot(externalRunA, Options{})
	if err != nil {
		t.Fatalf("Snapshot(external run A) error = %v", err)
	}
	normalizedExternalB, err := Snapshot(externalRunB, Options{})
	if err != nil {
		t.Fatalf("Snapshot(external run B) error = %v", err)
	}
	if !bytes.Equal(marshalSnapshot(t, normalizedExternalA), marshalSnapshot(t, normalizedExternalB)) {
		t.Fatalf("regenerated external parent IDs normalized differently: %s != %s", marshalSnapshot(t, normalizedExternalA), marshalSnapshot(t, normalizedExternalB))
	}
	if got := normalizedExternalA.Traces[0].Spans[0].Parent.Kind; got != model.ParentExternal {
		t.Fatalf("external parent kind = %q, want %q", got, model.ParentExternal)
	}
}

func TestSnapshotCanonicalizesEqualTimestampSiblingOrder(t *testing.T) {
	runA := model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{
		{SpanID: "root-a", Name: "root", StartTime: 10},
		{SpanID: "zeta-a", ParentSpanID: "root-a", Name: "zeta", StartTime: 20},
		{SpanID: "alpha-a", ParentSpanID: "root-a", Name: "alpha", StartTime: 20},
	}}}}
	runB := model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{
		{SpanID: "alpha-b", ParentSpanID: "root-b", Name: "alpha", StartTime: 200},
		{SpanID: "root-b", Name: "root", StartTime: 100},
		{SpanID: "zeta-b", ParentSpanID: "root-b", Name: "zeta", StartTime: 200},
	}}}}

	normalizedA, err := Snapshot(runA, Options{})
	if err != nil {
		t.Fatalf("Snapshot(run A) error = %v", err)
	}
	normalizedB, err := Snapshot(runB, Options{})
	if err != nil {
		t.Fatalf("Snapshot(run B) error = %v", err)
	}
	if !bytes.Equal(marshalSnapshot(t, normalizedA), marshalSnapshot(t, normalizedB)) {
		t.Fatalf("equal-timestamp siblings normalized differently:\nrun A: %s\nrun B: %s", marshalSnapshot(t, normalizedA), marshalSnapshot(t, normalizedB))
	}
	spans := normalizedA.Traces[0].Spans
	if names := []string{spans[0].Key.Name, spans[1].Key.Name, spans[2].Key.Name}; !reflect.DeepEqual(names, []string{"root", "alpha", "zeta"}) {
		t.Fatalf("canonical sibling order = %v, want root, alpha, zeta", names)
	}
}

func TestSnapshotRejectsInvalidParentGraphs(t *testing.T) {
	tests := []struct {
		name    string
		spans   []model.Span
		wantErr string
	}{
		{
			name:    "self parent",
			spans:   []model.Span{{SpanID: "a", ParentSpanID: "a", Name: "a"}},
			wantErr: "span cannot be its own parent",
		},
		{
			name: "cycle",
			spans: []model.Span{
				{SpanID: "a", ParentSpanID: "b", Name: "a"},
				{SpanID: "b", ParentSpanID: "a", Name: "b"},
			},
			wantErr: "parent relationships contain a cycle",
		},
		{
			name: "duplicate ID",
			spans: []model.Span{
				{SpanID: "a", Name: "a"},
				{SpanID: "a", Name: "b"},
			},
			wantErr: "duplicate span ID",
		},
		{
			name:    "missing ID",
			spans:   []model.Span{{Name: "a"}},
			wantErr: "span ID is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Snapshot(model.Snapshot{Traces: []model.Trace{{Spans: test.spans}}}, Options{})
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Snapshot() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestDurationBucketingBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		bucket   time.Duration
		want     time.Duration
		wantErr  bool
	}{
		{name: "exact disabled", duration: 17, bucket: 0, want: 17},
		{name: "below bucket", duration: 9, bucket: 10, want: 0},
		{name: "exact boundary", duration: 10, bucket: 10, want: 10},
		{name: "above boundary", duration: 11, bucket: 10, want: 10},
		{name: "maximum duration", duration: time.Duration(math.MaxInt64), bucket: 10, want: time.Duration(math.MaxInt64 - 7)},
		{name: "negative duration", duration: -1, bucket: 10, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := bucketDuration(test.duration, test.bucket)
			if (err != nil) != test.wantErr {
				t.Fatalf("bucketDuration() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("bucketDuration() = %s, want %s", got, test.want)
			}
		})
	}

	if _, err := Snapshot(model.Snapshot{}, Options{DurationBucket: -time.Nanosecond}); err == nil || !strings.Contains(err.Error(), "duration bucket must be non-negative") {
		t.Fatalf("Snapshot() negative bucket error = %v", err)
	}
}

func TestCanonicalAttributesAreSelectedSortedAndTyped(t *testing.T) {
	attributes := model.Attributes{
		"rpc.service":         {Type: model.AttributeValueInt, IntValue: -7},
		"request.id":          {Type: model.AttributeValueString, StringValue: "synthetic-run-id"},
		"http.route":          {Type: model.AttributeValueString, StringValue: "/checkout/{id}"},
		"rpc.method":          {Type: model.AttributeValueBytes, BytesValue: []byte("Charge")},
		"http.request.method": {Type: model.AttributeValueBool, BoolValue: true},
	}

	got, err := canonicalAttributes(attributes)
	if err != nil {
		t.Fatalf("canonicalAttributes() error = %v", err)
	}
	want := []model.NormalizedAttribute{
		{Key: "http.request.method", Type: model.AttributeValueBool, Value: "true"},
		{Key: "http.route", Type: model.AttributeValueString, Value: "/checkout/{id}"},
		{Key: "rpc.method", Type: model.AttributeValueBytes, Value: "Q2hhcmdl"},
		{Key: "rpc.service", Type: model.AttributeValueInt, Value: "-7"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonicalAttributes() = %#v, want %#v", got, want)
	}
}

func TestCanonicalAttributesIgnoreCompositeValues(t *testing.T) {
	attributes := model.Attributes{
		"http.route": {
			Type: model.AttributeValueArray,
			ArrayValue: []model.AttributeValue{
				{Type: model.AttributeValueString, StringValue: "/checkout"},
			},
		},
		"request.id": {Type: model.AttributeValueString, StringValue: "synthetic-run-id"},
	}

	got, err := canonicalAttributes(attributes)
	if err != nil {
		t.Fatalf("canonicalAttributes() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("canonicalAttributes() = %#v, want composite and unsafe values omitted", got)
	}
}

func TestSnapshotExtractsStringErrorTypeOutsideMatchAttributes(t *testing.T) {
	input := model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{{
		SpanID: "span",
		Name:   "operation",
		Status: model.StatusError,
		Attributes: model.Attributes{
			"error.type":        {Type: model.AttributeValueString, StringValue: "synthetic.Declined"},
			"exception.message": {Type: model.AttributeValueString, StringValue: "synthetic private detail"},
		},
	}}}}}

	got, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	span := got.Traces[0].Spans[0]
	if !span.ErrorTypePresent || span.ErrorType != "synthetic.Declined" {
		t.Fatalf("normalized error evidence = %q/%t, want synthetic.Declined/present", span.ErrorType, span.ErrorTypePresent)
	}
	if len(span.Attributes) != 0 {
		t.Fatalf("normalized match attributes = %#v, want error evidence excluded", span.Attributes)
	}
	encoded := string(marshalSnapshot(t, got))
	if strings.Contains(encoded, "synthetic private detail") {
		t.Fatalf("normalized snapshot contains exception message: %s", encoded)
	}

	input.Traces[0].Spans[0].Attributes["error.type"] = model.AttributeValue{Type: model.AttributeValueInt, IntValue: 7}
	got, err = Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("Snapshot(non-string error.type) error = %v", err)
	}
	if got.Traces[0].Spans[0].ErrorTypePresent {
		t.Fatalf("non-string error.type became evidence: %#v", got.Traces[0].Spans[0])
	}
}

func TestCanonicalDoubleValues(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "finite", value: 1.25, want: "1.25"},
		{name: "positive zero", value: 0, want: "0"},
		{name: "negative zero", value: math.Copysign(0, -1), want: "0"},
		{name: "not a number", value: math.NaN(), want: "NaN"},
		{name: "positive infinity", value: math.Inf(1), want: "Infinity"},
		{name: "negative infinity", value: math.Inf(-1), want: "-Infinity"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := canonicalAttributeValue(model.AttributeValue{Type: model.AttributeValueDouble, DoubleValue: test.value})
			if err != nil {
				t.Fatalf("canonicalAttributeValue() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("canonicalAttributeValue() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSnapshotDoesNotMutateInputAndCanonicalizesEmptyCollections(t *testing.T) {
	bytesValue := []byte("synthetic")
	input := model.Snapshot{Traces: []model.Trace{{
		ID: "trace",
		Spans: []model.Span{{
			SpanID:     "span-b",
			Name:       "operation-b",
			StartTime:  20,
			InputOrder: 2,
			Attributes: model.Attributes{
				"rpc.method": {Type: model.AttributeValueBytes, BytesValue: bytesValue},
			},
		}, {
			SpanID:     "span-a",
			Name:       "operation-a",
			StartTime:  10,
			InputOrder: 1,
		}},
	}}}
	wantInput := model.Snapshot{Traces: []model.Trace{{
		ID: "trace",
		Spans: []model.Span{{
			SpanID:     "span-b",
			Name:       "operation-b",
			StartTime:  20,
			InputOrder: 2,
			Attributes: model.Attributes{
				"rpc.method": {Type: model.AttributeValueBytes, BytesValue: []byte("synthetic")},
			},
		}, {
			SpanID:     "span-a",
			Name:       "operation-a",
			StartTime:  10,
			InputOrder: 1,
		}},
	}}}

	normalized, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if !reflect.DeepEqual(input, wantInput) {
		t.Fatalf("Snapshot() mutated input:\ngot:  %#v\nwant: %#v", input, wantInput)
	}
	normalized.Traces[0].Spans[0].Key.Name = "changed"
	if input.Traces[0].Spans[1].Name != "operation-a" || string(bytesValue) != "synthetic" {
		t.Fatal("normalized output aliases parsed input")
	}

	nilSnapshot, err := Snapshot(model.Snapshot{}, Options{})
	if err != nil {
		t.Fatalf("Snapshot(nil collections) error = %v", err)
	}
	emptySnapshot, err := Snapshot(model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{}}}}, Options{})
	if err != nil {
		t.Fatalf("Snapshot(empty collections) error = %v", err)
	}
	if !bytes.Equal(marshalSnapshot(t, nilSnapshot), marshalSnapshot(t, emptySnapshot)) {
		t.Fatalf("nil and empty collections normalized differently: %s != %s", marshalSnapshot(t, nilSnapshot), marshalSnapshot(t, emptySnapshot))
	}

	withAttributes := func(attributes model.Attributes) model.Snapshot {
		return model.Snapshot{Traces: []model.Trace{{Spans: []model.Span{{
			SpanID:     "span",
			Name:       "operation",
			Attributes: attributes,
		}}}}}
	}
	nilAttributes, err := Snapshot(withAttributes(nil), Options{})
	if err != nil {
		t.Fatalf("Snapshot(nil attributes) error = %v", err)
	}
	emptyAttributes, err := Snapshot(withAttributes(model.Attributes{}), Options{})
	if err != nil {
		t.Fatalf("Snapshot(empty attributes) error = %v", err)
	}
	if !bytes.Equal(marshalSnapshot(t, nilAttributes), marshalSnapshot(t, emptyAttributes)) {
		t.Fatalf("nil and empty attributes normalized differently: %s != %s", marshalSnapshot(t, nilAttributes), marshalSnapshot(t, emptyAttributes))
	}
}

func TestSnapshotAssignsGlobalOccurrencesAfterCanonicalOrdering(t *testing.T) {
	span := func(id string, duration time.Duration) model.Span {
		return model.Span{SpanID: id, Name: "repeat", ServiceName: "service", Kind: "SPAN_KIND_INTERNAL", Duration: duration}
	}
	runA := model.Snapshot{Traces: []model.Trace{
		{ID: "trace-b", Spans: []model.Span{span("b", 20)}},
		{ID: "trace-a", Spans: []model.Span{span("a", 10)}},
	}}
	runB := model.Snapshot{Traces: []model.Trace{
		{ID: "trace-c", Spans: []model.Span{span("c", 10)}},
		{ID: "trace-d", Spans: []model.Span{span("d", 20)}},
	}}
	got, err := Snapshot(runA, Options{})
	if err != nil {
		t.Fatalf("Snapshot(run A) error = %v", err)
	}
	want, err := Snapshot(runB, Options{})
	if err != nil {
		t.Fatalf("Snapshot(run B) error = %v", err)
	}
	if !bytes.Equal(marshalSnapshot(t, got), marshalSnapshot(t, want)) {
		t.Fatalf("reordered traces normalized differently:\nrun A: %s\nrun B: %s", marshalSnapshot(t, got), marshalSnapshot(t, want))
	}
	if got.Traces[0].Spans[0].Occurrence != 0 || got.Traces[1].Spans[0].Occurrence != 1 {
		t.Fatalf("occurrences = %d, %d; want 0, 1", got.Traces[0].Spans[0].Occurrence, got.Traces[1].Spans[0].Occurrence)
	}
	if got.Traces[0].Spans[0].Duration != 10 || got.Traces[1].Spans[0].Duration != 20 {
		t.Fatalf("canonical trace durations = %s, %s; want 10ns, 20ns", got.Traces[0].Spans[0].Duration, got.Traces[1].Spans[0].Duration)
	}
}

func TestSnapshotDeepChainHasLinearSizeCanonicalOutput(t *testing.T) {
	const spanCount = 1024
	spans := make([]model.Span, spanCount)
	for index := range spans {
		spanID := fmt.Sprintf("span-%04d", index)
		parentID := ""
		if index > 0 {
			parentID = spans[index-1].SpanID
		}
		spans[index] = model.Span{
			SpanID:       spanID,
			ParentSpanID: parentID,
			Name:         "operation",
			StartTime:    uint64(index),
		}
	}

	normalized, err := Snapshot(model.Snapshot{Traces: []model.Trace{{Spans: spans}}}, Options{})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got := len(normalized.Traces[0].Spans); got != spanCount {
		t.Fatalf("normalized span count = %d, want %d", got, spanCount)
	}
	if encodedSize := len(marshalSnapshot(t, normalized)); encodedSize > 1_000_000 {
		t.Fatalf("normalized JSON size = %d bytes, want a linear-size result below 1 MB", encodedSize)
	}
}

func parseFixture(t *testing.T, name string) model.Snapshot {
	t.Helper()
	file, err := os.Open(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer file.Close()
	snapshot, err := otlp.Parse(file)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return snapshot
}

func marshalSnapshot(t *testing.T, snapshot model.NormalizedSnapshot) []byte {
	t.Helper()
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func findTraceByRootName(t *testing.T, snapshot model.NormalizedSnapshot, name string) model.NormalizedTrace {
	t.Helper()
	for _, trace := range snapshot.Traces {
		if len(trace.Spans) > 0 && trace.Spans[0].Key.Name == name {
			return trace
		}
	}
	t.Fatalf("normalized trace with root %q not found", name)
	return model.NormalizedTrace{}
}
