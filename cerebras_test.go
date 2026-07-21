package main

import (
	"strings"
	"testing"
)

func TestCerebrasWrapper(t *testing.T) {
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false

	output, err := CerebrasWrapper("Mock prompt", true, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "# Cerebras") {
		t.Errorf("Expected Cerebras header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
	if !strings.Contains(output, "mocked Cerebras response") {
		t.Errorf("Expected mock content in output, got: %s", output)
	}
}

func TestCerebrasWrapperQuietMode(t *testing.T) {
	output, err := CerebrasWrapper("Mock prompt", true, false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if strings.Contains(output, "# Cerebras") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if !strings.Contains(output, "mocked Cerebras response") {
		t.Errorf("Expected mock content in quiet mode, got: %s", output)
	}
}
