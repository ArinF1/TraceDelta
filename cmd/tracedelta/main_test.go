package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--format", "yaml"},
			wantCode:  exitUsageError,
			wantError: "use text, json, or html",
		},
		{
			name:      "force without output",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--force"},
			wantCode:  exitUsageError,
			wantError: "--force requires --output",
		},
		{
			name:      "unknown flag",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--config", "policy.yml"},
			wantCode:  exitUsageError,
			wantError: "flag provided but not defined: -config",
		},
		{
			name:      "invalid absolute duration threshold",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--duration-threshold-absolute", "-1ms"},
			wantCode:  exitUsageError,
			wantError: "invalid --duration-threshold-absolute",
		},
		{
			name:      "empty additional redaction key",
			args:      []string{"compare", "--baseline", baseline, "--candidate", candidate, "--redact-attribute", " "},
			wantCode:  exitUsageError,
			wantError: "attribute key must not be empty",
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

func TestRunCompareOutputFilePolicy(t *testing.T) {
	baseline := fixturePath(t, "baseline.json")
	candidate := fixturePath(t, "candidate.json")
	directory := t.TempDir()

	newOutput := filepath.Join(directory, "report.json")
	var stdout strings.Builder
	var stderr strings.Builder
	code := run([]string{"compare", "--baseline", baseline, "--candidate", candidate, "--format", "json", "--output", newOutput}, &stdout, &stderr)
	if code != exitDifferences || stdout.Len() != 0 {
		t.Fatalf("run(new output) code/stdout = %d/%q, want differences and empty stdout; stderr=%q", code, stdout.String(), stderr.String())
	}
	contents, err := os.ReadFile(newOutput)
	if err != nil {
		t.Fatalf("ReadFile(new output) error = %v", err)
	}
	if !strings.Contains(string(contents), `"schemaVersion": "tracedelta.report/v1"`) {
		t.Fatalf("new output = %q, want JSON report", contents)
	}

	existingOutput := filepath.Join(directory, "existing.html")
	if err := os.WriteFile(existingOutput, []byte("sentinel"), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"compare", "--baseline", baseline, "--candidate", candidate, "--format", "html", "--output", existingOutput}, &stdout, &stderr)
	if code != exitUsageError || !strings.Contains(stderr.String(), "refuse to overwrite existing output") {
		t.Fatalf("run(existing output) code/stderr = %d/%q", code, stderr.String())
	}
	contents, err = os.ReadFile(existingOutput)
	if err != nil || string(contents) != "sentinel" {
		t.Fatalf("existing output changed without force: contents=%q err=%v", contents, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"compare", "--baseline", baseline, "--candidate", candidate, "--format", "html", "--output", existingOutput, "--force"}, &stdout, &stderr)
	if code != exitDifferences || stdout.Len() != 0 {
		t.Fatalf("run(force output) code/stdout = %d/%q; stderr=%q", code, stdout.String(), stderr.String())
	}
	contents, err = os.ReadFile(existingOutput)
	if err != nil || !strings.HasPrefix(string(contents), "<!doctype html>") {
		t.Fatalf("forced output = %q err=%v, want HTML report", contents, err)
	}

	baselineBefore, err := os.ReadFile(baseline)
	if err != nil {
		t.Fatalf("read baseline before self-target check: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"compare", "--baseline", baseline, "--candidate", candidate, "--output", baseline, "--force"}, &stdout, &stderr)
	if code != exitUsageError || !strings.Contains(stderr.String(), "must not refer to the baseline or candidate") {
		t.Fatalf("run(input output target) code/stderr = %d/%q", code, stderr.String())
	}
	baselineAfter, err := os.ReadFile(baseline)
	if err != nil || string(baselineAfter) != string(baselineBefore) {
		t.Fatalf("baseline changed by rejected output target: err=%v", err)
	}

	failedOutput := filepath.Join(directory, "parse-failure.txt")
	malformed := filepath.Join(directory, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"resourceSpans":[`), 0o600); err != nil {
		t.Fatalf("write malformed input: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"compare", "--baseline", malformed, "--candidate", candidate, "--output", failedOutput}, &stdout, &stderr)
	if code != exitUsageError {
		t.Fatalf("run(parse failure output) code = %d", code)
	}
	if _, err := os.Stat(failedOutput); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("parse failure created output: err=%v", err)
	}
}

func TestRunCompareRepeatableRedactionKeys(t *testing.T) {
	directory := t.TempDir()
	baselinePath := filepath.Join(directory, "baseline.json")
	candidatePath := filepath.Join(directory, "candidate.json")
	baseline := `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"checkout"}}]},"scopeSpans":[{"spans":[{"traceId":"11111111111111111111111111111111","spanId":"1111111111111111","name":"handle","kind":2,"startTimeUnixNano":"1","endTimeUnixNano":"2","attributes":[{"key":"http.route","value":{"stringValue":"/baseline"}}]}]}]}]}`
	candidate := strings.Replace(baseline, "/baseline", "/candidate", 1)
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(candidate), 0o600); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := run([]string{"compare", "--baseline", baselinePath, "--candidate", candidatePath}, &stdout, &stderr)
	if code != exitDifferences {
		t.Fatalf("run(default matching) code = %d; stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"compare", "--baseline", baselinePath, "--candidate", candidatePath,
		"--redact-attribute", " HTTP.ROUTE ",
		"--redact-attribute", "error.type",
	}, &stdout, &stderr)
	if code != exitNoDifferences || !strings.Contains(stdout.String(), "Result: no behavioral differences detected") {
		t.Fatalf("run(redacted matching) code/stdout/stderr = %d/%q/%q", code, stdout.String(), stderr.String())
	}
}

type shortWriter struct{}

func (shortWriter) Write(contents []byte) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	return len(contents) - 1, nil
}

func TestWriteReportOutputDetectsShortStdoutWrite(t *testing.T) {
	err := writeReportOutput(shortWriter{}, "", false, []byte("report"))
	if err == nil || !strings.Contains(err.Error(), "short write") {
		t.Fatalf("writeReportOutput() error = %v, want short write", err)
	}
}

func TestValidateOutputTargetRejectsFilesystemAliasOfInput(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.json")
	alias := filepath.Join(directory, "alias.json")
	if err := os.WriteFile(input, []byte("synthetic"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.Link(input, alias); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}
	err := validateOutputTarget(alias, input, filepath.Join(directory, "other.json"))
	if err == nil || !strings.Contains(err.Error(), "must not refer to the baseline or candidate") {
		t.Fatalf("validateOutputTarget() error = %v, want alias rejection", err)
	}
}

func TestRunCompareHTMLUsesSameResultPolicy(t *testing.T) {
	baseline := fixturePath(t, "baseline.json")
	candidate := fixturePath(t, "candidate.json")
	tests := []struct {
		name      string
		candidate string
		wantCode  int
		wantText  string
	}{
		{name: "differences", candidate: candidate, wantCode: exitDifferences, wantText: "Behavioral differences detected"},
		{name: "no differences", candidate: baseline, wantCode: exitNoDifferences, wantText: "No behavioral differences"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder
			code := run([]string{"compare", "--baseline", baseline, "--candidate", test.candidate, "--format", "html"}, &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("run() code = %d, want %d; stderr=%q", code, test.wantCode, stderr.String())
			}
			if !strings.HasPrefix(stdout.String(), "<!doctype html>") || !strings.Contains(stdout.String(), test.wantText) {
				t.Fatalf("run() HTML output missing document/outcome: %q", stdout.String())
			}
		})
	}
}

func TestRunCompareJSONUsesSameResultPolicy(t *testing.T) {
	baseline := fixturePath(t, "baseline.json")
	candidate := fixturePath(t, "candidate.json")
	tests := []struct {
		name       string
		candidate  string
		wantCode   int
		wantStatus string
		wantCount  int
	}{
		{name: "differences", candidate: candidate, wantCode: exitDifferences, wantStatus: "differences", wantCount: 4},
		{name: "no differences", candidate: baseline, wantCode: exitNoDifferences, wantStatus: "no_differences", wantCount: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder
			code := run([]string{"compare", "--baseline", baseline, "--candidate", test.candidate, "--format", "json"}, &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("run() code = %d, want %d; stderr=%q", code, test.wantCode, stderr.String())
			}
			var report struct {
				SchemaVersion string `json:"schemaVersion"`
				Result        struct {
					Status   string `json:"status"`
					ExitCode int    `json:"exitCode"`
				} `json:"result"`
				Findings []struct {
					Kind  string `json:"kind"`
					Field string `json:"field"`
					Span  struct {
						Name string `json:"name"`
					} `json:"span"`
				} `json:"findings"`
			}
			if err := json.Unmarshal([]byte(stdout.String()), &report); err != nil {
				t.Fatalf("json.Unmarshal() error = %v; output=%q", err, stdout.String())
			}
			if report.SchemaVersion != "tracedelta.report/v1" || report.Result.Status != test.wantStatus || report.Result.ExitCode != test.wantCode || len(report.Findings) != test.wantCount {
				t.Fatalf("JSON report = %#v, want status=%q exit=%d findings=%d", report, test.wantStatus, test.wantCode, test.wantCount)
			}
			if test.wantCount > 0 {
				want := [][3]string{
					{"ADDED", "", "inventory.reserve"},
					{"REMOVED", "", "cache.get"},
					{"CHANGED", "status", "checkout.handle"},
					{"CHANGED", "duration", "payment.charge"},
				}
				for index, expected := range want {
					finding := report.Findings[index]
					if finding.Kind != expected[0] || finding.Field != expected[1] || finding.Span.Name != expected[2] {
						t.Fatalf("finding[%d] = %#v, want %v", index, finding, expected)
					}
				}
			}
		})
	}
}

func TestRunCompareRequiresBothLatencyThresholds(t *testing.T) {
	directory := t.TempDir()
	baselinePath := filepath.Join(directory, "baseline.json")
	candidatePath := filepath.Join(directory, "candidate.json")
	baseline := `{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"11111111111111111111111111111111","spanId":"1111111111111111","name":"operation","kind":1,"startTimeUnixNano":"0","endTimeUnixNano":"20000000"}]}]}]}`
	candidate := strings.Replace(baseline, `"endTimeUnixNano":"20000000"`, `"endTimeUnixNano":"25000000"`, 1)
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(candidate), 0o600); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := run([]string{"compare", "--baseline", baselinePath, "--candidate", candidatePath}, &stdout, &stderr)
	if code != exitNoDifferences {
		t.Fatalf("run(default thresholds) code = %d, stderr=%q, stdout=%q", code, stderr.String(), stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"compare", "--baseline", baselinePath, "--candidate", candidatePath, "--duration-threshold-absolute", "5ms"}, &stdout, &stderr)
	if code != exitDifferences {
		t.Fatalf("run(5ms absolute) code = %d, stderr=%q, stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "thresholds: >=20% and >=5ms") {
		t.Fatalf("run(5ms absolute) stdout = %q, want effective thresholds", stdout.String())
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

func TestParseAbsoluteDurationThreshold(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{input: "10ms", want: 10 * time.Millisecond},
		{input: "0", want: 0},
		{input: " 1.5s ", want: 1500 * time.Millisecond},
		{input: "10", wantErr: true},
		{input: "-1ns", wantErr: true},
		{input: "", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := parseAbsoluteDurationThreshold(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseAbsoluteDurationThreshold() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Errorf("parseAbsoluteDurationThreshold() = %v, want %v", got, test.want)
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
