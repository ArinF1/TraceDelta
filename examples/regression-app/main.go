package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	traceID    = "11111111111111111111111111111111"
	rootSpanID = "1111111111111111"
	startNano  = uint64(1_700_000_000_000_000_000)
)

type spanSpec struct {
	Name       string
	SpanID     string
	ParentID   string
	StartAfter uint64
	Duration   uint64
	Kind       int
	StatusCode int
	ErrorType  string
}

type exportRequest struct {
	ResourceSpans []resourceSpans `json:"resourceSpans"`
}

type resourceSpans struct {
	Resource   resource     `json:"resource"`
	ScopeSpans []scopeSpans `json:"scopeSpans"`
}

type resource struct {
	Attributes []attribute `json:"attributes"`
}

type scopeSpans struct {
	Scope scope      `json:"scope"`
	Spans []wireSpan `json:"spans"`
}

type scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type wireSpan struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId,omitempty"`
	Name              string      `json:"name"`
	Kind              int         `json:"kind"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []attribute `json:"attributes,omitempty"`
	Status            status      `json:"status"`
}

type status struct {
	Code int `json:"code"`
}

type attribute struct {
	Key   string         `json:"key"`
	Value attributeValue `json:"value"`
}

type attributeValue struct {
	StringValue string `json:"stringValue"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "example regression app: %v\n", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("regression-app", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "", "new OTLP JSON output file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if *output == "" {
		return errors.New("--output is required")
	}

	data, err := renderScenario(exampleScenario())
	if err != nil {
		return err
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write output: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}

func renderScenario(specs []spanSpec) ([]byte, error) {
	spans := make([]wireSpan, 0, len(specs))
	for _, spec := range specs {
		attributes := []attribute(nil)
		if spec.ErrorType != "" {
			attributes = append(attributes, attribute{
				Key: "error.type",
				Value: attributeValue{
					StringValue: spec.ErrorType,
				},
			})
		}
		spans = append(spans, wireSpan{
			TraceID:           traceID,
			SpanID:            spec.SpanID,
			ParentSpanID:      spec.ParentID,
			Name:              spec.Name,
			Kind:              spec.Kind,
			StartTimeUnixNano: fmt.Sprint(startNano + spec.StartAfter),
			EndTimeUnixNano:   fmt.Sprint(startNano + spec.StartAfter + spec.Duration),
			Attributes:        attributes,
			Status:            status{Code: spec.StatusCode},
		})
	}

	request := exportRequest{ResourceSpans: []resourceSpans{{
		Resource: resource{Attributes: []attribute{{
			Key:   "service.name",
			Value: attributeValue{StringValue: "example-checkout"},
		}}},
		ScopeSpans: []scopeSpans{{
			Scope: scope{Name: "tracedelta.example", Version: "0.1.0"},
			Spans: spans,
		}},
	}}}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("encode OTLP JSON: %w", err)
	}
	return output.Bytes(), nil
}
