// Package report renders deterministic comparison reports.
package report

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

// Metadata identifies the compared inputs in a report.
type Metadata struct {
	BaselinePath  string
	CandidatePath string
}

// WriteText writes the current human-readable terminal report format.
func WriteText(w io.Writer, result diff.Result, metadata Metadata) error {
	var output strings.Builder
	output.WriteString("TraceDelta comparison\n\n")
	fmt.Fprintf(&output, "Baseline:  %s\n", escapeControls(metadata.BaselinePath))
	fmt.Fprintf(&output, "Candidate: %s\n\n", escapeControls(metadata.CandidatePath))
	output.WriteString("Summary:\n")
	fmt.Fprintf(&output, "  Added spans:    %d\n", result.AddedSpans)
	fmt.Fprintf(&output, "  Removed spans:  %d\n", result.RemovedSpans)
	fmt.Fprintf(&output, "  Changed spans:  %d\n\n", result.ChangedSpans)
	output.WriteString("Changes:\n")
	if len(result.Changes) == 0 {
		output.WriteString("  None\n")
	} else {
		for _, change := range result.Changes {
			fmt.Fprintf(&output, "  %-8s %s", change.Kind, escapeControls(change.SpanName))
			switch change.Field {
			case diff.FieldStatus:
				fmt.Fprintf(&output, " status %s -> %s", change.Before, change.After)
			case diff.FieldDuration:
				fmt.Fprintf(&output, " duration %s -> %s", change.Before, change.After)
			}
			output.WriteByte('\n')
		}
	}
	output.WriteByte('\n')
	if result.HasDifferences() {
		output.WriteString("Result: behavioral differences detected\n")
	} else {
		output.WriteString("Result: no behavioral differences detected\n")
	}

	if _, err := io.WriteString(w, output.String()); err != nil {
		return fmt.Errorf("write text report: %w", err)
	}
	return nil
}

func escapeControls(value string) string {
	var escaped strings.Builder
	for _, character := range value {
		switch character {
		case '\n':
			escaped.WriteString(`\n`)
		case '\r':
			escaped.WriteString(`\r`)
		case '\t':
			escaped.WriteString(`\t`)
		default:
			if unicode.IsControl(character) {
				fmt.Fprintf(&escaped, `\u{%x}`, character)
				continue
			}
			escaped.WriteRune(character)
		}
	}
	return escaped.String()
}
