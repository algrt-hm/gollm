package main

import (
	"strings"
	"testing"
)

func TestDeepseekWrapper(t *testing.T) {
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false

	output, err := DeepseekWrapper("Mock prompt", true, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "# DeepSeek") {
		t.Errorf("Expected Deepseek header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
	if !strings.Contains(output, "mocked DeepSeek response") {
		t.Errorf("Expected mock content in output, got: %s", output)
	}
}

func TestDeepseekWrapperQuietMode(t *testing.T) {
	output, err := DeepseekWrapper("Mock prompt", true, false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if strings.Contains(output, "# DeepSeek") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if !strings.Contains(output, "mocked DeepSeek response") {
		t.Errorf("Expected mock content in quiet mode, got: %s", output)
	}
}
