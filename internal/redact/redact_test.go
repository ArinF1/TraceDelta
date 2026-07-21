package redact

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ArinF1/TraceDelta/internal/model"
)

func TestSnapshotRemovesBuiltInSensitiveAttributesRecursively(t *testing.T) {
	input := testSnapshot()

	got, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	span := got.Traces[0].Spans[0]
	for _, attributes := range []model.Attributes{span.Attributes, span.Resource.Attributes, span.Scope.Attributes} {
		for key := range attributes {
			if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "email") {
				t.Fatalf("redacted key %q remains in %#v", key, attributes)
			}
		}
	}
	nested := span.Attributes["synthetic.nested"]
	if len(nested.ArrayValue) != 1 || len(nested.ArrayValue[0].KVListValue) != 1 || nested.ArrayValue[0].KVListValue[0].Key != "safe" {
		t.Fatalf("nested array/kvlist = %#v, want only safe entry", nested.ArrayValue)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	for _, sensitiveValue := range []string{"synthetic-password", "synthetic@example.invalid", "synthetic-token"} {
		if strings.Contains(string(encoded), sensitiveValue) {
			t.Fatalf("redacted snapshot contains sensitive test value %q", sensitiveValue)
		}
	}
}

func TestSnapshotAdditionalKeysOverrideSafeEvidenceAndServiceName(t *testing.T) {
	input := testSnapshot()

	got, err := Snapshot(input, Options{AdditionalKeys: []string{" HTTP.ROUTE ", "service.name"}})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	span := got.Traces[0].Spans[0]
	if _, exists := span.Attributes["http.route"]; exists {
		t.Fatal("caller-denied http.route remains")
	}
	if _, exists := span.Resource.Attributes["service.name"]; exists {
		t.Fatal("caller-denied service.name remains in resource")
	}
	if span.ServiceName != "" {
		t.Fatalf("derived service name = %q, want cleared", span.ServiceName)
	}
}

func TestSnapshotIsDeterministicAndDoesNotMutateInput(t *testing.T) {
	input := testSnapshot()
	before := marshalSnapshot(t, input)

	first, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("first Snapshot() error = %v", err)
	}
	second, err := Snapshot(input, Options{})
	if err != nil {
		t.Fatalf("second Snapshot() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("redaction is not deterministic: %#v != %#v", first, second)
	}
	if after := marshalSnapshot(t, input); after != before {
		t.Fatalf("Snapshot() mutated input: %s != %s", after, before)
	}
}

func TestSnapshotRejectsEmptyAdditionalKeyWithoutValues(t *testing.T) {
	input := testSnapshot()
	_, err := Snapshot(input, Options{AdditionalKeys: []string{""}})
	if err == nil || err.Error() != "additional redacted attribute key at index 0 must not be empty" {
		t.Fatalf("Snapshot() error = %v, want contextual key-index error", err)
	}
}

func TestBuiltInPolicyCoversCommonSensitiveKeysWithoutBlockingSafeEvidence(t *testing.T) {
	policy, err := newPolicy(nil)
	if err != nil {
		t.Fatalf("newPolicy() error = %v", err)
	}
	for _, key := range []string{
		"http.request.header.authorization",
		"http.response.header.set-cookie",
		"db.connection_string",
		"url.full",
		"user.id",
		"customer.account.id",
		"api_key",
		"privateKey",
		"accessToken",
		"error.message",
		"exception.stacktrace",
	} {
		if !policy.denied(key) {
			t.Errorf("built-in policy denied(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"http.route", "http.request.method", "rpc.service", "rpc.method", "error.type"} {
		if policy.denied(key) {
			t.Errorf("built-in policy denied(%q) = true, want fixed safe key retained", key)
		}
	}
}

func testSnapshot() model.Snapshot {
	return model.Snapshot{Traces: []model.Trace{{
		ID: "trace",
		Spans: []model.Span{{
			SpanID:      "span",
			ServiceName: "checkout-service",
			Attributes: model.Attributes{
				"http.route": {Type: model.AttributeValueString, StringValue: "/checkout"},
				"password":   {Type: model.AttributeValueString, StringValue: "synthetic-password"},
				"synthetic.nested": {
					Type: model.AttributeValueArray,
					ArrayValue: []model.AttributeValue{{
						Type: model.AttributeValueKVList,
						KVListValue: []model.AttributeKeyValue{
							{Key: "safe", Value: model.AttributeValue{Type: model.AttributeValueString, StringValue: "retained"}},
							{Key: "accessToken", Value: model.AttributeValue{Type: model.AttributeValueString, StringValue: "synthetic-token"}},
						},
					}},
				},
			},
			Resource: model.Resource{Attributes: model.Attributes{
				"service.name": {Type: model.AttributeValueString, StringValue: "checkout-service"},
				"user.email":   {Type: model.AttributeValueString, StringValue: "synthetic@example.invalid"},
			}},
			Scope: model.InstrumentationScope{Attributes: model.Attributes{
				"custom.password.value": {Type: model.AttributeValueBytes, BytesValue: []byte("synthetic-password")},
			}},
		}},
	}}}
}

func marshalSnapshot(t *testing.T, snapshot model.Snapshot) string {
	t.Helper()
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(encoded)
}
