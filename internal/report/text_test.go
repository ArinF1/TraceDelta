package report

import (
	"strings"
	"testing"
	"time"

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
			{Kind: diff.KindChanged, Field: diff.FieldDuration, SpanName: "payment.charge", Before: "120ms", After: "245ms", DurationThresholdRelative: 0.20, DurationThresholdAbsolute: 10 * time.Millisecond},
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
  CHANGED  payment.charge duration 120ms -> 245ms (thresholds: >=20% and >=10ms)

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

func TestWriteTextRendersSafeErrorTypeEvidenceAndMissingValue(t *testing.T) {
	result := diff.Result{
		ChangedSpans: 1,
		Changes: []diff.Change{{
			Kind:         diff.KindChanged,
			Field:        diff.FieldErrorType,
			SpanName:     "checkout.handle",
			After:        "synthetic\nDeclined",
			AfterPresent: true,
		}},
	}
	var output strings.Builder
	if err := WriteText(&output, result, Metadata{BaselinePath: "baseline.json", CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}
	if !strings.Contains(output.String(), `error.type (missing) -> synthetic\nDeclined`) {
		t.Fatalf("WriteText() output = %q, want escaped error.type evidence", output.String())
	}
}

func TestFormatEvidenceDistinguishesMissingAndEmpty(t *testing.T) {
	if got := formatEvidence("", false); got != "(missing)" {
		t.Fatalf("formatEvidence(missing) = %q", got)
	}
	if got := formatEvidence("", true); got != "(empty)" {
		t.Fatalf("formatEvidence(empty) = %q", got)
	}
}

func TestFormatPercentageIsDeterministic(t *testing.T) {
	for ratio, want := range map[float64]string{
		0:     "0%",
		0.125: "12.5%",
		0.20:  "20%",
		1:     "100%",
	} {
		if got := formatPercentage(ratio); got != want {
			t.Errorf("formatPercentage(%v) = %q, want %q", ratio, got, want)
		}
	}
}
