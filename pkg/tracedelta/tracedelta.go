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
	// Normalization controls deterministic pre-match normalization.
	Normalization NormalizationOptions
}

// DefaultOptions returns the documented initial comparison defaults.
func DefaultOptions() Options {
	return Options{DurationThreshold: 0.20}
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

	normalizationOptions := normalize.Options{DurationBucket: options.Normalization.DurationBucket}
	normalizedBaseline, err := normalize.Snapshot(baselineSnapshot, normalizationOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("normalize baseline traces: %w", err)
	}
	normalizedCandidate, err := normalize.Snapshot(candidateSnapshot, normalizationOptions)
	if err != nil {
		return Comparison{}, fmt.Errorf("normalize candidate traces: %w", err)
	}

	matches := match.Spans(normalizedBaseline, normalizedCandidate)
	comparison, err := diff.Compare(matches, diff.Options{DurationThreshold: options.DurationThreshold})
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
