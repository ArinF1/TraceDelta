// Command tracedelta compares runtime behavior captured in trace exports.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/example/tracedelta/pkg/tracedelta"
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
	format := flags.String("format", "text", "report format; currently only text")
	output := flags.String("output", "", "planned output file path; not implemented in the current build")

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
	if *format != "text" {
		writeError(stderr, fmt.Errorf("unsupported --format %q; the current build supports only text", *format))
		return exitUsageError
	}
	if *output != "" {
		writeError(stderr, errors.New("--output is not implemented in the current build; omit it to write to stdout"))
		return exitUsageError
	}

	threshold, err := parseDurationThreshold(*durationThreshold)
	if err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	options := tracedelta.DefaultOptions()
	options.DurationThreshold = threshold
	comparison, err := tracedelta.CompareFiles(*baselinePath, *candidatePath, options)
	if err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	if err := tracedelta.WriteText(stdout, comparison, *baselinePath, *candidatePath); err != nil {
		writeError(stderr, err)
		return exitUsageError
	}
	if comparison.HasDifferences() {
		return exitDifferences
	}
	return exitNoDifferences
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
