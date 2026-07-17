package report

import (
	"strings"
	"testing"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

func TestWriteTextRendersDeterministicReport(t *testing.T) {
	result := diff.Result{
		AddedSpans:   1,
		RemovedSpans: 1,
		ChangedSpans: 2,
		Changes: []diff.Change{
			{Kind: diff.KindAdded, SpanName: "inventory.reserve"},
			{Kind: diff.KindRemoved, SpanName: "cache.get"},
			{Kind: diff.KindChanged, Field: diff.FieldStatus, SpanName: "checkout.handle", Before: "OK", After: "ERROR"},
			{Kind: diff.KindChanged, Field: diff.FieldDuration, SpanName: "payment.charge", Before: "120ms", After: "245ms"},
		},
	}
	var output strings.Builder
	if err := WriteText(&output, result, Metadata{BaselinePath: "baseline.json", CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}

	want := `TraceDelta comparison

Baseline:  baseline.json
Candidate: candidate.json

Summary:
  Added spans:    1
  Removed spans:  1
  Changed spans:  2

Changes:
  ADDED    inventory.reserve
  REMOVED  cache.get
  CHANGED  checkout.handle status OK -> ERROR
  CHANGED  payment.charge duration 120ms -> 245ms

Result: behavioral differences detected
`
	if output.String() != want {
		t.Fatalf("WriteText() output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestWriteTextEscapesTerminalControlCharacters(t *testing.T) {
	result := diff.Result{
		AddedSpans: 1,
		Changes: []diff.Change{{
			Kind:     diff.KindAdded,
			SpanName: "inventory\n\x1breserve",
		}},
	}
	var output strings.Builder
	if err := WriteText(&output, result, Metadata{BaselinePath: "base\r.json", CandidatePath: "candidate\t.json"}); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}

	for _, want := range []string{`base\r.json`, `candidate\t.json`, `inventory\n\u{1b}reserve`} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("WriteText() output = %q, want it to contain %q", output.String(), want)
		}
	}
	if strings.Contains(output.String(), "\x1b") {
		t.Fatalf("WriteText() output contains a raw terminal escape: %q", output.String())
	}
}
