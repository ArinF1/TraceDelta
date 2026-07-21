package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

func FuzzReportersEscapeTraceDerivedStrings(f *testing.F) {
	f.Add("checkout.handle", "checkout-service", "synthetic.Timeout", "synthetic.Declined", "baseline.json", "candidate.json")
	f.Add("</td><script>alert(1)</script>", "service&name", "\x1b[31m", "line\nvalue", "<baseline>", "candidate\x00.json")

	f.Fuzz(func(t *testing.T, spanName, serviceName, before, after, baselinePath, candidatePath string) {
		result := diff.Result{
			ChangedSpans: 1,
			Changes: []diff.Change{{
				Kind:          diff.KindChanged,
				Field:         diff.FieldErrorType,
				SpanName:      spanName,
				ServiceName:   serviceName,
				Before:        before,
				After:         after,
				BeforePresent: true,
				AfterPresent:  true,
			}},
		}
		metadata := Metadata{BaselinePath: baselinePath, CandidatePath: candidatePath}

		var textOutput strings.Builder
		if err := WriteText(&textOutput, result, metadata); err != nil {
			t.Fatalf("WriteText() error = %v", err)
		}
		if strings.ContainsRune(textOutput.String(), '\x00') || strings.ContainsRune(textOutput.String(), '\x1b') {
			t.Fatal("text report retained an unsafe NUL or escape control")
		}

		var jsonOutput strings.Builder
		if err := WriteJSON(&jsonOutput, result, metadata); err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}
		if !json.Valid([]byte(jsonOutput.String())) {
			t.Fatal("WriteJSON() produced invalid JSON")
		}

		var htmlOutput strings.Builder
		if err := WriteHTML(&htmlOutput, result, metadata); err != nil {
			t.Fatalf("WriteHTML() error = %v", err)
		}
		html := strings.ToLower(htmlOutput.String())
		for _, forbidden := range []string{"<script", "<iframe", "javascript:", "<link", " src="} {
			if strings.Contains(html, forbidden) {
				t.Fatalf("WriteHTML() retained active content %q", forbidden)
			}
		}
	})
}
