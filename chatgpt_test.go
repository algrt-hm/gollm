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

	output := ChatGPTWrapper("Mock prompt", true, false, false)
	if !strings.Contains(output, "# ChatGPT") {
		t.Errorf("Expected ChatGPT header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
}

func TestChatGPTWrapperQuietMode(t *testing.T) {
	output := ChatGPTWrapper("Mock prompt", true, false, true)
	if strings.Contains(output, "# ChatGPT") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if output == "" {
		t.Error("Expected non-empty response in quiet mode")
	}
}
