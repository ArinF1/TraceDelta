// Command tracedelta compares runtime behavior captured in trace exports.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ArinF1/TraceDelta/pkg/tracedelta"
)

const (
	exitNoDifferences = 0
	exitDifferences   = 1
	exitUsageError    = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, errors.New("a command is required"))
		writeRootUsage(stderr)
		return exitUsageError
	}

	switch args[0] {
	case "compare":
		return runCompare(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		writeRootUsage(stdout)
		return exitNoDifferences
	default:
		writeError(stderr, fmt.Errorf("unknown command %q", args[0]))
		writeRootUsage(stderr)
		return exitUsageError
	}
}

func runCompare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("tracedelta compare", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	baselinePath := flags.String("baseline", "", "baseline OTLP JSON trace file (required)")
	candidatePath := flags.String("candidate", "", "candidate OTLP JSON trace file (required)")
	durationThreshold := flags.String("duration-threshold", "20%", "minimum relative duration increase to report")
	durationThresholdAbsolute := flags.String("duration-threshold-absolute", "10ms", "minimum absolute duration increase to report")
	format := flags.String("format", "text", "report format: text, json, or html")
	output := flags.String("output", "", "write the report to this new file instead of stdout")
	force := flags.Bool("force", false, "replace an existing --output file")
	var redactedAttributes stringListFlag
	flags.Var(&redactedAttributes, "redact-attribute", "additional attribute key to redact; repeatable")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeCompareUsage(stdout, flags)
			return exitNoDifferences
		}
		writeError(stderr, fmt.Errorf("parse compare flags: %w", err))
		writeCompareUsage(stderr, flags)
		return exitUsageError
	}
	if flags.NArg() != 0 {
		writeError(stderr, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " ")))
		return exitUsageError
	}
	if *baselinePath == "" {
		writeError(stderr, errors.New("--baseline is required"))
		return exitUsageError
	}
	if *candidatePath == "" {
		writeError(stderr, errors.New("--candidate is required"))
		return exitUsageError
	}
	if *format != "text" && *format != "json" && *format != "html" {
		writeError(stderr, fmt.Errorf("unsupported --format %q; use text, json, or html", *format))
		return exitUsageError
	}
	if *force && *output == "" {
		writeError(stderr, errors.New("--force requires --output"))
		return exitUsageError
	}
	if err := validateOutputTarget(*output, *baselinePath, *candidatePath); err != nil {
		writeError(stderr, err)
		return exitUsageError
	}

	threshold, err := parseDurationThreshold(*durationThreshold)
	if err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	absoluteThreshold, err := parseAbsoluteDurationThreshold(*durationThresholdAbsolute)
	if err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	options := tracedelta.DefaultOptions()
	options.DurationThreshold = threshold
	options.DurationThresholdAbsolute = absoluteThreshold
	options.RedactedAttributeKeys = append([]string(nil), redactedAttributes...)
	comparison, err := tracedelta.CompareFiles(*baselinePath, *candidatePath, options)
	if err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	var rendered bytes.Buffer
	var reportError error
	switch *format {
	case "text":
		reportError = tracedelta.WriteText(&rendered, comparison, *baselinePath, *candidatePath)
	case "json":
		reportError = tracedelta.WriteJSON(&rendered, comparison, *baselinePath, *candidatePath)
	case "html":
		reportError = tracedelta.WriteHTML(&rendered, comparison, *baselinePath, *candidatePath)
	}
	if reportError != nil {
		writeError(stderr, reportError)
		return exitUsageError
	}
	if err := writeReportOutput(stdout, *output, *force, rendered.Bytes()); err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	if comparison.HasDifferences() {
		return exitDifferences
	}
	return exitNoDifferences
}

type stringListFlag []string

func (values *stringListFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *stringListFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("attribute key must not be empty")
	}
	*values = append(*values, trimmed)
	return nil
}

func validateOutputTarget(outputPath, baselinePath, candidatePath string) error {
	if outputPath == "" {
		return nil
	}
	for _, inputPath := range []string{baselinePath, candidatePath} {
		same, err := sameFileTarget(outputPath, inputPath)
		if err != nil {
			return fmt.Errorf("validate --output path: %w", err)
		}
		if same {
			return errors.New("--output must not refer to the baseline or candidate input")
		}
	}
	return nil
}

func sameFileTarget(leftPath, rightPath string) (bool, error) {
	leftAbsolute, err := filepath.Abs(leftPath)
	if err != nil {
		return false, err
	}
	rightAbsolute, err := filepath.Abs(rightPath)
	if err != nil {
		return false, err
	}
	if leftAbsolute == rightAbsolute || (runtime.GOOS == "windows" && strings.EqualFold(leftAbsolute, rightAbsolute)) {
		return true, nil
	}
	leftInfo, leftErr := os.Stat(leftAbsolute)
	rightInfo, rightErr := os.Stat(rightAbsolute)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true, nil
	}
	if leftErr != nil && !errors.Is(leftErr, os.ErrNotExist) {
		return false, leftErr
	}
	if rightErr != nil && !errors.Is(rightErr, os.ErrNotExist) {
		return false, rightErr
	}
	return false, nil
}

func writeReportOutput(stdout io.Writer, outputPath string, force bool, contents []byte) error {
	if outputPath == "" {
		written, err := stdout.Write(contents)
		if err != nil {
			return fmt.Errorf("write report to stdout: %w", err)
		}
		if written != len(contents) {
			return fmt.Errorf("write report to stdout: %w", io.ErrShortWrite)
		}
		return nil
	}

	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(outputPath, flags, 0o600)
	if err != nil {
		if !force && errors.Is(err, os.ErrExist) {
			return fmt.Errorf("refuse to overwrite existing output %q; pass --force to replace it", outputPath)
		}
		return fmt.Errorf("open output %q: %w", outputPath, err)
	}
	removeOnError := !force
	written, err := file.Write(contents)
	if err != nil || written != len(contents) {
		file.Close()
		if removeOnError {
			_ = os.Remove(outputPath)
		}
		if err == nil {
			err = io.ErrShortWrite
		}
		return fmt.Errorf("write output %q: %w", outputPath, err)
	}
	if err := file.Close(); err != nil {
		if removeOnError {
			_ = os.Remove(outputPath)
		}
		return fmt.Errorf("close output %q: %w", outputPath, err)
	}
	return nil
}

func parseAbsoluteDurationThreshold(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	duration, err := time.ParseDuration(trimmed)
	if err != nil || duration < 0 {
		return 0, fmt.Errorf("invalid --duration-threshold-absolute %q; use a non-negative duration such as 10ms", value)
	}
	return duration, nil
}

func parseDurationThreshold(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "%") {
		return 0, fmt.Errorf("invalid --duration-threshold %q; use a non-negative percentage such as 20%%", value)
	}
	percentage, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(trimmed, "%")), 64)
	if err != nil || percentage < 0 || math.IsNaN(percentage) || math.IsInf(percentage, 0) {
		return 0, fmt.Errorf("invalid --duration-threshold %q; use a non-negative percentage such as 20%%", value)
	}
	return percentage / 100, nil
}

func writeError(w io.Writer, err error) {
	fmt.Fprintf(w, "Error: %v\n", err)
}

func writeRootUsage(w io.Writer) {
	fprintln(w, "Usage: tracedelta <command> [flags]")
	fprintln(w, "")
	fprintln(w, "Commands:")
	fprintln(w, "  compare   Compare baseline and candidate OTLP JSON trace files")
}

func writeCompareUsage(w io.Writer, flags *flag.FlagSet) {
	fprintln(w, "Usage: tracedelta compare --baseline FILE --candidate FILE [flags]")
	fprintln(w, "")
	fprintln(w, "Flags:")
	flags.SetOutput(w)
	flags.PrintDefaults()
}

func fprintln(w io.Writer, text string) {
	fmt.Fprintln(w, text)
}
