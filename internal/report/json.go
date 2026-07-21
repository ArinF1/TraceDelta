package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

const jsonSchemaVersion = "tracedelta.report/v1"

type jsonReport struct {
	SchemaVersion string        `json:"schemaVersion"`
	Inputs        jsonInputs    `json:"inputs"`
	Summary       jsonSummary   `json:"summary"`
	Result        jsonResult    `json:"result"`
	Findings      []jsonFinding `json:"findings"`
}

type jsonInputs struct {
	Baseline  jsonInput `json:"baseline"`
	Candidate jsonInput `json:"candidate"`
}

type jsonInput struct {
	Path string `json:"path"`
}

type jsonSummary struct {
	AddedSpans   int `json:"addedSpans"`
	RemovedSpans int `json:"removedSpans"`
	ChangedSpans int `json:"changedSpans"`
	Findings     int `json:"findings"`
}

type jsonResult struct {
	Status   string `json:"status"`
	ExitCode int    `json:"exitCode"`
}

type jsonFinding struct {
	Kind     diff.Kind    `json:"kind"`
	Field    diff.Field   `json:"field"`
	Span     jsonSpan     `json:"span"`
	Evidence jsonEvidence `json:"evidence"`
}

type jsonSpan struct {
	Name        string `json:"name"`
	ServiceName string `json:"serviceName"`
	Occurrence  int    `json:"occurrence"`
}

type jsonEvidence struct {
	Before            jsonEvidenceValue     `json:"before"`
	After             jsonEvidenceValue     `json:"after"`
	DurationThreshold jsonDurationThreshold `json:"durationThreshold"`
}

type jsonEvidenceValue struct {
	Present bool   `json:"present"`
	Value   string `json:"value"`
}

type jsonDurationThreshold struct {
	Present       bool    `json:"present"`
	RelativeRatio float64 `json:"relativeRatio"`
	Absolute      string  `json:"absolute"`
}

// WriteJSON writes the schema-versioned machine-readable report. The encoder
// escapes HTML-significant characters and buffers the complete document before
// touching the destination.
func WriteJSON(w io.Writer, result diff.Result, metadata Metadata) error {
	report := makeJSONReport(result, metadata)
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode JSON report: %w", err)
	}
	if _, err := w.Write(output.Bytes()); err != nil {
		return fmt.Errorf("write JSON report: %w", err)
	}
	return nil
}

func makeJSONReport(result diff.Result, metadata Metadata) jsonReport {
	status := "no_differences"
	exitCode := 0
	if result.HasDifferences() {
		status = "differences"
		exitCode = 1
	}

	findings := make([]jsonFinding, len(result.Changes))
	for index, change := range result.Changes {
		threshold := jsonDurationThreshold{}
		if change.Field == diff.FieldDuration {
			threshold = jsonDurationThreshold{
				Present:       true,
				RelativeRatio: change.DurationThresholdRelative,
				Absolute:      change.DurationThresholdAbsolute.String(),
			}
		}
		findings[index] = jsonFinding{
			Kind:  change.Kind,
			Field: change.Field,
			Span: jsonSpan{
				Name:        change.SpanName,
				ServiceName: change.ServiceName,
				Occurrence:  change.Occurrence,
			},
			Evidence: jsonEvidence{
				Before:            jsonEvidenceValue{Present: change.BeforePresent, Value: change.Before},
				After:             jsonEvidenceValue{Present: change.AfterPresent, Value: change.After},
				DurationThreshold: threshold,
			},
		}
	}

	return jsonReport{
		SchemaVersion: jsonSchemaVersion,
		Inputs: jsonInputs{
			Baseline:  jsonInput{Path: metadata.BaselinePath},
			Candidate: jsonInput{Path: metadata.CandidatePath},
		},
		Summary: jsonSummary{
			AddedSpans:   result.AddedSpans,
			RemovedSpans: result.RemovedSpans,
			ChangedSpans: result.ChangedSpans,
			Findings:     len(result.Changes),
		},
		Result:   jsonResult{Status: status, ExitCode: exitCode},
		Findings: findings,
	}
}
