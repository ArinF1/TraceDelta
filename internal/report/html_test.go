package report

import (
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

func TestWriteHTMLRendersStandaloneOrderedReport(t *testing.T) {
	result := diff.Result{
		AddedSpans:   1,
		RemovedSpans: 1,
		ChangedSpans: 2,
		Changes: []diff.Change{
			{Kind: diff.KindAdded, SpanName: "inventory.reserve", ServiceName: "inventory"},
			{Kind: diff.KindRemoved, SpanName: "cache.get", ServiceName: "checkout"},
			{Kind: diff.KindChanged, Field: diff.FieldStatus, SpanName: "checkout.handle", ServiceName: "checkout", Before: "OK", After: "ERROR", BeforePresent: true, AfterPresent: true},
			{Kind: diff.KindChanged, Field: diff.FieldDuration, SpanName: "payment.charge", ServiceName: "payments", Before: "120ms", After: "245ms", BeforePresent: true, AfterPresent: true, DurationThresholdRelative: 0.20, DurationThresholdAbsolute: 10 * time.Millisecond},
		},
	}
	var output strings.Builder
	if err := WriteHTML(&output, result, Metadata{BaselinePath: "baseline.json", CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	html := output.String()
	for _, want := range []string{
		"<!doctype html>",
		"Behavioral differences detected",
		">1</b><span>Added spans",
		">1</b><span>Removed spans",
		">2</b><span>Changed spans",
		">4</b><span>Total findings",
		"duration 120ms → 245ms (thresholds: ≥20% and ≥10ms)",
		"Self-contained offline report",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("WriteHTML() output missing %q", want)
		}
	}
	previous := -1
	for _, spanName := range []string{"inventory.reserve", "cache.get", "checkout.handle", "payment.charge"} {
		index := strings.Index(html, spanName)
		if index <= previous {
			t.Fatalf("finding %q index = %d after %d; output order changed", spanName, index, previous)
		}
		previous = index
	}
	for _, forbidden := range []string{"<script", "<link", "http://", "https://", " src="} {
		if strings.Contains(strings.ToLower(html), forbidden) {
			t.Fatalf("standalone HTML contains forbidden external/active content %q", forbidden)
		}
	}
}

func TestWriteHTMLEscapesTraceDerivedContentAndRendersEmptyState(t *testing.T) {
	result := diff.Result{AddedSpans: 1, Changes: []diff.Change{{
		Kind:        diff.KindAdded,
		SpanName:    `</td><script>alert("span")</script>`,
		ServiceName: "service&name\nline",
	}}}
	var output strings.Builder
	if err := WriteHTML(&output, result, Metadata{BaselinePath: `</code><script>alert("path")</script>`, CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	html := output.String()
	for _, unsafe := range []string{"<script>alert", "</code><script", "service&name"} {
		if strings.Contains(html, unsafe) {
			t.Fatalf("WriteHTML() contains unescaped %q: %s", unsafe, html)
		}
	}
	for _, escaped := range []string{"&lt;/td&gt;&lt;script&gt;", "service&amp;name\\nline"} {
		if !strings.Contains(html, escaped) {
			t.Fatalf("WriteHTML() missing escaped %q", escaped)
		}
	}

	output.Reset()
	if err := WriteHTML(&output, diff.Result{}, Metadata{}); err != nil {
		t.Fatalf("WriteHTML(no differences) error = %v", err)
	}
	if !strings.Contains(output.String(), "No behavioral differences") || !strings.Contains(output.String(), "No findings under the configured policy.") {
		t.Fatalf("WriteHTML(no differences) output missing pass/empty state")
	}
}
