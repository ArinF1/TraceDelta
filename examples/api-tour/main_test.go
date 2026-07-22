package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotsAreDeterministicValidOTLPJSON(t *testing.T) {
	for _, version := range []snapshotVersion{baselineVersion, candidateVersion} {
		first, err := renderSnapshot(version)
		if err != nil {
			t.Fatalf("renderSnapshot(%q) error = %v", version, err)
		}
		second, err := renderSnapshot(version)
		if err != nil {
			t.Fatalf("renderSnapshot(%q) second error = %v", version, err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("renderSnapshot(%q) is not deterministic", version)
		}
		if !json.Valid(first) {
			t.Fatalf("renderSnapshot(%q) is not valid JSON", version)
		}
	}
}

func TestSnapshotServerRoutesAndMethods(t *testing.T) {
	server := httptest.NewServer(snapshotHandler())
	defer server.Close()
	client := server.Client()
	client.Timeout = time.Second

	for _, path := range []string{"/baseline.json", "/candidate.json"} {
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		contents, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read/close %s errors = %v / %v", path, readErr, closeErr)
		}
		if response.StatusCode != http.StatusOK || !json.Valid(contents) {
			t.Fatalf("GET %s status = %d, valid JSON = %t", path, response.StatusCode, json.Valid(contents))
		}
		if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("GET %s Content-Type = %q", path, got)
		}
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/baseline.json", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST /baseline.json error = %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST /baseline.json status = %d, want 405", response.StatusCode)
	}

	response, err = client.Get(server.URL + "/missing")
	if err != nil {
		t.Fatalf("GET /missing error = %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /missing status = %d, want 404", response.StatusCode)
	}
}

func TestFetchSnapshotRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), maxSnapshotBytes+1))
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = time.Second

	_, err := fetchSnapshot(client, server.URL)
	if err == nil || !strings.Contains(err.Error(), "snapshot exceeds") {
		t.Fatalf("fetchSnapshot() error = %v, want size-limit error", err)
	}
}

func TestRunGeneratesEveryPublicAPIArtifact(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "tour")
	var stdout bytes.Buffer
	if err := run([]string{"--output-dir", outputDir}, &stdout); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	for _, name := range []string{"baseline.json", "candidate.json", "report.txt", "report.json", "report.html"} {
		contents, err := os.ReadFile(filepath.Join(outputDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", name, err)
		}
		if len(contents) == 0 {
			t.Fatalf("artifact %q is empty", name)
		}
	}

	textReport, err := os.ReadFile(filepath.Join(outputDir, "report.txt"))
	if err != nil {
		t.Fatalf("read text report: %v", err)
	}
	for _, finding := range []string{
		"ADDED    inventory.reserve",
		"REMOVED  cache.get",
		"status OK -> ERROR",
		"error.type example.ValidationError -> example.PolicyError",
		"duration 40ms -> 70ms",
	} {
		if !bytes.Contains(textReport, []byte(finding)) {
			t.Errorf("text report does not contain %q", finding)
		}
	}
	for _, redacted := range []string{"internal note", "@example.invalid"} {
		if bytes.Contains(textReport, []byte(redacted)) {
			t.Errorf("text report contains redacted value fragment %q", redacted)
		}
	}

	var jsonReport struct {
		SchemaVersion string `json:"schemaVersion"`
		Result        struct {
			Status   string `json:"status"`
			ExitCode int    `json:"exitCode"`
		} `json:"result"`
		Findings []json.RawMessage `json:"findings"`
	}
	jsonBytes, err := os.ReadFile(filepath.Join(outputDir, "report.json"))
	if err != nil {
		t.Fatalf("read JSON report: %v", err)
	}
	if err := json.Unmarshal(jsonBytes, &jsonReport); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}
	if jsonReport.SchemaVersion != "tracedelta.report/v1" || jsonReport.Result.Status != "differences" || jsonReport.Result.ExitCode != 1 || len(jsonReport.Findings) != 5 {
		t.Fatalf("JSON report summary = %#v, findings = %d", jsonReport, len(jsonReport.Findings))
	}

	htmlReport, err := os.ReadFile(filepath.Join(outputDir, "report.html"))
	if err != nil {
		t.Fatalf("read HTML report: %v", err)
	}
	if !bytes.Contains(htmlReport, []byte("<!doctype html>")) || bytes.Contains(htmlReport, []byte("<script")) {
		t.Fatal("HTML report is not the expected self-contained script-free document")
	}

	for _, api := range []string{"DefaultOptions", "Compare", "CompareFiles", "WriteText", "WriteJSON", "WriteHTML", "Comparison.HasDifferences(): true"} {
		if !strings.Contains(stdout.String(), api) {
			t.Errorf("stdout does not demonstrate %q", api)
		}
	}
	if got := strings.Count(stdout.String(), "\n  "); got != 5 {
		t.Fatalf("stdout typed change count = %d, want 5\n%s", got, stdout.String())
	}

	if err := run([]string{"--output-dir", outputDir}, io.Discard); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second run error = %v, want existing-directory refusal", err)
	}
}

func TestRenderSnapshotRejectsUnknownVersion(t *testing.T) {
	_, err := renderSnapshot("unknown")
	if err == nil || !strings.Contains(err.Error(), "unsupported snapshot version") {
		t.Fatalf("renderSnapshot(unknown) error = %v", err)
	}
}
