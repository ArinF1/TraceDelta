package tracedelta

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCompareFilesCanonicalizesEquivalentRuns(t *testing.T) {
	baseline := filepath.Join("..", "..", "testdata", "normalize-run-b.json")
	candidate := filepath.Join("..", "..", "testdata", "normalize-run-a.json")
	exactOptions := DefaultOptions()
	exactOptions.DurationThreshold = 0

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
