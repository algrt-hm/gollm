package main

import (
	"strings"
	"testing"
)

func TestChatGPTWrapper(t *testing.T) {
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false

	output, err := ChatGPTWrapper("Mock prompt", true, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "# ChatGPT") {
		t.Errorf("Expected ChatGPT header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
}

func TestChatGPTWrapperQuietMode(t *testing.T) {
	output, err := ChatGPTWrapper("Mock prompt", true, false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if strings.Contains(output, "# ChatGPT") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if output == "" {
		t.Error("Expected non-empty response in quiet mode")
	}
}
