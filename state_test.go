package main

import (
	"os"
	"testing"
)

func TestState(t *testing.T) {
	tmpFile := "test_state.json"
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Errorf("Error removing tmp file: %v", err)
		}
	}()

	state, err := LoadState(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading new state, got %v", err)
	}

	if !state.ShouldNotify("pkg1", "1.0") {
		t.Error("Expected ShouldNotify to be true for new package")
	}

	state.MarkNotified("pkg1", "1.0")
	if state.ShouldNotify("pkg1", "1.0") {
		t.Errorf(
			"Expected ShouldNotify to be false for already notified version",
		)
	}

	if !state.ShouldNotify("pkg1", "1.1") {
		t.Error("Expected ShouldNotify to be true for new version")
	}

	if err := state.Save(tmpFile); err != nil {
		t.Fatalf("Expected no error saving state, got %v", err)
	}

	state2, err := LoadState(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading saved state, got %v", err)
	}

	if state2.ShouldNotify("pkg1", "1.0") {
		t.Error("Expected ShouldNotify to be false for version saved in state")
	}
}
