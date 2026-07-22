package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const baselineStartNano = uint64(1_700_000_000_000_000_000)

type snapshotVersion string

const (
	baselineVersion  snapshotVersion = "baseline"
	candidateVersion snapshotVersion = "candidate"
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
	Note       string
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

func snapshotHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /baseline.json", serveSnapshot(baselineVersion))
	mux.HandleFunc("GET /candidate.json", serveSnapshot(candidateVersion))
	return mux
}

func serveSnapshot(version snapshotVersion) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		contents, err := renderSnapshot(version)
		if err != nil {
			http.Error(w, "could not render synthetic snapshot", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(contents)
	}
}

func renderSnapshot(version snapshotVersion) ([]byte, error) {
	traceID, rootSpanID, startNano, specs, err := scenario(version)
	if err != nil {
		return nil, err
	}
	spans := make([]wireSpan, 0, len(specs))
	for _, spec := range specs {
		attributes := []attribute{
			{Key: "example.internal.note", Value: attributeValue{StringValue: spec.Note}},
		}
		if spec.Name == "checkout.handle" {
			attributes = append(attributes, attribute{Key: "http.route", Value: attributeValue{StringValue: "/checkout"}})
		}
		if spec.ErrorType != "" {
			attributes = append(attributes, attribute{Key: "error.type", Value: attributeValue{StringValue: spec.ErrorType}})
		}
		parentID := spec.ParentID
		if parentID == "root" {
			parentID = rootSpanID
		}
		spans = append(spans, wireSpan{
			TraceID:           traceID,
			SpanID:            spec.SpanID,
			ParentSpanID:      parentID,
			Name:              spec.Name,
			Kind:              spec.Kind,
			StartTimeUnixNano: fmt.Sprint(startNano + spec.StartAfter),
			EndTimeUnixNano:   fmt.Sprint(startNano + spec.StartAfter + spec.Duration),
			Attributes:        attributes,
			Status:            status{Code: spec.StatusCode},
		})
	}

	request := exportRequest{ResourceSpans: []resourceSpans{{
		Resource: resource{Attributes: []attribute{
			{Key: "service.name", Value: attributeValue{StringValue: "api-tour"}},
			{Key: "user.email", Value: attributeValue{StringValue: string(version) + "@example.invalid"}},
		}},
		ScopeSpans: []scopeSpans{{
			Scope: scope{Name: "tracedelta.api-tour", Version: "1.0.0"},
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

func scenario(version snapshotVersion) (traceID, rootSpanID string, startNano uint64, specs []spanSpec, err error) {
	switch version {
	case baselineVersion:
		return "11111111111111111111111111111111", "1111111111111111", baselineStartNano, []spanSpec{
			{Name: "checkout.handle", SpanID: "1111111111111111", Duration: 120_000_000, Kind: 2, StatusCode: 1, Note: "baseline root note"},
			{Name: "payment.charge", SpanID: "2222222222222222", ParentID: "root", StartAfter: 10_000_000, Duration: 40_000_000, Kind: 3, StatusCode: 1, Note: "baseline payment note"},
			{Name: "cache.get", SpanID: "3333333333333333", ParentID: "root", StartAfter: 55_000_000, Duration: 10_000_000, Kind: 3, StatusCode: 1, Note: "baseline cache note"},
			{Name: "validation.check", SpanID: "4444444444444444", ParentID: "root", StartAfter: 70_000_000, Duration: 5_000_000, Kind: 1, StatusCode: 2, ErrorType: "example.ValidationError", Note: "baseline validation note"},
		}, nil
	case candidateVersion:
		return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaa", baselineStartNano + 900_000_000_000, []spanSpec{
			{Name: "checkout.handle", SpanID: "aaaaaaaaaaaaaaaa", Duration: 120_000_000, Kind: 2, StatusCode: 2, Note: "candidate root note"},
			{Name: "payment.charge", SpanID: "bbbbbbbbbbbbbbbb", ParentID: "root", StartAfter: 10_000_000, Duration: 70_000_000, Kind: 3, StatusCode: 1, Note: "candidate payment note"},
			{Name: "inventory.reserve", SpanID: "cccccccccccccccc", ParentID: "root", StartAfter: 55_000_000, Duration: 10_000_000, Kind: 3, StatusCode: 1, Note: "candidate inventory note"},
			{Name: "validation.check", SpanID: "dddddddddddddddd", ParentID: "root", StartAfter: 85_000_000, Duration: 5_000_000, Kind: 1, StatusCode: 2, ErrorType: "example.PolicyError", Note: "candidate validation note"},
		}, nil
	default:
		return "", "", 0, nil, fmt.Errorf("unsupported snapshot version %q", version)
	}
}
