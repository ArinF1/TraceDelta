// Command api-tour demonstrates every exported function in pkg/tracedelta.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/ArinF1/TraceDelta/pkg/tracedelta"
)

const maxSnapshotBytes = 1 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "TraceDelta API tour: %v\n", err)
		os.Exit(2)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("api-tour", flag.ContinueOnError)
	flags.SetOutput(stdout)
	outputDir := flags.String("output-dir", "", "new directory for generated traces and reports")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if *outputDir == "" {
		return errors.New("--output-dir is required")
	}

	comparison, absoluteOutputDir, err := generateTour(*outputDir)
	if err != nil {
		return err
	}
	return writeSummary(stdout, comparison, absoluteOutputDir)
}

func generateTour(outputDir string) (tracedelta.Comparison, string, error) {
	absoluteOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return tracedelta.Comparison{}, "", fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.Mkdir(absoluteOutputDir, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return tracedelta.Comparison{}, "", fmt.Errorf("output directory %q already exists; choose a new path", absoluteOutputDir)
		}
		return tracedelta.Comparison{}, "", fmt.Errorf("create output directory: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(absoluteOutputDir)
		}
	}()

	server := httptest.NewServer(snapshotHandler())
	defer server.Close()
	client := server.Client()
	client.Timeout = 2 * time.Second

	baselineBytes, err := fetchSnapshot(client, server.URL+"/baseline.json")
	if err != nil {
		return tracedelta.Comparison{}, "", fmt.Errorf("fetch baseline snapshot: %w", err)
	}
	candidateBytes, err := fetchSnapshot(client, server.URL+"/candidate.json")
	if err != nil {
		return tracedelta.Comparison{}, "", fmt.Errorf("fetch candidate snapshot: %w", err)
	}

	baselinePath := filepath.Join(absoluteOutputDir, "baseline.json")
	candidatePath := filepath.Join(absoluteOutputDir, "candidate.json")
	if err := writeNewFile(baselinePath, baselineBytes); err != nil {
		return tracedelta.Comparison{}, "", err
	}
	if err := writeNewFile(candidatePath, candidateBytes); err != nil {
		return tracedelta.Comparison{}, "", err
	}

	// DefaultOptions supplies TraceDelta's documented 20% and 10ms latency
	// thresholds. These two fields demonstrate optional normalization and
	// organization-specific redaction without changing the public defaults.
	options := tracedelta.DefaultOptions()
	options.Normalization.DurationBucket = time.Millisecond
	options.RedactedAttributeKeys = []string{"example.internal.note"}

	// Compare accepts any io.Reader, which is useful for in-memory or streamed
	// snapshots.
	streamComparison, err := tracedelta.Compare(
		bytes.NewReader(baselineBytes),
		bytes.NewReader(candidateBytes),
		options,
	)
	if err != nil {
		return tracedelta.Comparison{}, "", fmt.Errorf("compare in-memory snapshots: %w", err)
	}

	// CompareFiles is the convenience function for local artifacts.
	fileComparison, err := tracedelta.CompareFiles(baselinePath, candidatePath, options)
	if err != nil {
		return tracedelta.Comparison{}, "", fmt.Errorf("compare snapshot files: %w", err)
	}
	if !reflect.DeepEqual(streamComparison, fileComparison) {
		return tracedelta.Comparison{}, "", errors.New("reader and file comparisons produced different results")
	}

	labels := [2]string{"baseline.json", "candidate.json"}
	reports := []struct {
		name  string
		write func(io.Writer, tracedelta.Comparison, string, string) error
	}{
		{name: "report.txt", write: tracedelta.WriteText},
		{name: "report.json", write: tracedelta.WriteJSON},
		{name: "report.html", write: tracedelta.WriteHTML},
	}
	for _, report := range reports {
		path := filepath.Join(absoluteOutputDir, report.name)
		if err := writeReport(path, func(w io.Writer) error {
			return report.write(w, fileComparison, labels[0], labels[1])
		}); err != nil {
			return tracedelta.Comparison{}, "", err
		}
	}

	complete = true
	return fileComparison, absoluteOutputDir, nil
}

func fetchSnapshot(client *http.Client, url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", response.Status)
	}

	contents, err := io.ReadAll(io.LimitReader(response.Body, maxSnapshotBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(contents) > maxSnapshotBytes {
		return nil, fmt.Errorf("snapshot exceeds %d-byte limit", maxSnapshotBytes)
	}
	return contents, nil
}

func writeNewFile(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	written, writeErr := file.Write(contents)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write %q: %w", path, writeErr)
	}
	if written != len(contents) {
		return fmt.Errorf("write %q: %w", path, io.ErrShortWrite)
	}
	if closeErr != nil {
		return fmt.Errorf("close %q: %w", path, closeErr)
	}
	return nil
}

func writeReport(path string, render func(io.Writer) error) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create report %q: %w", path, err)
	}
	if err := render(file); err != nil {
		_ = file.Close()
		return fmt.Errorf("render report %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close report %q: %w", path, err)
	}
	return nil
}

func writeSummary(w io.Writer, comparison tracedelta.Comparison, outputDir string) error {
	var summary bytes.Buffer
	fmt.Fprintln(&summary, "TraceDelta public Go API tour completed.")
	fmt.Fprintf(&summary, "Artifacts: %s\n", outputDir)
	fmt.Fprintln(&summary, "DefaultOptions, Compare, CompareFiles, WriteText, WriteJSON, and WriteHTML all ran successfully.")
	fmt.Fprintf(&summary, "Comparison.HasDifferences(): %t\n", comparison.HasDifferences())
	fmt.Fprintf(&summary, "Summary: %d added, %d removed, %d changed spans\n", comparison.AddedSpans, comparison.RemovedSpans, comparison.ChangedSpans)
	fmt.Fprintln(&summary, "Typed changes:")
	for _, change := range comparison.Changes {
		writeChange(&summary, change)
	}

	written, err := w.Write(summary.Bytes())
	if err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	if written != summary.Len() {
		return fmt.Errorf("write summary: %w", io.ErrShortWrite)
	}
	return nil
}

func writeChange(w io.Writer, change tracedelta.Change) {
	field := string(change.Field)
	if field == "" {
		field = "span"
	}
	fmt.Fprintf(w, "  %-7s %-10s %s\n", change.Kind, field, change.SpanName)
}
