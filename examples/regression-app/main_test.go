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
	wantNames := []string{"checkout.handle", "payment.charge", "inventory.reserve"}
	for i, want := range wantNames {
		if specs[i].Name != want {
			t.Fatalf("span[%d].Name = %q, want %q", i, specs[i].Name, want)
		}
	}
	if specs[0].StatusCode != 2 {
		t.Fatalf("checkout status = %d, want ERROR (2)", specs[0].StatusCode)
	}
	if specs[1].Duration != 70_000_000 {
		t.Fatalf("payment duration = %d, want 70ms", specs[1].Duration)
	}
}
