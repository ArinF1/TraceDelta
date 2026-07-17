package tracedelta

import (
	"path/filepath"
	"testing"
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
