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

	output := GeminiWrapper("Mock prompt", true, false, false)
	if !strings.Contains(output, "# Gemini") {
		t.Errorf("Expected Gemini header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
}

func TestGeminiWrapperQuietMode(t *testing.T) {
	output := GeminiWrapper("Mock prompt", true, false, true)
	if strings.Contains(output, "# Gemini") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if output == "" {
		t.Error("Expected non-empty response in quiet mode")
	}
}
