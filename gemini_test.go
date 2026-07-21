package main

import (
	"strings"
	"testing"
)

func TestGeminiWrapper(t *testing.T) {
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false

	output, err := GeminiWrapper("Mock prompt", true, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "# Gemini") {
		t.Errorf("Expected Gemini header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
}

func TestGeminiWrapperQuietMode(t *testing.T) {
	output, err := GeminiWrapper("Mock prompt", true, false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if strings.Contains(output, "# Gemini") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if output == "" {
		t.Error("Expected non-empty response in quiet mode")
	}
}
