package tracedelta

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

func TestCompareFilesExample(t *testing.T) {
	baseline := filepath.Join("..", "..", "testdata", "baseline.json")
	candidate := filepath.Join("..", "..", "testdata", "candidate.json")
	comparison, err := CompareFiles(baseline, candidate, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareFiles() error = %v", err)
	}
	if comparison.AddedSpans != 1 || comparison.RemovedSpans != 1 || comparison.ChangedSpans != 2 {
		t.Fatalf("CompareFiles() summary = %#v, want 1 added, 1 removed, 2 changed", comparison)
	}
	if len(comparison.Changes) != 4 {
		t.Fatalf("len(changes) = %d, want 4", len(comparison.Changes))
	}
}

func TestDefaultOptionsUseV01LatencyThresholds(t *testing.T) {
	options := DefaultOptions()
	if options.DurationThreshold != 0.20 || options.DurationThresholdAbsolute != 10*time.Millisecond {
		t.Fatalf("DefaultOptions() = %#v, want 20%% and 10ms latency thresholds", options)
	}
}

func TestCompareFilesRepresentativeOTLPAgainstItself(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "otlp-representative.json")
	comparison, err := CompareFiles(fixture, fixture, DefaultOptions())
	if err != nil {
		t.Fatalf("CompareFiles() error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("CompareFiles() comparison = %#v, want no differences", comparison)
	}
}

func TestCompareOTLPFileJSONLRecordOrderIsIrrelevant(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "otlp-file.jsonl")
	contents, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(contents), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("fixture record count = %d, want 2", len(lines))
	}
	reversed := append(append([]byte{}, lines[1]...), '\n')
	reversed = append(reversed, lines[0]...)

	comparison, err := Compare(bytes.NewReader(contents), bytes.NewReader(reversed), DefaultOptions())
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("Compare() comparison = %#v, want record order to be irrelevant", comparison)
	}
}

func TestCompareOTLPFileUnusedNestedEventAndLinkDataIsNotEvidence(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "otlp-file.jsonl")
	contents, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	candidate := strings.ReplaceAll(string(contents), "synthetic.event", "synthetic.changed-event")
	candidate = strings.ReplaceAll(candidate, "synthetic=1", "synthetic=2")
	candidate = strings.ReplaceAll(candidate, `"alpha"`, `"changed-nested-value"`)

	comparison, err := Compare(bytes.NewReader(contents), strings.NewReader(candidate), DefaultOptions())
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("Compare() comparison = %#v, want unused accepted payloads excluded from evidence", comparison)
	}
}

func TestCompareCallerRedactionOverridesSafeMatchingAttribute(t *testing.T) {
	baseline := `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"checkout"}}]},"scopeSpans":[{"spans":[{"traceId":"11111111111111111111111111111111","spanId":"1111111111111111","name":"handle","kind":2,"startTimeUnixNano":"1","endTimeUnixNano":"2","attributes":[{"key":"http.route","value":{"stringValue":"/private/baseline"}}]}]}]}]}`
	candidate := strings.ReplaceAll(baseline, "/private/baseline", "/private/candidate")
	options := DefaultOptions()
	options.RedactedAttributeKeys = []string{"HTTP.ROUTE"}

	comparison, err := Compare(strings.NewReader(baseline), strings.NewReader(candidate), options)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("Compare() comparison = %#v, want caller-denied safe key excluded", comparison)
	}
}

func TestCompareBuiltInSensitiveValuesDoNotReachResultsOrErrors(t *testing.T) {
	baseline := `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"checkout"}},{"key":"user.email","value":{"stringValue":"baseline-person@example.invalid"}}]},"scopeSpans":[{"spans":[{"traceId":"11111111111111111111111111111111","spanId":"1111111111111111","name":"handle","kind":2,"startTimeUnixNano":"1","endTimeUnixNano":"2","attributes":[{"key":"http.route","value":{"stringValue":"/checkout"}},{"key":"password","value":{"stringValue":"baseline-password"}}]}]}]}]}`
	candidate := strings.ReplaceAll(baseline, "baseline-person@example.invalid", "candidate-person@example.invalid")
	candidate = strings.ReplaceAll(candidate, "baseline-password", "candidate-password")

	comparison, err := Compare(strings.NewReader(baseline), strings.NewReader(candidate), DefaultOptions())
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	encoded := fmt.Sprintf("%#v", comparison)
	for _, sensitiveValue := range []string{"baseline-person@example.invalid", "candidate-person@example.invalid", "baseline-password", "candidate-password"} {
		if strings.Contains(encoded, sensitiveValue) {
			t.Fatalf("comparison contains sensitive test value %q", sensitiveValue)
		}
	}
	if comparison.HasDifferences() {
		t.Fatalf("Compare() comparison = %#v, want sensitive-only changes ignored", comparison)
	}
	var report bytes.Buffer
	if err := WriteText(&report, comparison, "baseline.json", "candidate.json"); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}
	for _, sensitiveValue := range []string{"baseline-person@example.invalid", "candidate-person@example.invalid", "baseline-password", "candidate-password"} {
		if strings.Contains(report.String(), sensitiveValue) {
			t.Fatalf("text report contains sensitive test value %q", sensitiveValue)
		}
	}
	report.Reset()
	if err := WriteJSON(&report, comparison, "baseline.json", "candidate.json"); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	for _, sensitiveValue := range []string{"baseline-person@example.invalid", "candidate-person@example.invalid", "baseline-password", "candidate-password"} {
		if strings.Contains(report.String(), sensitiveValue) {
			t.Fatalf("JSON report contains sensitive test value %q", sensitiveValue)
		}
	}
	report.Reset()
	if err := WriteHTML(&report, comparison, "baseline.json", "candidate.json"); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	for _, sensitiveValue := range []string{"baseline-person@example.invalid", "candidate-person@example.invalid", "baseline-password", "candidate-password"} {
		if strings.Contains(report.String(), sensitiveValue) {
			t.Fatalf("HTML report contains sensitive test value %q", sensitiveValue)
		}
	}

	badOptions := DefaultOptions()
	badOptions.RedactedAttributeKeys = []string{""}
	_, err = Compare(strings.NewReader(baseline), strings.NewReader(candidate), badOptions)
	if err == nil || !strings.Contains(err.Error(), "redact baseline attributes") {
		t.Fatalf("Compare() redaction error = %v, want contextual error", err)
	}
	for _, sensitiveValue := range []string{"baseline-person@example.invalid", "baseline-password"} {
		if strings.Contains(err.Error(), sensitiveValue) {
			t.Fatalf("redaction error contains sensitive test value %q", sensitiveValue)
		}
	}
}

func TestCompareErrorTypeChangeAndCallerDenial(t *testing.T) {
	baseline := `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"checkout"}}]},"scopeSpans":[{"spans":[{"traceId":"11111111111111111111111111111111","spanId":"1111111111111111","name":"handle","kind":2,"startTimeUnixNano":"1","endTimeUnixNano":"2","attributes":[{"key":"http.route","value":{"stringValue":"/checkout"}},{"key":"error.type","value":{"stringValue":"synthetic.Timeout"}},{"key":"exception.message","value":{"stringValue":"baseline private detail"}}],"status":{"code":2,"message":"baseline status detail"}}]}]}]}`
	candidate := strings.ReplaceAll(baseline, "synthetic.Timeout", "synthetic.Declined")
	candidate = strings.ReplaceAll(candidate, "baseline private detail", "candidate private detail")
	candidate = strings.ReplaceAll(candidate, "baseline status detail", "candidate status detail")

	comparison, err := Compare(strings.NewReader(baseline), strings.NewReader(candidate), DefaultOptions())
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(comparison.Changes) != 1 || comparison.Changes[0].Field != diff.FieldErrorType || comparison.Changes[0].Before != "synthetic.Timeout" || comparison.Changes[0].After != "synthetic.Declined" {
		t.Fatalf("Compare() changes = %#v, want one safe error.type change", comparison.Changes)
	}
	encoded := fmt.Sprintf("%#v", comparison)
	for _, unsafe := range []string{"private detail", "status detail"} {
		if strings.Contains(encoded, unsafe) {
			t.Fatalf("comparison contains unsafe %s", unsafe)
		}
	}

	options := DefaultOptions()
	options.RedactedAttributeKeys = []string{"error.type"}
	comparison, err = Compare(strings.NewReader(baseline), strings.NewReader(candidate), options)
	if err != nil {
		t.Fatalf("Compare(redacted error.type) error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("Compare(redacted error.type) = %#v, want no differences", comparison)
	}
}

func TestCompareFilesCanonicalizesEquivalentRuns(t *testing.T) {
	baseline := filepath.Join("..", "..", "testdata", "normalize-run-b.json")
	candidate := filepath.Join("..", "..", "testdata", "normalize-run-a.json")
	exactOptions := DefaultOptions()
	exactOptions.DurationThreshold = 0
	exactOptions.DurationThresholdAbsolute = 0

	exactComparison, err := CompareFiles(baseline, candidate, exactOptions)
	if err != nil {
		t.Fatalf("CompareFiles(exact durations) error = %v", err)
	}
	if !exactComparison.HasDifferences() {
		t.Fatal("CompareFiles(exact durations) found no differences, want duration noise before bucketing")
	}

	bucketedOptions := exactOptions
	bucketedOptions.Normalization.DurationBucket = 10 * time.Nanosecond

	comparison, err := CompareFiles(baseline, candidate, bucketedOptions)
	if err != nil {
		t.Fatalf("CompareFiles(bucketed durations) error = %v", err)
	}
	if comparison.HasDifferences() {
		t.Fatalf("CompareFiles(bucketed durations) comparison = %#v, want no differences", comparison)
	}
}

func TestCompareFilesRejectsNegativeNormalizationDurationBucket(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "otlp-representative.json")
	options := DefaultOptions()
	options.Normalization.DurationBucket = -time.Nanosecond

	_, err := CompareFiles(fixture, fixture, options)
	if err == nil || !strings.Contains(err.Error(), "normalize baseline traces: duration bucket must be non-negative") {
		t.Fatalf("CompareFiles() error = %v, want contextual negative duration-bucket error", err)
	}
}

func TestCompareRejectsAmbiguousRepeatedTracesContextually(t *testing.T) {
	baseline := `{"resourceSpans":[{"scopeSpans":[{"spans":[
		{"traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","spanId":"1111111111111111","name":"private-operation","kind":2,"startTimeUnixNano":"100","endTimeUnixNano":"200","status":{"code":1}},
		{"traceId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","spanId":"2222222222222222","name":"private-operation","kind":2,"startTimeUnixNano":"300","endTimeUnixNano":"500","status":{"code":1}}
	]}]}]}`
	candidate := `{"resourceSpans":[{"scopeSpans":[{"spans":[
		{"traceId":"cccccccccccccccccccccccccccccccc","spanId":"3333333333333333","name":"private-operation","kind":2,"startTimeUnixNano":"600","endTimeUnixNano":"700","status":{"code":1}},
		{"traceId":"dddddddddddddddddddddddddddddddd","spanId":"4444444444444444","name":"private-operation","kind":2,"startTimeUnixNano":"800","endTimeUnixNano":"1000","status":{"code":1}}
	]}]}]}`

	_, err := Compare(strings.NewReader(baseline), strings.NewReader(candidate), DefaultOptions())
	if err == nil || !strings.Contains(err.Error(), "match traces: ambiguous trace match: 2 baseline and 2 candidate traces") {
		t.Fatalf("Compare() error = %v, want contextual ambiguity", err)
	}
	if strings.Contains(err.Error(), "private-operation") {
		t.Fatalf("Compare() ambiguity error leaked trace-derived name: %v", err)
	}
}

func TestCompareRejectsAmbiguousSiblingSpansContextually(t *testing.T) {
	baseline := `{"resourceSpans":[{"scopeSpans":[{"spans":[
		{"traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","spanId":"1111111111111111","name":"root","kind":2,"startTimeUnixNano":"100","endTimeUnixNano":"300","status":{"code":1}},
		{"traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","spanId":"2222222222222222","parentSpanId":"1111111111111111","name":"private-child","kind":3,"startTimeUnixNano":"200","endTimeUnixNano":"250","status":{"code":1}},
		{"traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","spanId":"3333333333333333","parentSpanId":"1111111111111111","name":"private-child","kind":3,"startTimeUnixNano":"200","endTimeUnixNano":"250","status":{"code":1}}
	]}]}]}`
	candidate := `{"resourceSpans":[{"scopeSpans":[{"spans":[
		{"traceId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","spanId":"4444444444444444","name":"root","kind":2,"startTimeUnixNano":"500","endTimeUnixNano":"700","status":{"code":1}},
		{"traceId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","spanId":"5555555555555555","parentSpanId":"4444444444444444","name":"private-child","kind":3,"startTimeUnixNano":"600","endTimeUnixNano":"650","status":{"code":1}},
		{"traceId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","spanId":"6666666666666666","parentSpanId":"4444444444444444","name":"private-child","kind":3,"startTimeUnixNano":"600","endTimeUnixNano":"650","status":{"code":1}}
	]}]}]}`

	_, err := Compare(strings.NewReader(baseline), strings.NewReader(candidate), DefaultOptions())
	if err == nil || !strings.Contains(err.Error(), "match spans: ambiguous span match: 2 baseline and 2 candidate spans") {
		t.Fatalf("Compare() error = %v, want contextual span ambiguity", err)
	}
	if strings.Contains(err.Error(), "private-child") {
		t.Fatalf("Compare() span ambiguity error leaked trace-derived name: %v", err)
	}
}
