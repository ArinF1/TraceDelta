package report

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ArinF1/TraceDelta/internal/diff"
)

func TestWriteJSONMatchesVersionedGoldenSchema(t *testing.T) {
	result := diff.Result{
		AddedSpans:   1,
		RemovedSpans: 1,
		ChangedSpans: 3,
		Changes: []diff.Change{
			{Kind: diff.KindAdded, SpanName: "inventory.reserve", ServiceName: "inventory", Occurrence: 0},
			{Kind: diff.KindRemoved, SpanName: "cache.get", ServiceName: "checkout", Occurrence: 1},
			{Kind: diff.KindChanged, Field: diff.FieldStatus, SpanName: "checkout.handle", ServiceName: "checkout", Before: "OK", After: "ERROR", BeforePresent: true, AfterPresent: true},
			{Kind: diff.KindChanged, Field: diff.FieldErrorType, SpanName: "payment.charge", ServiceName: "payments", After: "synthetic.Declined", AfterPresent: true},
			{Kind: diff.KindChanged, Field: diff.FieldDuration, SpanName: "payment.charge", ServiceName: "payments", Before: "120ms", After: "245ms", BeforePresent: true, AfterPresent: true, DurationThresholdRelative: 0.20, DurationThresholdAbsolute: 10 * time.Millisecond},
		},
	}
	var output strings.Builder
	if err := WriteJSON(&output, result, Metadata{BaselinePath: "baseline.json", CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	want := `{
  "schemaVersion": "tracedelta.report/v1",
  "inputs": {
    "baseline": {
      "path": "baseline.json"
    },
    "candidate": {
      "path": "candidate.json"
    }
  },
  "summary": {
    "addedSpans": 1,
    "removedSpans": 1,
    "changedSpans": 3,
    "findings": 5
  },
  "result": {
    "status": "differences",
    "exitCode": 1
  },
  "findings": [
    {
      "kind": "ADDED",
      "field": "",
      "span": {
        "name": "inventory.reserve",
        "serviceName": "inventory",
        "occurrence": 0
      },
      "evidence": {
        "before": {
          "present": false,
          "value": ""
        },
        "after": {
          "present": false,
          "value": ""
        },
        "durationThreshold": {
          "present": false,
          "relativeRatio": 0,
          "absolute": ""
        }
      }
    },
    {
      "kind": "REMOVED",
      "field": "",
      "span": {
        "name": "cache.get",
        "serviceName": "checkout",
        "occurrence": 1
      },
      "evidence": {
        "before": {
          "present": false,
          "value": ""
        },
        "after": {
          "present": false,
          "value": ""
        },
        "durationThreshold": {
          "present": false,
          "relativeRatio": 0,
          "absolute": ""
        }
      }
    },
    {
      "kind": "CHANGED",
      "field": "status",
      "span": {
        "name": "checkout.handle",
        "serviceName": "checkout",
        "occurrence": 0
      },
      "evidence": {
        "before": {
          "present": true,
          "value": "OK"
        },
        "after": {
          "present": true,
          "value": "ERROR"
        },
        "durationThreshold": {
          "present": false,
          "relativeRatio": 0,
          "absolute": ""
        }
      }
    },
    {
      "kind": "CHANGED",
      "field": "error.type",
      "span": {
        "name": "payment.charge",
        "serviceName": "payments",
        "occurrence": 0
      },
      "evidence": {
        "before": {
          "present": false,
          "value": ""
        },
        "after": {
          "present": true,
          "value": "synthetic.Declined"
        },
        "durationThreshold": {
          "present": false,
          "relativeRatio": 0,
          "absolute": ""
        }
      }
    },
    {
      "kind": "CHANGED",
      "field": "duration",
      "span": {
        "name": "payment.charge",
        "serviceName": "payments",
        "occurrence": 0
      },
      "evidence": {
        "before": {
          "present": true,
          "value": "120ms"
        },
        "after": {
          "present": true,
          "value": "245ms"
        },
        "durationThreshold": {
          "present": true,
          "relativeRatio": 0.2,
          "absolute": "10ms"
        }
      }
    }
  ]
}
`
	if output.String() != want {
		t.Fatalf("WriteJSON() output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestWriteJSONEscapesStringsAndReportsNoDifferencesPolicy(t *testing.T) {
	result := diff.Result{Changes: []diff.Change{{
		Kind:        diff.KindAdded,
		SpanName:    "</script>\noperation",
		ServiceName: "service&name",
	}}}
	var output strings.Builder
	if err := WriteJSON(&output, result, Metadata{BaselinePath: "base<line>.json", CandidatePath: "candidate.json"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	for _, unsafe := range []string{"</script>", "base<line>", "service&name"} {
		if strings.Contains(output.String(), unsafe) {
			t.Fatalf("WriteJSON() output contains unescaped %q: %s", unsafe, output.String())
		}
	}
	var decoded jsonReport
	if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Findings[0].Span.Name != "</script>\noperation" || decoded.Inputs.Baseline.Path != "base<line>.json" {
		t.Fatalf("decoded escaped values = %#v", decoded)
	}

	output.Reset()
	if err := WriteJSON(&output, diff.Result{}, Metadata{}); err != nil {
		t.Fatalf("WriteJSON(no differences) error = %v", err)
	}
	if err := json.Unmarshal([]byte(output.String()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(no differences) error = %v", err)
	}
	if decoded.Result.Status != "no_differences" || decoded.Result.ExitCode != 0 || decoded.Findings == nil || len(decoded.Findings) != 0 {
		t.Fatalf("no-differences policy = %#v, want status/exit 0 and [] findings", decoded)
	}
}

func TestWriteJSONDoesNotWritePartialDocumentOnEncodingError(t *testing.T) {
	result := diff.Result{Changes: []diff.Change{{
		Kind:                      diff.KindChanged,
		Field:                     diff.FieldDuration,
		SpanName:                  "operation",
		DurationThresholdRelative: math.NaN(),
	}}}
	var output strings.Builder
	err := WriteJSON(&output, result, Metadata{})
	if err == nil || !strings.Contains(err.Error(), "encode JSON report") {
		t.Fatalf("WriteJSON() error = %v, want contextual encoding error", err)
	}
	if output.Len() != 0 {
		t.Fatalf("WriteJSON() wrote partial output: %q", output.String())
	}
}
