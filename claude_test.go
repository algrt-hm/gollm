package main

import (
	"strings"
	"testing"
)

func TestClaudeMock(t *testing.T) {
	mock := ClaudeGenChatCompletionMock()
	if mock.Content == "" {
		t.Error("Expected non-empty mock content")
	}
	if mock.Model != DefaultModels.Claude {
		t.Errorf("Expected model %s, got %s", DefaultModels.Claude, mock.Model)
	}
	if mock.StopReason == "" {
		t.Error("Expected non-empty stop reason")
	}
}

func TestClaudeWrapper(t *testing.T) {
	if testingSuppressOutput {
		t.Cleanup(func() { RenderWithGlamourPtr = RenderWithGlamour })
		RenderWithGlamourPtr = func(s string) {}
	}
	quietMode = false

	output, err := ClaudeWrapper("Mock prompt", true, false, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(output, "# Claude") {
		t.Errorf("Expected Claude header in output, got: %s", output)
	}
	if !strings.Contains(output, "Model:") {
		t.Errorf("Expected 'Model:' status line in output, got: %s", output)
	}
	if !strings.Contains(output, "mocked Claude response") {
		t.Errorf("Expected mock content in output, got: %s", output)
	}
}

func TestClaudeWrapperQuietMode(t *testing.T) {
	output, err := ClaudeWrapper("Mock prompt", true, false, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if strings.Contains(output, "# Claude") {
		t.Errorf("Quiet mode should not contain header, got: %s", output)
	}
	if !strings.Contains(output, "mocked Claude response") {
		t.Errorf("Expected mock content in quiet mode, got: %s", output)
	}
}
