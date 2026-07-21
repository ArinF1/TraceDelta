package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestExampleScenarioIsDeterministicOTLPJSON(t *testing.T) {
	first, err := renderScenario(exampleScenario())
	if err != nil {
		t.Fatalf("render first scenario: %v", err)
	}
	second, err := renderScenario(exampleScenario())
	if err != nil {
		t.Fatalf("render second scenario: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("same scenario produced different bytes")
	}
	if !json.Valid(first) {
		t.Fatal("scenario output is not valid JSON")
	}
}

func TestExampleScenarioBaselineContract(t *testing.T) {
	specs := exampleScenario()
	if len(specs) != 3 {
		t.Fatalf("span count = %d, want 3", len(specs))
	}
	wantNames := []string{"checkout.handle", "payment.charge", "cache.get"}
	for i, want := range wantNames {
		if specs[i].Name != want {
			t.Fatalf("span[%d].Name = %q, want %q", i, specs[i].Name, want)
		}
	}
	if specs[0].StatusCode != 1 {
		t.Fatalf("checkout status = %d, want OK (1)", specs[0].StatusCode)
	}
	if specs[1].Duration != 40_000_000 {
		t.Fatalf("payment duration = %d, want 40ms", specs[1].Duration)
	}
}
