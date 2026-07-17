package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCompareExitCodes(t *testing.T) {
	baseline := fixturePath(t, "baseline.json")
	candidate := fixturePath(t, "candidate.json")

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
		wantError  string
	}{
		{
			name:       "differences",
			args:       []string{"compare", "--baseline", baseline, "--candidate", candidate},
			wantCode:   exitDifferences,
			wantOutput: "Result: behavioral differences detected",
		},
		{
			name:       "no differences",
			args:       []string{"compare", "--baseline", baseline, "--candidate", baseline},
			wantCode:   exitNoDifferences,
			wantOutput: "Result: no behavioral differences detected",
		},
		{
			name:      "missing required flag",
			args:      []string{"compare", "--candidate", candidate},
			wantCode:  exitUsageError,
			wantError: "--baseline is required",
		},
		{
			name:      "unsupported format",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--format", "json"},
			wantCode:  exitUsageError,
			wantError: "supports only text",
		},
		{
			name:      "planned output flag",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--output", "report.txt"},
			wantCode:  exitUsageError,
			wantError: "--output is not implemented",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder
			gotCode := run(test.args, &stdout, &stderr)
			if gotCode != test.wantCode {
				t.Errorf("run() code = %d, want %d; stderr=%q", gotCode, test.wantCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantOutput) {
				t.Errorf("run() stdout = %q, want it to contain %q", stdout.String(), test.wantOutput)
			}
			if !strings.Contains(stderr.String(), test.wantError) {
				t.Errorf("run() stderr = %q, want it to contain %q", stderr.String(), test.wantError)
			}
		})
	}
}

func TestRunCompareReportsMalformedInput(t *testing.T) {
	directory := t.TempDir()
	malformed := filepath.Join(directory, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"resourceSpans": [`), 0o600); err != nil {
		t.Fatalf("write malformed fixture: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := run([]string{
		"compare",
		"--baseline", malformed,
		"--candidate", fixturePath(t, "candidate.json"),
	}, &stdout, &stderr)
	if code != exitUsageError {
		t.Errorf("run() code = %d, want %d", code, exitUsageError)
	}
	if !strings.Contains(stderr.String(), "parse baseline traces: decode OTLP JSON") {
		t.Fatalf("run() stderr = %q, want helpful baseline parse error", stderr.String())
	}
}

func TestParseDurationThreshold(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{input: "20%", want: 0.20},
		{input: "0%", want: 0},
		{input: " 12.5% ", want: 0.125},
		{input: "20", wantErr: true},
		{input: "-1%", wantErr: true},
		{input: "NaN%", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := parseDurationThreshold(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseDurationThreshold() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Errorf("parseDurationThreshold() = %v, want %v", got, test.want)
			}
		})
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	return path
}
