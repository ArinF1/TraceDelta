// Package diff turns matched spans into semantic behavior changes.
package diff

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/ArinF1/TraceDelta/internal/match"
	"github.com/ArinF1/TraceDelta/internal/model"
)

// Kind classifies a reported change.
type Kind string

const (
	// KindAdded identifies a candidate-only span.
	KindAdded Kind = "ADDED"
	// KindRemoved identifies a baseline-only span.
	KindRemoved Kind = "REMOVED"
	// KindChanged identifies a changed field on a matched span.
	KindChanged Kind = "CHANGED"
)

// Field identifies the matched-span property that changed.
type Field string

const (
	// FieldNone is used for added and removed spans.
	FieldNone Field = ""
	// FieldStatus identifies an OpenTelemetry status-code change.
	FieldStatus Field = "status"
	// FieldErrorType identifies a changed safe error.type value while both
	// matched spans have ERROR status.
	FieldErrorType Field = "error.type"
	// FieldDuration identifies a meaningful duration change.
	FieldDuration Field = "duration"
)

// Change is one deterministic, reportable behavioral change.
type Change struct {
	Kind        Kind
	Field       Field
	SpanName    string
	ServiceName string
	Occurrence  int
	Before      string
	After       string
	// BeforePresent and AfterPresent distinguish absent evidence from a
	// present empty value. Added/removed changes have no before/after evidence.
	BeforePresent bool
	AfterPresent  bool
	// Duration thresholds are populated only for duration findings so every
	// report can render the effective policy that produced the finding.
	DurationThresholdRelative float64
	DurationThresholdAbsolute time.Duration
}

// Result summarizes all meaningful differences in a comparison.
type Result struct {
	Changes      []Change
	AddedSpans   int
	RemovedSpans int
	ChangedSpans int
}

// HasDifferences reports whether the comparison found any meaningful change.
func (result Result) HasDifferences() bool {
	return len(result.Changes) > 0
}

// Options controls semantic diff thresholds.
type Options struct {
	// DurationThreshold is a non-negative relative ratio, where 0.20 means 20%.
	DurationThreshold float64
	// DurationThresholdAbsolute is the minimum absolute candidate increase.
	DurationThresholdAbsolute time.Duration
}

// Compare detects additions, removals, status changes, and duration changes.
func Compare(matches match.Result, options Options) (Result, error) {
	if options.DurationThreshold < 0 || math.IsNaN(options.DurationThreshold) || math.IsInf(options.DurationThreshold, 0) {
		return Result{}, fmt.Errorf("duration threshold must be a finite, non-negative ratio")
	}
	if options.DurationThresholdAbsolute < 0 {
		return Result{}, fmt.Errorf("absolute duration threshold must be non-negative")
	}

	result := Result{
		AddedSpans:   len(matches.Added),
		RemovedSpans: len(matches.Removed),
	}
	for _, span := range matches.Added {
		result.Changes = append(result.Changes, Change{
			Kind:        KindAdded,
			Field:       FieldNone,
			SpanName:    span.Key.Name,
			ServiceName: span.Key.ServiceName,
			Occurrence:  span.Occurrence,
		})
	}
	for _, span := range matches.Removed {
		result.Changes = append(result.Changes, Change{
			Kind:        KindRemoved,
			Field:       FieldNone,
			SpanName:    span.Key.Name,
			ServiceName: span.Key.ServiceName,
			Occurrence:  span.Occurrence,
		})
	}
	for _, pair := range matches.Paired {
		changed := false
		if pair.Baseline.Status != pair.Candidate.Status {
			changed = true
			result.Changes = append(result.Changes, Change{
				Kind:          KindChanged,
				Field:         FieldStatus,
				SpanName:      pair.Baseline.Key.Name,
				ServiceName:   pair.Baseline.Key.ServiceName,
				Occurrence:    pair.Baseline.Occurrence,
				Before:        string(pair.Baseline.Status),
				After:         string(pair.Candidate.Status),
				BeforePresent: true,
				AfterPresent:  true,
			})
		}
		if errorTypeChanged(pair.Baseline, pair.Candidate) {
			changed = true
			result.Changes = append(result.Changes, Change{
				Kind:          KindChanged,
				Field:         FieldErrorType,
				SpanName:      pair.Baseline.Key.Name,
				ServiceName:   pair.Baseline.Key.ServiceName,
				Occurrence:    pair.Baseline.Occurrence,
				Before:        pair.Baseline.ErrorType,
				After:         pair.Candidate.ErrorType,
				BeforePresent: pair.Baseline.ErrorTypePresent,
				AfterPresent:  pair.Candidate.ErrorTypePresent,
			})
		}
		if durationChanged(pair.Baseline.Duration, pair.Candidate.Duration, options.DurationThreshold, options.DurationThresholdAbsolute) {
			changed = true
			result.Changes = append(result.Changes, Change{
				Kind:                      KindChanged,
				Field:                     FieldDuration,
				SpanName:                  pair.Baseline.Key.Name,
				ServiceName:               pair.Baseline.Key.ServiceName,
				Occurrence:                pair.Baseline.Occurrence,
				Before:                    pair.Baseline.Duration.String(),
				After:                     pair.Candidate.Duration.String(),
				BeforePresent:             true,
				AfterPresent:              true,
				DurationThresholdRelative: options.DurationThreshold,
				DurationThresholdAbsolute: options.DurationThresholdAbsolute,
			})
		}
		if changed {
			result.ChangedSpans++
		}
	}

	sort.SliceStable(result.Changes, func(i, j int) bool {
		return lessChange(result.Changes[i], result.Changes[j])
	})
	return result, nil
}

func errorTypeChanged(baseline, candidate model.NormalizedSpan) bool {
	if baseline.Status != model.StatusError || candidate.Status != model.StatusError {
		return false
	}
	return baseline.ErrorTypePresent != candidate.ErrorTypePresent || baseline.ErrorType != candidate.ErrorType
}

func durationChanged(baseline, candidate time.Duration, relativeThreshold float64, absoluteThreshold time.Duration) bool {
	if candidate <= baseline {
		return false
	}
	difference := candidate - baseline
	if difference < absoluteThreshold {
		return false
	}
	if baseline == 0 {
		return true
	}
	return float64(difference)/float64(baseline) >= relativeThreshold
}

func lessChange(left, right Change) bool {
	if kindRank(left.Kind) != kindRank(right.Kind) {
		return kindRank(left.Kind) < kindRank(right.Kind)
	}
	if left.SpanName != right.SpanName {
		return left.SpanName < right.SpanName
	}
	if left.ServiceName != right.ServiceName {
		return left.ServiceName < right.ServiceName
	}
	if left.Occurrence != right.Occurrence {
		return left.Occurrence < right.Occurrence
	}
	return fieldRank(left.Field) < fieldRank(right.Field)
}

func kindRank(kind Kind) int {
	switch kind {
	case KindAdded:
		return 0
	case KindRemoved:
		return 1
	case KindChanged:
		return 2
	default:
		return 3
	}
}

func fieldRank(field Field) int {
	switch field {
	case FieldStatus:
		return 0
	case FieldErrorType:
		return 1
	case FieldDuration:
		return 2
	default:
		return 2
	}
}
