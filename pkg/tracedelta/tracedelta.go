// Package tracedelta provides the public API for local trace comparison.
package tracedelta

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ArinF1/TraceDelta/internal/diff"
	"github.com/ArinF1/TraceDelta/internal/match"
	"github.com/ArinF1/TraceDelta/internal/normalize"
	"github.com/ArinF1/TraceDelta/internal/otlp"
	"github.com/ArinF1/TraceDelta/internal/redact"
	"github.com/ArinF1/TraceDelta/internal/report"
)

// NormalizationOptions controls the deterministic projection applied before
// matching.
type NormalizationOptions struct {
	// DurationBucket floors span durations to this width. Zero preserves exact
	// durations. This is separate from the relative regression threshold.
	DurationBucket time.Duration
}

// Options controls comparison behavior.
type Options struct {
	// DurationThreshold is a relative ratio, where 0.20 means a 20% increase.
	DurationThreshold float64
	// DurationThresholdAbsolute is the minimum absolute increase required in
	// addition to DurationThreshold.
	DurationThresholdAbsolute time.Duration
	// Normalization controls deterministic pre-match normalization.
	Normalization NormalizationOptions
	// RedactedAttributeKeys adds case-insensitive attribute keys to the built-in
	// credential and personal-data deny rules. Denied keys are removed before
	// normalization and always override the fixed safe matching set.
	RedactedAttributeKeys []string
}

// DefaultOptions returns the documented initial comparison defaults.
func DefaultOptions() Options {
	return Options{
		DurationThreshold:         0.20,
		DurationThresholdAbsolute: 10 * time.Millisecond,
	}
}

// Comparison is the result of comparing two trace snapshots.
type Comparison = diff.Result

// Change is one reportable behavioral change.
type Change = diff.Change

// Compare parses and compares two streams in TraceDelta's documented OTLP JSON subset.
func Compare(baseline, candidate io.Reader, options Options) (Comparison, error) {
	baselineSnapshot, err := otlp.Parse(baseline)
	if err != nil {
		return Comparison{}, fmt.Errorf("parse baseline traces: %w", err)
	}
	candidateSnapshot, err := otlp.Parse(candidate)
	if err != nil {
		return Comparison{}, fmt.Errorf("parse candidate traces: %w", err)
	}
	redactionOptions := redact.Options{AdditionalKeys: options.RedactedAttributeKeys}
	baselineSnapshot, err = redact.Snapshot(baselineSnapshot, redactionOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("redact baseline attributes: %w", err)
	}
	candidateSnapshot, err = redact.Snapshot(candidateSnapshot, redactionOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("redact candidate attributes: %w", err)
	}

	normalizationOptions := normalize.Options{DurationBucket: options.Normalization.DurationBucket}
	normalizedBaseline, err := normalize.Snapshot(baselineSnapshot, normalizationOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("normalize baseline traces: %w", err)
	}
	normalizedCandidate, err := normalize.Snapshot(candidateSnapshot, normalizationOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("normalize candidate traces: %w", err)
	}

	traceMatches, err := match.Traces(normalizedBaseline, normalizedCandidate)
	if err != nil {
		return Comparison{}, fmt.Errorf("match traces: %w", err)
	}
	matches, err := match.Spans(traceMatches)
	if err != nil {
		return Comparison{}, fmt.Errorf("match spans: %w", err)
	}
	comparison, err := diff.Compare(matches, diff.Options{
		DurationThreshold:         options.DurationThreshold,
		DurationThresholdAbsolute: options.DurationThresholdAbsolute,
	})
	if err != nil {
		return Comparison{}, fmt.Errorf("compare traces: %w", err)
	}
	return comparison, nil
}

// CompareFiles opens and compares two local files in the documented OTLP JSON subset.
func CompareFiles(baselinePath, candidatePath string, options Options) (Comparison, error) {
	baseline, err := os.Open(baselinePath)
	if err != nil {
		return Comparison{}, fmt.Errorf("open baseline %q: %w", baselinePath, err)
	}
	defer baseline.Close()

	candidate, err := os.Open(candidatePath)
	if err != nil {
		return Comparison{}, fmt.Errorf("open candidate %q: %w", candidatePath, err)
	}
	defer candidate.Close()

	return Compare(baseline, candidate, options)
}

// WriteText writes a human-readable comparison report.
func WriteText(w io.Writer, comparison Comparison, baselinePath, candidatePath string) error {
	return report.WriteText(w, comparison, report.Metadata{
		BaselinePath:  baselinePath,
		CandidatePath: candidatePath,
	})
}

// WriteJSON writes the schema-versioned machine-readable comparison report.
func WriteJSON(w io.Writer, comparison Comparison, baselinePath, candidatePath string) error {
	return report.WriteJSON(w, comparison, report.Metadata{
		BaselinePath:  baselinePath,
		CandidatePath: candidatePath,
	})
}

// WriteHTML writes the self-contained offline comparison report.
func WriteHTML(w io.Writer, comparison Comparison, baselinePath, candidatePath string) error {
	return report.WriteHTML(w, comparison, report.Metadata{
		BaselinePath:  baselinePath,
		CandidatePath: candidatePath,
	})
}
